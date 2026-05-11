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

// PinataClient handles uploading files and JSON to Pinata IPFS
type PinataClient struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewPinataClient creates a new Pinata client using keys from .env
func NewPinataClient() *PinataClient {
	return &PinataClient{
		apiKey:    os.Getenv("PINATA_API_KEY"),
		apiSecret: os.Getenv("PINATA_SECRET_KEY"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// PinResponse is the response from Pinata after a successful pin
type PinResponse struct {
	IpfsHash  string `json:"IpfsHash"`
	PinSize   int64  `json:"PinSize"`
	Timestamp string `json:"Timestamp"`
}

// UploadFile uploads a file (image) to Pinata
// Returns the IPFS hash (e.g. QmXxx...)
func (p *PinataClient) UploadFile(fileData []byte, fileName string) (*PinResponse, error) {
	// Build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Add pinata metadata
	pinataMetadata := fmt.Sprintf(`{"name": "%s"}`, fileName)
	if err := writer.WriteField("pinataMetadata", pinataMetadata); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	writer.Close()

	// Send request
	req, err := http.NewRequest("POST", "https://api.pinata.cloud/pinning/pinFileToIPFS", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("pinata_api_key", p.apiKey)
	req.Header.Set("pinata_secret_api_key", p.apiSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to Pinata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinata error %d: %s", resp.StatusCode, string(body))
	}

	var result PinResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode pinata response: %w", err)
	}

	return &result, nil
}

// UploadJSON uploads a JSON metadata object to Pinata
// Returns the IPFS hash
func (p *PinataClient) UploadJSON(metadata interface{}, name string) (*PinResponse, error) {
	// Build request body
	payload := map[string]interface{}{
		"pinataContent": metadata,
		"pinataMetadata": map[string]string{
			"name": name,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		"https://api.pinata.cloud/pinning/pinJSONToIPFS",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pinata_api_key", p.apiKey)
	req.Header.Set("pinata_secret_api_key", p.apiSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload JSON to Pinata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinata error %d: %s", resp.StatusCode, string(body))
	}

	var result PinResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode pinata response: %w", err)
	}

	return &result, nil
}

// BuildCIP68Metadata builds the CIP-68 compliant metadata JSON
// This is what gets stored in the reference NFT datum on-chain
func BuildCIP68Metadata(
	name string,
	description string,
	imageIPFS string,
	royalties float64,
	attributes map[string]string,
) map[string]interface{} {
	// CIP-68 metadata format version 1
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"image":       imageIPFS, // ipfs://Qm...
		"mediaType":   "image/png",
		"files": []map[string]interface{}{
			{
				"name":      name,
				"mediaType": "image/png",
				"src":       imageIPFS,
			},
		},
		"royalties": royalties,
		"version":   1, // CIP-68 version field
		"extra":     attributes,
	}
}
