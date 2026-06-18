package postgres

import (
	"context"
	"golaunch/internal/domain/entities"
	"golaunch/internal/infrastructure"

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
		VALUES ($1, $2, $3, $4, $5)
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

func (repo *ProjectRepository) GetByID(ctx context.Context, id string) (*entities.Project, error) {
	query := `
		SELECT id, name, unique_key, source_type, source_location, status, created_at, updated_at
		FROM projects
		WHERE id = $1
	`
	p := &entities.Project{}
	err := repo.DB.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.UniqueKey, &p.SourceType,
		&p.SourceLocation, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, entities.MapPostgresError(err)
	}
	return p, nil
}

func (repo *ProjectRepository) UpdateStatus(ctx context.Context, id string, status entities.ProjectStatus) error {
	_, err := repo.DB.Exec(ctx,
		`UPDATE projects SET status=$1, updated_at=NOW() WHERE id=$2`,
		status, id,
	)
	return err
}

func (repo *ProjectRepository) UpdatePortAndStatus(ctx context.Context, id string, port int, status entities.ProjectStatus) error {
	_, err := repo.DB.Exec(ctx,
		`UPDATE projects SET port=$1, status=$2, updated_at=NOW() WHERE id=$3`,
		port, status, id,
	)
	return err
}
