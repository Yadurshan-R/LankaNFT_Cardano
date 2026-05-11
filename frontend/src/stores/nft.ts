import { defineStore } from 'pinia'
import { ref } from 'vue'
import { prepareMint, mintNFT, getMyNFTs } from '@/services/nft'

export const useNFTStore = defineStore('nft', () => {
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const myNFTs = ref<any[]>([])
  const lastMintedTx = ref<string | null>(null)

  // Step 1 — upload to IPFS + store in DB
  async function prepare(formData: FormData) {
    isLoading.value = true
    error.value = null
    try {
      const result = await prepareMint(formData)
      return result
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to prepare mint'
      return null
    } finally {
      isLoading.value = false
    }
  }

  // Step 2 — mint on blockchain
  async function mint(nftId: string) {
    isLoading.value = true
    error.value = null
    try {
      const result = await mintNFT(nftId)
      lastMintedTx.value = result.tx_hash
      return result
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to mint NFT'
      return null
    } finally {
      isLoading.value = false
    }
  }

  // Fetch user NFTs for dashboard
  async function fetchMyNFTs() {
    isLoading.value = true
    try {
      const result = await getMyNFTs()
      myNFTs.value = result.nfts || []
    } catch (err: any) {
      error.value = 'Failed to fetch NFTs'
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading,
    error,
    myNFTs,
    lastMintedTx,
    prepare,
    mint,
    fetchMyNFTs,
  }
})