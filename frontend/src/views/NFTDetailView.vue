<template>
  <div class="nft-detail">
    <router-link to="/" class="back-link">
      <ArrowLeft :size="14" /> Back to Dashboard
    </router-link>

    <!-- Loading skeleton -->
    <div v-if="loading" class="detail-content">
      <div class="skeleton-left">
        <div class="skeleton-image" />
        <div class="skeleton-btn" />
      </div>
      <div class="skeleton-right">
        <div class="skeleton-line skeleton-line--title" />
        <div class="skeleton-line skeleton-line--short" />
        <div class="skeleton-line" />
        <div class="skeleton-line skeleton-line--short" />
      </div>
    </div>

    <!-- Not found -->
    <div v-else-if="!nft" class="error-state">
      <p>NFT not found.</p>
      <router-link to="/">← Back to Dashboard</router-link>
    </div>

    <!-- Detail -->
    <div v-else class="detail-content">
      <!-- Left: Image + Actions -->
      <div class="detail-left">
        <div class="detail-image-wrap">
          <img v-if="imageUrl" :src="imageUrl" :alt="nft.name" class="detail-image" />
          <div v-else class="detail-image-placeholder">
            <ImageIcon :size="64" color="#ccc" />
          </div>
          <div :class="['status-badge', `status-badge--${nft.status}`]">
            {{ nft.status }}
          </div>
          <div v-if="nft.privacy === 'private'" class="privacy-badge">
            <Lock :size="12" color="#fff" /> Private
          </div>
        </div>

        <!-- View on Cardanoscan — only shown after minting is confirmed -->
        <a
          v-if="nft.tx_hash"
          :href="`https://preprod.cardanoscan.io/transaction/${nft.tx_hash}`"
          target="_blank"
          rel="noopener noreferrer"
          class="cardanoscan-btn"
        >
          View Transaction on Cardanoscan →
        </a>

        <!-- ── Listing Section ─────────────────────────────────────── -->
        <!-- Only shown for minted NFTs that are not already listed -->
        <div v-if="nft.status === 'minted'" class="list-section">

          <!-- Success state — shown after successful listing -->
          <div v-if="listingConfirmed" class="listing-confirmed">
            <div class="confirmed-header">
              <CheckCircle :size="20" color="#085041" />
              <span>Listed Successfully!</span>
            </div>
            <div class="confirmed-details">
              <div class="confirmed-row">
                <span>Price</span>
                <strong>{{ listPrice }} ADA</strong>
              </div>
              <div class="confirmed-row">
                <span>Transaction</span>
                <a
                  :href="`https://preprod.cardanoscan.io/transaction/${listingTxHash}`"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="confirmed-link"
                >
                  View on Cardanoscan →
                </a>
              </div>
            </div>
            <!-- Allow relisting only after explicit cancel — not from here -->
            <p class="confirmed-note">
              Your NFT is now visible in Browse Mints.
            </p>
          </div>

          <!-- List form — shown when not yet listed -->
          <div v-else>
            <div v-if="!showListForm">
              <button class="list-btn" @click="showListForm = true">
                <Tag :size="14" /> List for Sale
              </button>
            </div>
            <div v-else class="list-form">
              <label class="price-label">Price in ADA</label>
              <input
                v-model.number="listPrice"
                type="number"
                placeholder="e.g. 10"
                class="price-input"
                min="2"
              />
              <p class="price-hint">Minimum 2 ADA · You receive sale price minus royalties</p>
              <div class="list-actions">
                <button class="btn-confirm" :disabled="listing" @click="handleList">
                  <Loader2 v-if="listing" :size="14" class="spin" />
                  {{ listing ? 'Submitting...' : 'Confirm Listing' }}
                </button>
                <button class="btn-cancel-list" @click="showListForm = false">Cancel</button>
              </div>
              <p v-if="listError" class="list-error">{{ listError }}</p>
            </div>
          </div>
        </div>

        <!-- ── Transfer Section ────────────────────────────────────────── -->
        <div v-if="nft.status === 'minted'" class="transfer-section">

          <!-- Transfer confirmed state -->
          <div v-if="transferConfirmed" class="transfer-confirmed">
            <div class="confirmed-header">
              <Send :size="16" color="#534AB7" />
              <span>NFT Sent Successfully!</span>
            </div>
            <div class="confirmed-details">
              <div class="confirmed-row">
                <span>Sent to</span>
                <code class="confirmed-addr">{{ transferAddress.slice(0, 20) }}...{{ transferAddress.slice(-6) }}</code>
              </div>
              <div class="confirmed-row">
                <span>Transaction</span>
                <a
                  :href="`https://preprod.cardanoscan.io/transaction/${transferTxHash}`"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="confirmed-link"
                >
                  View on Cardanoscan →
                </a>
              </div>
            </div>
            <p class="confirmed-note">This NFT has been removed from your dashboard.</p>
          </div>

          <!-- Transfer form -->
          <div v-else>
            <div v-if="!showTransferForm">
              <button class="transfer-btn" @click="showTransferForm = true">
                <Send :size="14" /> Transfer NFT
              </button>
            </div>
            <div v-else class="transfer-form">
              <label class="price-label">Recipient Address</label>
              <input
                v-model="transferAddress"
                class="price-input"
                placeholder="addr_test1..."
              />
              <p class="price-hint">Enter any valid Cardano preprod address</p>

              <!-- Fee estimate — always shown when form is open -->
              <!-- Transfer costs ~0.2 ADA network fee + 2 ADA min-UTxO sent with NFT -->
              <!-- The 2 ADA is sent TO the recipient, not lost — it is part of the transfer -->
              <div class="transfer-fee-info">
                <div class="fee-row">
                  <span>Min-ADA sent with NFT</span>
                  <span>2 ADA <span class="fee-note">(goes to recipient)</span></span>
                </div>
                <div class="fee-row">
                  <span>Network fee</span>
                  <span>~0.2 ADA</span>
                </div>
                <div class="fee-divider" />
                <div class="fee-row fee-row--total">
                  <span>You will spend</span>
                  <strong>~2.2 ADA</strong>
                </div>
              </div>

              <div class="list-actions">
                <button class="btn-confirm" :disabled="transferring || !transferAddress" @click="handleTransfer">
                  <Loader2 v-if="transferring" :size="14" class="spin" />
                  {{ transferring ? 'Sending NFT...' : 'Send NFT' }}
                </button>
                <button class="btn-cancel-list" @click="showTransferForm = false; transferError = ''">Cancel</button>
              </div>
              <p v-if="transferError" class="list-error">{{ transferError }}</p>
            </div>
          </div>
        </div>

        <!-- Already listed state -->
        <div v-if="nft.status === 'listed'" class="already-listed">
          <Store :size="16" color="#534AB7" />
          <span>This NFT is currently listed for sale in the marketplace.</span>
        </div>
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
              <span :class="['detail-value', `status--${nft.status}`]">{{ nft.status }}</span>
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

        <!-- Policy ID — unique identifier for this NFT's minting policy -->
        <div v-if="nft.policy_id" class="detail-section">
          <h3 class="detail-section-title">Policy ID</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.policy_id }}</code>
            <button class="copy-btn" @click="copy(nft.policy_id, 'policy')" title="Copy">
              <Check v-if="copied === 'policy'" :size="14" color="#085041" />
              <Copy v-else :size="14" color="#534AB7" />
            </button>
          </div>
        </div>

        <!-- Asset Name — CIP-68 token name stored on-chain -->
        <div class="detail-section">
          <h3 class="detail-section-title">Asset Name</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.asset_name }}</code>
            <button class="copy-btn" @click="copy(nft.asset_name, 'asset')" title="Copy">
              <Check v-if="copied === 'asset'" :size="14" color="#085041" />
              <Copy v-else :size="14" color="#534AB7" />
            </button>
          </div>
        </div>

        <!-- Transaction Hash — the Cardano tx that minted this NFT -->
        <div v-if="nft.tx_hash" class="detail-section">
          <h3 class="detail-section-title">Transaction Hash</h3>
          <div class="copy-row">
            <code class="code-value">{{ nft.tx_hash }}</code>
            <button class="copy-btn" @click="copy(nft.tx_hash!, 'tx')" title="Copy">
              <Check v-if="copied === 'tx'" :size="14" color="#085041" />
              <Copy v-else :size="14" color="#534AB7" />
            </button>
          </div>
        </div>

        <!-- IPFS URI — where the NFT image is permanently stored -->
        <div v-if="nft.image" class="detail-section">
          <h3 class="detail-section-title">IPFS Image</h3>
          <a
            :href="nft.image.replace('ipfs://', 'https://gateway.pinata.cloud/ipfs/')"
            target="_blank"
            rel="noopener noreferrer"
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
import {
  ArrowLeft,
  Check,
  CheckCircle,
  Copy,
  Image as ImageIcon,
  Loader2,
  Lock,
  Send,
  Store,
  Tag,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'
import { createListing } from '@/services/listing'
import { transferNFT } from '@/services/nft'

const route = useRoute()
const dashboard = useDashboardStore()

// UI state
const loading = ref(true)
const copied = ref<string | null>(null)

// Listing form state
const showListForm = ref(false)
const listPrice = ref(10)
const listing = ref(false)
const listError = ref('')

// Post-listing confirmation state
const listingConfirmed = ref(false)
const listingTxHash = ref('')

// Transfer state
const showTransferForm = ref(false)
const transferAddress = ref('')
const transferring = ref(false)
const transferError = ref('')
const transferConfirmed = ref(false)
const transferTxHash = ref('')

// Find the NFT from the store by URL param
const nft = computed(() =>
  dashboard.nfts.find((n) => n.id === route.params.id)
)

// Convert IPFS URI to a gateway URL for image display
const imageUrl = computed(() => {
  if (!nft.value?.image) return null
  if (nft.value.image.startsWith('ipfs://')) {
    return `https://gateway.pinata.cloud/ipfs/${nft.value.image.replace('ipfs://', '')}`
  }
  return nft.value.image
})

// Human-readable mint date
const formattedDate = computed(() => {
  if (!nft.value?.created_at) return '—'
  return new Date(nft.value.created_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'long', day: 'numeric',
  })
})

// Copy text to clipboard and show a temporary checkmark
async function copy(text: string, key: string) {
  await navigator.clipboard.writeText(text)
  copied.value = key
  setTimeout(() => (copied.value = null), 2000)
}

// Handle listing form submission
async function handleList() {
  if (!nft.value) return

  // Client-side validation before hitting the API
  if (listPrice.value < 2) {
    listError.value = 'Minimum price is 2 ADA'
    return
  }

  listing.value = true
  listError.value = ''

  try {
    const priceLovelace = Math.floor(listPrice.value * 1_000_000)
    const result = await createListing(nft.value.id, priceLovelace)

    // Show success confirmation with tx hash
    listingTxHash.value = result.tx_hash
    listingConfirmed.value = true
    showListForm.value = false

    // Reload dashboard to update NFT status to 'listed'
    await dashboard.loadDashboard()
  } catch (err: any) {
    // Show the error from the backend (e.g. duplicate listing message)
    listError.value = err.response?.data?.error || 'Failed to list NFT'
  } finally {
    listing.value = false
  }
}

async function handleTransfer() {
  if (!nft.value) return
  if (!transferAddress.value.startsWith('addr')) {
    transferError.value = 'Enter a valid Cardano address starting with addr'
    return
  }

  transferring.value = true
  transferError.value = ''

  try {
    const result = await transferNFT(nft.value.id, transferAddress.value)
    transferTxHash.value = result.tx_hash
    transferConfirmed.value = true
    showTransferForm.value = false
    // Reload dashboard to remove NFT from view
    await dashboard.loadDashboard()
  } catch (err: any) {
    transferError.value = err.response?.data?.error || 'Failed to transfer NFT'
  } finally {
    transferring.value = false
  }
}

onMounted(async () => {
  // Load dashboard data if not already in store
  if (dashboard.nfts.length === 0) await dashboard.loadDashboard()
  loading.value = false
})
</script>

<style scoped>
.nft-detail { max-width: 1000px; margin: 0 auto; padding: 32px; }

.back-link {
  font-size: 13px; color: #666; text-decoration: none;
  display: inline-flex; align-items: center; gap: 4px; margin-bottom: 24px;
}
.back-link:hover { color: #534AB7; }

.error-state { text-align: center; padding: 80px; color: #888; }

/* ── Skeleton ─────────────────────────────────────────────── */
.skeleton-left { display: flex; flex-direction: column; gap: 12px; }
.skeleton-image {
  aspect-ratio: 1; border-radius: 16px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.skeleton-btn {
  height: 44px; border-radius: 10px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.skeleton-right { display: flex; flex-direction: column; gap: 12px; padding-top: 8px; }
.skeleton-line {
  height: 14px; border-radius: 6px; width: 100%;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.skeleton-line--title { height: 28px; width: 60%; }
.skeleton-line--short { width: 40%; }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* ── Layout ───────────────────────────────────────────────── */
.detail-content {
  display: grid; grid-template-columns: 420px 1fr;
  gap: 48px; align-items: start;
}

/* ── Image ────────────────────────────────────────────────── */
.detail-image-wrap {
  position: relative; border-radius: 16px;
  overflow: hidden; aspect-ratio: 1; background: #f8f8f8;
}
.detail-image { width: 100%; height: 100%; object-fit: cover; }
.detail-image-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}

/* ── Badges ───────────────────────────────────────────────── */
.status-badge {
  position: absolute; top: 12px; left: 12px;
  font-size: 11px; font-weight: 600; padding: 4px 10px;
  border-radius: 20px; text-transform: uppercase;
  background: #f0f0f0; color: #666;
}
.status-badge--minted { background: #E1F5EE; color: #085041; }
.status-badge--listed  { background: #EEEDFE; color: #534AB7; }
.status-badge--pending { background: #FFF8E1; color: #7a5c00; }
.privacy-badge {
  position: absolute; top: 12px; right: 12px;
  font-size: 12px; font-weight: 500;
  background: rgba(0,0,0,0.6); color: #fff;
  padding: 4px 10px; border-radius: 20px;
  display: flex; align-items: center; gap: 4px;
}

/* ── Cardanoscan Button ───────────────────────────────────── */
.cardanoscan-btn {
  display: block; margin-top: 16px; padding: 12px;
  text-align: center; background: #534AB7; color: #fff;
  border-radius: 10px; text-decoration: none;
  font-size: 13px; font-weight: 500; transition: background 0.2s;
}
.cardanoscan-btn:hover { background: #3d35a0; }

/* ── Listing Section ──────────────────────────────────────── */
.list-section { margin-top: 16px; }

.list-btn {
  width: 100%; padding: 12px;
  background: #1a1a1a; color: #fff;
  border: none; border-radius: 10px;
  font-size: 14px; font-weight: 600; cursor: pointer;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  transition: background 0.2s;
}
.list-btn:hover { background: #333; }

.list-form { display: flex; flex-direction: column; gap: 8px; }
.price-label { font-size: 12px; color: #666; font-weight: 500; }
.price-input {
  padding: 10px 12px; border: 1.5px solid #e0e0e0;
  border-radius: 8px; font-size: 14px; outline: none;
}
.price-input:focus { border-color: #534AB7; }
.price-hint { font-size: 11px; color: #999; margin: 0; }

.list-actions { display: flex; gap: 8px; }
.btn-confirm {
  flex: 1; padding: 10px; background: #534AB7; color: #fff;
  border: none; border-radius: 8px; font-size: 14px; font-weight: 600;
  cursor: pointer; display: flex; align-items: center;
  justify-content: center; gap: 6px; transition: background 0.2s;
}
.btn-confirm:hover:not(:disabled) { background: #3d35a0; }
.btn-confirm:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-cancel-list {
  padding: 10px 16px; background: #f5f5f5;
  border: none; border-radius: 8px; font-size: 14px; cursor: pointer;
}
.list-error { color: #d32f2f; font-size: 13px; margin: 0; }

/* ── Listing Confirmed State ──────────────────────────────── */
.listing-confirmed {
  background: #E1F5EE; border: 1px solid #c3e6d8;
  border-radius: 12px; padding: 16px;
  display: flex; flex-direction: column; gap: 12px;
}
.confirmed-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 15px; font-weight: 600; color: #085041;
}
.confirmed-details { display: flex; flex-direction: column; gap: 6px; }
.confirmed-row {
  display: flex; justify-content: space-between;
  font-size: 13px; color: #444;
}
.confirmed-row strong { color: #085041; }
.confirmed-link { color: #534AB7; text-decoration: none; font-size: 13px; }
.confirmed-link:hover { text-decoration: underline; }
.confirmed-note { font-size: 12px; color: #666; margin: 0; }

/* ── Transfer Section ─────────────────────────────────────── */
.transfer-section { margin-top: 12px; }

.transfer-btn {
  width: 100%; padding: 12px;
  background: #f5f5f5; color: #333;
  border: 1.5px solid #e0e0e0; border-radius: 10px;
  font-size: 14px; font-weight: 600; cursor: pointer;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  transition: all 0.2s;
}
.transfer-btn:hover { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }

.transfer-form { display: flex; flex-direction: column; gap: 8px; }

/* Fee estimate box shown before confirming transfer */
.transfer-fee-info {
  background: #fafafa; border: 1px solid #e8e8e8;
  border-radius: 8px; padding: 12px;
  display: flex; flex-direction: column; gap: 6px;
}
.fee-row {
  display: flex; justify-content: space-between;
  font-size: 12px; color: #555;
}
.fee-row--total { margin-top: 2px; font-size: 13px; }
.fee-row--total strong { color: #534AB7; }
.fee-note { color: #888; font-size: 11px; }
.fee-divider { height: 1px; background: #e8e8e8; margin: 2px 0; }

/* Transfer confirmed card */
.transfer-confirmed {
  background: #EEEDFE; border: 1px solid #c5c1f0;
  border-radius: 12px; padding: 16px;
  display: flex; flex-direction: column; gap: 10px;
}
.transfer-confirmed .confirmed-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 600; color: #534AB7;
}
.confirmed-addr {
  font-family: monospace; font-size: 11px; color: #534AB7;
}

/* ── Already Listed ───────────────────────────────────────── */
.already-listed {
  margin-top: 16px; padding: 12px;
  background: #EEEDFE; border-radius: 10px;
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; color: #534AB7;
}

/* ── Right Panel ──────────────────────────────────────────── */
.detail-name { font-size: 28px; font-weight: 700; margin: 0 0 8px; }
.detail-desc { font-size: 14px; color: #666; margin: 0 0 24px; line-height: 1.6; }

.detail-section { margin-bottom: 24px; }
.detail-section-title {
  font-size: 12px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.5px; color: #888; margin: 0 0 8px;
}
.detail-rows { display: flex; flex-direction: column; gap: 8px; }
.detail-row {
  display: flex; justify-content: space-between;
  padding: 8px 0; border-bottom: 1px solid #f0f0f0; font-size: 14px;
}
.detail-label { color: #666; }
.detail-value { font-weight: 500; }
.status--minted { color: #085041; }
.status--listed  { color: #534AB7; }
.status--pending { color: #7a5c00; }

.copy-row {
  display: flex; align-items: center; gap: 8px;
  background: #f8f8f8; border-radius: 8px; padding: 10px 12px;
}
.code-value {
  font-family: monospace; font-size: 12px; color: #444;
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 0; flex-shrink: 0; display: flex; align-items: center;
}
.copy-btn:hover { opacity: 0.7; }

.ipfs-link {
  font-size: 12px; color: #534AB7;
  text-decoration: none; word-break: break-all; font-family: monospace;
}
.ipfs-link:hover { text-decoration: underline; }

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ── Mobile ───────────────────────────────────────────────── */
@media (max-width: 768px) {
  .nft-detail { padding: 16px; }
  .detail-content { grid-template-columns: 1fr; gap: 24px; }
}
</style>