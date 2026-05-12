// ─────────────────────────────────────────────────────────────────────────────
// lib/cip102.ak
//
// CIP-102 Royalty Datum Metadata Standard — Helper Library
//
// CIP-102 improves on CIP-27 by storing royalty information ON-CHAIN
// inside a datum attached to a special Royalty NFT (label 500).
//
// How it works:
//   1. When an NFT collection is created, a (500) Royalty NFT is minted
//   2. That Royalty NFT is locked at the royalty_lock script address
//   3. The datum on that UTxO contains: rate + recipient address(es)
//   4. Marketplaces read this datum as a reference input to enforce royalties
//   5. The collection owner can update royalty terms by spending + relocking
//
// Key advantage over CIP-27:
//   - Royalties are verifiable by smart contracts (trustless)
//   - Multiple royalty recipients supported (e.g. creator + platform split)
//   - Royalty terms can be updated by the owner without re-minting NFTs
//
// Reference:
//   https://cips.cardano.org/cip/CIP-102
// ─────────────────────────────────────────────────────────────────────────────

use aiken/collection/list

// ─── Types ───────────────────────────────────────────────────────────────────

/// A single royalty recipient
/// address: the Cardano address that receives royalty payments (as raw bytes)
/// rate:    royalty percentage expressed as a fraction of 1,000,000
///          e.g. 50000 = 5% (50000 / 1000000 = 0.05)
pub type RoyaltyRecipient {
  address: ByteArray,
  rate: Int,
}

/// The full royalty datum stored inside the (500) Royalty NFT UTxO
/// recipients: list of addresses and their royalty rates
///             rates must sum to <= 1,000,000 (100%)
/// min_ada:    minimum ADA that must be included in a royalty payment
pub type RoyaltyDatum {
  recipients: List<RoyaltyRecipient>,
  min_ada: Int,
}

// ─── Redeemer ─────────────────────────────────────────────────────────────────

/// Actions that can be performed on the royalty lock script
pub type RoyaltyRedeemer {
  // Owner updates the royalty terms (rate or recipient addresses)
  UpdateRoyalty
  // Owner burns the royalty NFT (removes royalty enforcement)
  BurnRoyalty
}

// ─── Validation Helpers ───────────────────────────────────────────────────────

/// Validate that all royalty rates sum to at most 1,000,000 (100%)
/// This prevents creators from accidentally setting > 100% royalties
pub fn validate_royalty_rates(recipients: List<RoyaltyRecipient>) -> Bool {
  let total_rate = list.foldr(recipients, 0, fn(r, acc) { acc + r.rate })
  total_rate <= 1_000_000
}

/// Convert a percentage to the CIP-102 rate format
/// e.g. percentage_to_rate(5) => 50000  (5% = 50000 / 1000000)
pub fn percentage_to_rate(percentage: Int) -> Int {
  percentage * 10_000
}

/// Convert a CIP-102 rate back to a human-readable percentage
/// e.g. rate_to_percentage(50000) => 5
pub fn rate_to_percentage(rate: Int) -> Int {
  rate / 10_000
}
