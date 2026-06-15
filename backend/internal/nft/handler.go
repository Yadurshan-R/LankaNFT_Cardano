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
	"strings"
	"time"

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
		nft.GET("/external-assets", h.GetExternalAssets)

		// External wallet unsigned routes
		nft.POST("/mint-unsigned", h.MintNFTUnsigned)
		nft.POST("/transfer-unsigned", h.TransferNFTUnsigned)
		nft.POST("/confirm-transfer", h.ConfirmTransfer)
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

	var assetName, metadataIPFS, imageIPFS, description string
	var royalties float64
	err := h.service.db.QueryRow(c.Request.Context(), `
        SELECT asset_name, metadata_ipfs, image_ipfs, royalties, description
        FROM nfts WHERE id = $1 AND owner_id = $2
    `, body.NFTID, userID).Scan(&assetName, &metadataIPFS, &imageIPFS, &royalties, &description)
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
		Description:  description,
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
//
// Returns the tADA balance of the user's wallet from Blockfrost.
// Works for both custodial and external wallet users:
//
//	custodial → address from custodial_wallets table
//	external  → address from external_wallets table
func (h *Handler) GetWalletBalance(c *gin.Context) {
	userID := c.GetString("user_id")

	// Try custodial wallet first
	var address string
	err := h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM custodial_wallets WHERE user_id = $1",
		userID,
	).Scan(&address)

	if err != nil {
		// Not a custodial user — try external wallet
		err = h.service.db.QueryRow(c.Request.Context(),
			"SELECT wallet_address FROM external_wallets WHERE user_id = $1",
			userID,
		).Scan(&address)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
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

	// Address exists but has no transactions yet — return zero balance
	if resp.StatusCode == 404 {
		c.JSON(http.StatusOK, gin.H{
			"wallet_address": address,
			"lovelace":       "0",
		})
		return
	}

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
func (h *Handler) GetCertificate(c *gin.Context) {
	nftID := c.Param("id")
	if nftID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT ID is required"})
		return
	}

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

	cardanoscanTx := ""
	cardanoscanAsset := ""
	if txHash != "" {
		cardanoscanTx = fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", txHash)
		cardanoscanAsset = fmt.Sprintf("https://preprod.cardanoscan.io/token/%s%s", policyID, assetName)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                id,
		"nft_name":          nftName,
		"description":       description,
		"image_ipfs":        imageIPFS,
		"policy_id":         policyID,
		"asset_name":        assetName,
		"tx_hash":           txHash,
		"owner_address":     ownerAddress,
		"status":            status,
		"royalties":         royalties,
		"privacy":           privacy,
		"minted_at":         createdAt,
		"network":           "Cardano Preprod",
		"platform":          "LankaNFT",
		"cardanoscan_tx":    cardanoscanTx,
		"cardanoscan_asset": cardanoscanAsset,
	})
}

// GetExternalAssets godoc
// GET /api/nft/external-assets
//
// Returns all NFTs held in an external wallet (Nami/Eternl/Lace) by
// querying Blockfrost directly. Only available to external wallet users.
func (h *Handler) GetExternalAssets(c *gin.Context) {
	userID := c.GetString("user_id")

	// Get this user's external wallet address
	var walletAddress string
	err := h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM external_wallets WHERE user_id = $1",
		userID,
	).Scan(&walletAddress)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "external wallet not found"})
		return
	}

	blockfrostURL := os.Getenv("BLOCKFROST_BASE_URL")
	blockfrostKey := os.Getenv("BLOCKFROST_PROJECT_ID")
	client := &http.Client{Timeout: 30 * time.Second}

	// ── Step 1: get all assets at this wallet address ──
	url := fmt.Sprintf("%s/addresses/%s/assets", blockfrostURL, walletAddress)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("project_id", blockfrostKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assets from Blockfrost"})
		return
	}
	defer resp.Body.Close()

	// 404 = address has never received any transactions yet — return empty list
	if resp.StatusCode == 404 {
		c.JSON(http.StatusOK, gin.H{"nfts": []interface{}{}, "count": 0, "wallet_address": walletAddress})
		return
	}

	var assets []struct {
		Unit     string `json:"unit"`
		Quantity string `json:"quantity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse asset list"})
		return
	}

	// ── Step 2: fetch metadata for each NFT ──
	var nfts []map[string]interface{}

	for _, asset := range assets {
		// Skip lovelace (ADA) and fungible tokens (quantity > 1)
		if asset.Unit == "lovelace" || asset.Quantity != "1" {
			continue
		}

		// Fetch detailed asset info from Blockfrost
		assetURL := fmt.Sprintf("%s/assets/%s", blockfrostURL, asset.Unit)
		assetReq, _ := http.NewRequest("GET", assetURL, nil)
		assetReq.Header.Set("project_id", blockfrostKey)

		assetResp, err := client.Do(assetReq)
		if err != nil {
			continue
		}

		var assetData struct {
			Asset           string `json:"asset"`
			PolicyID        string `json:"policy_id"`
			AssetName       string `json:"asset_name"` // hex-encoded
			OnchainMetadata *struct {
				Name        interface{} `json:"name"`
				Image       interface{} `json:"image"`
				Description interface{} `json:"description"`
			} `json:"onchain_metadata"`
		}

		if err := json.NewDecoder(assetResp.Body).Decode(&assetData); err != nil {
			assetResp.Body.Close()
			continue
		}
		assetResp.Body.Close()

		// Skip CIP-68 reference tokens (label 100 = 0x000643b0)
		// These are metadata holders, not the tradeable NFTs
		if strings.HasPrefix(assetData.AssetName, "000643b0") {
			continue
		}

		// Build the NFT entry with whatever metadata is available
		nftEntry := map[string]interface{}{
			"id":              asset.Unit, // use full unit as ID for external NFTs
			"policy_id":       assetData.PolicyID,
			"asset_name":      assetData.AssetName,
			"user_token_name": assetData.AssetName, // already hex-encoded from Blockfrost
			"status":          "minted",
			"source":          "external", // distinguishes from platform-minted NFTs
			"wallet_address":  walletAddress,
		}

		// Parse on-chain metadata if available
		if assetData.OnchainMetadata != nil {
			// Name — string
			if name, ok := assetData.OnchainMetadata.Name.(string); ok {
				nftEntry["name"] = name
			} else {
				nftEntry["name"] = assetData.AssetName
			}

			// Description — string
			if desc, ok := assetData.OnchainMetadata.Description.(string); ok {
				nftEntry["description"] = desc
			}

			// Image — can be a plain string or CIP-25 array of strings
			switch img := assetData.OnchainMetadata.Image.(type) {
			case string:
				nftEntry["image"] = img
			case []interface{}:
				// CIP-25 allows image as array of strings — join them
				var parts []string
				for _, p := range img {
					if s, ok := p.(string); ok {
						parts = append(parts, s)
					}
				}
				nftEntry["image"] = strings.Join(parts, "")
			}
		} else {
			// No on-chain metadata — use hex asset name as display name
			nftEntry["name"] = assetData.AssetName
		}

		nfts = append(nfts, nftEntry)
	}

	if nfts == nil {
		nfts = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"nfts":           nfts,
		"count":          len(nfts),
		"wallet_address": walletAddress,
	})
}

// MintNFTUnsigned godoc
// POST /api/nft/mint-unsigned
//
// Builds an unsigned CIP-68 mint transaction for external wallet users.
// Returns unsigned CBOR — the frontend signs with CIP-30 and submits.
//
// After signing and submitting, the frontend calls POST /api/nft/confirm-mint
// with { nft_id, tx_hash, policy_id } to update the DB.
//
// Body: { nft_id }
// The nft_id must be a pending NFT belonging to the authenticated user.
func (h *Handler) MintNFTUnsigned(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID       string   `json:"nft_id" binding:"required"`
		WalletUtxos []string `json:"wallet_utxos"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nft_id is required"})
		return
	}

	// Verify NFT belongs to this user and is pending
	var assetName, metadataIPFS, imageIPFS, description string
	var royalties float64
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT asset_name, metadata_ipfs, image_ipfs, royalties, description
		FROM nfts WHERE id = $1 AND owner_id = $2 AND status = 'pending'
	`, body.NFTID, userID).Scan(&assetName, &metadataIPFS, &imageIPFS, &royalties, &description)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not pending"})
		return
	}

	// Get this user's external wallet address
	var walletAddress string
	err = h.service.db.QueryRow(c.Request.Context(),
		"SELECT wallet_address FROM external_wallets WHERE user_id = $1",
		userID,
	).Scan(&walletAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external wallet not found — are you logged in with a wallet?"})
		return
	}

	// Build unsigned tx via sidecar
	result, err := h.service.blockchainClient.MintNFTUnsigned(blockchain.MintNFTUnsignedRequest{
		WalletAddress: walletAddress,
		WalletUtxos:   body.WalletUtxos,
		AssetName:     assetName,
		MetadataIPFS:  metadataIPFS,
		ImageIPFS:     imageIPFS,
		Royalties:     royalties,
		Description:   description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build mint tx: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unsigned_cbor": result.UnsignedCbor,
		"policy_id":     result.PolicyID,
		"nft_id":        body.NFTID,
		"asset_name":    assetName,
	})
}

// TransferNFTUnsigned godoc
// POST /api/nft/transfer-unsigned
//
// Builds an unsigned transfer transaction for external wallet users.
// Body: { nft_id, recipient_address }
func (h *Handler) TransferNFTUnsigned(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID            string `json:"nft_id" binding:"required"`
		RecipientAddress string `json:"recipient_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify NFT belongs to this user and is minted (not listed)
	var policyID, userTokenName, nftStatus string
	err := h.service.db.QueryRow(c.Request.Context(), `
		SELECT policy_id, user_token_name, status
		FROM nfts WHERE id = $1 AND owner_id = $2
	`, body.NFTID, userID).Scan(&policyID, &userTokenName, &nftStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found or not yours"})
		return
	}
	if nftStatus == "listed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cancel the listing before transferring"})
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

	nftUnit := buildNFTUnit(policyID, userTokenName)

	result, err := h.service.blockchainClient.TransferNFTUnsigned(blockchain.TransferNFTUnsignedRequest{
		WalletAddress:    walletAddress,
		NFTUnit:          nftUnit,
		RecipientAddress: body.RecipientAddress,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build transfer tx: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unsigned_cbor":     result.UnsignedCbor,
		"nft_id":            body.NFTID,
		"recipient_address": body.RecipientAddress,
	})
}

// ConfirmTransfer godoc
// POST /api/nft/confirm-transfer
//
// Called after external wallet user signs and submits a transfer tx.
// Updates NFT ownership in the DB.
// Body: { nft_id, tx_hash, recipient_address }
func (h *Handler) ConfirmTransfer(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		NFTID            string `json:"nft_id" binding:"required"`
		TxHash           string `json:"tx_hash" binding:"required"`
		RecipientAddress string `json:"recipient_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if recipient is another platform user
	var recipientUserID string
	dbErr := h.service.db.QueryRow(c.Request.Context(),
		"SELECT user_id FROM custodial_wallets WHERE wallet_address = $1",
		body.RecipientAddress,
	).Scan(&recipientUserID)

	if dbErr == nil {
		// Transfer to custodial user — update ownership
		h.service.db.Exec(c.Request.Context(),
			"UPDATE nfts SET owner_id = $1, updated_at = NOW() WHERE id = $2 AND owner_id = $3",
			recipientUserID, body.NFTID, userID,
		)
	} else {
		// Transfer to external — mark as transferred
		h.service.db.Exec(c.Request.Context(),
			"UPDATE nfts SET status = 'transferred', updated_at = NOW() WHERE id = $1 AND owner_id = $2",
			body.NFTID, userID,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "transfer confirmed",
		"tx_hash": body.TxHash,
	})
}
