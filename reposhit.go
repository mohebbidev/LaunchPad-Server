package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"

	"github.com/golang-migrate/migrate/"
	pgxdriver "github.com/golang-migrate/migrate/database/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Import the pgx stdlib driver
)

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

// RunMigrations executes database migrations using the golang-migrate library.
// It takes a pgxpool.Pool and the source URL for migration files.
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
	driver, err := pgxdriver.WithInstance(db, pgxdriver.Config{})
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
	// Use UpTo(nil) to apply all pending migrations
	version, dirty, err := m.UpTo(nil)
	if err != nil {
		// Handle the case where there are no changes gracefully
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No database migrations to apply")
			return nil
		}
		// Log the version and hash on other errors
		currentVersion, currentHash, _ := m.Version() // Best effort to get version/hash
		slog.Error("Failed to apply migrations", "error", err, "version", currentVersion, "hash", currentHash, "dirty", dirty)
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Log success
	slog.Info(fmt.Sprintf("Database migrations applied successfully. Current version: %d, Dirty: %t", version, dirty))
	return nil
}