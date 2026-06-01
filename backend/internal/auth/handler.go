package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"NFT_Minting_Platform/pkg/email"
)

// Handler holds the auth service
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all auth routes on the router
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/request-otp", h.RequestOTP)
		auth.POST("/verify-otp", h.VerifyOTP)
		auth.GET("/nonce", h.GetNonce)
		auth.POST("/wallet-verify", h.WalletVerify)
		auth.POST("/logout", h.Logout)
	}
}

// GetMe godoc
// GET /api/me
// Returns current authenticated context user ID and message.
// Prepares user parameters for upcoming external wallet tracking features.
func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"message": "you are authenticated",
	})
}

// RequestOTP godoc
// POST /auth/request-otp
// Body: { "email": "user@example.com" }
// Generates a 6-digit OTP and sends it via Email
func (h *Handler) RequestOTP(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid email is required"})
		return
	}

	// Generate and store OTP
	code, err := h.service.GenerateOTP(c.Request.Context(), body.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OTP"})
		return
	}

	// Send OTP via MAIL
	if err := email.SendOTP(body.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send OTP email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent to " + body.Email,
	})
}

// VerifyOTP godoc
// POST /auth/verify-otp
// Body: { "email": "user@example.com", "code": "123456" }
// Verifies OTP, creates user if new, returns JWT in httpOnly cookie
func (h *Handler) VerifyOTP(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and 6-digit code are required"})
		return
	}

	// Verify OTP and get or create user
	userID, err := h.service.VerifyOTP(c.Request.Context(), body.Email, body.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired OTP"})
		return
	}

	// Generate JWT token
	token, err := generateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Set JWT in httpOnly cookie
	// httpOnly = true means JavaScript cannot read it (XSS protection)
	c.SetCookie(
		"auth_token",               // cookie name
		token,                      // value
		3600*24*7,                  // max age: 7 days in seconds
		"/",                        // path
		os.Getenv("COOKIE_DOMAIN"), // domain from .env
		false,                      // secure: set true in production (HTTPS)
		true,                       // httpOnly: JS cannot access this cookie
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged in successfully",
		"user_id": userID,
	})
}

// GetNonce godoc
// GET /auth/nonce?wallet=addr1...
// Returns a random challenge string for wallet signature verification
func (h *Handler) GetNonce(c *gin.Context) {
	walletAddress := c.Query("wallet")
	if walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet address is required"})
		return
	}

	// Generate a unique nonce — wallet user signs this to prove ownership
	nonce := "Sign this message to login to NFT Minting Platform: " +
		walletAddress + " " +
		time.Now().Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"nonce": nonce,
	})
}

// WalletVerify godoc
// POST /auth/wallet-verify
// Body: { "wallet_address": "addr1...", "wallet_name": "lace", "signature": "..." }
// Verifies the wallet signature and returns JWT in httpOnly cookie
func (h *Handler) WalletVerify(c *gin.Context) {
	var body struct {
		WalletAddress string `json:"wallet_address" binding:"required"`
		WalletName    string `json:"wallet_name" binding:"required"`
		Signature     string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet_address, wallet_name and signature are required"})
		return
	}

	// TODO: verify the Cardano signature cryptographically
	// For now we trust the wallet address (add full verification in Phase 2)
	// Full verification will check the signature against the nonce using
	// the wallet's public key via the cardano-addresses library

	// Create or find user by wallet address
	userID, err := h.service.UpsertExternalWalletUser(
		c.Request.Context(),
		body.WalletAddress,
		body.WalletName,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process wallet login"})
		return
	}

	// Generate JWT token
	token, err := generateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Set JWT in httpOnly cookie
	c.SetCookie(
		"auth_token",
		token,
		3600*24*7,
		"/",
		os.Getenv("COOKIE_DOMAIN"),
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "wallet logged in successfully",
		"user_id": userID,
	})
}

// Logout godoc
// POST /auth/logout
// Clears the auth cookie
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie(
		"auth_token",
		"",
		-1, // negative max age = delete immediately
		"/",
		os.Getenv("COOKIE_DOMAIN"),
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// generateJWT creates a signed JWT token containing the user ID
// Token expires in 7 days
func generateJWT(userID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
