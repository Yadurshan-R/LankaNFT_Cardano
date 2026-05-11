package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client handles communication with the Node.js blockchain sidecar
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// NewClient creates a new blockchain service client
func NewClient() *Client {
	return &Client{
		baseURL: os.Getenv("BLOCKCHAIN_SERVICE_URL"),
		secret:  os.Getenv("BLOCKCHAIN_SERVICE_SECRET"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateWalletResponse is the response from /api/wallet/generate
type GenerateWalletResponse struct {
	Mnemonic []string `json:"mnemonic"`
	Address  string   `json:"address"`
}

// MintNFTRequest is sent to the blockchain sidecar for minting
type MintNFTRequest struct {
	Mnemonic     []string `json:"mnemonic"`
	AssetName    string   `json:"asset_name"`
	MetadataIPFS string   `json:"metadata_ipfs"`
	ImageIPFS    string   `json:"image_ipfs"`
	Royalties    float64  `json:"royalties"`
}

// MintNFTResponse is returned after successful mint
type MintNFTResponse struct {
	TxHash    string `json:"tx_hash"`
	PolicyID  string `json:"policy_id"`
	AssetName string `json:"asset_name"`
}

// GenerateWallet calls the blockchain sidecar to generate a new wallet
// Returns mnemonic words and wallet address
func (c *Client) GenerateWallet() (*GenerateWalletResponse, error) {
	resp, err := c.post("/api/wallet/generate", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GenerateWalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode wallet response: %w", err)
	}

	return &result, nil
}

// MintNFT calls the blockchain sidecar to mint a CIP-68 NFT
func (c *Client) MintNFT(req MintNFTRequest) (*MintNFTResponse, error) {
	resp, err := c.post("/api/mint/single", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result MintNFTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode mint response: %w", err)
	}

	return &result, nil
}

// post sends a POST request to the blockchain sidecar
// Automatically adds the shared secret header
func (c *Client) post(path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = strings.NewReader("{}")
	}

	url := c.baseURL + path
	log.Printf("Calling blockchain sidecar: %s", url) // add this log

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	// Shared secret — sidecar rejects requests without this
	req.Header.Set("X-Service-Secret", c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call blockchain service: %w", err)
	}

	// Log the response body for debugging
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Blockchain sidecar error %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("blockchain service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
