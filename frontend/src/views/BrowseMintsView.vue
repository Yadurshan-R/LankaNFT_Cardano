<!-- ─────────────────────────────────────────────────────────────────────────
  BrowseMintsView.vue — LankaNFT marketplace browse page
  
  Shows all active listings with search, sort, and price filter.
  Buy action calls the marketplace contract via the backend.
  All filtering is client-side on the fetched listings array.
──────────────────────────────────────────────────────────────────────────── -->
<template>
  <div class="browse">

    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Browse Mints</h1>
        <p class="page-sub">Discover and buy NFTs on Cardano Preprod.</p>
      </div>
    </div>

    <!-- Search + filter bar -->
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

    <!-- Loading skeleton -->
    <div v-if="loading" class="listings-grid">
      <div v-for="n in 8" :key="n" class="listing-card listing-card--skeleton">
        <div class="sk-image" />
        <div class="sk-body">
          <div class="sk-line sk-line--title" />
          <div class="sk-line sk-line--short" />
          <div class="sk-line sk-line--btn" />
        </div>
      </div>
    </div>

    <!-- Empty state -->
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

    <!-- Listings grid -->
    <div v-else class="listings-grid">
      <div
        v-for="listing in filteredListings"
        :key="listing.id"
        class="listing-card"
      >
        <!-- Image with hover overlay -->
        <div class="listing-img-wrap">
          <img
            v-if="imageUrl(listing.image_ipfs)"
            :src="imageUrl(listing.image_ipfs)"
            :alt="listing.nft_name"
            class="listing-img"
          />
          <div v-else class="listing-img-placeholder">
            <ImageIcon :size="32" color="#ddd" />
          </div>
          <!-- Hover overlay — hidden until card hovered via CSS -->
          <div class="listing-hover">
            <button
              class="listing-hover-btn"
              :disabled="buyingId === listing.id"
              @click="handleBuy(listing)"
            >
              <Loader2 v-if="buyingId === listing.id" :size="13" class="spin" />
              <ShoppingCart v-else :size="13" />
              {{ buyingId === listing.id ? 'Buying...' : 'Buy Now' }}
            </button>
          </div>
        </div>

        <!-- Card info -->
        <div class="listing-info">
          <div class="listing-name-row">
            <span class="listing-name">{{ listing.nft_name }}</span>
            <span class="listing-badge">Listed</span>
          </div>
          <div class="listing-royalty">{{ listing.royalties ?? 5 }}% royalty</div>
          <div class="listing-price">
            <span class="price-ada">{{ adaAmount(listing.price_lovelace) }} ₳</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Results count -->
    <div v-if="!loading && listings.length > 0" class="results-count">
      {{ filteredListings.length }} of {{ listings.length }} listings
    </div>

    <!-- Success toast -->
    <Transition name="toast">
      <div v-if="successMsg" class="toast toast--success">
        <CheckCircle :size="15" /> {{ successMsg }}
      </div>
    </Transition>

    <!-- Error toast -->
    <Transition name="toast">
      <div v-if="errorMsg" class="toast toast--error">
        <AlertCircle :size="15" /> {{ errorMsg }}
      </div>
    </Transition>

  </div>
</template>

<script setup lang="ts">
// All logic unchanged from original — only template and styles updated
import { computed, onMounted, ref } from 'vue'
import {
  AlertCircle, CheckCircle,
  Image as ImageIcon, Loader2,
  Search, ShoppingCart, SlidersHorizontal,
  Store, X,
} from 'lucide-vue-next'
import { getAllListings, buyListing } from '@/services/listing'

const listings  = ref<any[]>([])
const loading   = ref(true)
const buyingId  = ref<string | null>(null)
const successMsg = ref('')
const errorMsg   = ref('')

// Filters
const searchQuery = ref('')
const sortBy      = ref('newest')
const maxPrice    = ref<number | null>(null)

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

function clearFilters() {
  searchQuery.value = ''
  sortBy.value      = 'newest'
  maxPrice.value    = null
}

// Convert ipfs:// URI to HTTP gateway URL
function imageUrl(ipfs: string): string | undefined {
  if (!ipfs) return undefined
  return ipfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
    : ipfs
}

// Convert lovelace to ADA string
function adaAmount(lovelace: number): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', { maximumFractionDigits: 2 })
}

function showSuccess(msg: string) {
  successMsg.value = msg
  setTimeout(() => (successMsg.value = ''), 4000)
}

function showError(msg: string) {
  errorMsg.value = msg
  setTimeout(() => (errorMsg.value = ''), 4000)
}

async function handleBuy(listing: any) {
  buyingId.value = listing.id
  try {
    const result = await buyListing(listing.id)
    showSuccess(`NFT purchased! TX: ${result.tx_hash.slice(0, 16)}...`)
    listings.value = listings.value.filter((l) => l.id !== listing.id)
  } catch (err: any) {
    const raw = err.response?.data?.error || ''
    const friendly = raw.includes('Insufficient input')
      ? 'Not enough ADA in wallet.'
      : raw.includes('UTxO')
      ? 'Transaction failed. Please try again.'
      : 'Failed to buy NFT. Please try again.'
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
    showError('Failed to load listings.')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.browse { padding: 28px 32px; max-width: 1200px; }

/* Header */
.page-header { margin-bottom: 20px; }
.page-title  { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub    { font-size: 13px; color: #888; }

/* Filter bar */
.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  align-items: center;
  flex-wrap: wrap;
}

.search-wrap {
  flex: 1;
  min-width: 200px;
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 10px;
  padding: 9px 14px;
}
.search-icon  { color: #aaa; flex-shrink: 0; }
.search-input { border: none; outline: none; font-size: 13px; width: 100%; background: transparent; color: #111; }
.search-input::placeholder { color: #bbb; }

.filter-right { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

.select-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 10px;
  padding: 9px 12px;
}
.select-icon { color: #aaa; }
.sort-select { border: none; outline: none; font-size: 12px; background: transparent; cursor: pointer; color: #444; }

.price-wrap  { display: flex; align-items: center; gap: 6px; }
.price-label { font-size: 12px; color: #888; white-space: nowrap; }
.price-input {
  width: 72px;
  padding: 9px 10px;
  border: 1px solid #e8e8e8;
  border-radius: 10px;
  font-size: 12px;
  outline: none;
  background: #fff;
}
.price-input:focus { border-color: #534AB7; }

.btn-clear {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  border: 1px solid #e8e8e8;
  border-radius: 10px;
  background: #fff;
  font-size: 12px;
  color: #888;
  cursor: pointer;
}
.btn-clear:hover { border-color: #d32f2f; color: #d32f2f; }

/* Grid */
.listings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
}

/* Card */
.listing-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  overflow: hidden;
  transition: all 0.15s;
}
.listing-card:hover { border-color: #534AB7; transform: translateY(-2px); }

/* Image + hover overlay */
.listing-img-wrap {
  aspect-ratio: 1;
  background: #f8f8f8;
  position: relative;
  overflow: hidden;
}
.listing-img         { width: 100%; height: 100%; object-fit: cover; }
.listing-img-placeholder { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }

/* Hover overlay — only visible when card is hovered */
.listing-hover {
  position: absolute;
  inset: 0;
  background: rgba(83,74,183,0.88);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.15s;
}
.listing-card:hover .listing-hover { opacity: 1; }

.listing-hover-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #fff;
  color: #534AB7;
  border: none;
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.listing-hover-btn:hover:not(:disabled) { background: #f0effd; }
.listing-hover-btn:disabled { opacity: 0.7; cursor: not-allowed; }

/* Card info */
.listing-info { padding: 10px 12px; }

.listing-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  margin-bottom: 3px;
}
.listing-name {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.listing-badge {
  font-size: 9px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 20px;
  background: #EEEDFE;
  color: #534AB7;
  white-space: nowrap;
  flex-shrink: 0;
}
.listing-royalty { font-size: 10px; color: #aaa; margin-bottom: 4px; }
.listing-price   { display: flex; align-items: baseline; gap: 4px; }
.price-ada       { font-size: 15px; font-weight: 700; color: #534AB7; }

/* Results count */
.results-count { margin-top: 16px; font-size: 12px; color: #aaa; text-align: center; }

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 64px 20px;
  border: 1.5px dashed #e8e8e8;
  border-radius: 14px;
  text-align: center;
}
.empty-title { font-size: 15px; font-weight: 600; color: #444; }
.empty-desc  { font-size: 13px; color: #aaa; }

.btn-primary {
  padding: 9px 18px;
  background: #534AB7;
  color: #fff;
  border: none;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.btn-outline {
  padding: 9px 18px;
  background: transparent;
  color: #534AB7;
  border: 1px solid #534AB7;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

/* Skeleton */
.listing-card--skeleton { pointer-events: none; }
.sk-image {
  aspect-ratio: 1;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.sk-body { padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.sk-line {
  height: 11px;
  border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.sk-line--title { width: 70%; height: 14px; }
.sk-line--short { width: 40%; }
.sk-line--btn   { width: 100%; height: 32px; border-radius: 8px; margin-top: 4px; }

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Toast */
.toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  padding: 12px 18px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  box-shadow: 0 4px 20px rgba(0,0,0,0.1);
  z-index: 999;
  display: flex;
  align-items: center;
  gap: 8px;
}
.toast--success { background: #E1F5EE; color: #085041; }
.toast--error   { background: #FCEBEB; color: #791F1F; }

.toast-enter-active, .toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from, .toast-leave-to       { opacity: 0; transform: translateY(8px); }

/* Spinner */
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Mobile */
@media (max-width: 640px) {
  .browse         { padding: 16px; }
  .filter-bar     { flex-direction: column; }
  .search-wrap    { min-width: unset; width: 100%; }
  .listings-grid  { grid-template-columns: repeat(2, 1fr); gap: 10px; }
}
</style>