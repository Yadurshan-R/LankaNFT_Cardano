// ─────────────────────────────────────────────────────────────────────────────
// internal/listing/handler.go
//
// # Marketplace HTTP Handler
//
// Routes:
//
//	POST /api/listing/create   — list an NFT for sale (checks for duplicate listing)
//	POST /api/listing/buy      — purchase a listed NFT
//	POST /api/listing/cancel   — cancel a listing and reclaim NFT
//	GET  /api/listing/all      — get all active listings (public)
//	GET  /api/listing/mine     — get caller's active listings
//
// ─────────────────────────────────────────────────────────────────────────────
package listing

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"NFT_Minting_Platform/pkg/blockchain"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler holds the listing service
type Handler struct {
	service *Service
}

// NewHandler creates a new listing handler
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{service: NewService(db)}
}

// RegisterRoutes registers all listing routes under /api/listing
func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	l := protected.Group("/listing")
	{
		l.POST("/create", h.CreateListing)
		l.POST("/buy", h.BuyListing)
		l.POST("/cancel", h.CancelListing)
		l.GET("/all", h.GetAllListings)
		l.GET("/mine", h.GetMyListings)

		// External wallet unsigned routes
		l.POST("/create-unsigned", h.CreateListingUnsigned)
		l.POST("/buy-unsigned", h.BuyListingUnsigned)
		l.POST("/cancel-unsigned", h.CancelListingUnsigned)
		l.POST("/confirm-create", h.ConfirmCreateListing)
		l.POST("/confirm-buy", h.ConfirmBuyListing)
		l.POST("/confirm-cancel", h.ConfirmCancelListing)
	}
}

// CreateListing godoc
// POST /api/listing/create
//
// Lists an NFT for sale on the marketplace.
//
// Safety guarantees:
//  1. NFT must be in 'minted' status — prevents listing already-listed NFTs
//  2. Stale active listings for the same NFT are auto-cancelled in the DB
//     before the new listing is created. This handles cases where a previous
//     listing tx succeeded on-chain but the app crashed before updating DB
//     status, leaving a ghost 'active' record pointing to a spent UTxO.
//  3. Minimum price enforced at 2 ADA — below this the min-ADA UTxO
//     requirement makes the listing economically invalid.
//
// Body: { nft_id, price_lovelace }
func (h *Handler) CreateListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID         string  `json:"nft_id" binding:"required"`
		PriceLovelace int64   `json:"price_lovelace" binding:"required"`
		PriceADA      float64 `json:"price_ada"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.PriceLovelace < 2_000_000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum listing price is 2 ADA (2,000,000 lovelace)"})
		return
	}

	// Verify the NFT exists, belongs to this user, and is in 'minted' status.
	// 'listed' status means it is already on the marketplace.
	var policyID, assetName, userTokenName string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT policy_id, asset_name, user_token_name
		FROM nfts
		WHERE id = $1 AND owner_id = $2 AND status = 'minted'
	`, body.NFTID, userID).Scan(&policyID, &assetName, &userTokenName)
	if err != nil {
		// Distinguish between "not found" and "wrong status" for a clear error message
		var existingStatus string
		statusErr := h.service.db.QueryRow(c.Request.Context(), `
			SELECT status FROM nfts WHERE id = $1 AND owner_id = $2
		`, body.NFTID, userID).Scan(&existingStatus)

		if statusErr == nil && existingStatus == "listed" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "this NFT is already listed for sale — cancel the existing listing first",
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not in a listable state"})
		return
	}

	// Auto-cancel any stale active listings for this NFT.
	//
	// A stale listing exists when: a previous listing tx went through on-chain,
	// the NFT left the wallet, but the DB was never updated (app crash, timeout).
	// This leaves ghost 'active' records pointing to already-spent UTxOs.
	// When we create a fresh listing, those old records are meaningless —
	// marking them 'stale' prevents cancel from trying to spend a non-existent UTxO.
	_, err = h.service.db.Exec(c.Request.Context(), `
		UPDATE listings
		SET status = 'stale', updated_at = NOW()
		WHERE nft_id = $1 AND status = 'active'
	`, body.NFTID)
	if err != nil {
		// Non-fatal — log and continue. Stale records are a UX issue, not a blocker.
		log.Printf("[LISTING] Warning: failed to clear stale listings for NFT %s: %v", body.NFTID, err)
	}

	// For single NFTs, the royalty policy is the same as the minting policy
	royaltyPolicyID := policyID

	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[LISTING] wallet load failed for %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	nftUnit := buildNFTUnit(policyID, userTokenName)

	result, err := h.service.blockchainClient.ListNFT(blockchain.ListNFTRequest{
		Mnemonic:        mnemonic,
		NFTUnit:         nftUnit,
		PriceLovelace:   body.PriceLovelace,
		RoyaltyPolicyID: royaltyPolicyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list NFT: " + err.Error()})
		return
	}

	listingID, err := h.service.CreateListing(
		c.Request.Context(),
		body.NFTID, userID,
		result.TxHash,
		body.PriceLovelace,
		policyID, userTokenName,
		royaltyPolicyID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listing submitted but DB update failed"})
		return
	}

	log.Printf("[LISTING] NFT listed — listing: %s, nft: %s, tx: %s", listingID, body.NFTID, result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message":             "NFT listed successfully",
		"listing_id":          listingID,
		"tx_hash":             result.TxHash,
		"marketplace_address": result.MarketplaceAddress,
		"price_lovelace":      body.PriceLovelace,
		"price_ada":           float64(body.PriceLovelace) / 1_000_000,
		"cardanoscan":         fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", result.TxHash),
	})
}

// BuyListing godoc
// POST /api/listing/buy
//
// Purchases a listed NFT. Transfers NFT and ADA on-chain.
//
// Body: { listing_id }
func (h *Handler) BuyListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch full listing details — only active listings can be purchased
	var listingTxHash, scriptUTxO, sellerID string
	var nftPolicyID, nftAssetName, royaltyPolicyID string
	var priceLovelace int64

	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT listing_tx_hash, script_utxo, seller_id,
		       nft_policy_id, nft_asset_name, royalty_policy_id,
		       price_lovelace
		FROM listings
		WHERE id = $1 AND status = 'active'
	`, body.ListingID).Scan(
		&listingTxHash, &scriptUTxO, &sellerID,
		&nftPolicyID, &nftAssetName, &royaltyPolicyID,
		&priceLovelace,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found or not active"})
		return
	}

	// Prevent self-purchase — seller cannot buy their own NFT
	if sellerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot buy your own listing"})
		return
	}

	// Load buyer's custodial wallet
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[LISTING] wallet load failed for %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// Get seller's wallet address for the ADA payment output
	sellerAddress, err := h.service.GetAddressForUser(c.Request.Context(), sellerID)
	if err != nil {
		log.Printf("[LISTING] seller address lookup failed for %s: %v", sellerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get seller address"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)
	nftUnit := buildNFTUnit(nftPolicyID, nftAssetName)

	result, err := h.service.blockchainClient.BuyNFT(blockchain.BuyNFTRequest{
		Mnemonic:         mnemonic,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		SellerAddress:    sellerAddress,
		PriceLovelace:    priceLovelace,
		NFTUnit:          nftUnit,
		RoyaltyPolicyID:  royaltyPolicyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to buy NFT: " + err.Error()})
		return
	}

	// Mark listing as sold and notify seller
	h.service.MarkListingSold(c.Request.Context(), body.ListingID, result.TxHash, userID)

	// Transfer NFT ownership in the DB to the buyer
	h.service.db.Exec(c.Request.Context(), `
		UPDATE nfts SET owner_id = $1, status = 'minted', updated_at = NOW()
		WHERE policy_id = $2 AND user_token_name = $3
	`, userID, nftPolicyID, nftAssetName)

	var sellerEmail, nftName string
	h.service.db.QueryRow(c.Request.Context(), `
		SELECT u.email, n.nft_name
		FROM listings l
		JOIN users u ON u.id = l.seller_id
		JOIN nfts n ON n.id = l.nft_id
		WHERE l.id = $1
	`, body.ListingID).Scan(&sellerEmail, &nftName)

	log.Printf("[LISTING] NFT sold — listing: %s, nft: %s, price: %d lovelace, tx: %s",
		body.ListingID, nftName, priceLovelace, result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message":     "NFT purchased successfully",
		"tx_hash":     result.TxHash,
		"nft_name":    nftName,
		"price_ada":   float64(priceLovelace) / 1_000_000,
		"cardanoscan": fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", result.TxHash),
	})
}

// CancelListing godoc
// POST /api/listing/cancel
// Cancels an active listing and returns the NFT to the seller's wallet.
// Only the original seller can cancel their own listing.
// Body: { listing_id }
func (h *Handler) CancelListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the listing belongs to this user and is still active
	var scriptUTxO, nftPolicyID, nftAssetName string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT script_utxo, nft_policy_id, nft_asset_name
		FROM listings
		WHERE id = $1 AND seller_id = $2 AND status = 'active'
	`, body.ListingID, userID).Scan(&scriptUTxO, &nftPolicyID, &nftAssetName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found or not yours"})
		return
	}

	// Load seller's custodial wallet for signing the cancel transaction
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[LISTING] wallet load failed for %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)
	nftUnit := buildNFTUnit(nftPolicyID, nftAssetName)

	// Call the blockchain sidecar to cancel — returns NFT to seller
	result, err := h.service.blockchainClient.CancelListing(blockchain.CancelListingRequest{
		Mnemonic:         mnemonic,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		NFTUnit:          nftUnit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel listing: " + err.Error()})
		return
	}

	// Update listing status to 'cancelled' and restore NFT status to 'minted'
	h.service.MarkListingCancelled(c.Request.Context(), body.ListingID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Listing cancelled successfully",
		"tx_hash": result.TxHash,
	})
}

// GetAllListings godoc
// GET /api/listing/all
//
// Returns all active listings for the Browse Mints page.
//
// Includes is_own: true for listings created by the authenticated user.
// The frontend uses this to show "Your Listing" instead of a Buy button,
// preventing confusing UX where a seller sees their own NFT for sale
// and tries to buy it (which the backend would reject anyway).
func (h *Handler) GetAllListings(c *gin.Context) {
	// Get the authenticated user's ID to flag their own listings.
	// This endpoint requires auth so user_id is always available.
	userID := c.GetString("user_id")

	listings, err := h.service.GetActiveListings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings, "count": len(listings)})
}

// GetMyListings godoc
// GET /api/listing/mine
// Returns all listings created by the authenticated user (all statuses).
func (h *Handler) GetMyListings(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := h.service.db.Query(c.Request.Context(), `
		SELECT l.id, l.nft_id, l.script_utxo, l.price_lovelace,
		       l.nft_policy_id, l.nft_asset_name, l.status,
		       l.created_at,
		       n.nft_name, n.image_ipfs
		FROM listings l
		JOIN nfts n ON n.id = l.nft_id
		WHERE l.seller_id = $1
		ORDER BY l.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}
	defer rows.Close()

	var listings []map[string]interface{}
	for rows.Next() {
		var id, nftID, scriptUTxO, nftPolicyID, nftAssetName, status string
		var nftName, imageIPFS string
		var priceLovelace int64
		var createdAt time.Time
		if err := rows.Scan(
			&id, &nftID, &scriptUTxO, &priceLovelace,
			&nftPolicyID, &nftAssetName, &status,
			&createdAt,
			&nftName, &imageIPFS,
		); err != nil {
			continue
		}
		listings = append(listings, map[string]interface{}{
			"id":             id,
			"nft_id":         nftID,
			"script_utxo":    scriptUTxO,
			"price_lovelace": priceLovelace,
			"nft_policy_id":  nftPolicyID,
			"nft_asset_name": nftAssetName,
			"status":         status,
			"created_at":     createdAt.UTC().Format(time.RFC3339),
			"nft_name":       nftName,
			"image_ipfs":     imageIPFS,
		})
	}
	if listings == nil {
		listings = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"listings": listings, "count": len(listings)})
}

// buildNFTUnit converts a DB token name to the on-chain hex asset unit.
//
// The DB stores the CIP-68 token name as:
//
//	prefix (8 chars) + rawName (e.g. "aa")
//
// e.g. "001bc280aa"
//
// On-chain, Cardano requires the full hex encoding:
//
//	prefix (8 chars) + hex(rawName) (e.g. "6161")
//
// e.g. "001bc2806161"
//
// This function does the conversion so MeshSDK can find the UTxO correctly.
func buildNFTUnit(policyID, tokenName string) string {
	if len(tokenName) < 8 {
		return policyID + tokenName
	}
	prefix := tokenName[:8]                    // CIP-68 label prefix e.g. "001bc280"
	raw := tokenName[8:]                       // raw asset name e.g. "aa"
	encoded := hex.EncodeToString([]byte(raw)) // hex encoded e.g. "6161"
	return policyID + prefix + encoded
}

// splitUTxO splits a UTxO reference string "txHash#index" into its components.
// Scans from the right to handle tx hashes that may contain '#'.
func splitUTxO(utxo string) [2]string {
	for i := len(utxo) - 1; i >= 0; i-- {
		if utxo[i] == '#' {
			return [2]string{utxo[:i], utxo[i+1:]}
		}
	}
	return [2]string{utxo, "0"}
}

// CreateListingUnsigned godoc
// POST /api/listing/create-unsigned
// Body: { nft_id, price_lovelace }
func (h *Handler) CreateListingUnsigned(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID         string `json:"nft_id" binding:"required"`
		PriceLovelace int64  `json:"price_lovelace" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership and get NFT data
	var policyID, userTokenName string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT policy_id, user_token_name
		FROM nfts WHERE id = $1 AND owner_id = $2 AND status = 'minted'
	`, body.NFTID, userID).Scan(&policyID, &userTokenName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not minted"})
		return
	}
	royaltyPolicyID := policyID // royalty policy = minting policy for CIP-68 NFTs

	// Get external wallet address
	var walletAddress string
	err = h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM external_wallets WHERE user_id = $1", userID,
	).Scan(&walletAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external wallet not found"})
		return
	}

	nftUnit := buildNFTUnit(policyID, userTokenName)

	result, err := h.service.blockchainClient.ListNFTUnsigned(blockchain.ListNFTUnsignedRequest{
		WalletAddress:   walletAddress,
		NFTUnit:         nftUnit,
		PriceLovelace:   body.PriceLovelace,
		RoyaltyPolicyID: royaltyPolicyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build listing tx: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unsigned_cbor":     result.UnsignedCbor,
		"nft_id":            body.NFTID,
		"price_lovelace":    body.PriceLovelace,
		"nft_policy_id":     policyID,
		"nft_asset_name":    userTokenName,
		"royalty_policy_id": royaltyPolicyID,
	})
}

// ConfirmCreateListing godoc
// POST /api/listing/confirm-create
// Called after external wallet signs and submits the listing tx.
// Body: { nft_id, tx_hash, price_lovelace, nft_policy_id, nft_asset_name, royalty_policy_id }
func (h *Handler) ConfirmCreateListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID           string `json:"nft_id" binding:"required"`
		TxHash          string `json:"tx_hash" binding:"required"`
		PriceLovelace   int64  `json:"price_lovelace" binding:"required"`
		NFTPolicyID     string `json:"nft_policy_id" binding:"required"`
		NFTAssetName    string `json:"nft_asset_name" binding:"required"`
		RoyaltyPolicyID string `json:"royalty_policy_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listingID, err := h.service.CreateListing(
		c.Request.Context(),
		body.NFTID, userID, body.TxHash,
		body.PriceLovelace,
		body.NFTPolicyID, body.NFTAssetName,
		body.RoyaltyPolicyID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create listing: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "listing confirmed",
		"listing_id": listingID,
		"tx_hash":    body.TxHash,
	})
}

// BuyListingUnsigned godoc
// POST /api/listing/buy-unsigned
// Body: { listing_id }
func (h *Handler) BuyListingUnsigned(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch listing details
	var listingSellerID, scriptUTxO, sellerAddress, nftPolicyID, nftAssetName, royaltyPolicyID string
	var priceLovelace int64
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT l.seller_id, l.script_utxo, COALESCE(cw.wallet_address, ew.wallet_address),
		       l.price_lovelace, l.nft_policy_id, l.nft_asset_name, l.royalty_policy_id
		FROM listings l
		LEFT JOIN custodial_wallets cw ON cw.user_id = l.seller_id
		LEFT JOIN external_wallets ew ON ew.user_id = l.seller_id
		WHERE l.id = $1 AND l.status = 'active'
	`, body.ListingID).Scan(
		&listingSellerID, &scriptUTxO, &sellerAddress,
		&priceLovelace, &nftPolicyID, &nftAssetName, &royaltyPolicyID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "active listing not found"})
		return
	}

	// Prevent self-purchase
	if listingSellerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot buy your own listing"})
		return
	}

	// Get buyer's external wallet address
	var walletAddress string
	err = h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM external_wallets WHERE user_id = $1", userID,
	).Scan(&walletAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external wallet not found"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)
	nftUnit := buildNFTUnit(nftPolicyID, nftAssetName)

	result, err := h.service.blockchainClient.BuyNFTUnsigned(blockchain.BuyNFTUnsignedRequest{
		WalletAddress:    walletAddress,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		SellerAddress:    sellerAddress,
		PriceLovelace:    priceLovelace,
		NFTUnit:          nftUnit,
		RoyaltyPolicyID:  royaltyPolicyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build buy tx: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unsigned_cbor": result.UnsignedCbor,
		"listing_id":    body.ListingID,
	})
}

// ConfirmBuyListing godoc
// POST /api/listing/confirm-buy
// Body: { listing_id, tx_hash }
func (h *Handler) ConfirmBuyListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
		TxHash    string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the NFT ID to update ownership
	var nftID string
	h.service.db.QueryRow(c.Request.Context(),
		"SELECT nft_id FROM listings WHERE id = $1", body.ListingID,
	).Scan(&nftID)

	if err := h.service.MarkListingSold(c.Request.Context(), body.ListingID, body.TxHash, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm purchase"})
		return
	}

	// Update NFT ownership to buyer
	h.service.db.Exec(c.Request.Context(),
		"UPDATE nfts SET owner_id = $1, status = 'minted', updated_at = NOW() WHERE id = $2",
		userID, nftID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "purchase confirmed", "tx_hash": body.TxHash})
}

// CancelListingUnsigned godoc
// POST /api/listing/cancel-unsigned
// Body: { listing_id }
func (h *Handler) CancelListingUnsigned(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify seller owns this listing
	var scriptUTxO, nftPolicyID, nftAssetName string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT script_utxo, nft_policy_id, nft_asset_name
		FROM listings WHERE id = $1 AND seller_id = $2 AND status = 'active'
	`, body.ListingID, userID).Scan(&scriptUTxO, &nftPolicyID, &nftAssetName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "active listing not found or not yours"})
		return
	}

	// Get external wallet address
	var walletAddress string
	err = h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM external_wallets WHERE user_id = $1", userID,
	).Scan(&walletAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external wallet not found"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)
	nftUnit := buildNFTUnit(nftPolicyID, nftAssetName)

	result, err := h.service.blockchainClient.CancelListingUnsigned(blockchain.CancelListingUnsignedRequest{
		WalletAddress:    walletAddress,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		NFTUnit:          nftUnit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build cancel tx: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unsigned_cbor": result.UnsignedCbor,
		"listing_id":    body.ListingID,
	})
}

// ConfirmCancelListing godoc
// POST /api/listing/confirm-cancel
// Body: { listing_id, tx_hash }
func (h *Handler) ConfirmCancelListing(c *gin.Context) {
	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
		TxHash    string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.MarkListingCancelled(c.Request.Context(), body.ListingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm cancellation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancellation confirmed", "tx_hash": body.TxHash})
}
