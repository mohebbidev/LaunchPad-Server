package main

import (
	"context"
	"golaunch/internal/application"
	"golaunch/internal/infrastructure/config"
	"golaunch/internal/infrastructure/database/postgres"
	packageHttp "golaunch/internal/infrastructure/http"
	"golaunch/internal/queue"
	"log"
	nethttp "net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/docker/docker/libcontainerd/queue"
)

var (
	RunningMu sync.Mutex
	Running   = map[string]*exec.Cmd{}
	UploadDir = "./uploads"
	WorkDir   = "./work"
)

func main() {

	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	if err := os.MkdirAll(WorkDir, 0755); err != nil {
		log.Fatalf("Failed to create work directory: %v", err)
	}

	ctx := context.Background()

	configuration, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	dbDSN := postgres.BuildDSN(configuration.DB)
	dbPool, err := postgres.ConnectDB(ctx, dbDSN)

	if err != nil {
		log.Fatalf("cant connect db %v", err.Error())
	}

	postgres.RunMigrations(ctx, dbPool, "file:///home/nooberthanyall/Documents/projects/golaunch/migrations")
	defer func() {
		if dbPool != nil {
			log.Println("Closing database connection pool...")
			dbPool.Close()
		}
	}()

	mux := nethttp.NewServeMux()

	packageHttp.InitializeRoutes(ctx, dbPool, mux)

	projRunner := application.NewProjectRunner()
	workerPool := queue.NewWorkerPool(15, projRunner.Run)
	server := &nethttp.Server{
		Addr:    ":" + configuration.Server.Port,
		Handler: mux,

		// Open for later Timeouts
	}

	if err := server.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
