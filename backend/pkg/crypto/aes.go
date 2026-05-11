package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// Encrypt encrypts plaintext using AES-256-GCM
// Uses the AES_MASTER_KEY from .env
// Returns hex-encoded ciphertext
func Encrypt(plaintext string) (string, error) {
	key, err := getKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// GCM mode provides both encryption and authentication
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate a random nonce — different every time
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt: nonce is prepended to ciphertext so we can extract it on decrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts hex-encoded ciphertext using AES-256-GCM
func Decrypt(ciphertextHex string) (string, error) {
	key, err := getKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce from the beginning of ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// getKey reads and decodes the AES master key from .env
// Key must be exactly 32 bytes (64 hex chars) for AES-256
func getKey() ([]byte, error) {
	keyHex := os.Getenv("AES_MASTER_KEY")
	if keyHex == "" {
		return nil, errors.New("AES_MASTER_KEY is not set in .env")
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, errors.New("AES_MASTER_KEY is not valid hex")
	}

	if len(key) != 16 && len(key) != 32 {
		return nil, errors.New("AES_MASTER_KEY must be 16 or 32 bytes")
	}

	return key, nil
}
