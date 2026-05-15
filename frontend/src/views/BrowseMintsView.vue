<template>
  <div class="browse">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Browse Mints</h1>
        <p class="page-sub">Discover and buy NFTs on Cardano Preprod.</p>
      </div>
    </div>

    <!-- Search + Filter bar -->
    <div class="filter-bar">
      <div class="search-wrap">
        <Search :size="16" color="#888" class="search-icon" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by name..."
          class="search-input"
        />
      </div>
      <div class="filter-wrap">
        <SlidersHorizontal :size="16" color="#888" />
        <select v-model="sortBy" class="sort-select">
          <option value="newest">Newest first</option>
          <option value="price_asc">Price: Low to High</option>
          <option value="price_desc">Price: High to Low</option>
        </select>
      </div>
      <div class="price-filter">
        <span class="filter-label">Max price (ADA)</span>
        <input
          v-model.number="maxPrice"
          type="number"
          min="0"
          placeholder="Any"
          class="price-input"
        />
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="listings-grid">
      <div v-for="n in 8" :key="n" class="listing-card skeleton">
        <div class="skeleton-image" />
        <div class="skeleton-info">
          <div class="skeleton-line skeleton-line--title" />
          <div class="skeleton-line skeleton-line--short" />
          <div class="skeleton-line skeleton-line--btn" />
        </div>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredListings.length === 0" class="empty-state">
      <Store :size="48" color="#ccc" />
      <h3>{{ listings.length === 0 ? 'No listings yet' : 'No results found' }}</h3>
      <p>{{ listings.length === 0 ? 'Be the first to list an NFT for sale.' : 'Try adjusting your search or filters.' }}</p>
      <router-link v-if="listings.length === 0" to="/">
        <button class="btn-primary">Go to Dashboard</button>
      </router-link>
      <button v-else class="btn-outline" @click="clearFilters">Clear Filters</button>
    </div>

    <!-- Grid -->
    <div v-else class="listings-grid">
      <div
        v-for="listing in filteredListings"
        :key="listing.id"
        class="listing-card"
      >
        <!-- Image -->
        <div class="listing-image-wrap">
          <img
            v-if="imageUrl(listing.image_ipfs)"
            :src="imageUrl(listing.image_ipfs)"
            :alt="listing.nft_name"
            class="listing-image"
          />
          <div v-else class="listing-image-placeholder">
            <ImageIcon :size="40" color="#ccc" />
          </div>
        </div>

        <!-- Info -->
        <div class="listing-info">
          <h3 class="listing-name">{{ listing.nft_name }}</h3>
          <p v-if="listing.description" class="listing-desc">
            {{ listing.description }}
          </p>

          <div class="listing-price">
            <span class="price-amount">{{ adaAmount(listing.price_lovelace) }}</span>
            <span class="price-unit">tADA</span>
          </div>

          <div class="listing-meta">
            <User :size="12" color="#888" />
            <span class="meta-tag">{{ listing.seller_email }}</span>
          </div>

          <button
            class="btn-buy"
            :disabled="buyingId === listing.id"
            @click="handleBuy(listing)"
          >
            <Loader2 v-if="buyingId === listing.id" :size="14" class="spin" />
            <ShoppingCart v-else :size="14" />
            {{ buyingId === listing.id ? 'Buying...' : 'Buy Now' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Results count -->
    <div v-if="!loading && listings.length > 0" class="results-count">
      Showing {{ filteredListings.length }} of {{ listings.length }} listings
    </div>

    <!-- Success toast -->
    <div v-if="successMsg" class="toast toast--success">
      <CheckCircle :size="16" />
      {{ successMsg }}
    </div>

    <!-- Error toast -->
    <div v-if="errorMsg" class="toast toast--error">
      <AlertCircle :size="16" />
      {{ errorMsg }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  AlertCircle,
  CheckCircle,
  Image as ImageIcon,
  Loader2,
  Search,
  ShoppingCart,
  SlidersHorizontal,
  Store,
  User,
} from 'lucide-vue-next'
import { getAllListings, buyListing } from '@/services/listing'

const listings = ref<any[]>([])
const loading = ref(true)
const buyingId = ref<string | null>(null)
const successMsg = ref('')
const errorMsg = ref('')

// Filters
const searchQuery = ref('')
const sortBy = ref('newest')
const maxPrice = ref<number | null>(null)

const filteredListings = computed(() => {
  let result = [...listings.value]

  // Search by name
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter((l) => l.nft_name?.toLowerCase().includes(q))
  }

  // Max price filter
  if (maxPrice.value !== null && maxPrice.value > 0) {
    result = result.filter(
      (l) => l.price_lovelace / 1_000_000 <= maxPrice.value!
    )
  }

  // Sort
  if (sortBy.value === 'price_asc') {
    result.sort((a, b) => a.price_lovelace - b.price_lovelace)
  } else if (sortBy.value === 'price_desc') {
    result.sort((a, b) => b.price_lovelace - a.price_lovelace)
  }
  // newest = default order from API

  return result
})

function clearFilters() {
  searchQuery.value = ''
  sortBy.value = 'newest'
  maxPrice.value = null
}

function imageUrl(ipfs: string): string | undefined {
  if (!ipfs) return undefined
  if (ipfs.startsWith('ipfs://')) {
    return `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
  }
  return ipfs
}

function adaAmount(lovelace: number): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', {
    maximumFractionDigits: 2,
  })
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
    showError(err.response?.data?.error || 'Failed to buy NFT')
  } finally {
    buyingId.value = null
  }
}

onMounted(async () => {
  try {
    const data = await getAllListings()
    listings.value = data.listings || []
  } catch {
    showError('Failed to load listings')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.browse { max-width: 1100px; margin: 0 auto; padding: 32px; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}
.page-title { font-size: 28px; font-weight: 700; margin: 0 0 4px; }
.page-sub { font-size: 13px; color: #666; margin: 0; }

/* Filter bar */
.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  flex-wrap: wrap;
  align-items: center;
}
.search-wrap {
  flex: 1;
  min-width: 200px;
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  padding: 8px 12px;
}
.search-icon { flex-shrink: 0; }
.search-input {
  border: none; outline: none;
  font-size: 13px; width: 100%;
  background: transparent;
}
.filter-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  padding: 8px 12px;
}
.sort-select {
  border: none; outline: none;
  font-size: 13px; background: transparent;
  cursor: pointer;
}
.price-filter {
  display: flex;
  align-items: center;
  gap: 8px;
}
.filter-label { font-size: 12px; color: #666; white-space: nowrap; }
.price-input {
  width: 80px;
  padding: 8px 10px;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  font-size: 13px;
  outline: none;
}
.price-input:focus { border-color: #534AB7; }

/* Skeleton */
.skeleton { pointer-events: none; }
.skeleton-image {
  aspect-ratio: 1;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.skeleton-info { padding: 16px; display: flex; flex-direction: column; gap: 10px; }
.skeleton-line {
  height: 12px; border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.skeleton-line--title { width: 70%; height: 16px; }
.skeleton-line--short { width: 40%; }
.skeleton-line--btn { width: 100%; height: 36px; border-radius: 8px; margin-top: 4px; }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Empty */
.empty-state {
  text-align: center; padding: 80px;
  border: 1.5px dashed #e0e0e0;
  border-radius: 12px;
  display: flex; flex-direction: column;
  align-items: center; gap: 12px;
}
.empty-state h3 { font-size: 18px; font-weight: 600; margin: 0; }
.empty-state p { font-size: 14px; color: #888; margin: 0; }

/* Grid */
.listings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 20px;
}
.listing-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  overflow: hidden;
  transition: box-shadow 0.2s, transform 0.2s;
}
.listing-card:hover {
  box-shadow: 0 6px 24px rgba(0,0,0,0.09);
  transform: translateY(-2px);
}
.listing-image-wrap {
  aspect-ratio: 1; background: #f8f8f8; overflow: hidden;
}
.listing-image { width: 100%; height: 100%; object-fit: cover; }
.listing-image-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}
.listing-info { padding: 16px; }
.listing-name {
  font-size: 15px; font-weight: 600; margin: 0 0 4px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.listing-desc {
  font-size: 12px; color: #888; margin: 0 0 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.listing-price {
  display: flex; align-items: baseline; gap: 4px; margin-bottom: 8px;
}
.price-amount { font-size: 22px; font-weight: 700; color: #534AB7; }
.price-unit { font-size: 13px; color: #888; }
.listing-meta {
  display: flex; align-items: center; gap: 4px;
  margin-bottom: 12px;
}
.meta-tag { font-size: 11px; color: #888; }
.btn-buy {
  width: 100%; padding: 10px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 14px; font-weight: 600;
  cursor: pointer; transition: background 0.2s;
  display: flex; align-items: center;
  justify-content: center; gap: 6px;
}
.btn-buy:hover:not(:disabled) { background: #3d35a0; }
.btn-buy:disabled { opacity: 0.6; cursor: not-allowed; }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.btn-primary {
  padding: 10px 20px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 14px; font-weight: 600; cursor: pointer;
}
.btn-outline {
  padding: 10px 20px;
  background: transparent; color: #534AB7;
  border: 1.5px solid #534AB7; border-radius: 8px;
  font-size: 14px; font-weight: 600; cursor: pointer;
}

.results-count {
  margin-top: 16px;
  font-size: 12px; color: #888; text-align: center;
}

.toast {
  position: fixed; bottom: 24px; right: 24px;
  padding: 14px 20px; border-radius: 10px;
  font-size: 13px; font-weight: 500;
  box-shadow: 0 4px 20px rgba(0,0,0,0.12);
  z-index: 999;
  display: flex; align-items: center; gap: 8px;
}
.toast--success { background: #E1F5EE; color: #085041; }
.toast--error { background: #FCEBEB; color: #791F1F; }

/* Mobile responsive */
@media (max-width: 640px) {
  .browse { padding: 16px; }
  .filter-bar { flex-direction: column; }
  .search-wrap { min-width: unset; width: 100%; }
  .listings-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .page-title { font-size: 22px; }
}
</style>