// ─────────────────────────────────────────────────────────────────────────────
// stores/dashboard.ts
//
// Dashboard store — NFT collection, stats, wallet balance, listings
//
// Two user types handled here:
//   custodial — email login users. NFTs stored in our DB. Balance from
//               /api/nft/balance (custodial wallet address via Blockfrost).
//
//   external  — CIP-30 wallet users (Nami/Eternl/Lace). NFTs fetched
//               from Blockfrost directly via /api/nft/external-assets.
//               Balance fetched from /api/nft/balance (backend handles
//               both wallet types via external_wallets table lookup).
//               Stats computed from the returned NFT array.
//
// loadDashboard() is called:
//   - On app mount (App.vue, after checkAuth resolves)
//   - After any mutation (mint, list, buy, cancel, transfer)
// ─────────────────────────────────────────────────────────────────────────────
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getWalletBalance, getNFTStats, getMyNFTs } from '@/services/dashboard'
import { getMyListings } from '@/services/listing'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'

export const useDashboardStore = defineStore('dashboard', () => {

  // ── State ────────────────────────────────────────────────────────────────────
  const isLoading     = ref(false)
  const error         = ref<string | null>(null)
  const walletAddress = ref<string>('')
  const lovelace      = ref<string>('0')
  const totalNFTs     = ref(0)
  const mintedNFTs    = ref(0)
  const pendingNFTs   = ref(0)
  const nfts          = ref<any[]>([])
  const myListings    = ref<any[]>([])

  // ── Computed ──────────────────────────────────────────────────────────────────
  // Deduplicate by ID — prevents double-rendering during rapid reloads
  const uniqueNFTs = computed(() => {
    const seen = new Set<string>()
    return nfts.value.filter((n) => {
      if (seen.has(n.id)) return false
      seen.add(n.id)
      return true
    })
  })

  const mintedOnly  = computed(() => uniqueNFTs.value.filter((n) => n.status === 'minted'))
  const listedOnly  = computed(() => uniqueNFTs.value.filter((n) => n.status === 'listed'))
  const pendingOnly = computed(() => uniqueNFTs.value.filter((n) => n.status === 'pending'))

  // ── loadDashboard ─────────────────────────────────────────────────────────────
  async function loadDashboard() {
    isLoading.value = true
    error.value     = null

    const auth = useAuthStore()

    try {
      if (auth.walletType === 'external') {
        await loadExternalDashboard(auth)
      } else {
        await loadCustodialDashboard()
      }
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to load dashboard'
    } finally {
      isLoading.value = false
    }
  }

  // ── Custodial flow ────────────────────────────────────────────────────────────
  // NFTs from DB, balance from custodial wallet via Blockfrost
  async function loadCustodialDashboard() {
    const [balanceData, statsData, nftsData, listingsData] = await Promise.all([
      getWalletBalance().catch(() => null),
      getNFTStats(),
      getMyNFTs(),
      getMyListings().catch(() => ({ listings: [] })),
    ])

    if (balanceData) {
      walletAddress.value = balanceData.wallet_address || ''
      lovelace.value      = balanceData.lovelace || '0'
    }

    totalNFTs.value   = statsData.total   || 0
    mintedNFTs.value  = statsData.minted  || 0
    pendingNFTs.value = statsData.pending  || 0
    nfts.value        = nftsData.nfts     || []
    myListings.value  = listingsData.listings || []
  }

  // ── External wallet flow ──────────────────────────────────────────────────────
  // NFTs from Blockfrost (not DB), balance from external wallet via Blockfrost.
  // Stats are computed from the returned NFT array — no separate stats endpoint.
  async function loadExternalDashboard(auth: ReturnType<typeof useAuthStore>) {
    const [balanceData, externalAssetsRes, listingsData] = await Promise.all([
      // Balance works for both wallet types — backend checks external_wallets table
      getWalletBalance().catch(() => null),

      // External assets — fetched from Blockfrost for this wallet address
      api.get('/api/nft/external-assets').catch(() => ({ data: { nfts: [] } })),

      getMyListings().catch(() => ({ listings: [] })),
    ])

    // Wallet address comes from auth store (set during login)
    walletAddress.value = auth.walletAddress || balanceData?.wallet_address || ''
    lovelace.value      = balanceData?.lovelace || '0'

    const externalNFTs = externalAssetsRes.data?.nfts || []

    // Stats computed from the Blockfrost asset list
    // External wallets don't have pending NFTs — everything is already on-chain
    totalNFTs.value   = externalNFTs.length
    mintedNFTs.value  = externalNFTs.length
    pendingNFTs.value = 0

    // Merge external Blockfrost NFTs with any platform-minted NFTs in DB
    // A user might have minted some NFTs here and also have external ones
    let platformNFTs: any[] = []
    try {
      const platformRes = await getMyNFTs()
      platformNFTs = platformRes.nfts || []
    } catch {
      // No platform NFTs — that's fine
    }

    // Combine: platform NFTs first (have full metadata), external ones after
    // Deduplicate by policy_id + asset_name to avoid showing same NFT twice
    const seen = new Set<string>()
    const combined = [...platformNFTs, ...externalNFTs].filter((n) => {
      const key = `${n.policy_id}:${n.asset_name || n.user_token_name}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })

    nfts.value       = combined
    totalNFTs.value  = combined.length
    mintedNFTs.value = combined.length

    myListings.value = listingsData.listings || []
  }

  // ── Reset ─────────────────────────────────────────────────────────────────────
  // Called on logout to clear stale data
  function reset() {
    walletAddress.value = ''
    lovelace.value      = '0'
    totalNFTs.value     = 0
    mintedNFTs.value    = 0
    pendingNFTs.value   = 0
    nfts.value          = []
    myListings.value    = []
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
    reset,
  }
})