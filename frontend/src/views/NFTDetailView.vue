<template>
  <div class="nft-detail">
    <!-- Back -->
    <router-link to="/" class="back-link">← Back to Dashboard</router-link>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <p>Loading NFT details...</p>
    </div>

    <!-- Not found -->
    <div v-else-if="!nft" class="error-state">
      <p>NFT not found.</p>
      <router-link to="/">← Back to Dashboard</router-link>
    </div>

    <!-- Detail -->
    <div v-else class="detail-content">
      <!-- Left: Image -->
      <div class="detail-left">
        <div class="detail-image-wrap">
          <img
            v-if="imageUrl"
            :src="imageUrl"
            :alt="nft.name"
            class="detail-image"
          />
          <div v-else class="detail-image-placeholder">🖼</div>

          <!-- Badges -->
          <div :class="['status-badge', `status-badge--${nft.status}`]">
            {{ nft.status }}
          </div>
          <div v-if="nft.privacy === 'private'" class="privacy-badge">🔒 Private</div>
        </div>

        <!-- Cardanoscan -->
        <a
          v-if="nft.tx_hash"
          :href="`https://preprod.cardanoscan.io/transaction/${nft.tx_hash}`"
          target="_blank"
          class="cardanoscan-btn"
        >
          View Transaction on Cardanoscan →
        </a>
      </div>

      <!-- Right: Info -->
      <div class="detail-right">
        <h1 class="detail-name">{{ nft.name }}</h1>
        <p v-if="nft.description" class="detail-desc">{{ nft.description }}</p>

        <!-- Details table -->
        <div class="detail-section">
          <h3 class="detail-section-title">Details</h3>
          <div class="detail-rows">
            <div class="detail-row">
              <span class="detail-label">Status</span>
              <span :class="['detail-value', `status--${nft.status}`]">
                {{ nft.status }}
              </span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Royalties</span>
              <span class="detail-value">{{ nft.royalties }}%</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Privacy</span>
              <span class="detail-value">{{ nft.privacy }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Minted</span>
              <span class="detail-value">{{ formattedDate }}</span>
            </div>
          </div>
        </div>

        <!-- Policy ID -->
        <div v-if="nft.policy_id" class="detail-section">
          <h3 class="detail-section-title">Policy ID</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.policy_id }}</code>
            <button class="copy-btn" @click="copy(nft.policy_id, 'policy')">
              {{ copied === 'policy' ? '✓' : '⎘' }}
            </button>
          </div>
        </div>

        <!-- Asset Name -->
        <div class="detail-section">
          <h3 class="detail-section-title">Asset Name</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.asset_name }}</code>
            <button class="copy-btn" @click="copy(nft.asset_name, 'asset')">
              {{ copied === 'asset' ? '✓' : '⎘' }}
            </button>
          </div>
        </div>

        <!-- TX Hash -->
        <div v-if="nft.tx_hash" class="detail-section">
          <h3 class="detail-section-title">Transaction Hash</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.tx_hash }}</code>
            <button class="copy-btn" @click="copy(nft.tx_hash!, 'tx')">
              {{ copied === 'tx' ? '✓' : '⎘' }}
            </button>
          </div>
        </div>

        <!-- IPFS -->
        <div v-if="nft.image" class="detail-section">
          <h3 class="detail-section-title">IPFS Image</h3>
          <a
            :href="nft.image.replace('ipfs://', 'https://gateway.pinata.cloud/ipfs/')"
            target="_blank"
            class="ipfs-link"
          >
            {{ nft.image }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useDashboardStore } from '@/stores/dashboard'

const route = useRoute()
const dashboard = useDashboardStore()
const loading = ref(true)
const copied = ref<string | null>(null)

// Find the NFT from the dashboard store
// If not loaded yet, load the dashboard first
const nft = computed(() =>
  dashboard.nfts.find((n) => n.id === route.params.id)
)

const imageUrl = computed(() => {
  if (!nft.value?.image) return null
  if (nft.value.image.startsWith('ipfs://')) {
    const hash = nft.value.image.replace('ipfs://', '')
    return `https://gateway.pinata.cloud/ipfs/${hash}`
  }
  return nft.value.image
})

const formattedDate = computed(() => {
  if (!nft.value?.created_at) return '—'
  return new Date(nft.value.created_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
})

async function copy(text: string, key: string) {
  await navigator.clipboard.writeText(text)
  copied.value = key
  setTimeout(() => (copied.value = null), 2000)
}

onMounted(async () => {
  // Load dashboard data if not already loaded
  if (dashboard.nfts.length === 0) {
    await dashboard.loadDashboard()
  }
  loading.value = false
})
</script>

<style scoped>
.nft-detail { max-width: 1000px; margin: 0 auto; padding: 32px; }
.back-link {
  font-size: 13px; color: #666; text-decoration: none;
  display: block; margin-bottom: 24px;
}
.back-link:hover { color: #534AB7; }

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

.error-state { text-align: center; padding: 80px; color: #888; }

.detail-content {
  display: grid;
  grid-template-columns: 420px 1fr;
  gap: 48px;
  align-items: start;
}

/* Left */
.detail-image-wrap {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  aspect-ratio: 1;
  background: #f8f8f8;
}
.detail-image { width: 100%; height: 100%; object-fit: cover; }
.detail-image-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center;
  justify-content: center;
  font-size: 80px; color: #ccc;
}
.status-badge {
  position: absolute; top: 12px; left: 12px;
  font-size: 11px; font-weight: 600;
  padding: 4px 10px; border-radius: 20px;
  text-transform: uppercase;
  background: #f0f0f0; color: #666;
}
.status-badge--minted { background: #E1F5EE; color: #085041; }
.status-badge--pending { background: #FFF8E1; color: #7a5c00; }
.privacy-badge {
  position: absolute; top: 12px; right: 12px;
  font-size: 12px; font-weight: 500;
  background: rgba(0,0,0,0.6);
  color: #fff; padding: 4px 10px;
  border-radius: 20px;
}
.cardanoscan-btn {
  display: block; margin-top: 16px;
  padding: 12px; text-align: center;
  background: #534AB7; color: #fff;
  border-radius: 10px; text-decoration: none;
  font-size: 13px; font-weight: 500;
}
.cardanoscan-btn:hover { background: #3d35a0; }

/* Right */
.detail-name { font-size: 28px; font-weight: 700; margin: 0 0 8px; }
.detail-desc { font-size: 14px; color: #666; margin: 0 0 24px; line-height: 1.6; }

.detail-section { margin-bottom: 24px; }
.detail-section-title {
  font-size: 12px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #888; margin: 0 0 8px;
}
.detail-rows { display: flex; flex-direction: column; gap: 8px; }
.detail-row {
  display: flex; justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
}
.detail-label { color: #666; }
.detail-value { font-weight: 500; }
.status--minted { color: #085041; }
.status--pending { color: #7a5c00; }

.copy-row {
  display: flex; align-items: center;
  gap: 8px;
  background: #f8f8f8;
  border-radius: 8px;
  padding: 10px 12px;
}
.code-value {
  font-family: monospace; font-size: 12px;
  color: #444; flex: 1;
  overflow: hidden; text-overflow: ellipsis;
  white-space: nowrap;
}
.copy-btn {
  background: none; border: none;
  cursor: pointer; font-size: 16px;
  color: #534AB7; padding: 0;
  flex-shrink: 0;
}
.copy-btn:hover { opacity: 0.7; }

.ipfs-link {
  font-size: 12px; color: #534AB7;
  text-decoration: none; word-break: break-all;
  font-family: monospace;
}
.ipfs-link:hover { text-decoration: underline; }
</style>