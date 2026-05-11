package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the global database connection pool
// pgxpool handles multiple concurrent requests safely
var DB *pgxpool.Pool

// Connect initialises the PostgreSQL connection pool
// Call this once when the server starts
func Connect() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in .env")
	}

	// Create connection pool
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Ping the database to confirm connection is alive
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("Database is not reachable:", err)
	}

	DB = pool
	log.Println("Database connected successfully")
}

// Close shuts down the connection pool gracefully
// Call this when the server stops
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}
