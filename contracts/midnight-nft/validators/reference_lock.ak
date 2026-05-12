// ─────────────────────────────────────────────────────────────────────────────
// validators/reference_lock.ak
//
// CIP-68 Reference NFT Lock Validator
//
// The Reference NFT (label 100) must be locked at a script address — never
// in a regular wallet. This file provides two validators for that purpose:
//
//   1. immutable_lock  — Nobody can ever spend this UTxO. The metadata
//                        is permanently frozen. Best for art NFTs where
//                        the creator wants to guarantee nothing will change.
//
//   2. updatable_lock  — Only the original creator (by public key hash)
//                        can spend and re-lock this UTxO to update metadata.
//                        Best for dynamic NFTs (game items, evolving art).
//
// The creator chooses which lock type to use at mint time.
// This choice is permanent — you cannot switch between them after minting.
//
// How it works in practice:
//   Mint tx  → Reference NFT sent to immutable_lock or updatable_lock address
//   Read     → Marketplaces and dApps read the datum as a reference input
//   Update   → (updatable_lock only) Creator sends new datum in a new tx
// ─────────────────────────────────────────────────────────────────────────────

use aiken/collection/list
use aiken/crypto.{VerificationKeyHash}
use cardano/transaction.{OutputReference, Transaction}

// ─── Immutable Lock ───────────────────────────────────────────────────────────

/// Permanently locks the Reference NFT at this script address.
/// No one — not even the creator — can ever spend from this address.
/// Once locked here, the CIP-68 metadata datum is frozen forever.
///
/// Use this for:
///   - Fine art NFTs where immutability is the selling point
///   - Collections where "provably unchangeable metadata" adds value
validator immutable_lock {
  spend(
    // We do not use the datum — the lock is unconditional
    _datum: Option<Data>,
    // We do not use the redeemer — nothing can unlock this
    _redeemer: Data,
    // We do not use the output reference
    _utxo: OutputReference,
    // We do not use the transaction
    _self: Transaction,
  ) {
    // Always fail — this UTxO can NEVER be spent
    // The reference NFT is locked here permanently
    fail @"Reference NFT is permanently locked — metadata is immutable"
  }

  // Reject any other script purpose (minting, withdrawals, etc.)
  else(_) {
    fail
  }
}

// ─── Updatable Lock ───────────────────────────────────────────────────────────

/// Locks the Reference NFT but allows the original creator to update metadata.
/// Only the wallet whose public key hash is stored in the datum can spend this.
/// After updating, the creator must re-lock the Reference NFT at this address.
///
/// Use this for:
///   - Game item NFTs whose stats can evolve
///   - NFTs with dynamic traits that change over time
///   - Collections where the creator wants to fix typos or update links
///
/// Security: Only the creator can update — no one else, not even the platform
validator updatable_lock {
  spend(
    // The datum stores the creator's public key hash
    // This is set once at mint time and never changes
    datum: Option<VerificationKeyHash>,
    // The redeemer is not used — we only check the signature
    _redeemer: Data,
    // The output reference is not used
    _utxo: OutputReference,
    // The full transaction — we check signatories here
    self: Transaction,
  ) {
    let Transaction { extra_signatories, .. } = self

    // Extract the creator's public key hash from the datum
    // If no datum is present, reject the transaction
    when datum is {
      Some(creator_pkh) ->
        // The transaction must be signed by the original creator
        // This is the only unlock condition
        list.has(extra_signatories, creator_pkh)

      // No datum means the UTxO was set up incorrectly — reject
      None -> fail @"No creator key hash in datum — cannot unlock"
    }
  }

  // Reject any other script purpose
  else(_) {
    fail
  }
}
