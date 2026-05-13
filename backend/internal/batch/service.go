// ─────────────────────────────────────────────────────────────────────────────
// internal/batch/service.go
//
// Batch Minting Service
//
// Handles minting multiple NFTs in a single batch job.
// Each NFT in the batch is minted sequentially — one transaction per NFT.
// This is required because each NFT needs its own one-shot policy UTxO.
//
// Flow:
//   1. Create batch job record in DB
//   2. For each item: upload image + metadata to IPFS
//   3. For each item: mint on Cardano via blockchain sidecar
//   4. Update batch job progress after each item
// ─────────────────────────────────────────────────────────────────────────────

package batch

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"NFT_Minting_Platform/pkg/blockchain"
	"NFT_Minting_Platform/pkg/crypto"
	"NFT_Minting_Platform/pkg/ipfs"
)

// Service handles batch minting business logic
type Service struct {
	db               *pgxpool.Pool
	pinata           *ipfs.PinataClient
	blockchainClient *blockchain.Client
}

// NewService creates a new batch service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:               db,
		pinata:           ipfs.NewPinataClient(),
		blockchainClient: blockchain.NewClient(),
	}
}

// BatchItem represents one NFT in a batch upload
type BatchItem struct {
	RowOrder    int
	Name        string
	Description string
	Royalties   float64
	TotalSupply int
	Attributes  map[string]string
	ImageData   []byte
	ImageName   string
}

// BatchJob represents the full batch request
type BatchJob struct {
	OwnerID string
	Privacy string
	Items   []BatchItem
}

// BatchResult is returned after creating the batch job
type BatchResult struct {
	BatchID string
	Total   int
}

// ItemResult is the result of processing one batch item
type ItemResult struct {
	RowOrder int
	NFTID    string
	TxHash   string
	PolicyID string
	Status   string
	ErrorMsg string
}

// CreateBatchJob creates a new batch job record in the DB
// Returns the batch job ID
func (s *Service) CreateBatchJob(ctx context.Context, ownerID string, total int) (string, error) {
	var batchID string
	err := s.db.QueryRow(ctx, `
        INSERT INTO batch_jobs (owner_id, status, total)
        VALUES ($1, 'pending', $2)
        RETURNING id
    `, ownerID, total).Scan(&batchID)
	if err != nil {
		return "", fmt.Errorf("failed to create batch job: %w", err)
	}
	return batchID, nil
}

// ProcessBatchItem uploads one NFT to IPFS and stores it in DB
// Returns the NFT ID and IPFS URIs needed for minting
func (s *Service) ProcessBatchItem(
	ctx context.Context,
	batchID string,
	ownerID string,
	privacy string,
	item BatchItem,
) (nftID, imageIPFS, metadataIPFS, assetName string, err error) {
	// Upload image to Pinata
	imagePin, err := s.pinata.UploadFile(item.ImageData, item.ImageName)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to upload image: %w", err)
	}
	imageIPFS = "ipfs://" + imagePin.IpfsHash

	// Build and upload CIP-68 metadata
	metadata := ipfs.BuildCIP68Metadata(
		item.Name,
		item.Description,
		imageIPFS,
		item.Royalties,
		item.Attributes,
	)
	metaPin, err := s.pinata.UploadJSON(metadata, item.Name+"_metadata")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to upload metadata: %w", err)
	}
	metadataIPFS = "ipfs://" + metaPin.IpfsHash

	// Sanitize asset name for on-chain use
	assetName = sanitizeAssetName(item.Name)

	// Store NFT record in DB
	err = s.db.QueryRow(ctx, `
        INSERT INTO nfts (
            owner_id, policy_id, asset_name,
            ref_token_name, user_token_name,
            nft_name, description,
            image_ipfs, metadata_ipfs,
            royalties, total_supply, privacy, mint_type,
            status
        ) VALUES (
            $1, '', $2, $3, $4,
            $5, $6, $7, $8,
            $9, $10, $11, 'standard', 'pending'
        ) RETURNING id
    `,
		ownerID, assetName,
		"000643b0"+assetName,
		"001bc280"+assetName,
		item.Name, item.Description,
		imageIPFS, metadataIPFS,
		item.Royalties, item.TotalSupply, privacy,
	).Scan(&nftID)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to store NFT record: %w", err)
	}

	// Store batch item record
	_, err = s.db.Exec(ctx, `
        INSERT INTO batch_items (
            batch_id, nft_id, row_order,
            nft_name, description, royalties,
            total_supply, image_name, status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'uploaded')
    `,
		batchID, nftID, item.RowOrder,
		item.Name, item.Description, item.Royalties,
		item.TotalSupply, item.ImageName,
	)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to store batch item: %w", err)
	}

	// Store IPFS pins
	s.db.Exec(ctx, `
        INSERT INTO ipfs_pins (nft_id, ipfs_hash, pin_type, file_name, file_size)
        VALUES ($1, $2, 'image', $3, $4)
        ON CONFLICT (ipfs_hash) DO NOTHING
    `, nftID, imagePin.IpfsHash, item.ImageName, imagePin.PinSize)

	s.db.Exec(ctx, `
        INSERT INTO ipfs_pins (nft_id, ipfs_hash, pin_type, file_name)
        VALUES ($1, $2, 'metadata', $3)
        ON CONFLICT (ipfs_hash) DO NOTHING
    `, nftID, metaPin.IpfsHash, item.Name+"_metadata.json")

	return nftID, imageIPFS, metadataIPFS, assetName, nil
}

// MintBatchItem mints one NFT on Cardano and updates DB
func (s *Service) MintBatchItem(
	ctx context.Context,
	batchID string,
	nftID string,
	mnemonic []string,
	assetName string,
	metadataIPFS string,
	imageIPFS string,
	royalties float64,
) error {
	// Call blockchain sidecar to mint
	mintResult, err := s.blockchainClient.MintNFT(blockchain.MintNFTRequest{
		Mnemonic:     mnemonic,
		AssetName:    assetName,
		MetadataIPFS: metadataIPFS,
		ImageIPFS:    imageIPFS,
		Royalties:    royalties,
	})
	if err != nil {
		// Mark batch item as failed
		s.db.Exec(ctx, `
            UPDATE batch_items SET status = 'failed', error_msg = $1
            WHERE batch_id = $2 AND nft_id = $3
        `, err.Error(), batchID, nftID)

		// Increment failed count on batch job
		s.db.Exec(ctx, `
            UPDATE batch_jobs SET failed = failed + 1, updated_at = NOW()
            WHERE id = $1
        `, batchID)

		return fmt.Errorf("mint failed: %w", err)
	}

	// Update NFT status to minted
	s.db.Exec(ctx, `
        UPDATE nfts
        SET status = 'minted', tx_hash = $1, policy_id = $2, updated_at = NOW()
        WHERE id = $3
    `, mintResult.TxHash, mintResult.PolicyID, nftID)

	// Update batch item status
	s.db.Exec(ctx, `
        UPDATE batch_items SET status = 'minted'
        WHERE batch_id = $1 AND nft_id = $2
    `, batchID, nftID)

	// Increment minted count on batch job
	s.db.Exec(ctx, `
        UPDATE batch_jobs SET minted = minted + 1, updated_at = NOW()
        WHERE id = $1
    `, batchID)

	return nil
}

// UpdateBatchStatus updates the overall batch job status
func (s *Service) UpdateBatchStatus(ctx context.Context, batchID, status string) error {
	_, err := s.db.Exec(ctx, `
        UPDATE batch_jobs SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, status, batchID)
	return err
}

// GetBatchStatus returns the current status of a batch job
func (s *Service) GetBatchStatus(ctx context.Context, batchID, ownerID string) (map[string]interface{}, error) {
	var id, status, owner string
	var total, uploaded, minted, failed int
	var createdAt interface{}

	err := s.db.QueryRow(ctx, `
        SELECT id, owner_id, status, total, uploaded, minted, failed, created_at
        FROM batch_jobs
        WHERE id = $1 AND owner_id = $2
    `, batchID, ownerID).Scan(
		&id, &owner, &status, &total,
		&uploaded, &minted, &failed, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("batch job not found: %w", err)
	}

	return map[string]interface{}{
		"id":         id,
		"status":     status,
		"total":      total,
		"uploaded":   uploaded,
		"minted":     minted,
		"failed":     failed,
		"created_at": createdAt,
	}, nil
}

// GetWalletForUser retrieves and decrypts the custodial wallet mnemonic
func (s *Service) GetWalletForUser(ctx context.Context, userID string) ([]string, string, error) {
	var encryptedMnemonic, walletAddress string
	err := s.db.QueryRow(ctx, `
        SELECT encrypted_mnemonic, wallet_address
        FROM custodial_wallets
        WHERE user_id = $1
    `, userID).Scan(&encryptedMnemonic, &walletAddress)
	if err != nil {
		return nil, "", fmt.Errorf("wallet not found for user: %w", err)
	}

	mnemonicStr, err := crypto.Decrypt(encryptedMnemonic)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	words := strings.Split(mnemonicStr, " ")
	return words, walletAddress, nil
}

// sanitizeAssetName removes spaces and special chars for on-chain use
// Cardano asset names must be valid bytes and max 32 bytes long
func sanitizeAssetName(name string) string {
	result := strings.ReplaceAll(name, " ", "_")
	var clean strings.Builder
	for _, ch := range result {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' {
			clean.WriteRune(ch)
		}
	}
	// Cardano asset name max = 32 bytes
	// Truncate if longer
	s := clean.String()
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}
