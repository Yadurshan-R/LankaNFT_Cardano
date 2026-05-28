<template>
  <div class="nft-detail">
    <router-link to="/" class="back-link">
      <ArrowLeft :size="14" /> Back to Dashboard
    </router-link>

    <div v-if="loading" class="detail-grid">
      <div class="skeleton skeleton--image" />
      <div class="sk-right">
        <div class="skeleton skeleton--title" />
        <div class="skeleton skeleton--line" />
        <div class="skeleton skeleton--line" style="width:60%" />
        <div class="skeleton skeleton--line" />
      </div>
    </div>

    <div v-else-if="!nft" class="empty-state">
      <ImageIcon :size="40" color="#ddd" />
      <p class="empty-title">NFT not found</p>
      <router-link to="/"><button class="btn-primary">Go to Dashboard</button></router-link>
    </div>

    <div v-else class="detail-grid">
      <!-- ── Left column ── -->
      <div class="detail-left">
        <div class="img-wrap">
          <img v-if="imageUrl" :src="imageUrl" :alt="nft.name" class="nft-img" />
          <div v-else class="nft-img-placeholder">
            <ImageIcon :size="64" color="#ddd" />
          </div>
          <span :class="['status-badge', `status-badge--${nft.status}`]">{{ nft.status }}</span>
          <span v-if="nft.privacy === 'private'" class="privacy-badge">
            <Lock :size="11" /> Private
          </span>
        </div>

        <a
          v-if="nft.tx_hash"
          :href="`https://preprod.cardanoscan.io/transaction/${nft.tx_hash}`"
          target="_blank"
          rel="noopener noreferrer"
          class="btn-scan"
        >
          <ExternalLink :size="13" />
          View on Cardanoscan
        </a>

        <!-- ── List for Sale ── -->
        <div v-if="nft.status === 'minted'" class="action-section">
          <div v-if="listingConfirmed" class="confirmed-card confirmed-card--green">
            <div class="confirmed-header"><CheckCircle :size="16" /> Listed Successfully</div>
            <div class="confirmed-row">
              <span>Price</span><strong>{{ listPrice }} ADA</strong>
            </div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="`https://preprod.cardanoscan.io/transaction/${listingTxHash}`"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">Your NFT is now visible in Browse Mints.</p>
          </div>
          <div v-else>
            <button v-if="!showListForm" class="btn-action btn-action--dark" @click="showListForm = true">
              <Tag :size="14" /> List for Sale
            </button>
            <div v-else class="form-card">
              <label class="form-label">Price in ADA</label>
              <input v-model.number="listPrice" type="number" placeholder="e.g. 10"
                class="form-input" min="2" />
              <p class="form-hint">Minimum 2 ADA · You receive sale price minus royalties</p>
              <div class="form-actions">
                <button class="btn-primary" :disabled="listing" @click="showListConfirm = true">
                  Review & Confirm
                </button>
                <button class="btn-ghost" @click="showListForm = false">Cancel</button>
              </div>
              <p v-if="listError" class="form-error">{{ listError }}</p>
            </div>
          </div>
        </div>

        <!-- ── Cancel Listing ── -->
        <div v-if="nft.status === 'listed'" class="action-section">
          <div v-if="cancelConfirmed" class="confirmed-card confirmed-card--green">
            <div class="confirmed-header"><CheckCircle :size="16" /> Listing Cancelled</div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="`https://preprod.cardanoscan.io/transaction/${cancelTxHash}`"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">Your NFT has been returned to your wallet.</p>
          </div>
          <div v-else>
            <div class="listed-notice">
              <Store :size="14" />
              This NFT is currently listed in the marketplace.
            </div>
            <button
              v-if="!showCancelConfirm"
              class="btn-action btn-action--outline-red"
              @click="showCancelConfirm = true"
            >
              <X :size="14" /> Cancel Listing
            </button>
            <div v-else class="form-card form-card--warn">
              <div class="warn-header">
                <AlertCircle :size="15" color="#854F0B" /> Cancel this listing?
              </div>
              <p class="form-hint">
                The NFT will be returned to your wallet. You'll pay ~0.17 ADA network fee.
              </p>
              <div class="form-actions">
                <button class="btn-danger" :disabled="cancelling" @click="handleCancel">
                  <Loader2 v-if="cancelling" :size="13" class="spin" />
                  {{ cancelling ? 'Cancelling...' : 'Yes, Cancel Listing' }}
                </button>
                <button class="btn-ghost" @click="showCancelConfirm = false">Keep Listed</button>
              </div>
              <p v-if="cancelError" class="form-error">{{ cancelError }}</p>
            </div>
          </div>
        </div>

        <!-- ── Transfer NFT ── -->
        <div v-if="nft.status === 'minted'" class="action-section">
          <div v-if="transferConfirmed" class="confirmed-card confirmed-card--purple">
            <div class="confirmed-header"><Send :size="14" /> NFT Sent Successfully</div>
            <div class="confirmed-row">
              <span>Sent to</span>
              <code class="confirmed-addr">{{ transferAddress.slice(0,16) }}...{{ transferAddress.slice(-6) }}</code>
            </div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="`https://preprod.cardanoscan.io/transaction/${transferTxHash}`"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">This NFT has been removed from your dashboard.</p>
          </div>
          <div v-else>
            <button v-if="!showTransferForm" class="btn-action btn-action--outline" @click="showTransferForm = true">
              <Send :size="14" /> Transfer NFT
            </button>
            <div v-else class="form-card">
              <label class="form-label">Recipient Address</label>
              <input v-model="transferAddress" class="form-input" placeholder="addr_test1..." />
              <p class="form-hint">Enter any valid Cardano preprod address</p>
              <div class="fee-box">
                <div class="fee-row"><span>Min-ADA sent with NFT</span><span>2 ADA <span class="fee-note">(goes to recipient)</span></span></div>
                <div class="fee-row"><span>Network fee</span><span>~0.2 ADA</span></div>
                <div class="fee-divider" />
                <div class="fee-row fee-row--total"><span>You will spend</span><strong>~2.2 ADA</strong></div>
              </div>
              <div class="form-actions">
                <button class="btn-primary" :disabled="transferring || !transferAddress" @click="handleTransfer">
                  <Loader2 v-if="transferring" :size="13" class="spin" />
                  {{ transferring ? 'Sending...' : 'Send NFT' }}
                </button>
                <button class="btn-ghost" @click="showTransferForm = false; transferError = ''">Cancel</button>
              </div>
              <p v-if="transferError" class="form-error">{{ transferError }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Right column ── -->
      <div class="detail-right">
        <h1 class="detail-name">{{ nft.name }}</h1>
        <p v-if="nft.description" class="detail-desc">{{ nft.description }}</p>

        <div class="info-panel">
          <div class="info-panel-title">Details</div>
          <div class="info-row">
            <span class="info-label">Status</span>
            <span :class="['info-value', `info-value--${nft.status}`]">{{ nft.status }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Royalties</span>
            <span class="info-value">{{ nft.royalties }}%</span>
          </div>
          <div class="info-row">
            <span class="info-label">Privacy</span>
            <span class="info-value">{{ nft.privacy }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Minted</span>
            <span class="info-value">{{ formattedDate }}</span>
          </div>
        </div>

        <div v-if="nft.policy_id" class="info-panel">
          <div class="info-panel-title">Policy ID</div>
          <div class="copy-row">
            <code class="code-val">{{ nft.policy_id }}</code>
            <button class="copy-btn" @click="copy(nft.policy_id, 'policy')">
              <Check v-if="copied === 'policy'" :size="13" color="#085041" />
              <Copy v-else :size="13" color="#534AB7" />
            </button>
          </div>
        </div>

        <div class="info-panel">
          <div class="info-panel-title">Asset Name</div>
          <div class="copy-row">
            <code class="code-val">{{ nft.asset_name }}</code>
            <button class="copy-btn" @click="copy(nft.asset_name, 'asset')">
              <Check v-if="copied === 'asset'" :size="13" color="#085041" />
              <Copy v-else :size="13" color="#534AB7" />
            </button>
          </div>
        </div>

        <div v-if="nft.tx_hash" class="info-panel">
          <div class="info-panel-title">Transaction Hash</div>
          <div class="copy-row">
            <code class="code-val">{{ nft.tx_hash }}</code>
            <button class="copy-btn" @click="copy(nft.tx_hash!, 'tx')">
              <Check v-if="copied === 'tx'" :size="13" color="#085041" />
              <Copy v-else :size="13" color="#534AB7" />
            </button>
          </div>
        </div>

        <div v-if="nft.image" class="info-panel">
          <div class="info-panel-title">IPFS Image</div>
          <a :href="nft.image.replace('ipfs://', 'https://gateway.pinata.cloud/ipfs/')"
            target="_blank" rel="noopener noreferrer" class="ipfs-link">
            {{ nft.image }}
          </a>
        </div>
      </div>
    </div>

    <ListConfirmModal
      :show="showListConfirm"
      :nft="nft"
      :price="listPrice"
      :loading="listing"
      @confirm="handleList"
      @cancel="showListConfirm = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  AlertCircle, ArrowLeft, Check, CheckCircle, Copy,
  ExternalLink, Image as ImageIcon,
  Loader2, Lock, Send, Store, Tag, X,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'
import { createListing, cancelListing } from '@/services/listing'
import { transferNFT } from '@/services/nft'
import ListConfirmModal from '@/components/marketplace/ListConfirmModal.vue'

const route     = useRoute()
const dashboard = useDashboardStore()

const loading = ref(true)
const copied  = ref<string | null>(null)

// ── Listing flow ──────────────────────────────────────────────────────────────
const showListForm    = ref(false)
const showListConfirm = ref(false)
const listPrice       = ref(10)
const listing         = ref(false)
const listError       = ref('')
const listingConfirmed = ref(false)
const listingTxHash   = ref('')

// ── Cancel flow ───────────────────────────────────────────────────────────────
const showCancelConfirm = ref(false)
const cancelling        = ref(false)
const cancelError       = ref('')
const cancelConfirmed   = ref(false)
const cancelTxHash      = ref('')

// ── Transfer flow ─────────────────────────────────────────────────────────────
const showTransferForm  = ref(false)
const transferAddress   = ref('')
const transferring      = ref(false)
const transferError     = ref('')
const transferConfirmed = ref(false)
const transferTxHash    = ref('')

// ── Computed ──────────────────────────────────────────────────────────────────
const nft = computed(() =>
  dashboard.nfts.find((n) => n.id === route.params.id)
)

const imageUrl = computed(() => {
  if (!nft.value?.image) return null
  return nft.value.image.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${nft.value.image.replace('ipfs://', '')}`
    : nft.value.image
})

const formattedDate = computed(() => {
  if (!nft.value?.created_at) return '—'
  return new Date(nft.value.created_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'long', day: 'numeric',
  })
})

// Find the active listing for this NFT (needed for cancel)
const activeListing = computed(() =>
  dashboard.myListings.find(
    (l: any) => l.nft_id === nft.value?.id && l.status === 'active'
  )
)

// ── Methods ───────────────────────────────────────────────────────────────────
async function copy(text: string, key: string) {
  await navigator.clipboard.writeText(text)
  copied.value = key
  setTimeout(() => (copied.value = null), 2000)
}

async function handleList() {
  if (!nft.value) return
  if (listPrice.value < 2) {
    listError.value = 'Minimum price is 2 ADA'
    showListConfirm.value = false
    return
  }
  listing.value = true
  listError.value = ''
  try {
    const result = await createListing(nft.value.id, Math.floor(listPrice.value * 1_000_000))
    listingTxHash.value = result.tx_hash
    listingConfirmed.value = true
    showListForm.value = false
    showListConfirm.value = false
    await dashboard.loadDashboard()
  } catch (err: any) {
    listError.value = err.response?.data?.error || 'Failed to list NFT'
    showListConfirm.value = false
  } finally {
    listing.value = false
  }
}

async function handleCancel() {
  if (!activeListing.value) {
    cancelError.value = 'Could not find the active listing — try refreshing.'
    return
  }
  cancelling.value = true
  cancelError.value = ''
  try {
    const result = await cancelListing(activeListing.value.id)
    cancelTxHash.value = result.tx_hash
    cancelConfirmed.value = true
    showCancelConfirm.value = false
    await dashboard.loadDashboard()
  } catch (err: any) {
    cancelError.value = err.response?.data?.error || 'Failed to cancel listing'
  } finally {
    cancelling.value = false
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
    await dashboard.loadDashboard()
  } catch (err: any) {
    transferError.value = err.response?.data?.error || 'Failed to transfer NFT'
  } finally {
    transferring.value = false
  }
}

onMounted(async () => {
  if (dashboard.nfts.length === 0) await dashboard.loadDashboard()
  loading.value = false
})
</script>

<style scoped>
.nft-detail { padding: 28px 32px; max-width: 1100px; }

.back-link {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 13px; color: #888; text-decoration: none;
  margin-bottom: 24px; transition: color 0.15s;
}
.back-link:hover { color: #534AB7; }

.detail-grid {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 40px;
  align-items: start;
}

/* ── Left column ── */
.detail-left { display: flex; flex-direction: column; gap: 12px; }

.img-wrap {
  position: relative; aspect-ratio: 1;
  border-radius: 16px; overflow: hidden;
  background: #f8f8f8; border: 1px solid #f0f0f0;
}
.nft-img             { width: 100%; height: 100%; object-fit: cover; }
.nft-img-placeholder { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }

.status-badge {
  position: absolute; top: 12px; left: 12px;
  font-size: 10px; font-weight: 600; padding: 4px 10px;
  border-radius: 20px; text-transform: uppercase;
}
.status-badge--minted  { background: #E1F5EE; color: #085041; }
.status-badge--listed  { background: #EEEDFE; color: #534AB7; }
.status-badge--pending { background: #FFF8E1; color: #7a5c00; }

.privacy-badge {
  position: absolute; top: 12px; right: 12px;
  display: flex; align-items: center; gap: 4px;
  font-size: 10px; font-weight: 500;
  background: rgba(0,0,0,0.55); color: #fff;
  padding: 4px 10px; border-radius: 20px;
}

.btn-scan {
  display: flex; align-items: center; justify-content: center; gap: 6px;
  padding: 11px; background: #534AB7; color: #fff;
  border-radius: 10px; text-decoration: none;
  font-size: 13px; font-weight: 600; transition: background 0.15s;
}
.btn-scan:hover { background: #3d35a0; }

.action-section { display: flex; flex-direction: column; gap: 8px; }

.btn-action {
  display: flex; align-items: center; justify-content: center;
  gap: 6px; width: 100%; padding: 11px;
  border-radius: 10px; font-size: 13px; font-weight: 600;
  cursor: pointer; transition: all 0.15s; border: none;
}
.btn-action--dark         { background: #1a1a1a; color: #fff; }
.btn-action--dark:hover   { background: #333; }
.btn-action--outline      { background: transparent; color: #534AB7; border: 1px solid #534AB7; }
.btn-action--outline:hover { background: #EEEDFE; }
.btn-action--outline-red  { background: transparent; color: #d32f2f; border: 1px solid #d32f2f; }
.btn-action--outline-red:hover { background: #FCEBEB; }

.listed-notice {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 14px; background: #EEEDFE;
  border-radius: 10px; font-size: 13px; color: #534AB7;
  margin-bottom: 4px;
}

.form-card {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 12px; padding: 16px;
  display: flex; flex-direction: column; gap: 8px;
}
.form-card--warn { border-color: #FAEEDA; background: #FFFBF5; }

.warn-header {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600; color: #854F0B;
}

.form-label { font-size: 12px; font-weight: 500; color: #555; }
.form-input {
  padding: 9px 12px; border: 1px solid #e8e8e8;
  border-radius: 8px; font-size: 13px; outline: none;
}
.form-input:focus { border-color: #534AB7; }
.form-hint  { font-size: 11px; color: #aaa; }
.form-error { font-size: 12px; color: #d32f2f; }

.form-actions { display: flex; gap: 8px; }

.btn-primary {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 9px 16px; background: #534AB7; color: #fff;
  border: none; border-radius: 8px; font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-primary:hover:not(:disabled) { background: #3d35a0; }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-danger {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 9px 16px; background: #d32f2f; color: #fff;
  border: none; border-radius: 8px; font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-danger:hover:not(:disabled) { background: #b71c1c; }
.btn-danger:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-ghost {
  padding: 9px 14px; background: #f5f5f5; border: none;
  border-radius: 8px; font-size: 13px; cursor: pointer;
}
.btn-ghost:hover { background: #ececec; }

.fee-box {
  background: #fafafa; border: 1px solid #f0f0f0;
  border-radius: 8px; padding: 12px;
  display: flex; flex-direction: column; gap: 5px;
}
.fee-row         { display: flex; justify-content: space-between; font-size: 12px; color: #555; }
.fee-row--total  { font-size: 13px; font-weight: 500; }
.fee-row--total strong { color: #534AB7; }
.fee-note        { font-size: 10px; color: #aaa; }
.fee-divider     { height: 1px; background: #ebebeb; margin: 2px 0; }

.confirmed-card {
  border-radius: 12px; padding: 14px;
  display: flex; flex-direction: column; gap: 8px;
}
.confirmed-card--green  { background: #E1F5EE; border: 1px solid #b8e0cc; }
.confirmed-card--purple { background: #EEEDFE; border: 1px solid #c5c1f0; }

.confirmed-header {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600;
}
.confirmed-card--green  .confirmed-header { color: #085041; }
.confirmed-card--purple .confirmed-header { color: #534AB7; }

.confirmed-row { display: flex; justify-content: space-between; font-size: 12px; color: #444; }
.confirmed-row strong { color: #085041; }
.confirmed-link { color: #534AB7; text-decoration: none; font-size: 12px; }
.confirmed-link:hover { text-decoration: underline; }
.confirmed-addr { font-family: monospace; font-size: 11px; color: #534AB7; }
.confirmed-note { font-size: 11px; color: #888; }

/* ── Right column ── */
.detail-right { display: flex; flex-direction: column; gap: 16px; }
.detail-name  { font-size: 26px; font-weight: 600; margin: 0; }
.detail-desc  { font-size: 13px; color: #888; line-height: 1.6; margin: 0; }

.info-panel {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 12px; padding: 16px;
}
.info-panel-title {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #aaa; margin-bottom: 10px;
}
.info-row {
  display: flex; justify-content: space-between;
  align-items: center; padding: 8px 0;
  border-bottom: 1px solid #f5f5f5; font-size: 13px;
}
.info-row:last-child { border-bottom: none; }
.info-label { color: #888; }
.info-value { font-weight: 500; }
.info-value--minted  { color: #085041; }
.info-value--listed  { color: #534AB7; }
.info-value--pending { color: #7a5c00; }

.copy-row {
  display: flex; align-items: center; gap: 8px;
  background: #f8f8f8; border-radius: 8px; padding: 10px 12px;
}
.code-val {
  font-family: monospace; font-size: 11px; color: #444;
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 0; flex-shrink: 0; display: flex; align-items: center;
}
.copy-btn:hover { opacity: 0.7; }

.ipfs-link {
  font-family: monospace; font-size: 11px; color: #534AB7;
  text-decoration: none; word-break: break-all; line-height: 1.5;
}
.ipfs-link:hover { text-decoration: underline; }

/* Skeleton */
.skeleton {
  border-radius: 12px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.skeleton--image { aspect-ratio: 1; }
.skeleton--title { height: 28px; width: 60%; margin-bottom: 8px; }
.skeleton--line  { height: 14px; margin-bottom: 6px; }
.sk-right        { display: flex; flex-direction: column; gap: 10px; padding-top: 8px; }

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.empty-state {
  display: flex; flex-direction: column; align-items: center;
  gap: 12px; padding: 80px; text-align: center;
}
.empty-title { font-size: 16px; font-weight: 600; color: #444; }

@media (max-width: 768px) {
  .nft-detail  { padding: 16px; }
  .detail-grid { grid-template-columns: 1fr; gap: 24px; }
}
</style>