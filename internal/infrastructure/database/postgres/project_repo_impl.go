package postgres

import (
	"context"
	"gowsrunner/internal/domain/entities"
	"gowsrunner/internal/infrastructure"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	DB *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{
		DB: db,
	}
}

func (repo *ProjectRepository) Create(ctx context.Context, p *entities.Project) (string, error) {

	var projectID string
	query := `
		INSERT INTO projects (name, unique_key, source_location, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	

	err := infrastructure.Retry(ctx, 3, func() error {
		return repo.DB.QueryRow(
			ctx,
			query,
			// p.UserID,
			p.Name,
			p.UniqueKey,
			p.SourceLocation,
			p.Status,
			p.CreatedAt,
		).Scan(&projectID)
	})

	err = entities.MapPostgresError(err)

	if err != nil {
		return "", err
	}
	return projectID, nil
}


