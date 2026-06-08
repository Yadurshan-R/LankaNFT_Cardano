// ─────────────────────────────────────────────────────────────────────────────
// services/listing.ts
//
// Listing API service — wraps all marketplace backend calls.
//
// Two flows per action:
//   Custodial  → direct action endpoint (backend signs with mnemonic)
//   External   → unsigned endpoint → CIP-30 sign → submit → confirm
// ─────────────────────────────────────────────────────────────────────────────
import api from './api'

// ── Read ──────────────────────────────────────────────────────────────────────

export async function getAllListings() {
  const res = await api.get('/api/listing/all')
  return res.data
}

export async function getMyListings() {
  const res = await api.get('/api/listing/mine')
  return res.data
}

// ── Custodial List ────────────────────────────────────────────────────────────

export async function createListing(nftId: string, priceLovelace: number) {
  const res = await api.post('/api/listing/create', {
    nft_id:         nftId,
    price_lovelace: priceLovelace,
  })
  return res.data
}

// ── External wallet List ──────────────────────────────────────────────────────
// Returns unsigned CBOR + all data needed for confirmCreateListing.

export async function createListingUnsigned(nftId: string, priceLovelace: number) {
  const res = await api.post('/api/listing/create-unsigned', {
    nft_id:         nftId,
    price_lovelace: priceLovelace,
  })
  return res.data
  // Returns: { unsigned_cbor, nft_id, price_lovelace, nft_policy_id, nft_asset_name, royalty_policy_id }
}

export async function confirmCreateListing(
  nftId:           string,
  txHash:          string,
  priceLovelace:   number,
  nftPolicyId:     string,
  nftAssetName:    string,
  royaltyPolicyId: string,
) {
  const res = await api.post('/api/listing/confirm-create', {
    nft_id:            nftId,
    tx_hash:           txHash,
    price_lovelace:    priceLovelace,
    nft_policy_id:     nftPolicyId,
    nft_asset_name:    nftAssetName,
    royalty_policy_id: royaltyPolicyId,
  })
  return res.data
}

// ── Custodial Buy ─────────────────────────────────────────────────────────────

export async function buyListing(listingId: string) {
  const res = await api.post('/api/listing/buy', { listing_id: listingId })
  return res.data
}

// ── External wallet Buy ───────────────────────────────────────────────────────

export async function buyListingUnsigned(listingId: string) {
  const res = await api.post('/api/listing/buy-unsigned', { listing_id: listingId })
  return res.data
  // Returns: { unsigned_cbor, listing_id }
}

export async function confirmBuyListing(listingId: string, txHash: string) {
  const res = await api.post('/api/listing/confirm-buy', {
    listing_id: listingId,
    tx_hash:    txHash,
  })
  return res.data
}

// ── Custodial Cancel ──────────────────────────────────────────────────────────

export async function cancelListing(listingId: string) {
  const res = await api.post('/api/listing/cancel', { listing_id: listingId })
  return res.data
}

// ── External wallet Cancel ────────────────────────────────────────────────────

export async function cancelListingUnsigned(listingId: string) {
  const res = await api.post('/api/listing/cancel-unsigned', { listing_id: listingId })
  return res.data
  // Returns: { unsigned_cbor, listing_id }
}

export async function confirmCancelListing(listingId: string, txHash: string) {
  const res = await api.post('/api/listing/confirm-cancel', {
    listing_id: listingId,
    tx_hash:    txHash,
  })
  return res.data
}