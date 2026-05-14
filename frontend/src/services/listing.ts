import api from './api'

export async function getAllListings() {
  const res = await api.get('/api/listing/all')
  return res.data
}

export async function getMyListings() {
  const res = await api.get('/api/listing/mine')
  return res.data
}

export async function createListing(nftId: string, priceLovelace: number) {
  const res = await api.post('/api/listing/create', {
    nft_id: nftId,
    price_lovelace: priceLovelace,
  })
  return res.data
}

export async function buyListing(listingId: string) {
  const res = await api.post('/api/listing/buy', { listing_id: listingId })
  return res.data
}

export async function cancelListing(listingId: string) {
  const res = await api.post('/api/listing/cancel', { listing_id: listingId })
  return res.data
}