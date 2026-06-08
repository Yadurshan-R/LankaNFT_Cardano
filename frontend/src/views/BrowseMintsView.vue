<template>
  <div class="browse">

    <router-link to="/" class="back-link" title="Back to Dashboard">
      <ArrowLeft :size="16" />
    </router-link>

    <div class="page-header">
      <div>
        <h1 class="page-title">Browse Mints</h1>
        <p class="page-sub">Discover and collect NFTs on Cardano Preprod.</p>
      </div>
    </div>

    <div class="filter-bar">
      <div class="search-wrap">
        <Search :size="15" class="search-icon" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by name..."
          class="search-input"
        />
      </div>
      <div class="filter-right">
        <div class="select-wrap">
          <SlidersHorizontal :size="14" class="select-icon" />
          <select v-model="sortBy" class="sort-select">
            <option value="newest">Newest first</option>
            <option value="price_asc">Price: Low to High</option>
            <option value="price_desc">Price: High to Low</option>
          </select>
        </div>
        <div class="price-wrap">
          <span class="price-label">Max (₳)</span>
          <input
            v-model.number="maxPrice"
            type="number"
            min="0"
            placeholder="Any"
            class="price-input"
          />
        </div>
        <button v-if="hasFilters" class="btn-clear" @click="clearFilters">
          <X :size="13" /> Clear
        </button>
      </div>
    </div>

    <div v-if="loading" class="listings-grid">
      <div v-for="n in 8" :key="n" class="listing-card listing-card--skeleton">
        <div class="sk-image" />
        <div class="sk-body">
          <div class="sk-line sk-line--title" />
          <div class="sk-line sk-line--price" />
        </div>
      </div>
    </div>

    <div v-else-if="filteredListings.length === 0" class="empty-state">
      <Store :size="40" color="#ddd" />
      <p class="empty-title">
        {{ listings.length === 0 ? 'No listings yet' : 'No results found' }}
      </p>
      <p class="empty-desc">
        {{ listings.length === 0
          ? 'Be the first to list an NFT for sale.'
          : 'Try adjusting your search or filters.' }}
      </p>
      <button v-if="listings.length > 0" class="btn-outline" @click="clearFilters">
        Clear filters
      </button>
      <router-link v-else to="/">
        <button class="btn-primary">Go to Dashboard</button>
      </router-link>
    </div>

    <div v-else class="listings-grid">
      <div
        v-for="listing in filteredListings"
        :key="listing.id"
        class="listing-card"
        @click="selectedListing = listing"
      >
        <div class="listing-img-wrap">
          <img
            v-if="imageUrl(listing.image_ipfs)"
            :src="imageUrl(listing.image_ipfs)"
            :alt="listing.nft_name"
            class="listing-img"
          />
          <div v-else class="listing-img-placeholder">
            <ImageIcon :size="28" color="#ddd" />
          </div>
          <div class="listing-hover">
            <span class="hover-label">
              <Eye :size="13" /> View Details
            </span>
          </div>
        </div>

        <div class="listing-body">
          <p class="listing-name">{{ listing.nft_name }}</p>
          <p class="listing-price">{{ adaAmount(listing.price_lovelace) }} ₳</p>
        </div>
      </div>
    </div>

    <div v-if="!loading && filteredListings.length > 0" class="results-count">
      {{ filteredListings.length }} of {{ listings.length }} listings
    </div>

    <ListingDetailPanel
      :listing="selectedListing"
      :loading="buyingId !== null"
      @confirm="handleBuy(selectedListing)"
      @close="selectedListing = null"
    />

    <Transition name="toast">
      <div v-if="errorMsg" class="toast toast--error">
        <AlertCircle :size="15" /> {{ errorMsg }}
      </div>
    </Transition>

    <BuySuccessModal
      :show="!!buySuccessData"
      :tx-hash="buySuccessData?.txHash || ''"
      :nft-name="buySuccessData?.nftName || ''"
      :image-ipfs="buySuccessData?.imageIpfs"
      :price-lovelace="buySuccessData?.priceLovelace"
      :policy-id="buySuccessData?.policyId"
      :asset-name="buySuccessData?.assetName"
      @close="buySuccessData = null"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  AlertCircle, ArrowLeft, CheckCircle, Eye,
  Image as ImageIcon, Search, SlidersHorizontal,
  Store, X,
} from 'lucide-vue-next'
import {
  getAllListings, buyListing,
  buyListingUnsigned, confirmBuyListing,
} from '@/services/listing'
import { submitSignedTx }   from '@/services/nft'
import { useAuthStore }     from '@/stores/auth'
import { useWalletSession } from '@/composables/useWalletSession'
import { useDashboardStore } from '@/stores/dashboard'
import ListingDetailPanel from '@/components/marketplace/ListingDetailPanel.vue'
import BuySuccessModal    from '@/components/marketplace/BuySuccessModal.vue'

// ── State ─────────────────────────────────────────────────────────────────────
const listings        = ref<any[]>([])
const loading         = ref(true)
const buyingId        = ref<string | null>(null)
const selectedListing = ref<any | null>(null)
const errorMsg        = ref('')

const buySuccessData = ref<{
  txHash:        string
  nftName:       string
  imageIpfs?:    string
  priceLovelace?: number
  policyId?:     string
  assetName?:    string
} | null>(null)

const auth          = useAuthStore()
const walletSession = useWalletSession()
const dashboard     = useDashboardStore()

// Filters
const searchQuery = ref('')
const sortBy      = ref('newest')
const maxPrice    = ref<number | null>(null)

// ── Computed ──────────────────────────────────────────────────────────────────
const hasFilters = computed(() =>
  searchQuery.value.trim() !== '' || maxPrice.value !== null
)
const filteredListings = computed(() => {
  let result = [...listings.value]
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter((l) => l.nft_name?.toLowerCase().includes(q))
  }
  if (maxPrice.value !== null && maxPrice.value > 0) {
    result = result.filter((l) => l.price_lovelace / 1_000_000 <= maxPrice.value!)
  }
  if (sortBy.value === 'price_asc')  result.sort((a, b) => a.price_lovelace - b.price_lovelace)
  if (sortBy.value === 'price_desc') result.sort((a, b) => b.price_lovelace - a.price_lovelace)
  return result
})

// ── Methods ───────────────────────────────────────────────────────────────────
function clearFilters() {
  searchQuery.value = ''
  sortBy.value      = 'newest'
  maxPrice.value    = null
}

function imageUrl(ipfs: string): string | undefined {
  if (!ipfs) return undefined
  return ipfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
    : ipfs
}

function adaAmount(lovelace: number): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', { maximumFractionDigits: 2 })
}

function showError(msg: string) {
  errorMsg.value = msg
  setTimeout(() => (errorMsg.value = ''), 5000)
}

async function handleBuy(listing: any) {
  if (!listing) return
  buyingId.value = listing.id

  try {
    let txHash: string

    if (auth.walletType === 'external') {
      const unsignedRes = await buyListingUnsigned(listing.id)
      const witnessCbor = await walletSession.signOnly(unsignedRes.unsigned_cbor)
      txHash            = await submitSignedTx(unsignedRes.unsigned_cbor, witnessCbor)
      await confirmBuyListing(listing.id, txHash)
    } else {
      const result = await buyListing(listing.id)
      txHash = result.tx_hash
    }

    listings.value        = listings.value.filter((l) => l.id !== listing.id)
    selectedListing.value = null

    buySuccessData.value = {
      txHash,
      nftName:       listing.nft_name,
      imageIpfs:     listing.image_ipfs,
      priceLovelace: listing.price_lovelace,
      policyId:      listing.nft_policy_id,
      assetName:     listing.nft_asset_name,
    }

    await dashboard.loadDashboard()

  } catch (err: any) {
    const raw = err?.message || err.response?.data?.error || ''
    const friendly =
      raw.includes('Insufficient input') ? 'Not enough ADA in your wallet.' :
      raw.includes('UTxO')               ? 'Transaction failed. Please try again.' :
      raw.includes('collateral')         ? 'Wallet needs a 5 ADA collateral UTxO.' :
      raw.includes('User declined')      ? 'Transaction cancelled.' :
                                           'Purchase failed. Please try again.'
    showError(friendly)
  } finally {
    buyingId.value = null
  }
}

onMounted(async () => {
  try {
    const data = await getAllListings()
    listings.value = data.listings || []
  } catch {
    showError('Failed to load listings. Please refresh the page.')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.browse { padding: 28px 32px; max-width: 1200px; }

.back-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  color: #888; text-decoration: none;
  margin-bottom: 20px; transition: all 0.15s;
}
.back-link:hover { background: #EEEDFE; color: #534AB7; }

.page-header { margin-bottom: 20px; }
.page-title  { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub    { font-size: 13px; color: #888; }

/* Filter bar */
.filter-bar {
  display: flex; gap: 10px; margin-bottom: 24px;
  align-items: center; flex-wrap: wrap;
}
.search-wrap {
  flex: 1; min-width: 200px;
  display: flex; align-items: center; gap: 8px;
  background: #fff; border: 1px solid #e8e8e8;
  border-radius: 10px; padding: 10px 14px;
}
.search-icon  { color: #aaa; flex-shrink: 0; }
.search-input { border: none; outline: none; font-size: 13px; width: 100%; background: transparent; color: #111; }
.search-input::placeholder { color: #bbb; }
.filter-right { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.select-wrap {
  display: flex; align-items: center; gap: 6px;
  background: #fff; border: 1px solid #e8e8e8;
  border-radius: 10px; padding: 10px 12px;
}
.select-icon { color: #aaa; }
.sort-select  { border: none; outline: none; font-size: 12px; background: transparent; cursor: pointer; color: #444; }
.price-wrap   { display: flex; align-items: center; gap: 6px; }
.price-label  { font-size: 12px; color: #888; white-space: nowrap; }
.price-input  {
  width: 72px; padding: 10px; border: 1px solid #e8e8e8;
  border-radius: 10px; font-size: 12px; outline: none; background: #fff;
}
.price-input:focus { border-color: #534AB7; }
.btn-clear {
  display: flex; align-items: center; gap: 4px;
  padding: 9px 12px; border: 1px solid #e8e8e8;
  border-radius: 10px; background: #fff;
  font-size: 12px; color: #888; cursor: pointer; transition: all 0.15s;
}
.btn-clear:hover { border-color: #d32f2f; color: #d32f2f; }

/* ── NFT grid — 4 columns, proper sizing ── */
.listings-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

/* ── NFT card ── */
.listing-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
}
.listing-card:hover {
  border-color: #534AB7;
  transform: translateY(-3px);
  box-shadow: 0 8px 24px rgba(83,74,183,0.1);
}

/* Image area */
.listing-img-wrap {
  aspect-ratio: 1;
  background: #f5f5f5;
  position: relative;
  overflow: hidden;
}
.listing-img {
  width: 100%; height: 100%;
  object-fit: contain; display: block;
  background: #f8f8f8;
  transition: transform 0.2s ease;
}
.listing-card:hover .listing-img { transform: scale(1.02); }
.listing-img-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}

/* Hover overlay — "View Details" label */
.listing-hover {
  position: absolute; inset: 0;
  background: rgba(83,74,183,0.82);
  display: flex; align-items: center; justify-content: center;
  opacity: 0; transition: opacity 0.2s ease;
}
.listing-card:hover .listing-hover { opacity: 1; }

.hover-label {
  display: flex; align-items: center; gap: 6px;
  background: #fff; color: #534AB7;
  padding: 8px 18px; border-radius: 20px;
  font-size: 13px; font-weight: 600;
}

/* Card body */
.listing-body { padding: 10px 12px 12px; border-top: 1px solid #f5f5f5; }
.listing-name {
  font-size: 13px; font-weight: 600; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin: 0 0 3px;
}
.listing-price {
  font-size: 15px; font-weight: 700; color: #534AB7; margin: 0;
}

/* Results count */
.results-count { margin-top: 20px; font-size: 12px; color: #aaa; text-align: center; }

/* Empty state */
.empty-state {
  display: flex; flex-direction: column; align-items: center;
  gap: 8px; padding: 80px 20px;
  border: 1.5px dashed #e8e8e8; border-radius: 14px; text-align: center;
}
.empty-title { font-size: 15px; font-weight: 600; color: #444; }
.empty-desc  { font-size: 13px; color: #aaa; }
.btn-primary {
  padding: 9px 18px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}
.btn-outline {
  padding: 9px 18px; background: transparent; color: #534AB7;
  border: 1px solid #534AB7; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}

/* Skeleton */
.listing-card--skeleton { pointer-events: none; }
.sk-image {
  aspect-ratio: 1;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.sk-body  { padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; }
.sk-line  {
  border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.sk-line--title { height: 13px; width: 70%; }
.sk-line--price { height: 16px; width: 40%; }
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

/* Toast */
.toast {
  position: fixed; bottom: 24px; right: 24px;
  padding: 12px 18px; border-radius: 10px;
  font-size: 13px; font-weight: 500;
  box-shadow: 0 4px 20px rgba(0,0,0,0.1);
  z-index: 999; display: flex; align-items: center; gap: 8px;
}
.toast--success { background: #E1F5EE; color: #085041; }
.toast--error   { background: #FCEBEB; color: #791F1F; }
.toast-enter-active, .toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from, .toast-leave-to       { opacity: 0; transform: translateY(8px); }

/* Mobile */
@media (max-width: 900px) {
  .listings-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 640px) {
  .browse        { padding: 16px; }
  .filter-bar    { flex-direction: column; }
  .search-wrap   { min-width: unset; width: 100%; }
  .listings-grid { grid-template-columns: repeat(2, 1fr); gap: 10px; }
}
</style>