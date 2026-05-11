package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"NFT_Minting_Platform/internal/auth"
	"NFT_Minting_Platform/internal/db"
	"NFT_Minting_Platform/internal/middleware"
	"NFT_Minting_Platform/internal/nft"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()
	defer db.Close()

	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// CORS — allow Vue frontend to call Go backend
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // required for cookies
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "NFT Minting Platform API is running",
		})
	})

	// Auth routes
	authService := auth.NewService(db.DB)
	authHandler := auth.NewHandler(authService)
	authHandler.RegisterRoutes(router)

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		protected.GET("/me", func(c *gin.Context) {
			userID := c.GetString("user_id")
			c.JSON(200, gin.H{
				"user_id": userID,
				"message": "you are authenticated",
			})
		})

		// NFT routes — protected
		nftHandler := nft.NewHandler(db.DB)
		nftHandler.RegisterRoutes(protected)
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
