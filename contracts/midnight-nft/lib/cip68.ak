// ─────────────────────────────────────────────────────────────────────────────
// lib/cip68.ak
//
// CIP-68 Datum Metadata Standard — Helper Library
//
// CIP-68 defines how NFT metadata is stored on-chain using datums.
// Every NFT under this standard consists of TWO tokens under the same policy:
//
//   (100) Reference NFT  — locked at a script address, holds the metadata datum
//   (222) User NFT       — lives in the owner's wallet, represents ownership
//
// The token name structure is:
//   full_token_name = label_prefix ++ base_asset_name
//
// Reference:
//   https://cips.cardano.org/cip/CIP-68
// ─────────────────────────────────────────────────────────────────────────────

use aiken/primitive/bytearray

// ─── Token Label Prefixes ─────────────────────────────────────────────────────
// These are the 4-byte hex prefixes defined in CIP-68 (via CIP-67 label registry)

/// (100) Reference NFT prefix — 0x000643b0
/// This token holds the on-chain metadata datum
/// It must be locked at a script address (never in a wallet)
pub const reference_prefix: ByteArray = #"000643b0"

/// (222) User NFT prefix — 0x001bc280
/// This token represents ownership and lives in the user's wallet
pub const user_prefix: ByteArray = #"001bc280"

/// (500) Royalty NFT prefix — 0x001f4d70
/// Used by CIP-102 to store royalty information on-chain
/// Locked at the royalty_lock script address
pub const royalty_prefix: ByteArray = #"001f4d70"

// ─── Token Name Builders ──────────────────────────────────────────────────────

/// Build the full reference NFT token name
/// Example: build_ref_name("MyNFT") => 0x000643b0 ++ "MyNFT"
pub fn build_ref_name(base_name: ByteArray) -> ByteArray {
  bytearray.concat(reference_prefix, base_name)
}

/// Build the full user NFT token name
/// Example: build_user_name("MyNFT") => 0x001bc280 ++ "MyNFT"
pub fn build_user_name(base_name: ByteArray) -> ByteArray {
  bytearray.concat(user_prefix, base_name)
}

/// Build the full royalty NFT token name
/// Example: build_royalty_name("MyNFT") => 0x001f4d70 ++ "MyNFT"
pub fn build_royalty_name(base_name: ByteArray) -> ByteArray {
  bytearray.concat(royalty_prefix, base_name)
}
