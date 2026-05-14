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

type Handler struct {
	service *Service
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{service: NewService(db)}
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum price is 2 ADA (2000000 lovelace)"})
		return
	}

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

	royaltyPolicyID := policyID

	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// DB stores "001bc280" + rawName, on-chain is "001bc280" + hex(rawName)
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

	log.Printf("NFT listed — listing: %s, tx: %s", listingID, result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message":             "NFT listed successfully",
		"listing_id":          listingID,
		"tx_hash":             result.TxHash,
		"marketplace_address": result.MarketplaceAddress,
		"price_lovelace":      body.PriceLovelace,
		"cardanoscan":         fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", result.TxHash),
	})
}

func (h *Handler) BuyListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	if sellerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot buy your own listing"})
		return
	}

	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	_, sellerAddress, err := h.service.GetWalletForUser(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get seller address"})
		return
	}

	utxoParts := splitUTxO(scriptUTxO)

	// Fix: apply hex encoding to asset name for on-chain unit
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

	h.service.MarkListingSold(c.Request.Context(), body.ListingID, result.TxHash, userID)

	h.service.db.Exec(c.Request.Context(), `
        UPDATE nfts SET owner_id = $1, status = 'minted', updated_at = NOW()
        WHERE policy_id = $2 AND user_token_name = $3
    `, userID, nftPolicyID, nftAssetName)

	c.JSON(http.StatusOK, gin.H{
		"message":     "NFT purchased successfully",
		"tx_hash":     result.TxHash,
		"cardanoscan": fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", result.TxHash),
	})
}

func (h *Handler) CancelListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		ListingID string `json:"listing_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	// Fix: apply hex encoding to asset name for on-chain unit
	nftUnit := buildNFTUnit(nftPolicyID, nftAssetName)

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

	h.service.MarkListingCancelled(c.Request.Context(), body.ListingID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Listing cancelled successfully",
		"tx_hash": result.TxHash,
	})
}

func (h *Handler) GetAllListings(c *gin.Context) {
	listings, err := h.service.GetActiveListings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"listings": listings, "count": len(listings)})
}

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

// buildNFTUnit converts DB token name to on-chain hex unit
// DB stores: policyID + "001bc280" + rawName (e.g. "aa")
// On-chain:  policyID + "001bc280" + hex(rawName) (e.g. "6161")
func buildNFTUnit(policyID, tokenName string) string {
	if len(tokenName) < 8 {
		return policyID + tokenName
	}
	prefix := tokenName[:8]                    // "001bc280"
	raw := tokenName[8:]                       // "aa"
	encoded := hex.EncodeToString([]byte(raw)) // "6161"
	return policyID + prefix + encoded
}

// splitUTxO splits "txHash#index" into [txHash, index]
func splitUTxO(utxo string) [2]string {
	for i := len(utxo) - 1; i >= 0; i-- {
		if utxo[i] == '#' {
			return [2]string{utxo[:i], utxo[i+1:]}
		}
	}
	return [2]string{utxo, "0"}
}
