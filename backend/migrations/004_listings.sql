-- Migration 004: NFT Marketplace Listings
-- Tracks NFTs listed for sale on the Midnight marketplace

CREATE TABLE IF NOT EXISTS listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nft_id          UUID NOT NULL REFERENCES nfts(id) ON DELETE CASCADE,
    seller_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- On-chain data
    listing_tx_hash VARCHAR(64),          -- tx that locked NFT at marketplace
    script_utxo     VARCHAR(200),         -- txHash#index of the listing UTxO
    price_lovelace  BIGINT NOT NULL,      -- asking price in lovelace

    -- NFT identifiers (cached for quick lookup)
    nft_policy_id   VARCHAR(56) NOT NULL,
    nft_asset_name  VARCHAR(255) NOT NULL,
    royalty_policy_id VARCHAR(56),        -- policy ID of (500) royalty token

    -- Status
    status          VARCHAR(20) DEFAULT 'active',
    -- 'active' → 'sold' → 'cancelled'

    -- Sale details (filled when sold)
    sale_tx_hash    VARCHAR(64),
    buyer_id        UUID REFERENCES users(id),

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_listings_nft       ON listings(nft_id);
CREATE INDEX IF NOT EXISTS idx_listings_seller    ON listings(seller_id);
CREATE INDEX IF NOT EXISTS idx_listings_status    ON listings(status);
CREATE INDEX IF NOT EXISTS idx_listings_policy    ON listings(nft_policy_id);