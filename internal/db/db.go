package db

import (
	"fmt"
	"time"

	"github.com/ementau/ementau-backend/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Connect opens a connection pool to PostgreSQL using sqlx and verifies it.
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	database, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(25)
	database.SetConnMaxLifetime(5 * time.Minute)
	database.SetConnMaxIdleTime(5 * time.Minute)

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return database, nil
}
