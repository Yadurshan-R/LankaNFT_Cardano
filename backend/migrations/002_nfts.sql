-- Migration 002: NFT tables
-- Stores minted NFTs and their metadata

-- Main NFTs table
-- Stores every NFT minted on the platform
CREATE TABLE IF NOT EXISTS nfts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- CIP-68 token info
    policy_id           VARCHAR(64) NOT NULL,   -- Cardano policy ID (hex)
    asset_name          VARCHAR(64) NOT NULL,   -- base name without prefix
    ref_token_name      VARCHAR(128) NOT NULL,  -- (100) prefixed name
    user_token_name     VARCHAR(128) NOT NULL,  -- (222) prefixed name

    -- Metadata
    nft_name            VARCHAR(255) NOT NULL,  -- display name
    description         TEXT,
    image_ipfs          VARCHAR(255) NOT NULL,  -- ipfs:// URI of image
    metadata_ipfs       VARCHAR(255) NOT NULL,  -- ipfs:// URI of metadata JSON

    -- Mint settings
    royalties           NUMERIC(5,2) DEFAULT 0, -- percentage e.g. 5.00
    total_supply        INTEGER DEFAULT 1,
    privacy             VARCHAR(10) DEFAULT 'public', -- 'public' or 'private'
    mint_type           VARCHAR(10) DEFAULT 'standard', -- 'standard' or 'lazy'

    -- Cardano transaction
    tx_hash             VARCHAR(128),           -- tx hash after minting
    utxo_ref            VARCHAR(255),           -- one-shot UTxO used for policy

    -- Status
    status              VARCHAR(20) DEFAULT 'pending',
    -- 'pending' → 'minting' → 'minted' → 'listed' → 'sold'

    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);

-- Attributes table
-- Stores CIP-68 trait attributes (key/value pairs per NFT)
CREATE TABLE IF NOT EXISTS nft_attributes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nft_id      UUID NOT NULL REFERENCES nfts(id) ON DELETE CASCADE,
    trait_type  VARCHAR(100) NOT NULL,  -- e.g. "Background"
    value       VARCHAR(255) NOT NULL,  -- e.g. "Blue"
    created_at  TIMESTAMP DEFAULT NOW()
);

-- IPFS pins table
-- Tracks everything we have pinned to Pinata
CREATE TABLE IF NOT EXISTS ipfs_pins (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nft_id      UUID REFERENCES nfts(id) ON DELETE SET NULL,
    ipfs_hash   VARCHAR(255) NOT NULL UNIQUE, -- Qm... or bafy... hash
    pin_type    VARCHAR(20) NOT NULL,          -- 'image' or 'metadata'
    file_name   VARCHAR(255),
    file_size   BIGINT,
    pinned_at   TIMESTAMP DEFAULT NOW()
);

-- Index for fast lookup by owner
CREATE INDEX IF NOT EXISTS idx_nfts_owner ON nfts(owner_id);

-- Index for fast lookup by policy
CREATE INDEX IF NOT EXISTS idx_nfts_policy ON nfts(policy_id);

-- Index for fast lookup by status
CREATE INDEX IF NOT EXISTS idx_nfts_status ON nfts(status);