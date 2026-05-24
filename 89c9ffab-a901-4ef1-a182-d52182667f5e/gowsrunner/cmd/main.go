package main

import (
	"context"
	"gowsrunner/internal/infrastructure/config"
	"gowsrunner/internal/infrastructure/database/postgres"
	packageHttp "gowsrunner/internal/infrastructure/http"
	"log"
	nethttp "net/http"
	"os"
	"os/exec"
	"sync"
)

var (
	RunningMu sync.Mutex
	Running   = map[string]*exec.Cmd{}
	UploadDir = "./uploads"
	WorkDir = "./work"
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

	defer func() {
		if dbPool != nil {
			log.Println("Closing database connection pool...")
			dbPool.Close()
		}
	}()


	mux := nethttp.NewServeMux()

	packageHttp.InitializeRoutes(ctx, dbPool, mux)

	server := &nethttp.Server{
		Addr: ":" + configuration.Server.Port,
		Handler: mux,

		// Open for later Timeouts
	}

	if err := server.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

