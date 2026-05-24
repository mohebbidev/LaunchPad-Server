package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowsrunner/internal/infrastructure/config"
	"time"

	"log/slog"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
	// "github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BuildDSN(cfg config.DBConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)
}

func ConnectDB(ctx context.Context, connURI string) (*pgxpool.Pool, error) {
	if connURI == "" {
		return nil, errors.New("connection URI is empty")
	}

	config, err := pgxpool.ParseConfig(connURI)
	if err != nil {
		// Use slog for structured logging
		slog.Error("Failed to parse connection URI", "error", err, "uri", connURI)
		return nil, fmt.Errorf("failed to parse connection URI: %w", err)
	}

	// Configure pool settings for optimal performance.
	// Adjust MaxConns based on your application's needs and database capacity.
	config.MaxConns = 10
	config.HealthCheckPeriod = 5 * time.Minute // Periodically check connection health

	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("Unable to create connection pool", "error", err, "uri", connURI)
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Ping the database to verify the connection is live.
	err = dbPool.Ping(ctx)
	if err != nil {
		dbPool.Close() // Close the pool if ping fails
		slog.Error("Database connection test failed", "error", err, "uri", connURI)
		return nil, fmt.Errorf("database connection test failed: %w", err)
	}

	slog.Info("Successfully connected to the database", "uri", connURI)
	return dbPool, nil
}

func RunMigrations(ctx context.Context, dbPool *pgxpool.Pool, sourceURL string) error {
	if dbPool == nil {
		return errors.New("database pool is nil")
	}
	if sourceURL == "" {
		return errors.New("migration source URL is empty")
	}

	// Get the connection string from the pool's configuration
	// This ensures we use the same connection details the pool is using.
	connString := dbPool.Config().ConnString()

	// sql.Open requires the "pgx/stdlib" driver to be imported.
	// We are opening a *separate* DB connection specifically for the migrate library.
	// This is the standard pattern for golang-migrate with pgx.
	db, err := sql.Open("pgx", connString)
	if err != nil {
		slog.Error("Failed to open DB connection for migrations", "error", err)
		return fmt.Errorf("failed to open DB connection for migrations: %w", err)
	}
	defer db.Close() // Ensure the migration DB connection is closed

	// Verify the new connection is working
	if err := db.PingContext(ctx); err != nil {
		slog.Error("Migration DB connection test failed", "error", err)
		return fmt.Errorf("migration DB connection test failed: %w", err)
	}

	// Use the pgx driver for golang-migrate
	// Pass a pointer to Config{} as required by the function signature.
	driver, err := pgxdriver.WithInstance(db, &pgxdriver.Config{}) // Changed to pass a pointer
	if err != nil {
		slog.Error("Failed to create pgx migrate driver instance", "error", err)
		return fmt.Errorf("failed to create pgx migrate driver instance: %w", err)
	}

	// Create a new migrate instance
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "pgx", driver)
	if err != nil {
		slog.Error("Failed to create new migrate instance", "error", err, "sourceURL", sourceURL)
		return fmt.Errorf("failed to create new migrate instance: %w", err)
	}

	// Apply pending migrations
	// Up() returns two values: the number of migrations applied (int) and an error.
	err = m.Up() // Assign the returned values correctly
	if err != nil {
		// Handle the case where there are no changes gracefully
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No database migrations to apply")
			return nil
		}
		// Log the version and hash on other errors
		// m.Version() returns current version and dirty status.
		currentVersion, dirty, _ := m.Version()
		slog.Error("Failed to apply migrations", "error", err, "version", currentVersion, "dirty", dirty)
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Log success
	slog.Info("Database migrations applied successfully.")
	return nil
}

// func RunMigrations(db *pgxpool.Pool, sourceURL string) error {
// 	dbDriverName := "pgx"

// 	sqlDB, err := sql.Open(dbDriverName, db.Config().ConnString())

// 	if err != nil {
// 		return fmt.Errorf("failed to open *sql.DB for migrations using pgx/v5 stdlib driver: %w", err)
// 	}

// 	defer sqlDB.Close()

// 	driver, err := pgxdriver.WithInstance(sqlDB, &pgxdriver.Config{})

// 	if err != nil {
// 		return fmt.Errorf("failed to create migrate instance with db instance: %w", err)
// 	}

// 	m, err := migrate.NewWithDatabaseInstance(sourceURL, dbDriverName, driver)

// 	if err != nil {
// 		return fmt.Errorf("failed to create migrate instance: %w", err)
// 	}

// 	err = m.Up()

// 	if err != nil {

// 		if err == migrate.ErrNoChange {
// 			return nil
// 		}

// 		version, hash, _ := m.Version()

// 		log.Printf("Current version: %d, Hash: %s", version, hash)
// 		return fmt.Errorf("failed to run Up migrations: %w", err)
// 	}

// 	log.Println("Migrations got run successfully!")
// 	return nil
// }

// func ConnectDB(ctx context.Context, connURI string) (*pgxpool.Pool, error) {

// 	if connURI == "" {
// 		log.Fatal("Env Not Found 'DATABASE_URI'")
// 	}

// 	poolCfg, err := pgxpool.ParseConfig(connURI)

// 	if err != nil {
// 		log.Fatalf("Cant COnnect to DB: %v", err)
// 	}

// 	poolCfg.MaxConns = 10

// 	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("connect db: %w", err)
// 	}

// 	if err := pool.Ping(ctx); err != nil {
// 		pool.Close()
// 		return nil, fmt.Errorf("ping db: %w", err)
// 	}

// 	return pool, nil
// }
