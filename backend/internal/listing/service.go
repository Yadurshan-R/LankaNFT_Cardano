// ─────────────────────────────────────────────────────────────────────────────
// internal/listing/service.go
//
// Marketplace Listing Service
//
// Handles the business logic for listing, buying, and cancelling NFTs.
// Works with the blockchain sidecar to build and submit Cardano transactions.
// ─────────────────────────────────────────────────────────────────────────────

package listing

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"NFT_Minting_Platform/pkg/blockchain"
	"NFT_Minting_Platform/pkg/crypto"
	"NFT_Minting_Platform/pkg/email"
)

// Service handles listing business logic
type Service struct {
	db               *pgxpool.Pool
	blockchainClient *blockchain.Client
}

// NewService creates a new listing service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:               db,
		blockchainClient: blockchain.NewClient(),
	}
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

// CreateListing stores a new listing in the DB after on-chain confirmation
func (s *Service) CreateListing(
	ctx context.Context,
	nftID string,
	sellerID string,
	txHash string,
	priceLovelace int64,
	nftPolicyID string,
	nftAssetName string,
	royaltyPolicyID string,
) (string, error) {
	scriptUTxO := txHash + "#0"

	var listingID string
	err := s.db.QueryRow(ctx, `
        INSERT INTO listings (
            nft_id, seller_id,
            listing_tx_hash, script_utxo,
            price_lovelace,
            nft_policy_id, nft_asset_name,
            royalty_policy_id,
            status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active')
        RETURNING id
    `,
		nftID, sellerID,
		txHash, scriptUTxO,
		priceLovelace,
		nftPolicyID, nftAssetName,
		royaltyPolicyID,
	).Scan(&listingID)
	if err != nil {
		return "", fmt.Errorf("failed to create listing: %w", err)
	}

	// Update NFT status to listed
	s.db.Exec(ctx, `
        UPDATE nfts SET status = 'listed', updated_at = NOW()
        WHERE id = $1
    `, nftID)

	return listingID, nil
}

// GetActiveListings returns all active listings
func (s *Service) GetActiveListings(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(ctx, `
        SELECT
            l.id, l.nft_id, l.seller_id,
            l.listing_tx_hash, l.script_utxo,
            l.price_lovelace,
            l.nft_policy_id, l.nft_asset_name,
            l.royalty_policy_id,
            l.created_at,
            n.nft_name, n.description, n.image_ipfs,
            u.email as seller_email
        FROM listings l
        JOIN nfts n ON n.id = l.nft_id
        JOIN users u ON u.id = l.seller_id
        WHERE l.status = 'active'
        ORDER BY l.created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []map[string]interface{}
	for rows.Next() {
		var id, nftID, sellerID, txHash, scriptUTxO string
		var nftPolicyID, nftAssetName, royaltyPolicyID string
		var nftName, description, imageIPFS, sellerEmail string
		var priceLovelace int64
		var createdAt interface{}

		err := rows.Scan(
			&id, &nftID, &sellerID,
			&txHash, &scriptUTxO,
			&priceLovelace,
			&nftPolicyID, &nftAssetName,
			&royaltyPolicyID,
			&createdAt,
			&nftName, &description, &imageIPFS,
			&sellerEmail,
		)
		if err != nil {
			return nil, err
		}

		listings = append(listings, map[string]interface{}{
			"id":                id,
			"nft_id":            nftID,
			"seller_id":         sellerID,
			"listing_tx_hash":   txHash,
			"script_utxo":       scriptUTxO,
			"price_lovelace":    priceLovelace,
			"nft_policy_id":     nftPolicyID,
			"nft_asset_name":    nftAssetName,
			"royalty_policy_id": royaltyPolicyID,
			"created_at":        createdAt,
			"nft_name":          nftName,
			"description":       description,
			"image_ipfs":        imageIPFS,
			"seller_email":      sellerEmail,
		})
	}

	if listings == nil {
		listings = []map[string]interface{}{}
	}

	return listings, nil
}

// MarkListingSold updates listing status to sold and notifies the seller
func (s *Service) MarkListingSold(
	ctx context.Context,
	listingID string,
	saleTxHash string,
	buyerID string,
) error {
	_, err := s.db.Exec(ctx, `
		UPDATE listings
		SET status = 'sold',
		    sale_tx_hash = $1,
		    buyer_id = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, saleTxHash, buyerID, listingID)
	if err != nil {
		return err
	}

	// Send email notification to seller — fire and forget, don't block the response
	go func() {
		var sellerEmail, buyerEmail, nftName string
		var priceLovelace int64
		err := s.db.QueryRow(context.Background(), `
			SELECT
				u_seller.email,
				u_buyer.email,
				n.nft_name,
				l.price_lovelace
			FROM listings l
			JOIN users u_seller ON u_seller.id = l.seller_id
			JOIN users u_buyer  ON u_buyer.id  = $1
			JOIN nfts n         ON n.id         = l.nft_id
			WHERE l.id = $2
		`, buyerID, listingID).Scan(&sellerEmail, &buyerEmail, &nftName, &priceLovelace)
		if err != nil {
			return
		}

		email.SendSaleNotification(sellerEmail, nftName, buyerEmail, priceLovelace, saleTxHash)
	}()

	return nil
}

// MarkListingCancelled updates listing status to cancelled
func (s *Service) MarkListingCancelled(ctx context.Context, listingID string) error {
	_, err := s.db.Exec(ctx, `
        UPDATE listings
        SET status = 'cancelled', updated_at = NOW()
        WHERE id = $1
    `, listingID)
	if err != nil {
		return err
	}

	// Get nft_id and restore NFT status
	var nftID string
	s.db.QueryRow(ctx, `SELECT nft_id FROM listings WHERE id = $1`, listingID).Scan(&nftID)
	if nftID != "" {
		s.db.Exec(ctx, `UPDATE nfts SET status = 'minted', updated_at = NOW() WHERE id = $1`, nftID)
	}

	return nil
}
