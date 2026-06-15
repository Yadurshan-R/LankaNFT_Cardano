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
			Timeout: 180 * time.Second, // Updated to 180s for Blockfrost retries
		},
	}
}

// ─── Single Mint ──────────────────────────────────────────────────────────────

type MintNFTRequest struct {
	Mnemonic     []string `json:"mnemonic"`
	AssetName    string   `json:"asset_name"`
	MetadataIPFS string   `json:"metadata_ipfs"`
	ImageIPFS    string   `json:"image_ipfs"`
	Royalties    float64  `json:"royalties"`
	Description  string   `json:"description"`
}

type MintNFTResponse struct {
	TxHash    string `json:"tx_hash"`
	PolicyID  string `json:"policy_id"`
	AssetName string `json:"asset_name"`
}

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

type BatchMintItem struct {
	NFTID        string  `json:"nft_id"`
	AssetName    string  `json:"asset_name"`
	MetadataIPFS string  `json:"metadata_ipfs"`
	ImageIPFS    string  `json:"image_ipfs"`
	Royalties    float64 `json:"royalties"`
	Description  string  `json:"description"`
}

type BatchMintRequest struct {
	Mnemonic []string        `json:"mnemonic"`
	Items    []BatchMintItem `json:"items"`
}

type BatchMintTokenResult struct {
	NFTID     string `json:"nft_id"`
	AssetName string `json:"asset_name"`
	RefToken  string `json:"ref_token"`
	UserToken string `json:"user_token"`
}

type BatchMintResponse struct {
	TxHash   string                 `json:"tx_hash"`
	PolicyID string                 `json:"policy_id"`
	Minted   int                    `json:"minted"`
	Tokens   []BatchMintTokenResult `json:"tokens"`
}

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

// ─── Marketplace ──────────────────────────────────────────────────────────────

// ListNFTRequest is sent to the sidecar to lock an NFT at the marketplace script
type ListNFTRequest struct {
	Mnemonic        []string `json:"mnemonic"`
	NFTUnit         string   `json:"nft_unit"`
	PriceLovelace   int64    `json:"price_lovelace"`
	RoyaltyPolicyID string   `json:"royalty_policy_id"`
}

// ListNFTResponse is returned after successful listing
type ListNFTResponse struct {
	TxHash             string `json:"tx_hash"`
	ScriptUTxO         string `json:"script_utxo"`
	MarketplaceAddress string `json:"marketplace_address"`
}

// ListNFT calls the sidecar to list an NFT for sale on the marketplace
func (c *Client) ListNFT(req ListNFTRequest) (*ListNFTResponse, error) {
	resp, err := c.post("/api/marketplace/list", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ListNFTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}
	return &result, nil
}

// BuyNFTRequest is sent to the sidecar to purchase a listed NFT
type BuyNFTRequest struct {
	Mnemonic         []string `json:"mnemonic"`
	ListingUTxOHash  string   `json:"listing_utxo_hash"`
	ListingUTxOIndex string   `json:"listing_utxo_index"`
	SellerAddress    string   `json:"seller_address"`
	PriceLovelace    int64    `json:"price_lovelace"`
	NFTUnit          string   `json:"nft_unit"`
	RoyaltyPolicyID  string   `json:"royalty_policy_id"`
}

// BuyNFTResponse is returned after successful purchase
type BuyNFTResponse struct {
	TxHash string `json:"tx_hash"`
}

// BuyNFT calls the sidecar to purchase a listed NFT
func (c *Client) BuyNFT(req BuyNFTRequest) (*BuyNFTResponse, error) {
	resp, err := c.post("/api/marketplace/buy", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BuyNFTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode buy response: %w", err)
	}
	return &result, nil
}

// CancelListingRequest is sent to the sidecar to cancel a listing
type CancelListingRequest struct {
	Mnemonic         []string `json:"mnemonic"`
	ListingUTxOHash  string   `json:"listing_utxo_hash"`
	ListingUTxOIndex string   `json:"listing_utxo_index"`
	NFTUnit          string   `json:"nft_unit"`
}

// CancelListingResponse is returned after successful cancellation
type CancelListingResponse struct {
	TxHash string `json:"tx_hash"`
}

// CancelListing calls the sidecar to cancel a marketplace listing
func (c *Client) CancelListing(req CancelListingRequest) (*CancelListingResponse, error) {
	resp, err := c.post("/api/marketplace/cancel", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CancelListingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode cancel response: %w", err)
	}
	return &result, nil
}

// ─── Transfer ─────────────────────────────────────────────────────────────────

// TransferNFTRequest is sent to the sidecar to transfer an NFT to another address
type TransferNFTRequest struct {
	Mnemonic         []string `json:"mnemonic"`
	NFTUnit          string   `json:"nft_unit"`
	RecipientAddress string   `json:"recipient_address"`
}

// TransferNFTResponse is returned after successful transfer
type TransferNFTResponse struct {
	TxHash           string `json:"tx_hash"`
	NFTUnit          string `json:"nft_unit"`
	RecipientAddress string `json:"recipient_address"`
}

// TransferNFT calls the sidecar to send an NFT to another address
func (c *Client) TransferNFT(req TransferNFTRequest) (*TransferNFTResponse, error) {
	resp, err := c.post("/api/transfer", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result TransferNFTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer response: %w", err)
	}
	return &result, nil
}

// ─── Wallet ───────────────────────────────────────────────────────────────────

type GenerateWalletResponse struct {
	Mnemonic []string `json:"mnemonic"`
	Address  string   `json:"address"`
}

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

// ─── Unsigned Mint ────────────────────────────────────────────────────────────

// MintNFTUnsignedRequest is sent to the sidecar to build an unsigned mint tx.
// Used for external wallet users — wallet_address replaces mnemonic.
type MintNFTUnsignedRequest struct {
	WalletAddress string   `json:"wallet_address"`
	WalletUtxos   []string `json:"wallet_utxos"`
	AssetName     string   `json:"asset_name"`
	MetadataIPFS  string   `json:"metadata_ipfs"`
	ImageIPFS     string   `json:"image_ipfs"`
	Royalties     float64  `json:"royalties"`
	Description   string   `json:"description"`
}

// MintNFTUnsignedResponse contains the unsigned CBOR + policy info.
// Frontend signs the CBOR, submits, then calls /api/nft/confirm-mint.
type MintNFTUnsignedResponse struct {
	UnsignedCbor  string `json:"unsigned_cbor"`
	PolicyID      string `json:"policy_id"`
	AssetName     string `json:"asset_name"`
	RefTokenName  string `json:"ref_token_name"`
	UserTokenName string `json:"user_token_name"`
}

func (c *Client) MintNFTUnsigned(req MintNFTUnsignedRequest) (*MintNFTUnsignedResponse, error) {
	resp, err := c.post("/api/mint/single-unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result MintNFTUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode unsigned mint response: %w", err)
	}
	return &result, nil
}

// ─── Unsigned Batch Mint ──────────────────────────────────────────────────────

type BatchMintUnsignedRequest struct {
	WalletAddress string          `json:"wallet_address"`
	WalletUtxos   []string        `json:"wallet_utxos"`
	Items         []BatchMintItem `json:"items"`
}

type BatchMintUnsignedToken struct {
	NFTID     string `json:"nft_id"`
	AssetName string `json:"asset_name"`
	RefToken  string `json:"ref_token"`
	UserToken string `json:"user_token"`
}

type BatchMintUnsignedResponse struct {
	UnsignedCbor string                   `json:"unsigned_cbor"`
	PolicyID     string                   `json:"policy_id"`
	LockSlot     int64                    `json:"lock_slot"`
	Minted       int                      `json:"minted"`
	Tokens       []BatchMintUnsignedToken `json:"tokens"`
}

func (c *Client) BatchMintUnsigned(req BatchMintUnsignedRequest) (*BatchMintUnsignedResponse, error) {
	resp, err := c.post("/api/mint/batch-unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result BatchMintUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode batch unsigned response: %w", err)
	}
	return &result, nil
}

// ─── Unsigned List ────────────────────────────────────────────────────────────

type ListNFTUnsignedRequest struct {
	WalletAddress   string `json:"wallet_address"`
	NFTUnit         string `json:"nft_unit"`
	PriceLovelace   int64  `json:"price_lovelace"`
	RoyaltyPolicyID string `json:"royalty_policy_id"`
}

type ListNFTUnsignedResponse struct {
	UnsignedCbor string `json:"unsigned_cbor"`
}

func (c *Client) ListNFTUnsigned(req ListNFTUnsignedRequest) (*ListNFTUnsignedResponse, error) {
	resp, err := c.post("/api/marketplace/list-unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result ListNFTUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode unsigned list response: %w", err)
	}
	return &result, nil
}

// ─── Unsigned Buy ─────────────────────────────────────────────────────────────

type BuyNFTUnsignedRequest struct {
	WalletAddress    string `json:"wallet_address"`
	ListingUTxOHash  string `json:"listing_utxo_hash"`
	ListingUTxOIndex string `json:"listing_utxo_index"`
	SellerAddress    string `json:"seller_address"`
	PriceLovelace    int64  `json:"price_lovelace"`
	NFTUnit          string `json:"nft_unit"`
	RoyaltyPolicyID  string `json:"royalty_policy_id"`
}

type BuyNFTUnsignedResponse struct {
	UnsignedCbor string `json:"unsigned_cbor"`
}

func (c *Client) BuyNFTUnsigned(req BuyNFTUnsignedRequest) (*BuyNFTUnsignedResponse, error) {
	resp, err := c.post("/api/marketplace/buy-unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result BuyNFTUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode unsigned buy response: %w", err)
	}
	return &result, nil
}

// ─── Unsigned Cancel ──────────────────────────────────────────────────────────

type CancelListingUnsignedRequest struct {
	WalletAddress    string `json:"wallet_address"`
	ListingUTxOHash  string `json:"listing_utxo_hash"`
	ListingUTxOIndex string `json:"listing_utxo_index"`
	NFTUnit          string `json:"nft_unit"`
}

type CancelListingUnsignedResponse struct {
	UnsignedCbor string `json:"unsigned_cbor"`
}

func (c *Client) CancelListingUnsigned(req CancelListingUnsignedRequest) (*CancelListingUnsignedResponse, error) {
	resp, err := c.post("/api/marketplace/cancel-unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result CancelListingUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode unsigned cancel response: %w", err)
	}
	return &result, nil
}

// ─── Unsigned Transfer ────────────────────────────────────────────────────────

type TransferNFTUnsignedRequest struct {
	WalletAddress    string `json:"wallet_address"`
	NFTUnit          string `json:"nft_unit"`
	RecipientAddress string `json:"recipient_address"`
}

type TransferNFTUnsignedResponse struct {
	UnsignedCbor     string `json:"unsigned_cbor"`
	NFTUnit          string `json:"nft_unit"`
	RecipientAddress string `json:"recipient_address"`
}

func (c *Client) TransferNFTUnsigned(req TransferNFTUnsignedRequest) (*TransferNFTUnsignedResponse, error) {
	resp, err := c.post("/api/transfer/unsigned", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result TransferNFTUnsignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode unsigned transfer response: %w", err)
	}
	return &result, nil
}

// ─── Submit Signed Transaction ────────────────────────────────────────────────

// SubmitTxRequest sends a signed CBOR transaction to the sidecar for submission.
type SubmitTxRequest struct {
	UnsignedCbor string `json:"unsigned_cbor"`
	WitnessCbor  string `json:"witness_cbor"`
}

// SubmitTxResponse returns the submitted transaction hash.
type SubmitTxResponse struct {
	TxHash string `json:"tx_hash"`
}

// SubmitTx submits a signed CBOR transaction via the sidecar's Blockfrost connection.
// Used for external wallet users — wallet signs, backend submits reliably.
func (c *Client) SubmitTx(req SubmitTxRequest) (*SubmitTxResponse, error) {
	resp, err := c.post("/api/submit", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result SubmitTxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode submit response: %w", err)
	}
	return &result, nil
}

// ─── HTTP Helper ──────────────────────────────────────────────────────────────

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
