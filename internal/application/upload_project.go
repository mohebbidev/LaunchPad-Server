package application

import (
	"context"
	"gowsrunner/internal/domain/entities"
	"gowsrunner/internal/domain/repository"
	"gowsrunner/internal/infrastructure/utils"
	"io"
	"path/filepath"
)

type UploadProjectUseCase struct {
	ProjectRepo repository.ProjectRepository
	Storage     repository.Storage
	UploadDir   string
	WorkDir     string
}

type UploadInput struct {
	// UserID   domain.UserID
	File     io.Reader
	Filename string
}

type UploadOutput struct {
	ProjectID string
	UniqueKey string
}

func NewUploadProjectUseCase(
	projectRepo repository.ProjectRepository,
	storageRepo repository.Storage) *UploadProjectUseCase {

	return &UploadProjectUseCase{
		ProjectRepo: projectRepo,
		Storage:     storageRepo,
	}
}

func (uc *UploadProjectUseCase) Execute(
	ctx context.Context, input UploadInput) (*UploadOutput, error) {

	uniqueID := utils.NewID()
	zipPath := filepath.Join(uc.UploadDir, uniqueID+".zip")
	extractPath := filepath.Join(uc.WorkDir, uniqueID)

	if err := uc.Storage.Save(zipPath, input.File); err != nil {
		return nil, err
	}

	if err := uc.Storage.Unzip(zipPath, extractPath); err != nil {
		return nil, err
	}

	project := entities.NewProject(
		input.Filename,
		uniqueID,
		"zip",
		extractPath,
	)

	projID, err := uc.ProjectRepo.Create(ctx, project)

	if err != nil {
		return nil, err
	}

	return &UploadOutput{
		ProjectID: projID,
		UniqueKey: project.UniqueKey,
	}, nil
}
