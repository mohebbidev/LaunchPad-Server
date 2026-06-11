package application

import (
	"context"
	"fmt"
	"gowsrunner/internal/domain/entities"
	"gowsrunner/internal/domain/repository"
	"sync/atomic"
)

var portCounter atomic.Int32

func init() {
	portCounter.Store(3000)
}

func nextPort() int {
	return int(portCounter.Add(1))
}

type LogLine struct {
	Stream string
	Text string
}

type RunProjectUseCase struct {
	ProjectRepo repository.ProjectRepository
}


func NewRunProjectUseCase(repo repository.ProjectRepository) *RunProjectUseCase {
	return &RunProjectUseCase{ProjectRepo: repo}
}

func (uc *RunProjectUseCase) Execute(
	ctx context.Context,
	projectID string,
) (<-chan LogLine, error) {


	project, err := uc.ProjectRepo.GetByID(ctx, projectID)

	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if project.Status == entities.StatusRunning {
		return nil, fmt.Errorf("project is already running on port %d", project.Port)
	}

	port := nextPort()

	if err := uc.ProjectRepo.UpdatePortAndStatus(ctx, projectID, port, entities.StatusBuilding); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	logCh := make(chan LogLine, 64)

	go func(){
		defer close(logCh)

		send := func(stream, text string) {
			select{
			case logCh <- LogLine{Stream: stream, Text: text}:
			case <-ctx.Done():
			}
		}

		path := project.SourceLocation

		send("[stdout]", fmt.Sprintf("[runner] npm install in %s", path))

		
	}
}