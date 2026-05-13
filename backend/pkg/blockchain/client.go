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
			// Batch mints can take longer — 120 second timeout
			Timeout: 120 * time.Second,
		},
	}
}

// ─── Single Mint ──────────────────────────────────────────────────────────────

// MintNFTRequest is sent to the blockchain sidecar for single NFT minting
type MintNFTRequest struct {
	Mnemonic     []string `json:"mnemonic"`
	AssetName    string   `json:"asset_name"`
	MetadataIPFS string   `json:"metadata_ipfs"`
	ImageIPFS    string   `json:"image_ipfs"`
	Royalties    float64  `json:"royalties"`
}

// MintNFTResponse is returned after successful single mint
type MintNFTResponse struct {
	TxHash    string `json:"tx_hash"`
	PolicyID  string `json:"policy_id"`
	AssetName string `json:"asset_name"`
}

// MintNFT calls the blockchain sidecar to mint a single CIP-68 NFT
// Uses a one-shot Plutus policy — unique policy ID per NFT
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

// ─── Batch Mint ───────────────────────────────────────────────────────────────

// BatchMintItem represents one NFT in a batch mint request
type BatchMintItem struct {
	NFTID        string  `json:"nft_id"`
	AssetName    string  `json:"asset_name"`
	MetadataIPFS string  `json:"metadata_ipfs"`
	ImageIPFS    string  `json:"image_ipfs"`
	Royalties    float64 `json:"royalties"`
}

// BatchMintRequest is sent to the blockchain sidecar for batch minting
type BatchMintRequest struct {
	Mnemonic []string        `json:"mnemonic"`
	Items    []BatchMintItem `json:"items"`
}

// BatchMintTokenResult is one minted token in the batch response
type BatchMintTokenResult struct {
	NFTID     string `json:"nft_id"`
	AssetName string `json:"asset_name"`
	RefToken  string `json:"ref_token"`
	UserToken string `json:"user_token"`
}

// BatchMintResponse is returned after successful batch mint
type BatchMintResponse struct {
	TxHash   string                 `json:"tx_hash"`
	PolicyID string                 `json:"policy_id"`
	Minted   int                    `json:"minted"`
	Tokens   []BatchMintTokenResult `json:"tokens"`
}

// BatchMintNFTs calls the blockchain sidecar to mint all NFTs in one transaction
// Uses a native script collection policy — all NFTs share one policy ID
// This is how NMKR handles batch collections — fast, no UTxO conflicts
func (c *Client) BatchMintNFTs(req BatchMintRequest) (*BatchMintResponse, error) {
	resp, err := c.post("/api/mint/batch", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BatchMintResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode batch mint response: %w", err)
	}

	return &result, nil
}

// ─── Wallet ───────────────────────────────────────────────────────────────────

// GenerateWalletResponse is the response from /api/wallet/generate
type GenerateWalletResponse struct {
	Mnemonic []string `json:"mnemonic"`
	Address  string   `json:"address"`
}

// GenerateWallet calls the blockchain sidecar to generate a new custodial wallet
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

// ─── HTTP Helper ──────────────────────────────────────────────────────────────

// post sends a POST request to the blockchain sidecar
// Automatically adds the shared secret header for authentication
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
	log.Printf("Calling blockchain sidecar: %s", url)

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Secret", c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call blockchain service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Blockchain sidecar error %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("blockchain service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
