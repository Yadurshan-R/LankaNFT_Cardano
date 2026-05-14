<template>
  <div class="browse">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Browse Mints</h1>
        <p class="page-sub">Discover and buy NFTs on Cardano Preprod.</p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <p>Loading listings...</p>
    </div>

    <!-- Empty -->
    <div v-else-if="listings.length === 0" class="empty-state">
      <div class="empty-icon">🏪</div>
      <h3>No listings yet</h3>
      <p>Be the first to list an NFT for sale.</p>
      <router-link to="/">
        <button class="btn-primary">Go to Dashboard</button>
      </router-link>
    </div>

    <!-- Grid -->
    <div v-else class="listings-grid">
      <div
        v-for="listing in listings"
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
          <div v-else class="listing-image-placeholder">🖼</div>
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
            <span class="meta-tag">{{ listing.seller_email }}</span>
          </div>

          <!-- Buy button -->
          <button
            class="btn-buy"
            :disabled="buyingId === listing.id"
            @click="handleBuy(listing)"
          >
            {{ buyingId === listing.id ? 'Buying...' : 'Buy Now' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Success toast -->
    <div v-if="successMsg" class="toast toast--success">
      {{ successMsg }}
    </div>

    <!-- Error toast -->
    <div v-if="errorMsg" class="toast toast--error">
      {{ errorMsg }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAllListings, buyListing } from '@/services/listing'

const listings = ref<any[]>([])
const loading = ref(true)
const buyingId = ref<string | null>(null)
const successMsg = ref('')
const errorMsg = ref('')

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
    // Remove purchased listing from view
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
  margin-bottom: 32px;
}
.page-title { font-size: 28px; font-weight: 700; margin: 0 0 4px; }
.page-sub { font-size: 13px; color: #666; margin: 0; }

.loading-state {
  display: flex; flex-direction: column;
  align-items: center; gap: 16px;
  padding: 80px; color: #888;
}
.spinner {
  width: 32px; height: 32px;
  border: 3px solid #f0f0f0;
  border-top-color: #534AB7;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.empty-state {
  text-align: center; padding: 80px;
  border: 1.5px dashed #e0e0e0;
  border-radius: 12px;
  display: flex; flex-direction: column;
  align-items: center; gap: 12px;
}
.empty-icon { font-size: 48px; }
.empty-state h3 { font-size: 18px; font-weight: 600; margin: 0; }
.empty-state p { font-size: 14px; color: #888; margin: 0; }

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
  display: flex; align-items: center;
  justify-content: center; font-size: 48px; color: #ccc;
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

.listing-meta { margin-bottom: 12px; }
.meta-tag {
  font-size: 11px; background: #f5f5f5;
  color: #666; padding: 2px 8px; border-radius: 4px;
}

.btn-buy {
  width: 100%; padding: 10px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 14px; font-weight: 600;
  cursor: pointer; transition: background 0.2s;
}
.btn-buy:hover:not(:disabled) { background: #3d35a0; }
.btn-buy:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-primary {
  padding: 10px 20px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 14px; font-weight: 600;
  cursor: pointer;
}

.toast {
  position: fixed; bottom: 24px; right: 24px;
  padding: 14px 20px; border-radius: 10px;
  font-size: 13px; font-weight: 500;
  box-shadow: 0 4px 20px rgba(0,0,0,0.12);
  z-index: 999;
}
.toast--success { background: #E1F5EE; color: #085041; }
.toast--error { background: #FCEBEB; color: #791F1F; }
</style>