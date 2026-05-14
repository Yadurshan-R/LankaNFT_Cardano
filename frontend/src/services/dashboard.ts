import api from './api'

// Get wallet address and tADA balance
export async function getWalletBalance() {
  const res = await api.get('/api/nft/balance')
  return res.data
}

// Get NFT statistics (total, minted, pending)
export async function getNFTStats() {
  const res = await api.get('/api/nft/stats')
  return res.data
}

// Get all NFTs owned by logged in user
export async function getMyNFTs() {
  const res = await api.get('/api/nft/my-nfts')
  return res.data
}