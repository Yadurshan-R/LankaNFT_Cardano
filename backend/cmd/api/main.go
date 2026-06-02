package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"NFT_Minting_Platform/internal/auth"
	"NFT_Minting_Platform/internal/batch"
	"NFT_Minting_Platform/internal/db"
	"NFT_Minting_Platform/internal/listing"
	"NFT_Minting_Platform/internal/middleware"
	"NFT_Minting_Platform/internal/nft"
	"NFT_Minting_Platform/pkg/blockchain"
)

// ─── Per-user rate limiter ─────────────────────────────────────────────────────
// Each user gets a token bucket: 5 mint requests per minute
// Prevents custodial wallet draining and spam minting

type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*userLimiter)
	limitersMu sync.Mutex
)

func getLimiter(userID string) *rate.Limiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()

	ul, exists := limiters[userID]
	if !exists {
		// 5 requests per minute, burst of 3
		ul = &userLimiter{
			limiter:  rate.NewLimiter(rate.Every(time.Minute/5), 3),
			lastSeen: time.Now(),
		}
		limiters[userID] = ul
	}
	ul.lastSeen = time.Now()
	return ul.limiter
}

// cleanupLimiters removes limiters for users not seen in the last 10 minutes
func cleanupLimiters() {
	for {
		time.Sleep(10 * time.Minute)
		limitersMu.Lock()
		for id, ul := range limiters {
			if time.Since(ul.lastSeen) > 10*time.Minute {
				delete(limiters, id)
			}
		}
		limitersMu.Unlock()
	}
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Connect()
	defer db.Close()

	// Start limiter cleanup goroutine
	go cleanupLimiters()

	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "NFT Minting Platform API is running",
		})
	})

	authService := auth.NewService(db.DB)
	authHandler := auth.NewHandler(authService)
	authHandler.RegisterRoutes(router)

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())

	// Apply rate limiting as a separate middleware on the protected router
	// This wraps specific paths without re-registering them, ensuring user_id is loaded
	protected.Use(func(c *gin.Context) {
		path := c.FullPath()
		if path == "/api/nft/mint" || path == "/api/batch/mint" {
			userID := c.GetString("user_id")
			if userID != "" {
				limiter := getLimiter(userID)
				if !limiter.Allow() {
					c.JSON(http.StatusTooManyRequests, gin.H{
						"error": "Too many mint requests. Please wait a moment.",
					})
					c.Abort()
					return
				}
			}
		}
		c.Next()
	})

	{
		// Wires up the authenticated context handler from the auth package
		protected.GET("/me", authHandler.GetMe)

		nftHandler := nft.NewHandler(db.DB)
		nftHandler.RegisterRoutes(protected)
		nftHandler.RegisterPublicRoutes(router) // Certificate endpoint — no auth required

		batchHandler := batch.NewHandler(db.DB)
		batchHandler.RegisterRoutes(protected)

		listingHandler := listing.NewHandler(db.DB)
		listingHandler.RegisterRoutes(protected)

		// POST /api/submit — submit a signed tx from external wallet via backend
		protected.POST("/submit", func(c *gin.Context) {
			var body struct {
				UnsignedCbor string `json:"unsigned_cbor" binding:"required"`
				WitnessCbor  string `json:"witness_cbor" binding:"required"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unsigned_cbor and witness_cbor are required"})
				return
			}
			blockchainClient := blockchain.NewClient()
			result, err := blockchainClient.SubmitTx(blockchain.SubmitTxRequest{
				UnsignedCbor: body.UnsignedCbor,
				WitnessCbor:  body.WitnessCbor,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit transaction: " + err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"tx_hash": result.TxHash})
		})
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
