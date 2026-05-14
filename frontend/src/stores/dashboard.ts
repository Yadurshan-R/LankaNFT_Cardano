import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getWalletBalance, getNFTStats, getMyNFTs } from '@/services/dashboard'

export const useDashboardStore = defineStore('dashboard', () => {
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Wallet
  const walletAddress = ref<string>('')
  const lovelace = ref<string>('0')

  // Stats
  const totalNFTs = ref(0)
  const mintedNFTs = ref(0)
  const pendingNFTs = ref(0)

  // NFTs
  const nfts = ref<any[]>([])

  // Convert lovelace to ADA (1 ADA = 1,000,000 lovelace)
  function adaBalance(): string {
    const ada = parseInt(lovelace.value) / 1_000_000
    return ada.toLocaleString('en-US', { maximumFractionDigits: 2 })
  }

  // Load everything for the dashboard
  async function loadDashboard() {
    isLoading.value = true
    error.value = null

    try {
      // Run all 3 calls in parallel for speed
      const [balanceData, statsData, nftsData] = await Promise.all([
        getWalletBalance().catch(() => null),
        getNFTStats(),
        getMyNFTs(),
      ])

      if (balanceData) {
        walletAddress.value = balanceData.wallet_address || ''
        lovelace.value = balanceData.lovelace || '0'
      }

      totalNFTs.value = statsData.total || 0
      mintedNFTs.value = statsData.minted || 0
      pendingNFTs.value = statsData.pending || 0

      nfts.value = nftsData.nfts || []
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
    adaBalance,
    loadDashboard,
  }
})