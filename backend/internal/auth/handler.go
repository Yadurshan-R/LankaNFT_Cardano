// ─────────────────────────────────────────────────────────────────────────────
// internal/auth/service.go
//
// # Auth Service — OTP generation, verification, and wallet user management
//
// Responsibilities:
//   - GenerateOTP: create 6-digit code, store in DB with 10min expiry
//   - VerifyOTP: validate code, create user + custodial wallet on first login
//   - UpsertExternalWalletUser: create or find user by wallet address
//   - GetUserInfo: return wallet_type + wallet_address for /api/me
//
// Two user types:
//
//	custodial — signed in via Email OTP, wallet managed by LankaNFT
//	external  — signed in via CIP-30 wallet (Nami/Eternl/Lace), user holds keys
//
// ─────────────────────────────────────────────────────────────────────────────
package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"NFT_Minting_Platform/pkg/blockchain"
	"NFT_Minting_Platform/pkg/crypto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles all auth business logic
type Service struct {
	db               *pgxpool.Pool
	blockchainClient *blockchain.Client
}

// NewService creates a new auth service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:               db,
		blockchainClient: blockchain.NewClient(),
	}
}

// GenerateOTP generates a random 6-digit code and stores it in the DB
func (s *Service) GenerateOTP(ctx context.Context, email string) (string, error) {
	code := ""
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP: %w", err)
		}
		code += n.String()
	}

	expiry := time.Now().Add(10 * time.Minute)

	// Delete any existing unused OTPs for this email
	_, err := s.db.Exec(ctx,
		"DELETE FROM otp_codes WHERE email = $1 AND used = FALSE",
		email,
	)
	if err != nil {
		return "", fmt.Errorf("failed to clear old OTPs: %w", err)
	}

	_, err = s.db.Exec(ctx,
		"INSERT INTO otp_codes (email, code, expires_at) VALUES ($1, $2, $3)",
		email, code, expiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	return code, nil
}

// VerifyOTP checks if the given code is valid for the email.
// Returns the user ID — creates user + custodial wallet if first login.
func (s *Service) VerifyOTP(ctx context.Context, email, code string) (string, error) {
	var otpID string
	err := s.db.QueryRow(ctx, `
		SELECT id FROM otp_codes
		WHERE email = $1
		AND code = $2
		AND used = FALSE
		AND expires_at > NOW()
	`, email, code).Scan(&otpID)
	if err != nil {
		return "", fmt.Errorf("invalid or expired OTP")
	}

	// Mark OTP as used so it cannot be reused
	_, err = s.db.Exec(ctx,
		"UPDATE otp_codes SET used = TRUE WHERE id = $1",
		otpID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	// Check if user already exists
	var userID string
	err = s.db.QueryRow(ctx,
		"SELECT id FROM users WHERE email = $1",
		email,
	).Scan(&userID)

	if err != nil {
		// New user — create account with custodial wallet type
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (email, wallet_type)
			VALUES ($1, 'custodial')
			RETURNING id
		`, email).Scan(&userID)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}

		// Generate custodial wallet via blockchain sidecar
		walletData, err := s.blockchainClient.GenerateWallet()
		if err != nil {
			return "", fmt.Errorf("failed to generate wallet: %w", err)
		}

		// Encrypt mnemonic before storing — never store plaintext
		mnemonicStr := strings.Join(walletData.Mnemonic, " ")
		encryptedMnemonic, err := crypto.Encrypt(mnemonicStr)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt mnemonic: %w", err)
		}

		_, err = s.db.Exec(ctx, `
			INSERT INTO custodial_wallets (user_id, wallet_address, encrypted_mnemonic)
			VALUES ($1, $2, $3)
		`, userID, walletData.Address, encryptedMnemonic)
		if err != nil {
			return "", fmt.Errorf("failed to store wallet: %w", err)
		}
	}

	return userID, nil
}

// UpsertExternalWalletUser creates or finds a user by wallet address.
// Called when a user logs in via CIP-30 (Nami/Eternl/Lace).
// External wallet users hold their own keys — LankaNFT never touches them.
func (s *Service) UpsertExternalWalletUser(ctx context.Context, walletAddress, walletName string) (string, error) {
	// Check if this external wallet is already registered
	var userID string
	err := s.db.QueryRow(ctx,
		"SELECT user_id FROM external_wallets WHERE wallet_address = $1",
		walletAddress,
	).Scan(&userID)

	if err != nil {
		// New wallet user — create user record first
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (wallet_type)
			VALUES ('external')
			RETURNING id
		`).Scan(&userID)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}

		// Store the external wallet address + which wallet app they used
		_, err = s.db.Exec(ctx, `
			INSERT INTO external_wallets (user_id, wallet_address, wallet_name)
			VALUES ($1, $2, $3)
		`, userID, walletAddress, walletName)
		if err != nil {
			return "", fmt.Errorf("failed to store wallet: %w", err)
		}
	}

	return userID, nil
}

// GetUserInfo returns the wallet_type and wallet_address for a given user.
//
// Used by GET /api/me so the frontend knows:
//   - wallet_type: 'custodial' → user signed in via email, LankaNFT holds keys
//   - wallet_type: 'external' → user signed in via CIP-30, they hold their own keys
//   - wallet_address: the on-chain address, used for Blockfrost queries
//
// This is called on every page load (via checkAuth in auth store) so the
// frontend always knows which user type it's dealing with.
func (s *Service) GetUserInfo(ctx context.Context, userID string) (walletType string, walletAddress string, err error) {
	// Get wallet type from users table
	err = s.db.QueryRow(ctx,
		"SELECT COALESCE(wallet_type, 'custodial') FROM users WHERE id = $1",
		userID,
	).Scan(&walletType)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	// Get wallet address from the appropriate table
	if walletType == "custodial" {
		s.db.QueryRow(ctx,
			"SELECT COALESCE(wallet_address, '') FROM custodial_wallets WHERE user_id = $1",
			userID,
		).Scan(&walletAddress)
	} else {
		s.db.QueryRow(ctx,
			"SELECT COALESCE(wallet_address, '') FROM external_wallets WHERE user_id = $1",
			userID,
		).Scan(&walletAddress)
	}

	return walletType, walletAddress, nil
}
