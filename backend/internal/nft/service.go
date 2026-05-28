package nft

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"NFT_Minting_Platform/pkg/blockchain"
	"NFT_Minting_Platform/pkg/crypto"
	"NFT_Minting_Platform/pkg/ipfs"
)

// Service handles NFT business logic
type Service struct {
	db               *pgxpool.Pool
	pinata           *ipfs.PinataClient
	blockchainClient *blockchain.Client
}

// NewService creates a new NFT service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:               db,
		pinata:           ipfs.NewPinataClient(),
		blockchainClient: blockchain.NewClient(),
	}
}

// MintRequest holds all data needed to mint a single NFT
type MintRequest struct {
	OwnerID     string
	Name        string
	Description string
	Royalties   float64
	TotalSupply int
	Privacy     string
	ImageData   []byte
	ImageName   string
	Attributes  map[string]string
}

// MintResult holds the result after minting
type MintResult struct {
	NFTID        string
	TxHash       string
	PolicyID     string
	AssetName    string
	ImageIPFS    string
	MetadataIPFS string
}

// PrepareMint uploads to IPFS and stores NFT record
// Returns NFT ID and data needed for the blockchain transaction
func (s *Service) PrepareMint(ctx context.Context, req MintRequest) (*MintResult, error) {
	log.Printf("[MINT] Starting prepare for NFT: %s, image size: %d bytes", req.Name, len(req.ImageData))

	// Step 1 — Upload image to Pinata
	log.Printf("[MINT] Uploading image to Pinata...")
	imagePin, err := s.pinata.UploadFile(req.ImageData, req.ImageName)
	if err != nil {
		log.Printf("[MINT] Image upload failed: %v", err)
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	log.Printf("[MINT] Image uploaded: %s", imagePin.IpfsHash)

	imageIPFS := "ipfs://" + imagePin.IpfsHash

	// Step 2 — Build CIP-68 metadata JSON
	metadata := ipfs.BuildCIP68Metadata(
		req.Name,
		req.Description,
		imageIPFS,
		req.Royalties,
		req.Attributes,
	)

	// Step 3 — Upload metadata JSON to Pinata
	metaPin, err := s.pinata.UploadJSON(metadata, req.Name+"_metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to upload metadata: %w", err)
	}
	metadataIPFS := "ipfs://" + metaPin.IpfsHash

	// Step 4 — Generate asset name (sanitize NFT name for on-chain use)
	// Remove spaces and special chars — Cardano asset names are bytes
	assetName := sanitizeAssetName(req.Name)

	// Step 5 — Store NFT record in DB with status 'pending'
	var nftID string
	err = s.db.QueryRow(ctx, `
        INSERT INTO nfts (
            owner_id, policy_id, asset_name,
            ref_token_name, user_token_name,
            nft_name, description,
            image_ipfs, metadata_ipfs,
            royalties, total_supply, privacy, mint_type,
            status
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7, $8, $9,
            $10, $11, $12, 'standard',
            'pending'
        ) RETURNING id
    `,
		req.OwnerID,
		"", // policy_id filled after blockchain tx
		assetName,
		"000643b0"+assetName, // ref token name (100 prefix)
		"001bc280"+assetName, // user token name (222 prefix)
		req.Name,
		req.Description,
		imageIPFS,
		metadataIPFS,
		req.Royalties,
		req.TotalSupply,
		req.Privacy,
	).Scan(&nftID)
	if err != nil {
		return nil, fmt.Errorf("failed to store NFT record: %w", err)
	}

	// Step 6 — Store image pin record — ignore if already exists (same file = same IPFS hash)
	_, err = s.db.Exec(ctx, `
        INSERT INTO ipfs_pins (nft_id, ipfs_hash, pin_type, file_name, file_size)
        VALUES ($1, $2, 'image', $3, $4)
        ON CONFLICT (ipfs_hash) DO NOTHING
    `, nftID, imagePin.IpfsHash, req.ImageName, imagePin.PinSize)
	if err != nil {
		return nil, fmt.Errorf("failed to store image pin: %w", err)
	}

	// Step 7 — Store metadata pin record — ignore if already exists
	_, err = s.db.Exec(ctx, `
        INSERT INTO ipfs_pins (nft_id, ipfs_hash, pin_type, file_name)
        VALUES ($1, $2, 'metadata', $3)
        ON CONFLICT (ipfs_hash) DO NOTHING
    `, nftID, metaPin.IpfsHash, req.Name+"_metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to store metadata pin: %w", err)
	}

	// Step 8 — Store attributes
	for traitType, value := range req.Attributes {
		_, err = s.db.Exec(ctx, `
            INSERT INTO nft_attributes (nft_id, trait_type, value)
            VALUES ($1, $2, $3)
        `, nftID, traitType, value)
		if err != nil {
			return nil, fmt.Errorf("failed to store attribute: %w", err)
		}
	}

	return &MintResult{
		NFTID:        nftID,
		AssetName:    assetName,
		ImageIPFS:    imageIPFS,
		MetadataIPFS: metadataIPFS,
	}, nil
}

// UpdateMintStatus updates NFT status and tx hash after blockchain confirmation
func (s *Service) UpdateMintStatus(ctx context.Context, nftID, txHash, policyID string) error {
	_, err := s.db.Exec(ctx, `
        UPDATE nfts
        SET status = 'minted', tx_hash = $1, policy_id = $2, updated_at = NOW()
        WHERE id = $3
    `, txHash, policyID, nftID)
	return err
}

// GetUserNFTs returns all NFTs owned by a user
func (s *Service) GetUserNFTs(ctx context.Context, ownerID string) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(ctx, `
        SELECT id, nft_name, description, image_ipfs,
               policy_id, asset_name, status, tx_hash,
               royalties, privacy, created_at
        FROM nfts
        WHERE owner_id = $1
        AND status NOT IN ('failed', 'transferred')
        ORDER BY created_at DESC
    `, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nfts []map[string]interface{}
	for rows.Next() {
		var id, name, desc, image, policyID, assetName, status, privacy string
		var txHash *string
		var royalties float64
		var createdAt interface{}

		err := rows.Scan(&id, &name, &desc, &image,
			&policyID, &assetName, &status, &txHash,
			&royalties, &privacy, &createdAt)
		if err != nil {
			return nil, err
		}

		nfts = append(nfts, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"image":       image,
			"policy_id":   policyID,
			"asset_name":  assetName,
			"status":      status,
			"tx_hash":     txHash,
			"royalties":   royalties,
			"privacy":     privacy,
			"created_at":  createdAt,
		})
	}

	return nfts, nil
}

// GetWalletForUser retrieves and decrypts the custodial wallet mnemonic
// Returns mnemonic words for signing — never logs or stores decrypted value
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

	// Decrypt mnemonic
	mnemonicStr, err := crypto.Decrypt(encryptedMnemonic)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	// Split back into words
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

// GetUserStats returns NFT statistics for a user
func (s *Service) GetUserStats(ctx context.Context, ownerID string) (map[string]interface{}, error) {
	var total, minted, pending int
	err := s.db.QueryRow(ctx, `
        SELECT
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'minted') as minted,
            COUNT(*) FILTER (WHERE status = 'pending') as pending
        FROM nfts
        WHERE owner_id = $1
        AND status NOT IN ('failed', 'transferred')
    `, ownerID).Scan(&total, &minted, &pending)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total":   total,
		"minted":  minted,
		"pending": pending,
	}, nil
}
