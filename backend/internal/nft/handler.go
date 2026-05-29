package nft

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"NFT_Minting_Platform/pkg/blockchain"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler holds the NFT service
type Handler struct {
	service *Service
}

// NewHandler creates a new NFT handler
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		service: NewService(db),
	}
}

// RegisterRoutes registers all NFT routes
func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	nft := protected.Group("/nft")
	{
		nft.POST("/prepare-mint", h.PrepareMint)
		nft.POST("/confirm-mint", h.ConfirmMint)
		nft.POST("/mint", h.MintNFT)
		nft.POST("/transfer", h.TransferNFT) // Day 8: NFT transfer
		nft.GET("/my-nfts", h.GetMyNFTs)
		nft.GET("/wallet", h.GetWallet)
		nft.GET("/stats", h.GetStats)
		nft.GET("/balance", h.GetWalletBalance)
	}
}

// TransferNFT godoc
// POST /api/nft/transfer
//
// Transfers an NFT from the authenticated user's custodial wallet
// to any recipient address (custodial or external Cardano address).
//
// Body: { nft_id, recipient_address }
func (h *Handler) TransferNFT(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID            string `json:"nft_id" binding:"required"`
		RecipientAddress string `json:"recipient_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate recipient address — must start with addr_test1 (preprod)
	// Prevents accidental transfers to invalid or mainnet addresses
	if len(body.RecipientAddress) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient address"})
		return
	}

	// Verify NFT belongs to this user and is in a transferable state
	// Cannot transfer listed NFTs — must cancel listing first
	var policyID, userTokenName, nftStatus string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT policy_id, user_token_name, status
		FROM nfts
		WHERE id = $1 AND owner_id = $2
	`, body.NFTID, userID).Scan(&policyID, &userTokenName, &nftStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not yours"})
		return
	}

	// Block transfer of listed NFTs — NFT is locked at marketplace script
	if nftStatus == "listed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cannot transfer a listed NFT — cancel the listing first",
		})
		return
	}

	// Block transfer of pending NFTs — tx not yet confirmed on-chain
	if nftStatus == "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "NFT is still pending — wait for minting to confirm",
		})
		return
	}

	// Load sender's custodial wallet mnemonic
	mnemonic, senderAddress, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// Prevent self-transfer
	if senderAddress == body.RecipientAddress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer to your own address"})
		return
	}

	// Build the on-chain NFT unit from policy ID and token name
	// DB stores raw token name; on-chain needs hex encoded
	nftUnit := buildNFTUnit(policyID, userTokenName)

	// Call the blockchain sidecar to build and submit the transfer tx
	result, err := h.service.blockchainClient.TransferNFT(blockchain.TransferNFTRequest{
		Mnemonic:         mnemonic,
		NFTUnit:          nftUnit,
		RecipientAddress: body.RecipientAddress,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to transfer NFT: " + err.Error(),
		})
		return
	}

	// Check if recipient is another custodial user on this platform
	// If so, transfer ownership in DB so it appears in their dashboard
	var recipientUserID string
	dbErr := h.service.db.QueryRow(c.Request.Context(), `
		SELECT user_id FROM custodial_wallets WHERE wallet_address = $1
	`, body.RecipientAddress).Scan(&recipientUserID)

	if dbErr == nil && recipientUserID != "" {
		// Recipient is a platform user — update NFT ownership in DB
		h.service.db.Exec(c.Request.Context(), `
			UPDATE nfts
			SET owner_id = $1, updated_at = NOW()
			WHERE id = $2
		`, recipientUserID, body.NFTID)
	} else {
		// Recipient is an external wallet — mark NFT as transferred
		// Remove from sender's dashboard since it left the platform
		h.service.db.Exec(c.Request.Context(), `
			UPDATE nfts
			SET status = 'transferred', updated_at = NOW()
			WHERE id = $1
		`, body.NFTID)
	}

	log.Printf("NFT transferred — nft: %s, from: %s, to: %s, tx: %s",
		body.NFTID, senderAddress, body.RecipientAddress, result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message":           "NFT transferred successfully",
		"tx_hash":           result.TxHash,
		"recipient_address": body.RecipientAddress,
		"cardanoscan":       fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", result.TxHash),
	})
}

// buildNFTUnit converts DB token name to on-chain hex unit
// DB stores: "001bc280" + rawName — On-chain: "001bc280" + hex(rawName)
func buildNFTUnit(policyID, tokenName string) string {
	if len(tokenName) < 8 {
		return policyID + tokenName
	}
	prefix := tokenName[:8]
	raw := tokenName[8:]
	encoded := hex.EncodeToString([]byte(raw))
	return policyID + prefix + encoded
}

// PrepareMint godoc
// POST /api/nft/prepare-mint
func (h *Handler) PrepareMint(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	royalties, _ := strconv.ParseFloat(c.PostForm("royalties"), 64)
	if royalties < 0 || royalties > 100 {
		royalties = 0
	}

	totalSupply, _ := strconv.Atoi(c.PostForm("total_supply"))
	if totalSupply < 1 {
		totalSupply = 1
	}

	privacy := c.PostForm("privacy")
	if privacy != "private" {
		privacy = "public"
	}

	// Validate asset name — Cardano enforces 32 byte max on token names
	// CIP-68 prefix takes 4 bytes, leaving 28 bytes for the actual asset name
	nftName := c.PostForm("name")
	if len(nftName) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT name is required"})
		return
	}
	if len(nftName) > 28 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("NFT name is too long (%d chars). Maximum is 28 characters.", len(nftName)),
		})
		return
	}

	req := MintRequest{
		OwnerID:     userID,
		Name:        nftName,
		Description: c.PostForm("description"),
		Royalties:   royalties,
		TotalSupply: totalSupply,
		Privacy:     privacy,
		ImageData:   fileData,
		ImageName:   header.Filename,
		Attributes:  map[string]string{},
	}

	result, err := h.service.PrepareMint(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nft_id":        result.NFTID,
		"asset_name":    result.AssetName,
		"image_ipfs":    result.ImageIPFS,
		"metadata_ipfs": result.MetadataIPFS,
		"message":       "NFT prepared — ready to mint on blockchain",
	})
}

// MintNFT godoc
// POST /api/nft/mint
func (h *Handler) MintNFT(c *gin.Context) {
	userID := c.GetString("user_id")
	var body struct {
		NFTID string `json:"nft_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nft_id is required"})
		return
	}

	var assetName, metadataIPFS, imageIPFS string
	var royalties float64
	err := h.service.db.QueryRow(c.Request.Context(), `
        SELECT asset_name, metadata_ipfs, image_ipfs, royalties
        FROM nfts WHERE id = $1 AND owner_id = $2
    `, body.NFTID, userID).Scan(&assetName, &metadataIPFS, &imageIPFS, &royalties)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetWalletForUser error for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet: " + err.Error()})
		return
	}

	mintResult, err := h.service.blockchainClient.MintNFT(blockchain.MintNFTRequest{
		Mnemonic:     mnemonic,
		AssetName:    assetName,
		MetadataIPFS: metadataIPFS,
		ImageIPFS:    imageIPFS,
		Royalties:    royalties,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint: " + err.Error()})
		return
	}

	err = h.service.UpdateMintStatus(
		c.Request.Context(),
		body.NFTID,
		mintResult.TxHash,
		mintResult.PolicyID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mint succeeded but failed to update DB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "NFT minted successfully on Cardano Preprod",
		"tx_hash":   mintResult.TxHash,
		"policy_id": mintResult.PolicyID,
		"nft_id":    body.NFTID,
	})
}

// ConfirmMint godoc
// POST /api/nft/confirm-mint
func (h *Handler) ConfirmMint(c *gin.Context) {
	var body struct {
		NFTID    string `json:"nft_id" binding:"required"`
		TxHash   string `json:"tx_hash" binding:"required"`
		PolicyID string `json:"policy_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err := h.service.UpdateMintStatus(
		c.Request.Context(),
		body.NFTID,
		body.TxHash,
		body.PolicyID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update mint status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "NFT minted successfully",
		"tx_hash": body.TxHash,
	})
}

// GetWallet godoc
// GET /api/nft/wallet
// Returns the custodial wallet address for the authenticated user
func (h *Handler) GetWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	_, address, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet_address": address})
}

// GetMyNFTs godoc
// GET /api/nft/my-nfts
// Returns all NFTs owned by the authenticated user
func (h *Handler) GetMyNFTs(c *gin.Context) {
	userID := c.GetString("user_id")

	nfts, err := h.service.GetUserNFTs(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch NFTs"})
		return
	}

	if nfts == nil {
		nfts = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"nfts":  nfts,
		"count": len(nfts),
	})
}

// GetStats godoc
// GET /api/nft/stats
// Returns NFT statistics for the authenticated user
func (h *Handler) GetStats(c *gin.Context) {
	userID := c.GetString("user_id")

	stats, err := h.service.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetWalletBalance godoc
// GET /api/nft/balance
// Returns the tADA balance of the user's custodial wallet from Blockfrost
func (h *Handler) GetWalletBalance(c *gin.Context) {
	userID := c.GetString("user_id")

	// Get wallet address
	_, address, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	// Fetch balance from Blockfrost
	blockfrostURL := os.Getenv("BLOCKFROST_BASE_URL")
	blockfrostKey := os.Getenv("BLOCKFROST_PROJECT_ID")

	url := fmt.Sprintf("%s/addresses/%s", blockfrostURL, address)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("project_id", blockfrostKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch balance"})
		return
	}
	defer resp.Body.Close()

	// Parse Blockfrost response
	var bf struct {
		Amount []struct {
			Unit     string `json:"unit"`
			Quantity string `json:"quantity"`
		} `json:"amount"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&bf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse balance"})
		return
	}

	// Find lovelace amount
	lovelace := "0"
	for _, a := range bf.Amount {
		if a.Unit == "lovelace" {
			lovelace = a.Quantity
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet_address": address,
		"lovelace":       lovelace,
	})
}

// RegisterPublicRoutes registers NFT routes that require no authentication.
// Called from main.go on the base router (not the protected group).
// Currently only the certificate endpoint is public — all others require JWT.
func (h *Handler) RegisterPublicRoutes(router *gin.Engine) {
	router.GET("/api/certificate/:id", h.GetCertificate)
}

// GetCertificate godoc
// GET /api/certificate/:id
//
// Returns public certificate data for a minted NFT.
// No authentication required — certificate URLs are designed to be
// shared publicly (with buyers, galleries, on social media).
//
// Security decisions:
//   - Only 'minted' and 'listed' NFTs are returned. Pending/failed/transferred
//     NFTs return 404 — no certificate until the NFT exists on-chain.
//   - Owner email is never exposed — only the wallet address, which is
//     already public on the Cardano blockchain.
//   - NFT existence is not confirmed on 404 — prevents enumeration attacks.
func (h *Handler) GetCertificate(c *gin.Context) {
	nftID := c.Param("id")
	if nftID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT ID is required"})
		return
	}

	// Fetch NFT details + owner wallet address in one query.
	// LEFT JOIN on custodial_wallets — some NFTs may have been transferred
	// to external wallets, in which case owner_address will be empty string.
	var (
		id, nftName, description, imageIPFS string
		policyID, assetName, txHash         string
		privacy, status                     string
		royalties                           float64
		createdAt, ownerAddress             string
	)

	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT
			n.id,
			n.nft_name,
			n.description,
			n.image_ipfs,
			n.policy_id,
			n.asset_name,
			COALESCE(n.tx_hash, ''),
			n.privacy,
			n.status,
			n.royalties,
			TO_CHAR(n.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			COALESCE(cw.wallet_address, '') AS owner_address
		FROM nfts n
		LEFT JOIN custodial_wallets cw ON cw.user_id = n.owner_id
		WHERE n.id = $1
		  AND n.status IN ('minted', 'listed')
	`, nftID).Scan(
		&id, &nftName, &description, &imageIPFS,
		&policyID, &assetName, &txHash,
		&privacy, &status, &royalties,
		&createdAt, &ownerAddress,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Certificate not found. The NFT may not exist or may not be minted yet.",
		})
		return
	}

	// Build Cardanoscan links — empty string if no tx hash yet
	cardanoscanTx := ""
	cardanoscanAsset := ""
	if txHash != "" {
		cardanoscanTx = fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", txHash)
		cardanoscanAsset = fmt.Sprintf("https://preprod.cardanoscan.io/token/%s%s", policyID, assetName)
	}

	c.JSON(http.StatusOK, gin.H{
		// Core NFT identity
		"id":          id,
		"nft_name":    nftName,
		"description": description,
		"image_ipfs":  imageIPFS,

		// On-chain proof of authenticity
		"policy_id":     policyID,
		"asset_name":    assetName,
		"tx_hash":       txHash,
		"owner_address": ownerAddress,

		// Certificate metadata
		"status":    status,
		"royalties": royalties,
		"privacy":   privacy,
		"minted_at": createdAt,
		"network":   "Cardano Preprod",
		"platform":  "LankaNFT",

		// Direct links for verification
		"cardanoscan_tx":    cardanoscanTx,
		"cardanoscan_asset": cardanoscanAsset,
	})
}
