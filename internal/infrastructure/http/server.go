package http

import (
	"context"
	"gowsrunner/internal/application"
	"gowsrunner/internal/infrastructure/database/postgres"
	handler "gowsrunner/internal/infrastructure/http/handlers"
	"gowsrunner/internal/infrastructure/storage"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitializeUploadHandler(db *pgxpool.Pool) *handler.UploadHandler {
	dbRepo := postgres.NewProjectRepository(db)
	storageRepo := storage.Storage{}

    // Define your paths
    uploadDir := "./uploads"
    workDir := "./projects"

	useCase := application.NewUploadProjectUseCase(
        dbRepo, 
        &storageRepo, 
        uploadDir, 
        workDir,
    )
    
	return handler.NewUplaodHandler(useCase)
}

func InitializeRoutes(ctx context.Context, db *pgxpool.Pool, mux *http.ServeMux) {
	
	uploadHandler := InitializeUploadHandler(db)

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHandler.ServeHTTP(ctx, w, r)
	})
}

