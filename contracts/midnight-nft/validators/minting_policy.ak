use aiken/collection/dict
use aiken/collection/list
use aiken/primitive/bytearray
use cardano/assets.{PolicyId}
use cardano/transaction.{OutputReference, Transaction}

// ─── CIP-68 Token Label Prefixes ─────────────────────────────────────────────
// These are the standard prefixes defined in CIP-68
// (100) = Reference NFT — holds metadata datum, locked at script address
// (222) = User NFT — goes to the buyer/owner wallet
const reference_prefix: ByteArray = #"000643b0"

const user_prefix: ByteArray = #"001bc280"

// ─── Redeemer ─────────────────────────────────────────────────────────────────
// pub keyword makes this type visible to other modules
pub type MintRedeemer {
  // Mint: creates one reference NFT + one user NFT
  // asset_name is the base name without any prefix (e.g. "MidnightApe001")
  Mint { asset_name: ByteArray }
  // Burn: destroys both tokens in the pair
  Burn
}

// ─── Validator ────────────────────────────────────────────────────────────────
// One-shot minting policy for CIP-68 NFTs
// "One-shot" = parameterized by a specific UTxO reference
// Once that UTxO is consumed, this policy can never mint again
// This guarantees each NFT is unique and supply is fixed
validator cip68_mint(
  // The UTxO that must be consumed — locks the policy to a single mint
  utxo_ref: OutputReference,
) {
  mint(redeemer: MintRedeemer, policy_id: PolicyId, self: Transaction) {
    let Transaction { inputs, mint, .. } = self

    when redeemer is {
      // ── Mint action ───────────────────────────────────────────────────────
      Mint { asset_name: name } -> {
        // Check 1: The one-shot UTxO must be consumed in this transaction
        // This ensures the policy can only ever mint once
        let utxo_consumed =
          list.any(inputs, fn(input) { input.output_reference == utxo_ref })

        // Build the full token names with CIP-68 prefixes
        let ref_token_name = bytearray.concat(reference_prefix, name)
        let user_token_name = bytearray.concat(user_prefix, name)

        // Check 2: Exactly 1 reference NFT must be minted
        let ref_minted =
          assets.quantity_of(mint, policy_id, ref_token_name) == 1

        // Check 3: Exactly 1 user NFT must be minted
        let user_minted =
          assets.quantity_of(mint, policy_id, user_token_name) == 1

        // All three checks must pass
        utxo_consumed && ref_minted && user_minted
      }

      // ── Burn action ───────────────────────────────────────────────────────
      // Allow burning — all minted quantities must be negative
      Burn ->
        assets.tokens(mint, policy_id)
          |> dict.values()
          |> list.all(fn(qty) { qty < 0 })
    }
  }

  // Reject any other purpose (spend, withdraw, etc.)
  else(_) {
    fail
  }
}
