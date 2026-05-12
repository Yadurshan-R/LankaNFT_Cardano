// ─────────────────────────────────────────────────────────────────────────────
// validators/minting_policy.ak
//
// CIP-68 + CIP-102 Compliant NFT Minting Policy
//
// This is the core minting validator for the Midnight NFT Platform.
// It supports two minting modes:
//
//   MintSingle     — One-shot policy for individual NFTs
//                    Parameterized by a UTxO reference (consumed at mint time)
//                    Each NFT gets its own unique policy ID
//                    Use for: standalone art pieces, 1/1 NFTs
//
//   MintCollection — Owner-signature policy for NFT collections
//                    Parameterized by the creator's public key hash
//                    All NFTs in the collection share one policy ID
//                    Use for: PFP collections, game item sets
//
// Both modes mint THREE tokens per NFT (CIP-68 + CIP-102 compliant):
//   (100) Reference NFT  → locked at reference_lock script with metadata datum
//   (222) User NFT       → sent to the creator/buyer wallet
//   (500) Royalty NFT    → locked at royalty_lock script with royalty datum
//                          (only minted ONCE per collection, not per NFT)
//
// Burn:
//   Both modes allow burning by checking all minted quantities are negative
//
// References:
//   CIP-68: https://cips.cardano.org/cip/CIP-68
//   CIP-102: https://cips.cardano.org/cip/CIP-102
// ─────────────────────────────────────────────────────────────────────────────

use aiken/collection/dict
use aiken/collection/list
use aiken/crypto.{VerificationKeyHash}
use cardano/assets.{PolicyId}
use cardano/transaction.{OutputReference, Transaction}
use cip68.{build_ref_name, build_royalty_name, build_user_name}

// ─── Redeemer ─────────────────────────────────────────────────────────────────

/// The redeemer tells the validator which action is being performed
pub type MintRedeemer {
  // ── Single NFT mint ───────────────────────────────────────────────────────
  // Mints one CIP-68 NFT pair (100 + 222) and one royalty NFT (500)
  // The one-shot UTxO is consumed to make this policy unique and non-repeatable
  // asset_name: the base name of the NFT (without any prefix)
  // include_royalty: whether to mint the (500) royalty NFT in this transaction
  //   true  = first mint in a collection (also mints royalty NFT)
  //   false = subsequent mints (royalty NFT already exists)
  MintSingle { asset_name: ByteArray, include_royalty: Bool }

  // ── Collection NFT mint ───────────────────────────────────────────────────
  // Mints one CIP-68 NFT pair (100 + 222) under a shared collection policy
  // The policy is controlled by the creator's public key — not one-shot
  // Multiple NFTs can be minted under the same policy (same collection)
  // asset_name: the base name of this specific NFT in the collection
  // include_royalty: same as MintSingle — only true for the first mint
  MintCollection { asset_name: ByteArray, include_royalty: Bool }

  // ── Burn ──────────────────────────────────────────────────────────────────
  // Destroys tokens — all quantities in this policy must be negative
  Burn
}

// ─── Single NFT Validator ─────────────────────────────────────────────────────

/// One-shot minting policy for individual NFTs
/// Parameterized by a specific UTxO reference
/// Once that UTxO is consumed, this policy ID can never be used again
/// This guarantees each single NFT has a completely unique policy
validator cip68_mint_single(
  // The UTxO that must be consumed to authorise this mint
  // Consuming it makes the policy one-shot and unique
  utxo_ref: OutputReference,
) {
  mint(redeemer: MintRedeemer, policy_id: PolicyId, self: Transaction) {
    let Transaction { inputs, mint, .. } = self

    when redeemer is {
      // ── Mint single NFT ─────────────────────────────────────────────────
      MintSingle { asset_name: name, include_royalty: with_royalty } -> {
        // Check 1: The one-shot UTxO must be consumed in this transaction
        // This is what makes the policy unique — it can never mint again
        let utxo_consumed =
          list.any(inputs, fn(input) { input.output_reference == utxo_ref })

        // Build the full CIP-68 token names with their label prefixes
        let ref_token_name = build_ref_name(name)
        let user_token_name = build_user_name(name)
        let royalty_token_name = build_royalty_name(name)

        // Check 2: Exactly 1 reference NFT (100) must be minted
        let ref_minted =
          assets.quantity_of(mint, policy_id, ref_token_name) == 1

        // Check 3: Exactly 1 user NFT (222) must be minted
        let user_minted =
          assets.quantity_of(mint, policy_id, user_token_name) == 1

        // Check 4: If include_royalty is true, exactly 1 royalty NFT (500)
        // must also be minted. If false, royalty NFT must NOT be minted here
        let royalty_check =
          if with_royalty {
            assets.quantity_of(mint, policy_id, royalty_token_name) == 1
          } else {
            assets.quantity_of(mint, policy_id, royalty_token_name) == 0
          }

        // All checks must pass
        utxo_consumed && ref_minted && user_minted && royalty_check
      }

      // MintCollection redeemer is not valid for the single NFT validator
      MintCollection { .. } ->
        fail @"Use cip68_mint_collection for collection NFTs"

      // ── Burn ────────────────────────────────────────────────────────────
      // All token quantities under this policy must be negative (burning)
      Burn ->
        assets.tokens(mint, policy_id)
          |> dict.values()
          |> list.all(fn(qty) { qty < 0 })
    }
  }

  // Reject any other script purpose
  else(_) {
    fail
  }
}

// ─── Collection NFT Validator ─────────────────────────────────────────────────

/// Owner-signature minting policy for NFT collections
/// Parameterized by the creator's public key hash
/// All NFTs in the same collection share this policy ID
/// The creator can mint multiple NFTs by signing each transaction
validator cip68_mint_collection(
  // The collection creator's public key hash
  // Every mint transaction must be signed by this wallet
  creator_pkh: VerificationKeyHash,
) {
  mint(redeemer: MintRedeemer, policy_id: PolicyId, self: Transaction) {
    let Transaction { extra_signatories, mint, .. } = self

    when redeemer is {
      // MintSingle redeemer is not valid for the collection validator
      MintSingle { .. } -> fail @"Use cip68_mint_single for individual NFTs"

      // ── Mint collection NFT ─────────────────────────────────────────────
      MintCollection { asset_name: name, include_royalty: with_royalty } -> {
        // Check 1: Transaction must be signed by the collection creator
        // This is the authorisation mechanism for collection minting
        let signed_by_creator = list.has(extra_signatories, creator_pkh)

        // Build the full CIP-68 token names with their label prefixes
        let ref_token_name = build_ref_name(name)
        let user_token_name = build_user_name(name)
        let royalty_token_name = build_royalty_name(name)

        // Check 2: Exactly 1 reference NFT (100) must be minted
        let ref_minted =
          assets.quantity_of(mint, policy_id, ref_token_name) == 1

        // Check 3: Exactly 1 user NFT (222) must be minted
        let user_minted =
          assets.quantity_of(mint, policy_id, user_token_name) == 1

        // Check 4: Royalty NFT (500) only on first mint of the collection
        let royalty_check =
          if with_royalty {
            assets.quantity_of(mint, policy_id, royalty_token_name) == 1
          } else {
            assets.quantity_of(mint, policy_id, royalty_token_name) == 0
          }

        // All checks must pass
        signed_by_creator && ref_minted && user_minted && royalty_check
      }

      // ── Burn ────────────────────────────────────────────────────────────
      // All token quantities under this policy must be negative (burning)
      Burn ->
        assets.tokens(mint, policy_id)
          |> dict.values()
          |> list.all(fn(qty) { qty < 0 })
    }
  }

  // Reject any other script purpose
  else(_) {
    fail
  }
}
