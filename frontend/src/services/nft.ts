import api from './api'

// Upload asset and metadata to IPFS, store NFT record in DB
export async function prepareMint(formData: FormData) {
  const res = await api.post('/api/nft/prepare-mint', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data
}

// Mint the NFT on Cardano blockchain
export async function mintNFT(nftId: string) {
  const res = await api.post('/api/nft/mint', { nft_id: nftId })
  return res.data
}

// Get all NFTs owned by logged in user
export async function getMyNFTs() {
  const res = await api.get('/api/nft/my-nfts')
  return res.data
}