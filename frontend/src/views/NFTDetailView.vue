<template>
  <div class="nft-detail">
    <div class="nft-nav">
      <router-link to="/" class="back-link" title="Back to Dashboard">
        <ArrowLeft :size="16" />
      </router-link>

      <div class="nft-nav-arrows">
        <button
          class="nav-arrow"
          :disabled="!prevNFT"
          @click="goToPrev"
          title="Previous NFT (←)"
        >
          <ChevronLeft :size="18" />
        </button>
        <span class="nav-position">
          {{ currentIndex + 1 }} / {{ dashboard.nfts.length }}
        </span>
        <button
          class="nav-arrow"
          :disabled="!nextNFT"
          @click="goToNext"
          title="Next NFT (→)"
        >
          <ChevronRight :size="18" />
        </button>
      </div>
    </div>

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

        <div class="secondary-actions">
          <a
            v-if="viewAssetUrl"
            :href="viewAssetUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn-secondary"
            title="View on Cardanoscan"
          >
            <ExternalLink :size="13" /> View Asset
          </a>

          <router-link
            v-if="nft.status === 'minted' || nft.status === 'listed'"
            :to="`/certificate/${nft.id}`"
            target="_blank"
            class="btn-secondary"
            title="View Certificate"
          >
            <Award :size="13" /> Certificate
          </router-link>
        </div>

        <div v-if="nft.status === 'pending'" class="action-section">
          <div v-if="retryMintConfirmed" class="confirmed-card confirmed-card--green">
            <div class="confirmed-header">
              <CheckCircle :size="16" /> Minted Successfully!
            </div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a
                :href="cardanoscanTxUrl(retryMintTxHash)"
                target="_blank"
                rel="noopener noreferrer"
                class="confirmed-link"
              >
                View →
              </a>
            </div>
            <div class="confirmed-row">
              <span>Asset</span>
              <a
                :href="cardanoscanTokenUrl(nft.policy_id, nft.user_token_name)"
                target="_blank"
                rel="noopener noreferrer"
                class="confirmed-link"
              >
                View Asset →
              </a>
            </div>
            <p class="confirmed-note">⚡ Confirms on-chain in ~20 seconds</p>
          </div>

          <template v-else>
            <div class="listed-notice" style="background: #FFF8E1; color: #7a5c00; border: 1px solid #f0d080;">
              ⏳ This NFT was prepared but not minted yet. Complete minting now.
            </div>
            <button
              class="btn-primary-action"
              :disabled="retryMinting"
              @click="handleRetryMint"
            >
              <Loader2 v-if="retryMinting" :size="15" class="spin" />
              <Sparkles v-else :size="15" />
              {{ retryMinting ? 'Minting...' : 'Mint Now — One Click' }}
            </button>
            <p v-if="retryMintError" class="form-error">{{ retryMintError }}</p>
          </template>
        </div>

        <div v-if="nft.status === 'minted'" class="action-section">
          <div v-if="listingConfirmed" class="confirmed-card confirmed-card--green">
            <div class="confirmed-header"><CheckCircle :size="16" /> Listed Successfully</div>
            <div class="confirmed-row">
              <span>Price</span><strong>{{ listPrice }} ADA</strong>
            </div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="cardanoscanTxUrl(listingTxHash)"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">Your NFT is now visible in Browse Mints.</p>
          </div>
          <template v-else>
            <button v-if="!showListForm" class="btn-primary-action" @click="showListForm = true">
              <Tag :size="15" /> List for Sale
            </button>
            <div v-else class="form-card">
              <label class="form-label">Price in ADA</label>
              <input v-model.number="listPrice" type="number" placeholder="e.g. 10"
                class="form-input" min="2" />
              <p class="form-hint">Minimum 2 ADA · You receive sale price minus royalties</p>
              <div class="form-actions">
                <button class="btn-confirm-action" :disabled="listing" @click="showListConfirm = true">
                  Review & Confirm
                </button>
                <button class="btn-ghost" @click="showListForm = false">Cancel</button>
              </div>
              <p v-if="listError" class="form-error">{{ listError }}</p>
            </div>
          </template>
        </div>

        <div v-if="nft.status === 'listed'" class="action-section">
          <div v-if="cancelConfirmed" class="confirmed-card confirmed-card--green">
            <div class="confirmed-header"><CheckCircle :size="16" /> Listing Cancelled</div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="cardanoscanTxUrl(cancelTxHash)"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">Your NFT has been returned to your wallet.</p>
          </div>
          <template v-else>
            <div class="listed-notice">
              <Store :size="14" /> This NFT is currently listed in the marketplace.
            </div>
            <button
              v-if="!showCancelConfirm"
              class="btn-outline-red"
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
          </template>
        </div>

        <div v-if="nft.status === 'minted'" class="transfer-section">
          <div v-if="transferConfirmed" class="confirmed-card confirmed-card--purple">
            <div class="confirmed-header"><Send :size="14" /> NFT Sent Successfully</div>
            <div class="confirmed-row">
              <span>Sent to</span>
              <code class="confirmed-addr">{{ transferAddress.slice(0,16) }}...{{ transferAddress.slice(-6) }}</code>
            </div>
            <div class="confirmed-row">
              <span>Transaction</span>
              <a :href="cardanoscanTxUrl(transferTxHash)"
                target="_blank" rel="noopener noreferrer" class="confirmed-link">View →</a>
            </div>
            <p class="confirmed-note">This NFT has been removed from your dashboard.</p>
          </div>
          <template v-else>
            <button class="btn-transfer-toggle" @click="showTransferForm = !showTransferForm">
              <Send :size="13" />
              {{ showTransferForm ? 'Cancel Transfer' : 'Transfer to Another Wallet' }}
              <ChevronDown :size="13" :class="{ 'rotate-icon': showTransferForm }" />
            </button>
            <div v-if="showTransferForm" class="form-card">
              <label class="form-label">Recipient Address</label>
              <input v-model="transferAddress" class="form-input" placeholder="addr_test1..." />
              <p class="form-hint">Enter any valid Cardano Preprod address</p>
              <div class="fee-box">
                <div class="fee-row"><span>Min-ADA with NFT</span><span>2 ADA <span class="fee-note">(goes to recipient)</span></span></div>
                <div class="fee-row"><span>Network fee</span><span>~0.2 ADA</span></div>
                <div class="fee-divider" />
                <div class="fee-row fee-row--total"><span>You will spend</span><strong>~2.2 ADA</strong></div>
              </div>
              <div class="form-actions">
                <button class="btn-confirm-action" :disabled="transferring || !transferAddress" @click="handleTransfer">
                  <Loader2 v-if="transferring" :size="13" class="spin" />
                  {{ transferring ? 'Sending...' : 'Send NFT' }}
                </button>
              </div>
              <p v-if="transferError" class="form-error">{{ transferError }}</p>
            </div>
          </template>
        </div>

      </div>

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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  AlertCircle, ArrowLeft, Award, Check, CheckCircle, ChevronDown, ChevronLeft, ChevronRight, Copy,
  ExternalLink, Image as ImageIcon,
  Loader2, Lock, Send, Sparkles, Store, Tag, X,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'
import { useAuthStore }      from '@/stores/auth'
import { useWalletSession }  from '@/composables/useWalletSession'
import { cardanoscanTokenUrl, cardanoscanTxUrl } from '@/utils/cardano'
import {
  createListing, cancelListing,
  createListingUnsigned, confirmCreateListing,
  cancelListingUnsigned, confirmCancelListing,
} from '@/services/listing'
import {
  transferNFT, transferNFTUnsigned, confirmTransfer,
  mintNFT, mintNFTUnsigned, confirmMint, submitSignedTx,
} from '@/services/nft'
import ListConfirmModal from '@/components/marketplace/ListConfirmModal.vue'
import TxSuccessCard from '@/components/shared/TxSuccessCard.vue'

const route         = useRoute()
const router        = useRouter()
const dashboard     = useDashboardStore()
const auth          = useAuthStore()
const walletSession = useWalletSession()

const loading = ref(true)
const copied  = ref<string | null>(null)

// ── Navigation ────────────────────────────────────────────────────────────────
const currentIndex = computed(() =>
  dashboard.nfts.findIndex((n: any) => n.id === route.params.id)
)
const prevNFT = computed(() =>
  currentIndex.value > 0 ? dashboard.nfts[currentIndex.value - 1] : null
)
const nextNFT = computed(() =>
  currentIndex.value < dashboard.nfts.length - 1
    ? dashboard.nfts[currentIndex.value + 1]
    : null
)
function goToPrev() { if (prevNFT.value) router.push(`/nft/${prevNFT.value.id}`) }
function goToNext() { if (nextNFT.value) router.push(`/nft/${nextNFT.value.id}`) }
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowLeft')  goToPrev()
  if (e.key === 'ArrowRight') goToNext()
}

// ── Listing flow ──────────────────────────────────────────────────────────────
const showListForm     = ref(false)
const showListConfirm  = ref(false)
const listPrice        = ref(10)
const listing          = ref(false)
const listError        = ref('')
const listingConfirmed = ref(false)
const listingTxHash    = ref('')

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

// ── Pending re-mint flow ──────────────────────────────────────────────────────
const retryMinting      = ref(false)
const retryMintError    = ref('')
const retryMintConfirmed = ref(false)
const retryMintTxHash   = ref('')

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
const ipfsUrl = computed(() => {
  const ipfs = nft.value?.image || nft.value?.image_ipfs
  if (!ipfs) return '#'
  return `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
})
const cardanoscanUrl = computed(() => {
  if (!nft.value?.policy_id || !nft.value?.user_token_name) return '#'
  return cardanoscanTokenUrl(nft.value.policy_id, nft.value.user_token_name)
})
const formattedDate = computed(() => {
  if (!nft.value?.created_at) return '—'
  return new Date(nft.value.created_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'long', day: 'numeric',
  })
})

// Builds the correct Cardanoscan token URL regardless of how the NFT was loaded
// (platform DB, external Blockfrost, or older records without user_token_name)
const viewAssetUrl = computed(() => {
  const p = nft.value?.policy_id
  if (!p) return null
  // user_token_name = "001bc280RawName" (with CIP-68 prefix)
  // asset_name = "RawName" (no prefix) — used as fallback
  const t = nft.value?.user_token_name
    || ('001bc280' + (nft.value?.asset_name || ''))
  if (!t || t === '001bc280') return null
  return cardanoscanTokenUrl(p, t)
})

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

// ── List ──────────────────────────────────────────────────────────────────────
async function handleList() {
  if (!nft.value) return
  if (listPrice.value < 2) {
    listError.value = 'Minimum price is 2 ADA'
    showListConfirm.value = false
    return
  }
  listing.value = true
  listError.value = ''
  const priceLovelace = Math.floor(listPrice.value * 1_000_000)

  try {
    if (auth.walletType === 'external') {
      const walletUtxos  = await walletSession.getUtxos()
      const unsignedRes  = await createListingUnsigned(nft.value.id, priceLovelace)
      const witnessCbor  = await walletSession.signOnly(unsignedRes.unsigned_cbor)
      const txHash       = await submitSignedTx(unsignedRes.unsigned_cbor, witnessCbor)
      await confirmCreateListing(
        unsignedRes.nft_id,
        txHash,
        unsignedRes.price_lovelace,
        unsignedRes.nft_policy_id,
        unsignedRes.nft_asset_name,
        unsignedRes.royalty_policy_id,
      )
      listingTxHash.value    = txHash
      listingConfirmed.value = true
    } else {
      const result = await createListing(nft.value.id, priceLovelace)
      listingTxHash.value    = result.tx_hash
      listingConfirmed.value = true
    }
    showListForm.value    = false
    showListConfirm.value = false
    await dashboard.loadDashboard()
  } catch (err: any) {
    listError.value = err?.message || err.response?.data?.error || 'Failed to list NFT'
    showListConfirm.value = false
  } finally {
    listing.value = false
  }
}

// ── Cancel ────────────────────────────────────────────────────────────────────
async function handleCancel() {
  if (!activeListing.value) {
    cancelError.value = 'Could not find the active listing — try refreshing.'
    return
  }
  cancelling.value  = true
  cancelError.value = ''

  try {
    if (auth.walletType === 'external') {
      const unsignedRes = await cancelListingUnsigned(activeListing.value.id)
      const witnessCbor = await walletSession.signOnly(unsignedRes.unsigned_cbor)
      const txHash      = await submitSignedTx(unsignedRes.unsigned_cbor, witnessCbor)
      await confirmCancelListing(activeListing.value.id, txHash)
      cancelTxHash.value    = txHash
      cancelConfirmed.value = true
    } else {
      const result = await cancelListing(activeListing.value.id)
      cancelTxHash.value    = result.tx_hash
      cancelConfirmed.value = true
    }
    showCancelConfirm.value = false
    await dashboard.loadDashboard()
  } catch (err: any) {
    cancelError.value = err?.message || err.response?.data?.error || 'Failed to cancel listing'
  } finally {
    cancelling.value = false
  }
}

// ── Transfer ──────────────────────────────────────────────────────────────────
async function handleTransfer() {
  if (!nft.value) return
  if (!transferAddress.value.startsWith('addr')) {
    transferError.value = 'Enter a valid Cardano address starting with addr'
    return
  }
  transferring.value  = true
  transferError.value = ''

  try {
    if (auth.walletType === 'external') {
      const unsignedRes = await transferNFTUnsigned(nft.value.id, transferAddress.value)
      const witnessCbor = await walletSession.signOnly(unsignedRes.unsigned_cbor)
      const txHash      = await submitSignedTx(unsignedRes.unsigned_cbor, witnessCbor)
      await confirmTransfer(nft.value.id, txHash, transferAddress.value)
      
      transferTxHash.value    = txHash
      transferConfirmed.value = true
      showTransferForm.value  = false

      // Show success for 3 seconds, then reload dashboard and navigate home
      setTimeout(async () => {
        await dashboard.loadDashboard()
        router.push('/')
      }, 3000)
    } else {
      const result = await transferNFT(nft.value.id, transferAddress.value)
      transferTxHash.value    = result.tx_hash
      transferConfirmed.value = true
      showTransferForm.value  = false
      
      // SAME 3 second delay then go to dashboard
      setTimeout(async () => {
        await dashboard.loadDashboard()
        router.push('/')
      }, 3000)
    }
  } catch (err: any) {
    transferError.value = err?.message || err.response?.data?.error || 'Failed to transfer NFT'
  } finally {
    transferring.value = false
  }
}

// ── Pending re-mint ───────────────────────────────────────────────────────────
// NFT is already in DB (prepare-mint ran). Just mint it — no IPFS upload needed.
async function handleRetryMint() {
  if (!nft.value) return
  retryMinting.value   = true
  retryMintError.value = ''

  try {
    if (auth.walletType === 'external') {
      const walletUtxos = await walletSession.getUtxos()
      // mintNFTUnsigned accepts wallet_utxos — backend fetches NFT data from DB
      const unsignedRes = await mintNFTUnsigned(nft.value.id, walletUtxos)
      const witnessCbor = await walletSession.signOnly(unsignedRes.unsigned_cbor)
      const txHash      = await submitSignedTx(unsignedRes.unsigned_cbor, witnessCbor)
      await confirmMint(nft.value.id, txHash, unsignedRes.policy_id)
      retryMintTxHash.value    = txHash
      retryMintConfirmed.value = true
    } else {
      // Custodial: call /api/nft/mint directly (backend has mnemonic)
      const result = await mintNFT(nft.value.id)
      retryMintTxHash.value    = result.tx_hash
      retryMintConfirmed.value = true
    }
    await dashboard.loadDashboard()
  } catch (err: any) {
    retryMintError.value = err?.message || err.response?.data?.error || 'Minting failed. Please try again.'
  } finally {
    retryMinting.value = false
  }
}

onMounted(async () => {
  if (dashboard.nfts.length === 0) await dashboard.loadDashboard()
  loading.value = false
  window.addEventListener('keydown', handleKeydown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.nft-detail { padding: 28px 32px; max-width: 1100px; }

/* ── Navigation ── */
.nft-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.nft-nav-arrows {
  display: flex;
  align-items: center;
  gap: 4px;
}

.nav-arrow {
  display: flex; align-items: center; justify-content: center;
  width: 34px; height: 34px;
  border: 1px solid #e8e8e8; border-radius: 8px;
  background: #fff; color: #444;
  cursor: pointer; transition: all 0.15s;
}
.nav-arrow:hover:not(:disabled) {
  border-color: #534AB7;
  color: #534AB7;
  background: #EEEDFE;
}
.nav-arrow:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.nav-position {
  font-size: 12px; color: #aaa;
  padding: 0 8px; white-space: nowrap;
}

.back-link {
  display: inline-flex;
  align-items: center;
  width: 32px;
  height: 32px;
  justify-content: center;
  border-radius: 8px;
  color: #888;
  text-decoration: none;
  transition: all 0.15s;
}
.back-link:hover {
  background: #EEEDFE;
  color: #534AB7;
}

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
.nft-img             { width: 100%; height: 100%; object-fit: contain ; }
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

.action-section { display: flex; flex-direction: column; gap: 8px; }

.secondary-actions {
  display: flex;
  gap: 8px;
}

.btn-secondary {
  flex: 1;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  padding: 9px 12px;
  background: #fff; color: #534AB7;
  border: 1px solid #e8e8e8;
  border-radius: 10px; text-decoration: none;
  font-size: 12px; font-weight: 600;
  transition: all 0.15s; cursor: pointer;
}
.btn-secondary:hover { border-color: #534AB7; background: #EEEDFE; }

.btn-primary-action {
  display: flex; align-items: center; justify-content: center; gap: 7px;
  width: 100%; padding: 13px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 14px; font-weight: 700;
  cursor: pointer; transition: background 0.15s;
}
.btn-primary-action:hover { background: #3d35a0; }

.btn-confirm-action {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 9px 16px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-confirm-action:hover:not(:disabled) { background: #3d35a0; }
.btn-confirm-action:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-outline-red {
  display: flex; align-items: center; justify-content: center; gap: 6px;
  width: 100%; padding: 10px;
  background: transparent; color: #d32f2f;
  border: 1px solid #d32f2f; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: all 0.15s;
}
.btn-outline-red:hover { background: #FCEBEB; }

.btn-transfer-toggle {
  display: flex; align-items: center; gap: 6px;
  width: 100%; padding: 10px 14px;
  background: #fafafa; color: #666;
  border: 1px solid #e8e8e8; border-radius: 10px;
  font-size: 12px; font-weight: 500;
  cursor: pointer; transition: all 0.15s;
  justify-content: center;
}
.btn-transfer-toggle:hover { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }

.rotate-icon { transform: rotate(180deg); transition: transform 0.2s; }

.transfer-section { display: flex; flex-direction: column; gap: 8px; }

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