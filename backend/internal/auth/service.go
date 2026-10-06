// ─────────────────────────────────────────────────────────────────────────────
// internal/auth/handler.go
//
// # Auth HTTP Handlers — OTP login, wallet connect, session management
//
// Routes registered here:
//
//	POST /auth/request-otp    — send 6-digit code to email
//	POST /auth/verify-otp     — verify code, set JWT cookie
//	GET  /auth/nonce          — get challenge string for wallet signing
//	POST /auth/wallet-verify  — verify wallet signature, set JWT cookie
//	POST /auth/logout         — clear JWT cookie
//	GET  /api/me              — return current user info (wallet_type + address)
//
// /api/me is registered here (not in main.go) to keep auth logic in one place.
// It now returns wallet_type and wallet_address so the frontend can immediately
// determine which user type is logged in and adjust the UI accordingly.
// ─────────────────────────────────────────────────────────────────────────────
package auth

import (
	"log"
	"net/http"
	"os"
	"time"

	"NFT_Minting_Platform/pkg/email"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
//
// Returns the authenticated user's identity and wallet information.
// Called by the frontend on every page load (checkAuth in auth store).
//
// Response includes:
//
//	user_id        — UUID used throughout the platform
//	wallet_type    — 'custodial' (email login) or 'external' (CIP-30 wallet)
//	wallet_address — on-chain Cardano address for balance and asset queries
//
// The frontend uses wallet_type to decide which UI flows to show:
//
//	custodial → backend handles all signing (mnemonic-based)
//	external  → frontend signs with CIP-30, backend returns unsigned CBOR
func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	walletType, walletAddress, err := h.service.GetUserInfo(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":        userID,
		"wallet_type":    walletType,
		"wallet_address": walletAddress,
		"message":        "you are authenticated",
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

	code, err := h.service.GenerateOTP(c.Request.Context(), body.Email)
	if err != nil {
		log.Printf("[OTP] generate failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OTP"})
		return
	}

	if err := email.SendOTP(body.Email, code); err != nil {
		log.Printf("[OTP] email failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send OTP email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to " + body.Email})
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

	userID, err := h.service.VerifyOTP(c.Request.Context(), body.Email, body.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired OTP"})
		return
	}

	token, err := generateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// httpOnly = true prevents JavaScript from reading the cookie (XSS protection)
	c.SetCookie("auth_token", token, 3600*24*7, "/", os.Getenv("COOKIE_DOMAIN"), false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged in successfully", "user_id": userID})
}

// GetNonce godoc
// GET /auth/nonce?wallet=addr1...
// Returns a unique challenge string for wallet signature verification.
// The wallet user signs this string to prove they own the private key.
func (h *Handler) GetNonce(c *gin.Context) {
	walletAddress := c.Query("wallet")
	if walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet address is required"})
		return
	}

	nonce := "Sign this message to login to LankaNFT: " +
		walletAddress + " " +
		time.Now().Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{"nonce": nonce})
}

// WalletVerify godoc
// POST /auth/wallet-verify
// Body: { "wallet_address": "addr1...", "wallet_name": "nami", "signature": "..." }
//
// Verifies CIP-30 wallet ownership and returns JWT in httpOnly cookie.
// Creates a new user account on first login.
//
// TODO: verify the Cardano signature cryptographically (Phase 2).
// Currently trusts the wallet address. Full verification requires checking
// the Ed25519 signature against the nonce using the wallet's public key.
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

	userID, err := h.service.UpsertExternalWalletUser(
		c.Request.Context(),
		body.WalletAddress,
		body.WalletName,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process wallet login"})
		return
	}

	token, err := generateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.SetCookie("auth_token", token, 3600*24*7, "/", os.Getenv("COOKIE_DOMAIN"), false, true)
	c.JSON(http.StatusOK, gin.H{"message": "wallet logged in successfully", "user_id": userID})
}

// Logout godoc
// POST /auth/logout
// Clears the auth cookie — client is immediately unauthenticated
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", os.Getenv("COOKIE_DOMAIN"), false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// generateJWT creates a signed JWT token containing the user ID.
// Token expires in 7 days. Secret is loaded from JWT_SECRET env var.
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
