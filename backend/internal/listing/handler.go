// ─────────────────────────────────────────────────────────────────────────────
// internal/listing/handler.go
//
// Marketplace HTTP Handler
//
// Routes:
//   POST /api/listing/create   — list an NFT for sale
//   POST /api/listing/buy      — purchase a listed NFT
//   POST /api/listing/cancel   — cancel a listing
//   GET  /api/listing/all      — get all active listings
//   GET  /api/listing/mine     — get caller's active listings
// ─────────────────────────────────────────────────────────────────────────────

package listing

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

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

// RegisterRoutes registers all listing routes
func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	l := protected.Group("/listing")
	{
		l.POST("/create", h.CreateListing)
		l.POST("/buy", h.BuyListing)
		l.POST("/cancel", h.CancelListing)
		l.GET("/all", h.GetAllListings)
		l.GET("/mine", h.GetMyListings)
	}
}

// CreateListing godoc
// POST /api/listing/create
// Lists an NFT for sale on the marketplace
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

	// Minimum price 2 ADA
	if body.PriceLovelace < 2_000_000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum price is 2 ADA (2000000 lovelace)"})
		return
	}

	// Get NFT details from DB
	var policyID, assetName, userTokenName string
	err := h.service.db.QueryRow(c.Request.Context(), `
        SELECT policy_id, asset_name, user_token_name
        FROM nfts
        WHERE id = $1 AND owner_id = $2 AND status = 'minted'
    `, body.NFTID, userID).Scan(&policyID, &assetName, &userTokenName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not minted yet"})
		return
	}

	// Get royalty policy ID from the royalty NFT associated with this policy
	// For now we use the same policy ID — single NFTs use their own policy
	royaltyPolicyID := policyID

	// Get custodial wallet
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// DB stores "001bc280" + rawName, but on-chain it is "001bc280" + hex(rawName)
	// MeshSDK needs the full hex unit to find the UTxO
	tokenPrefix := userTokenName[:8]               // "001bc280"
	rawName := userTokenName[8:]                   // "aa"
	nameHex := hex.EncodeToString([]byte(rawName)) // "6161"
	nftUnit := policyID + tokenPrefix + nameHex

	// Call blockchain sidecar to build and submit listing tx
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

	// Store listing in DB
	listingID, err := h.service.CreateListing(
		c.Request.Context(),
		body.NFTID,
		userID,
		result.TxHash,
		body.PriceLovelace,
		policyID,
		userTokenName,
		royaltyPolicyID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listing submitted but DB update failed"})
		return
	}

	log.Printf("NFT listed — listing: %s, tx: %s", listingID, result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message":             "NFT listed successfully",
		"listing_id":          listingID,
		"tx_hash":             result.TxHash,
		"marketplace_address": result.MarketplaceAddress,
		"price_lovelace":      body.PriceLovelace,
		"cardanoscan": fmt.Sprintf(
			"https://preprod.cardanoscan.io/transaction/%s", result.TxHash,
		),
	})
}

// BuyListing godoc
// POST /api/listing/buy
// Purchase a listed NFT
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

	// Get listing details
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

	// Prevent buying your own listing
	if sellerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot buy your own listing"})
		return
	}

	// Get buyer wallet
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// Get seller address for payment
	_, sellerAddress, err := h.service.GetWalletForUser(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get seller address"})
		return
	}

	// Parse script UTxO
	utxoParts := splitUTxO(scriptUTxO)

	// Call blockchain sidecar to buy
	result, err := h.service.blockchainClient.BuyNFT(blockchain.BuyNFTRequest{
		Mnemonic:         mnemonic,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		SellerAddress:    sellerAddress,
		PriceLovelace:    priceLovelace,
		NFTUnit:          nftPolicyID + nftAssetName,
		RoyaltyPolicyID:  royaltyPolicyID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to buy NFT: " + err.Error()})
		return
	}

	// Mark listing as sold
	h.service.MarkListingSold(c.Request.Context(), body.ListingID, result.TxHash, userID)

	// Transfer NFT ownership in DB
	h.service.db.Exec(c.Request.Context(), `
        UPDATE nfts SET owner_id = $1, status = 'minted', updated_at = NOW()
        WHERE policy_id = $2 AND asset_name = $3
    `, userID, nftPolicyID, nftAssetName)

	c.JSON(http.StatusOK, gin.H{
		"message": "NFT purchased successfully",
		"tx_hash": result.TxHash,
		"cardanoscan": fmt.Sprintf(
			"https://preprod.cardanoscan.io/transaction/%s", result.TxHash,
		),
	})
}

// CancelListing godoc
// POST /api/listing/cancel
// Cancel a listing and reclaim NFT
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

	// Get listing — verify seller owns it
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

	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)

	result, err := h.service.blockchainClient.CancelListing(blockchain.CancelListingRequest{
		Mnemonic:         mnemonic,
		ListingUTxOHash:  utxoParts[0],
		ListingUTxOIndex: utxoParts[1],
		NFTUnit:          nftPolicyID + nftAssetName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel listing: " + err.Error()})
		return
	}

	h.service.MarkListingCancelled(c.Request.Context(), body.ListingID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Listing cancelled successfully",
		"tx_hash": result.TxHash,
	})
}

// GetAllListings godoc
// GET /api/listing/all
func (h *Handler) GetAllListings(c *gin.Context) {
	listings, err := h.service.GetActiveListings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"listings": listings, "count": len(listings)})
}

// GetMyListings godoc
// GET /api/listing/mine
func (h *Handler) GetMyListings(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := h.service.db.Query(c.Request.Context(), `
        SELECT l.id, l.script_utxo, l.price_lovelace,
               l.nft_policy_id, l.nft_asset_name, l.status,
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
		var id, scriptUTxO, nftPolicyID, nftAssetName, status string
		var nftName, imageIPFS string
		var priceLovelace int64
		if err := rows.Scan(
			&id, &scriptUTxO, &priceLovelace,
			&nftPolicyID, &nftAssetName, &status,
			&nftName, &imageIPFS,
		); err != nil {
			continue
		}
		listings = append(listings, map[string]interface{}{
			"id":             id,
			"script_utxo":    scriptUTxO,
			"price_lovelace": priceLovelace,
			"nft_policy_id":  nftPolicyID,
			"nft_asset_name": nftAssetName,
			"status":         status,
			"nft_name":       nftName,
			"image_ipfs":     imageIPFS,
		})
	}

	if listings == nil {
		listings = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings, "count": len(listings)})
}

// splitUTxO splits "txHash#index" into ["txHash", "index"]
func splitUTxO(utxo string) [2]string {
	for i := len(utxo) - 1; i >= 0; i-- {
		if utxo[i] == '#' {
			return [2]string{utxo[:i], utxo[i+1:]}
		}
	}
	return [2]string{utxo, "0"}
}
