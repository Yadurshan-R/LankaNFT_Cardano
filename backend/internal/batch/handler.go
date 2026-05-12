// ─────────────────────────────────────────────────────────────────────────────
// internal/batch/handler.go
//
// Batch Minting HTTP Handler
//
// Routes:
//   POST /api/batch/prepare  — upload all files to IPFS, create batch job
//   POST /api/batch/mint     — mint all prepared NFTs sequentially
//   GET  /api/batch/:id      — get batch job status and progress
// ─────────────────────────────────────────────────────────────────────────────

package batch

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler holds the batch service
type Handler struct {
	service *Service
}

// NewHandler creates a new batch handler
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		service: NewService(db),
	}
}

// RegisterRoutes registers all batch routes on the protected router group
func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	b := protected.Group("/batch")
	{
		b.POST("/prepare", h.PrepareBatch)
		b.POST("/mint", h.MintBatch)
		b.GET("/:id", h.GetBatchStatus)
	}
}

// PrepareBatch godoc
// POST /api/batch/prepare
// Accepts multiple files + metadata, uploads all to IPFS
// Creates a batch job and stores all NFT records in DB
// Returns batch_id for use in /api/batch/mint
//
// Form fields:
//
//	privacy          string  — 'public' or 'private'
//	files[]          []File  — image files (one per NFT)
//	names[]          []string — NFT names
//	descriptions[]   []string — NFT descriptions
//	royalties[]      []string — royalty percentages
//	total_supplies[] []string — total supplies
func (h *Handler) PrepareBatch(c *gin.Context) {
	userID := c.GetString("user_id")

	// Parse multipart form — max 500MB for batch uploads
	if err := c.Request.ParseMultipartForm(500 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	form := c.Request.MultipartForm

	// Get all uploaded files
	files := form.File["files[]"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one file is required"})
		return
	}

	// Parse privacy setting
	privacy := c.PostForm("privacy")
	if privacy != "private" {
		privacy = "public"
	}

	// Parse metadata arrays — one entry per file
	names := form.Value["names[]"]
	descriptions := form.Value["descriptions[]"]
	royaltiesStr := form.Value["royalties[]"]
	suppliesStr := form.Value["total_supplies[]"]

	// Validate that names array matches files count
	if len(names) != len(files) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf(
				"number of names (%d) must match number of files (%d)",
				len(names), len(files),
			),
		})
		return
	}

	// Create batch job in DB
	batchID, err := h.service.CreateBatchJob(c.Request.Context(), userID, len(files))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create batch job"})
		return
	}

	// Process each file — upload to IPFS and store in DB
	results := []map[string]interface{}{}
	uploadedCount := 0

	for i, fileHeader := range files {
		// Read file bytes
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Batch %s: failed to open file %d: %v", batchID, i, err)
			continue
		}
		fileData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			log.Printf("Batch %s: failed to read file %d: %v", batchID, i, err)
			continue
		}

		// Parse per-item metadata with safe fallbacks
		name := ""
		if i < len(names) {
			name = names[i]
		}
		if name == "" {
			name = fmt.Sprintf("NFT_%d", i+1)
		}

		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i]
		}

		royalties := 0.0
		if i < len(royaltiesStr) {
			royalties, _ = strconv.ParseFloat(royaltiesStr[i], 64)
		}

		totalSupply := 1
		if i < len(suppliesStr) {
			totalSupply, _ = strconv.Atoi(suppliesStr[i])
			if totalSupply < 1 {
				totalSupply = 1
			}
		}

		// Upload to IPFS and store NFT record
		item := BatchItem{
			RowOrder:    i + 1,
			Name:        name,
			Description: desc,
			Royalties:   royalties,
			TotalSupply: totalSupply,
			Attributes:  map[string]string{},
			ImageData:   fileData,
			ImageName:   fileHeader.Filename,
		}

		nftID, imageIPFS, metadataIPFS, assetName, err := h.service.ProcessBatchItem(
			c.Request.Context(),
			batchID,
			userID,
			privacy,
			item,
		)
		if err != nil {
			log.Printf("Batch %s: failed to process item %d (%s): %v", batchID, i+1, name, err)
			results = append(results, map[string]interface{}{
				"row":    i + 1,
				"name":   name,
				"status": "failed",
				"error":  err.Error(),
			})
			continue
		}

		uploadedCount++
		results = append(results, map[string]interface{}{
			"row":           i + 1,
			"nft_id":        nftID,
			"name":          name,
			"asset_name":    assetName,
			"image_ipfs":    imageIPFS,
			"metadata_ipfs": metadataIPFS,
			"status":        "uploaded",
		})
	}

	// Update batch job uploaded count
	h.service.db.Exec(c.Request.Context(), `
		UPDATE batch_jobs SET uploaded = $1, status = 'uploading', updated_at = NOW()
		WHERE id = $2
	`, uploadedCount, batchID)

	c.JSON(http.StatusOK, gin.H{
		"batch_id": batchID,
		"total":    len(files),
		"uploaded": uploadedCount,
		"items":    results,
		"message":  fmt.Sprintf("%d of %d NFTs uploaded to IPFS", uploadedCount, len(files)),
	})
}

// MintBatch godoc
// POST /api/batch/mint
// Mints all uploaded NFTs in a batch job sequentially
// Each NFT is minted in its own transaction
// Body: { "batch_id": "uuid" }
func (h *Handler) MintBatch(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		BatchID string `json:"batch_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id is required"})
		return
	}

	// Get custodial wallet mnemonic
	mnemonic, _, err := h.service.GetWalletForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallet: " + err.Error()})
		return
	}

	// Get all pending NFTs in this batch
	rows, err := h.service.db.Query(c.Request.Context(), `
		SELECT n.id, n.asset_name, n.metadata_ipfs, n.image_ipfs, n.royalties
		FROM nfts n
		JOIN batch_items bi ON bi.nft_id = n.id
		WHERE bi.batch_id = $1
		AND n.owner_id = $2
		AND n.status = 'pending'
		ORDER BY bi.row_order ASC
	`, body.BatchID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch batch items"})
		return
	}
	defer rows.Close()

	// Collect all items to mint
	type mintItem struct {
		NFTID        string
		AssetName    string
		MetadataIPFS string
		ImageIPFS    string
		Royalties    float64
	}
	var items []mintItem

	for rows.Next() {
		var item mintItem
		if err := rows.Scan(
			&item.NFTID, &item.AssetName,
			&item.MetadataIPFS, &item.ImageIPFS, &item.Royalties,
		); err != nil {
			continue
		}
		items = append(items, item)
	}
	rows.Close()

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no pending NFTs found in this batch"})
		return
	}

	// Update batch status to minting
	h.service.UpdateBatchStatus(c.Request.Context(), body.BatchID, "minting")

	// Mint each NFT sequentially
	results := []map[string]interface{}{}
	mintedCount := 0
	failedCount := 0

	for _, item := range items {
		err := h.service.MintBatchItem(
			c.Request.Context(),
			body.BatchID,
			item.NFTID,
			mnemonic,
			item.AssetName,
			item.MetadataIPFS,
			item.ImageIPFS,
			item.Royalties,
		)
		if err != nil {
			log.Printf("Batch %s: failed to mint NFT %s: %v", body.BatchID, item.NFTID, err)
			failedCount++
			results = append(results, map[string]interface{}{
				"nft_id": item.NFTID,
				"name":   item.AssetName,
				"status": "failed",
				"error":  err.Error(),
			})
			continue
		}

		mintedCount++
		results = append(results, map[string]interface{}{
			"nft_id": item.NFTID,
			"name":   item.AssetName,
			"status": "minted",
		})
	}

	// Update final batch status
	finalStatus := "completed"
	if failedCount > 0 && mintedCount == 0 {
		finalStatus = "failed"
	} else if failedCount > 0 {
		finalStatus = "completed" // partial success still completed
	}
	h.service.UpdateBatchStatus(c.Request.Context(), body.BatchID, finalStatus)

	c.JSON(http.StatusOK, gin.H{
		"batch_id": body.BatchID,
		"status":   finalStatus,
		"total":    len(items),
		"minted":   mintedCount,
		"failed":   failedCount,
		"items":    results,
		"message":  fmt.Sprintf("%d of %d NFTs minted successfully", mintedCount, len(items)),
	})
}

// GetBatchStatus godoc
// GET /api/batch/:id
// Returns the current status and progress of a batch job
func (h *Handler) GetBatchStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	batchID := c.Param("id")

	status, err := h.service.GetBatchStatus(c.Request.Context(), batchID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch job not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}
