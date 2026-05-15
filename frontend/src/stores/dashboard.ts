import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getWalletBalance, getNFTStats, getMyNFTs } from '@/services/dashboard'
import { getMyListings } from '@/services/listing'

export const useDashboardStore = defineStore('dashboard', () => {
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const walletAddress = ref<string>('')
  const lovelace = ref<string>('0')
  const totalNFTs = ref(0)
  const mintedNFTs = ref(0)
  const pendingNFTs = ref(0)
  const nfts = ref<any[]>([])
  const myListings = ref<any[]>([])

  const uniqueNFTs = computed(() => {
    const seen = new Set<string>()
    return nfts.value.filter((n) => {
      if (seen.has(n.id)) return false
      seen.add(n.id)
      return true
    })
  })

  const mintedOnly = computed(() =>
    uniqueNFTs.value.filter((n) => n.status === 'minted')
  )
  const listedOnly = computed(() =>
    uniqueNFTs.value.filter((n) => n.status === 'listed')
  )
  const pendingOnly = computed(() =>
    uniqueNFTs.value.filter((n) => n.status === 'pending')
  )

  async function loadDashboard() {
    isLoading.value = true
    error.value = null
    try {
      const [balanceData, statsData, nftsData, listingsData] = await Promise.all([
        getWalletBalance().catch(() => null),
        getNFTStats(),
        getMyNFTs(),
        getMyListings().catch(() => ({ listings: [] })),
      ])
      if (balanceData) {
        walletAddress.value = balanceData.wallet_address || ''
        lovelace.value = balanceData.lovelace || '0'
      }
      totalNFTs.value = statsData.total || 0
      mintedNFTs.value = statsData.minted || 0
      pendingNFTs.value = statsData.pending || 0
      nfts.value = nftsData.nfts || []
      myListings.value = listingsData.listings || []
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to load dashboard'
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading,
    error,
    walletAddress,
    lovelace,
    totalNFTs,
    mintedNFTs,
    pendingNFTs,
    nfts,
    myListings,
    uniqueNFTs,
    mintedOnly,
    listedOnly,
    pendingOnly,
    loadDashboard,
  }
})