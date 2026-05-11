-- Migration 001: Initial schema
-- Creates the core tables needed for auth and custodial wallets

-- Users table
-- Stores both email OTP users and external wallet users
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) UNIQUE,                -- null for external wallet users
    wallet_type VARCHAR(20) NOT NULL,               -- 'custodial' or 'external'
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- Custodial wallets table
-- Stores AES-256-GCM encrypted mnemonics for email OTP users
-- The mnemonic is NEVER stored in plaintext
CREATE TABLE IF NOT EXISTS custodial_wallets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address      VARCHAR(255) NOT NULL UNIQUE, -- Cardano bech32 address
    encrypted_mnemonic  TEXT NOT NULL,                -- AES-256-GCM encrypted
    created_at          TIMESTAMP DEFAULT NOW()
);

-- External wallets table
-- Stores wallet addresses for Lace/Nami/Eternl users
-- We never hold their private keys
CREATE TABLE IF NOT EXISTS external_wallets (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address VARCHAR(255) NOT NULL UNIQUE, -- Cardano bech32 address
    wallet_name    VARCHAR(50),                  -- 'lace', 'nami', 'eternl'
    created_at     TIMESTAMP DEFAULT NOW()
);

-- OTP codes table
-- Stores one-time passwords for email auth
-- Codes expire after OTP_EXPIRY_MINUTES and are deleted after use
CREATE TABLE IF NOT EXISTS otp_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      VARCHAR(255) NOT NULL,
    code       VARCHAR(6) NOT NULL,       -- 6 digit code
    expires_at TIMESTAMP NOT NULL,        -- set to NOW() + OTP_EXPIRY_MINUTES
    used       BOOLEAN DEFAULT FALSE,     -- marked true after verification
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for fast OTP lookup by email
CREATE INDEX IF NOT EXISTS idx_otp_codes_email ON otp_codes(email);

-- Index for fast user lookup by email
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);