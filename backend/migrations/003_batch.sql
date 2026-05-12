-- Migration 003: Batch minting tables
-- Tracks groups of NFTs being minted together in a single batch job

-- Batch jobs table
-- One row per batch mint operation
CREATE TABLE IF NOT EXISTS batch_jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(20) DEFAULT 'pending',
    -- 'pending' → 'uploading' → 'minting' → 'completed' → 'failed'
    total       INTEGER NOT NULL DEFAULT 0,  -- total NFTs in this batch
    uploaded    INTEGER NOT NULL DEFAULT 0,  -- successfully uploaded to IPFS
    minted      INTEGER NOT NULL DEFAULT 0,  -- successfully minted on-chain
    failed      INTEGER NOT NULL DEFAULT 0,  -- failed items
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- Batch items table
-- One row per NFT inside a batch job
CREATE TABLE IF NOT EXISTS batch_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id      UUID NOT NULL REFERENCES batch_jobs(id) ON DELETE CASCADE,
    nft_id        UUID REFERENCES nfts(id) ON DELETE SET NULL,
    row_order     INTEGER NOT NULL,           -- position in the batch (1, 2, 3...)
    nft_name      VARCHAR(255) NOT NULL,
    description   TEXT,
    royalties     NUMERIC(5,2) DEFAULT 0,
    total_supply  INTEGER DEFAULT 1,
    attributes    JSONB DEFAULT '{}',         -- trait key/value pairs
    image_name    VARCHAR(255),               -- original uploaded filename
    status        VARCHAR(20) DEFAULT 'pending',
    -- 'pending' → 'uploaded' → 'minted' → 'failed'
    error_msg     TEXT,                       -- error message if failed
    created_at    TIMESTAMP DEFAULT NOW()
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_batch_items_batch  ON batch_items(batch_id);
CREATE INDEX IF NOT EXISTS idx_batch_jobs_owner   ON batch_jobs(owner_id);
CREATE INDEX IF NOT EXISTS idx_batch_jobs_status  ON batch_jobs(status);