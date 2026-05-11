package nft

import (
	"io"
	"net/http"
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
		nft.GET("/my-nfts", h.GetMyNFTs)
	}
}

// PrepareMint godoc
// POST /api/nft/prepare-mint
// Multipart form: file + metadata fields
// Uploads to IPFS, stores in DB, returns mint params for blockchain
func (h *Handler) PrepareMint(c *gin.Context) {
	userID := c.GetString("user_id")

	// Parse multipart form (max 100MB)
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Read file bytes
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Parse royalties
	royalties, _ := strconv.ParseFloat(c.PostForm("royalties"), 64)
	if royalties < 0 || royalties > 100 {
		royalties = 0
	}

	// Parse total supply
	totalSupply, _ := strconv.Atoi(c.PostForm("total_supply"))
	if totalSupply < 1 {
		totalSupply = 1
	}

	// Parse privacy
	privacy := c.PostForm("privacy")
	if privacy != "private" {
		privacy = "public"
	}

	// Build mint request
	req := MintRequest{
		OwnerID:     userID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		Royalties:   royalties,
		TotalSupply: totalSupply,
		Privacy:     privacy,
		ImageData:   fileData,
		ImageName:   header.Filename,
		Attributes:  map[string]string{}, // TODO: parse from form
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NFT name is required"})
		return
	}

	// Upload to IPFS + store in DB
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
// Calls blockchain sidecar to mint the NFT on-chain
// Body: { "nft_id": "uuid" }
func (h *Handler) MintNFT(c *gin.Context) {
	userID := c.GetString("user_id")
	var body struct {
		NFTID string `json:"nft_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nft_id is required"})
		return
	}

	// Get NFT details from DB
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

	// Get and decrypt custodial wallet
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet"})
		return
	}

	// Call blockchain sidecar to mint
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

	// Update NFT status in DB
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
// Called after blockchain tx is submitted
// Updates NFT status to 'minted' with tx hash
func (h *Handler) ConfirmMint(c *gin.Context) {
	var body struct {
		NFTID    string `json:"nft_id" binding:"required"`
		TxHash   string `json:"tx_hash" binding:"required"`
		PolicyID string `json:"policy_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
