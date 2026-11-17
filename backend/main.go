package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/learnthor/backend/db"
	"github.com/learnthor/backend/router"
)

func main() {
	// ── Environment ───────────────────────────────────────────────────────────
	// Load .env if present. In production the vars are injected by the platform
	// (Docker, Railway, etc.) so a missing file is not an error.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading environment variables from system")
	}

	// ── Database ─────────────────────────────────────────────────────────────
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// ── HTTP server ───────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := router.New(database)

	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
