package main

import (
	"log"

	"github.com/ementau/ementau-backend/internal/config"
	"github.com/ementau/ementau-backend/internal/db"
	"github.com/ementau/ementau-backend/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	router := handlers.NewRouter(cfg, database)

	addr := ":" + cfg.Port
	log.Printf("starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
