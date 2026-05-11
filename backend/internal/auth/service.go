package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"NFT_Minting_Platform/pkg/blockchain"
	"NFT_Minting_Platform/pkg/crypto"
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
	// Generate 6 random digits
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

	// Store new OTP in database
	_, err = s.db.Exec(ctx,
		"INSERT INTO otp_codes (email, code, expires_at) VALUES ($1, $2, $3)",
		email, code, expiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	return code, nil
}

// VerifyOTP checks if the given code is valid for the email
// Returns the user ID — creates user + wallet if first time
func (s *Service) VerifyOTP(ctx context.Context, email, code string) (string, error) {
	// Check OTP exists, is not used, and not expired
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
		// New user — create account
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (email, wallet_type)
			VALUES ($1, 'custodial')
			RETURNING id
		`, email).Scan(&userID)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}

		// Generate wallet via blockchain sidecar
		walletData, err := s.blockchainClient.GenerateWallet()
		if err != nil {
			return "", fmt.Errorf("failed to generate wallet: %w", err)
		}

		// Join mnemonic words into single string for encryption
		mnemonicStr := strings.Join(walletData.Mnemonic, " ")

		// Encrypt the mnemonic before storing — never store plaintext
		encryptedMnemonic, err := crypto.Encrypt(mnemonicStr)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt mnemonic: %w", err)
		}

		// Store encrypted mnemonic and wallet address in DB
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

// UpsertExternalWalletUser creates or finds a user by wallet address
// Used for Lace/Nami/Eternl login
func (s *Service) UpsertExternalWalletUser(ctx context.Context, walletAddress, walletName string) (string, error) {
	// Check if external wallet already exists
	var userID string
	err := s.db.QueryRow(ctx,
		"SELECT user_id FROM external_wallets WHERE wallet_address = $1",
		walletAddress,
	).Scan(&userID)

	if err != nil {
		// New wallet — create user first
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (wallet_type)
			VALUES ('external')
			RETURNING id
		`).Scan(&userID)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}

		// Store the external wallet
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
