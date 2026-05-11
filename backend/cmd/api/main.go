package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"NFT_Minting_Platform/internal/auth"
	"NFT_Minting_Platform/internal/db"
	"NFT_Minting_Platform/internal/middleware"
	resendClient "NFT_Minting_Platform/pkg/resend"
)

func main() {
	// Load .env file — must be first thing that runs
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to PostgreSQL
	db.Connect()
	defer db.Close()

	// Initialise Resend email client
	resendClient.Init()

	// Set Gin to debug mode for development
	gin.SetMode(gin.DebugMode)

	// Create router with logger + panic recovery
	router := gin.Default()

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

	// Protected routes — JWT required
	// All future routes (NFT, marketplace, wallet) go inside this group
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		// Placeholder — we will add real routes here in Day 2+
		protected.GET("/me", func(c *gin.Context) {
			userID := c.GetString("user_id")
			c.JSON(200, gin.H{
				"user_id": userID,
				"message": "you are authenticated",
			})
		})
	}

	// Get port from .env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
