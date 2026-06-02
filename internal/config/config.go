package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from the environment.
type Config struct {
	AppEnv     string
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBDSN      string
	JWTSecret  string
}

// Load reads configuration from a .env file (if present) and the environment.
func Load() (*Config, error) {
	// .env is optional; ignore the error if it doesn't exist.
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:     getEnv("APP_ENV", "development"),
		Port:       getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBDSN:      os.Getenv("DB_DSN"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string. It prefers DB_DSN when set,
// otherwise it is assembled from the individual DB_* variables.
func (c *Config) DSN() string {
	if c.DBDSN != "" {
		return c.DBDSN
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
