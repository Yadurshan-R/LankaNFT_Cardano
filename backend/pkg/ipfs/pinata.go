// ─────────────────────────────────────────────────────────────────────────────
// pkg/ipfs/pinata.go
//
// Pinata IPFS client for uploading NFT images and metadata.
//
// Authentication: API key + secret (v1 pinning API).
// Endpoints used:
//   - POST /pinning/pinFileToIPFS  — image upload
//   - POST /pinning/pinJSONToIPFS  — metadata upload
//
// Timeout: 3 minutes. Pinata free tier is slow on files over 1MB.
// Recommended image size: under 1MB for reliable uploads.
// ─────────────────────────────────────────────────────────────────────────────
package ipfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// Maximum file size we allow before rejecting at the application layer.
// Pinata accepts larger files but they cause timeouts on the free tier.
// 10MB is a generous limit that still completes reliably.
const MaxFileSizeBytes = 10 * 1024 * 1024 // 10MB

// PinataClient handles uploading files and JSON to Pinata IPFS.
// Uses API key + secret authentication (not JWT).
type PinataClient struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewPinataClient creates a new Pinata client using credentials from environment.
// Timeout is set to 3 minutes — Pinata free tier is slow on files over 1MB
// and the default 60s causes frequent timeouts on real NFT images.
func NewPinataClient() *PinataClient {
	return &PinataClient{
		apiKey:    os.Getenv("PINATA_API_KEY"),
		apiSecret: os.Getenv("PINATA_SECRET_KEY"),
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

// PinResponse is the response from Pinata after a successful pin operation.
type PinResponse struct {
	IpfsHash  string `json:"IpfsHash"`
	PinSize   int64  `json:"PinSize"`
	Timestamp string `json:"Timestamp"`
}

// UploadFile uploads a binary file (image, video, etc.) to Pinata IPFS.
// Returns the pin response containing the IPFS hash (e.g. "QmXxx...").
//
// Rejects files over MaxFileSizeBytes before attempting upload to avoid
// wasting time on requests that will timeout on Pinata's free tier.
func (p *PinataClient) UploadFile(fileData []byte, fileName string) (*PinResponse, error) {
	// Guard: reject oversized files before attempting upload
	if len(fileData) > MaxFileSizeBytes {
		return nil, fmt.Errorf(
			"file too large: %dMB. Maximum allowed size is %dMB. Please compress your image before minting",
			len(fileData)/1024/1024,
			MaxFileSizeBytes/1024/1024,
		)
	}

	// Build multipart form body with file + Pinata metadata
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Write the file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart file field: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data to multipart body: %w", err)
	}

	// Write Pinata metadata field (sets the pin name in Pinata dashboard)
	pinataMetadata := fmt.Sprintf(`{"name": "%s"}`, fileName)
	if err := writer.WriteField("pinataMetadata", pinataMetadata); err != nil {
		return nil, fmt.Errorf("failed to write pinataMetadata field: %w", err)
	}
	writer.Close()

	// Build and send the request
	req, err := http.NewRequest("POST", "https://api.pinata.cloud/pinning/pinFileToIPFS", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinata upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("pinata_api_key", p.apiKey)
	req.Header.Set("pinata_secret_api_key", p.apiSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"Pinata upload failed (file: %s, size: %.1fMB) — if this persists, try a smaller image: %w",
			fileName,
			float64(len(fileData))/1024/1024,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Pinata returned error %d: %s", resp.StatusCode, string(body))
	}

	var result PinResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Pinata response: %w", err)
	}

	return &result, nil
}

// UploadJSON uploads a JSON metadata object to Pinata IPFS.
// Used for CIP-68 reference metadata after the image is already uploaded.
// Returns the pin response containing the IPFS hash.
func (p *PinataClient) UploadJSON(metadata interface{}, name string) (*PinResponse, error) {
	// Pinata's JSON pin endpoint wraps content in a pinataContent field
	payload := map[string]interface{}{
		"pinataContent": metadata,
		"pinataMetadata": map[string]string{
			"name": name,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		"https://api.pinata.cloud/pinning/pinJSONToIPFS",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinata JSON upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pinata_api_key", p.apiKey)
	req.Header.Set("pinata_secret_api_key", p.apiSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Pinata JSON upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Pinata returned error %d for JSON upload: %s", resp.StatusCode, string(body))
	}

	var result PinResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Pinata JSON response: %w", err)
	}

	return &result, nil
}

// BuildCIP68Metadata constructs the CIP-68 compliant metadata JSON object.
// This is stored in the reference NFT (100 token) datum on-chain and
// serves as the canonical source of truth for NFT metadata.
//
// CIP-68 spec: https://cips.cardano.org/cips/cip68/
func BuildCIP68Metadata(
	name string,
	description string,
	imageIPFS string,
	royalties float64,
	attributes map[string]string,
) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"image":       imageIPFS, // ipfs://Qm... — gateway resolved by wallets
		"mediaType":   "image/png",
		"files": []map[string]interface{}{
			{
				"name":      name,
				"mediaType": "image/png",
				"src":       imageIPFS,
			},
		},
		"royalties": royalties,
		"version":   1, // CIP-68 metadata version
		"extra":     attributes,
	}
}
