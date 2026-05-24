package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"gowsrunner/internal/infrastructure/config"
	"log"

	"github.com/golang-migrate/migrate/v4"
	// "github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)



var DB *pgx.Conn

func RunMigrations(db *pgxpool.Pool, sourceURL string) error {
	dbDriverName := "pgx"

	sqlDB, err := sql.Open(dbDriverName, db.Config().ConnString())

	if err != nil {
		return fmt.Errorf("failed to open *sql.DB for migrations using pgx/v5 stdlib driver: %w", err)
	}

	defer sqlDB.Close()

	driver, err := pgxdriver.WithInstance(sqlDB, &pgxdriver.Config{})

	if err != nil {
		return fmt.Errorf("failed to create migrate instance with db instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, dbDriverName, driver)

	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()

	if err != nil {

		if err == migrate.ErrNoChange {
			return nil
		}

		version, hash, _ := m.Version()

		log.Printf("Current version: %d, Hash: %s", version, hash)
		return fmt.Errorf("failed to run Up migrations: %w", err)
	}

	log.Println("Migrations got run successfully!")
	return nil
}

func BuildDSN(cfg config.DBConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)
}

func ConnectDB(ctx context.Context, connURI string) (*pgxpool.Pool, error) {

	if connURI == "" {
		log.Fatal("Env Not Found 'DATABASE_URI'")
	}

	poolCfg, err := pgxpool.ParseConfig(connURI)

	if err != nil {
		log.Fatalf("Cant COnnect to DB: %v", err)
	}

	poolCfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}





