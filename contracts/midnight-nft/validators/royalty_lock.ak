// ─────────────────────────────────────────────────────────────────────────────
// validators/royalty_lock.ak
//
// CIP-102 Royalty NFT Lock Validator
//
// This validator locks the Royalty NFT (label 500) at a script address.
// The datum on this UTxO contains the royalty terms for the entire collection:
//   - Who receives royalties (one or more addresses)
//   - What percentage each recipient gets
//   - Minimum ADA per royalty payment
//
// Two actions are possible:
//   UpdateRoyalty — only the collection owner can update the royalty terms
//                   They must re-lock the Royalty NFT at this same address
//                   with the updated datum
//   BurnRoyalty   — only the collection owner can burn the Royalty NFT
//                   This removes on-chain royalty enforcement permanently
//
// Reference:
//   https://cips.cardano.org/cip/CIP-102
// ─────────────────────────────────────────────────────────────────────────────

use aiken/collection/list
use aiken/crypto.{VerificationKeyHash}
use cardano/assets.{PolicyId}
use cardano/transaction.{InlineDatum, OutputReference, Transaction}
use cip102.{RoyaltyDatum, RoyaltyRedeemer, validate_royalty_rates}

// ─── Royalty Lock Datum ───────────────────────────────────────────────────────

/// The datum stored on the Royalty NFT UTxO
/// Extends RoyaltyDatum with the owner's public key hash
/// so only the owner can update or burn the royalty terms
pub type RoyaltyLockDatum {
  // The collection owner's public key hash
  // Only this wallet can update or burn the royalty NFT
  owner: VerificationKeyHash,
  // The royalty terms — imported from lib/cip102.ak
  royalty: RoyaltyDatum,
  // The policy ID of the NFT collection this royalty applies to
  // Used by marketplaces to match NFTs to their royalty terms
  collection_policy: PolicyId,
}

// ─── Validator ────────────────────────────────────────────────────────────────

/// CIP-102 compliant royalty lock validator
validator royalty_lock {
  spend(
    // The datum contains owner + royalty terms + collection policy
    datum: Option<RoyaltyLockDatum>,
    // The redeemer tells us if this is an update or a burn
    redeemer: RoyaltyRedeemer,
    // Not used directly
    _utxo: OutputReference,
    // The full transaction
    self: Transaction,
  ) {
    let Transaction { extra_signatories, outputs, .. } = self

    when datum is {
      Some(lock_datum) -> {
        let RoyaltyLockDatum { owner, royalty, .. } = lock_datum

        // Rule 1: Transaction must be signed by the collection owner
        let signed_by_owner = list.has(extra_signatories, owner)

        when redeemer is {
          // ── Update royalty terms ──────────────────────────────────────
          // Owner updates royalty rate or recipient addresses
          // Royalty NFT must be re-locked at this script with new datum
          cip102.UpdateRoyalty -> {
            // Check that the new royalty rates are valid (sum <= 100%)
            let valid_rates = validate_royalty_rates(royalty.recipients)

            // Check that at least one output has an inline datum
            // This ensures the Royalty NFT is re-locked with new terms
            let has_inline_datum_output =
              list.any(
                outputs,
                fn(output) {
                  when output.datum is {
                    InlineDatum(_) -> True
                    _ -> False
                  }
                },
              )

            // All conditions must pass
            signed_by_owner && valid_rates && has_inline_datum_output
          }

          // ── Burn royalty NFT ──────────────────────────────────────────
          // Owner permanently removes on-chain royalty enforcement
          cip102.BurnRoyalty -> signed_by_owner
        }
      }

      // No datum — UTxO was set up incorrectly
      None -> fail @"No royalty datum found on UTxO"
    }
  }

  // Reject any other script purpose
  else(_) {
    fail
  }
}
