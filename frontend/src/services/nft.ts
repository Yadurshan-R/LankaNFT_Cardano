// ─────────────────────────────────────────────────────────────────────────────
// services/nft.ts
//
// NFT API service — wraps all NFT-related backend calls.
//
// Two mint flows:
//   Custodial  → prepareMint() → mintNFT()
//   External   → prepareMint() → mintNFTUnsigned() → [CIP-30 sign] → confirmMint()
//
// Two transfer flows:
//   Custodial  → transferNFT()
//   External   → transferNFTUnsigned() → [CIP-30 sign] → confirmTransfer()
// ─────────────────────────────────────────────────────────────────────────────
import api from './api'

// Upload asset and metadata to IPFS, store NFT record in DB.
// Same for both custodial and external wallet users — IPFS upload
// doesn't require signing.
export async function prepareMint(formData: FormData) {
  const res = await api.post('/api/nft/prepare-mint', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data
}

// ── Custodial mint ────────────────────────────────────────────────────────────
// Backend decrypts mnemonic, sidecar signs and submits.
export async function mintNFT(nftId: string) {
  const res = await api.post('/api/nft/mint', { nft_id: nftId })
  return res.data
}

// ── External wallet mint ──────────────────────────────────────────────────────
// Returns unsigned CBOR — frontend signs with CIP-30 then calls confirmMint.
export async function mintNFTUnsigned(nftId: string, walletUtxos: string[] = []) {
  const res = await api.post('/api/nft/mint-unsigned', { 
    nft_id:       nftId,
    wallet_utxos: walletUtxos,
  })
  return res.data // { unsigned_cbor, policy_id, nft_id, asset_name }
}

// Called after external wallet signs and submits the mint tx.
// Updates DB: sets status='minted', tx_hash, policy_id.
export async function confirmMint(nftId: string, txHash: string, policyId: string) {
  const res = await api.post('/api/nft/confirm-mint', {
    nft_id:    nftId,
    tx_hash:   txHash,
    policy_id: policyId,
  })
  return res.data
}

// ── Get NFTs ──────────────────────────────────────────────────────────────────
export async function getMyNFTs() {
  const res = await api.get('/api/nft/my-nfts')
  return res.data
}

// ── Custodial transfer ────────────────────────────────────────────────────────
export async function transferNFT(nftId: string, recipientAddress: string) {
  const res = await api.post('/api/nft/transfer', {
    nft_id:            nftId,
    recipient_address: recipientAddress,
  })
  return res.data
}

// ── External wallet transfer ──────────────────────────────────────────────────
// Returns unsigned CBOR — frontend signs with CIP-30 then calls confirmTransfer.
export async function transferNFTUnsigned(nftId: string, recipientAddress: string) {
  const res = await api.post('/api/nft/transfer-unsigned', {
    nft_id:            nftId,
    recipient_address: recipientAddress,
  })
  return res.data // { unsigned_cbor, nft_id, recipient_address }
}

// Called after external wallet signs and submits the transfer tx.
export async function confirmTransfer(nftId: string, txHash: string, recipientAddress: string) {
  const res = await api.post('/api/nft/confirm-transfer', {
    nft_id:            nftId,
    tx_hash:           txHash,
    recipient_address: recipientAddress,
  })
  return res.data
}

// ── Submit Signed Transaction ─────────────────────────────────────────────────
// Submit a signed transaction to Cardano via the backend.
// 
// WHY two parameters:
//   Lace's signTx() returns only the WITNESS SET, not the full transaction.
//   The backend needs the original unsigned tx (body) + the witness set
//   to assemble a complete, valid transaction before submitting.
//
// Returns the tx hash on success.
export async function submitSignedTx(unsignedCbor: string, witnessCbor: string): Promise<string> {
  const res = await api.post('/api/submit', {
    unsigned_cbor: unsignedCbor,
    witness_cbor:  witnessCbor,
  })
  return res.data.tx_hash
}