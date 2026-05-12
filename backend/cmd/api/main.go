package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"NFT_Minting_Platform/internal/auth"
	"NFT_Minting_Platform/internal/batch"
	"NFT_Minting_Platform/internal/db"
	"NFT_Minting_Platform/internal/middleware"
	"NFT_Minting_Platform/internal/nft"
)

func main() {
	// Load .env file — must be first thing that runs
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to PostgreSQL
	db.Connect()
	defer db.Close()

	// Set Gin to debug mode for development
	gin.SetMode(gin.DebugMode)

	// Create router with logger + panic recovery
	router := gin.Default()

	// CORS — allow Vue frontend to call Go backend
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // required for cookies
	}))

	// Health check — no auth needed
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "NFT Minting Platform API is running",
		})
	})

	// Auth routes — no JWT needed (these are how you get the JWT)
	authService := auth.NewService(db.DB)
	authHandler := auth.NewHandler(authService)
	authHandler.RegisterRoutes(router)

	// Protected routes — JWT required for all routes below
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		// Confirm authenticated user
		protected.GET("/me", func(c *gin.Context) {
			userID := c.GetString("user_id")
			c.JSON(200, gin.H{
				"user_id": userID,
				"message": "you are authenticated",
			})
		})

		// Single NFT minting routes
		nftHandler := nft.NewHandler(db.DB)
		nftHandler.RegisterRoutes(protected)

		// Batch NFT minting routes
		batchHandler := batch.NewHandler(db.DB)
		batchHandler.RegisterRoutes(protected)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
