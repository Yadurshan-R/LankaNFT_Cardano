<!-- ─────────────────────────────────────────────────────────────────────────
  BuyConfirmModal.vue — Purchase confirmation modal for LankaNFT marketplace

  Shows a full price breakdown before the buyer commits to a transaction.
  Parent passes the listing object. Modal emits 'confirm' or 'cancel'.

  Price breakdown includes:
    - NFT price (from listing)
    - Royalty amount (calculated from listing.royalties %)
    - Network fee (~0.17 ADA estimated)
    - Min-ADA sent with NFT (~2 ADA, returned to buyer as UTxO)
    - Total the buyer will spend
──────────────────────────────────────────────────────────────────────────── -->
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="listing" class="modal-overlay" @click.self="onCancel">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">

          <!-- Header -->
          <div class="modal-header">
            <h2 id="modal-title" class="modal-title">Confirm Purchase</h2>
            <button class="modal-close" :disabled="loading" @click="onCancel" aria-label="Close">
              <X :size="18" />
            </button>
          </div>

          <!-- NFT preview -->
          <div class="modal-nft">
            <div class="nft-thumb">
              <img v-if="imgUrl" :src="imgUrl" :alt="listing.nft_name" class="nft-img" />
              <ImageIcon v-else :size="28" color="#ddd" />
            </div>
            <div class="nft-meta">
              <div class="nft-name">{{ listing.nft_name }}</div>
              <div class="nft-royalty">{{ listing.royalties ?? 5 }}% royalty on every resale</div>
            </div>
          </div>

          <!-- Price breakdown -->
          <div class="breakdown">
            <div class="breakdown-label">Price breakdown</div>

            <div class="breakdown-row">
              <span>NFT price</span>
              <span>{{ adaAmount(listing.price_lovelace) }} ₳</span>
            </div>
            <div class="breakdown-row">
              <span>Royalty ({{ listing.royalties ?? 5 }}%)</span>
              <span>{{ royaltyAda }} ₳</span>
            </div>
            <div class="breakdown-row">
              <span>Network fee <span class="note">(estimated)</span></span>
              <span>~0.17 ₳</span>
            </div>
            <div class="breakdown-row">
              <span>Min-ADA <span class="note">(returned to you as UTxO)</span></span>
              <span>~2.00 ₳</span>
            </div>

            <div class="breakdown-divider" />

            <div class="breakdown-row breakdown-row--total">
              <span>Total you pay</span>
              <strong>~{{ totalAda }} ₳</strong>
            </div>
          </div>

          <!-- Wallet warning -->
          <div class="modal-warning">
            <AlertCircle :size="14" />
            Make sure your wallet has at least
            <strong>{{ minRequired }} ₳</strong> before confirming.
          </div>

          <!-- Actions -->
          <div class="modal-actions">
            <button class="btn-cancel" :disabled="loading" @click="onCancel">
              Cancel
            </button>
            <button class="btn-confirm" :disabled="loading" @click="onConfirm">
              <Loader2 v-if="loading" :size="14" class="spin" />
              <ShoppingCart v-else :size="14" />
              {{ loading ? 'Processing...' : 'Confirm Purchase' }}
            </button>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { AlertCircle, Image as ImageIcon, Loader2, ShoppingCart, X } from 'lucide-vue-next'

// ── Props ─────────────────────────────────────────────────────────────────
const props = defineProps<{
  listing: any | null   // the listing being purchased — null means modal is hidden
  loading: boolean      // true while the buy transaction is in progress
}>()

// ── Emits ─────────────────────────────────────────────────────────────────
const emit = defineEmits<{
  (e: 'confirm'): void  // user confirmed — parent calls buyListing()
  (e: 'cancel'):  void  // user cancelled or clicked backdrop
}>()

// ── Computed ──────────────────────────────────────────────────────────────

// Convert IPFS URI to HTTP gateway URL for image display
const imgUrl = computed(() => {
  const ipfs = props.listing?.image_ipfs
  if (!ipfs) return null
  return ipfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
    : ipfs
})

// Royalty amount in ADA (royalty % of NFT price)
const royaltyAda = computed(() => {
  if (!props.listing) return '0.00'
  const pct    = (props.listing.royalties ?? 5) / 100
  const royalty = (props.listing.price_lovelace / 1_000_000) * pct
  return royalty.toFixed(2)
})

// Total = NFT price + network fee + min-ADA
// Note: royalty is paid from the sale price by the smart contract,
// not additionally charged to the buyer
const totalAda = computed(() => {
  if (!props.listing) return '0.00'
  const price      = props.listing.price_lovelace / 1_000_000
  const networkFee = 0.17
  const minAda     = 2
  return (price + networkFee + minAda).toFixed(2)
})

// Minimum wallet balance recommended before buying
const minRequired = computed(() => {
  return Math.ceil(parseFloat(totalAda.value)) + 1
})

// ── Helpers ───────────────────────────────────────────────────────────────

function adaAmount(lovelace: number): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

function onConfirm() { emit('confirm') }
function onCancel()  { if (!props.loading) emit('cancel') }
</script>

<style scoped>
/* ── Overlay ────────────────────────────────────────────────────────────── */
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex; align-items: center; justify-content: center;
  z-index: 500; padding: 20px;
}

/* ── Modal box ────────────────────────────────────────────────────────────── */
.modal {
  background: #fff;
  border-radius: 16px;
  width: 100%; max-width: 420px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
  overflow: hidden;
}

/* Header */
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 18px 20px 0;
}
.modal-title { font-size: 16px; font-weight: 600; }
.modal-close {
  background: none; border: none; cursor: pointer;
  color: #888; padding: 4px; border-radius: 6px;
}
.modal-close:hover:not(:disabled) { background: #f5f5f5; color: #333; }
.modal-close:disabled { opacity: 0.4; cursor: not-allowed; }

/* NFT preview */
.modal-nft {
  display: flex; align-items: center; gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}
.nft-thumb {
  width: 56px; height: 56px; border-radius: 10px;
  background: #f8f8f8; overflow: hidden; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
}
.nft-img     { width: 100%; height: 100%; object-fit: cover; }
.nft-name    { font-size: 14px; font-weight: 600; margin-bottom: 3px; }
.nft-royalty { font-size: 11px; color: #888; }

/* Price breakdown */
.breakdown { padding: 16px 20px; }
.breakdown-label {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #aaa; margin-bottom: 10px;
}
.breakdown-row {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; color: #555; padding: 5px 0;
}
.breakdown-row--total {
  font-size: 15px; font-weight: 600; color: #111;
  padding-top: 8px;
}
.breakdown-row--total strong { color: #534AB7; font-size: 17px; }
.breakdown-divider { height: 1px; background: #f0f0f0; margin: 6px 0; }
.note { font-size: 10px; color: #aaa; }

/* Warning */
.modal-warning {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  margin: 0 20px 16px;
  padding: 10px 12px;
  background: #FFF8E1; border-radius: 8px;
  font-size: 12px; color: #7a5c00; line-height: 1.5;
}
.modal-warning strong { color: #5a3e00; }

/* Actions */
.modal-actions {
  display: flex; gap: 10px;
  padding: 0 20px 20px;
}
.btn-cancel {
  flex: 1; padding: 11px;
  background: #f5f5f5; border: none;
  border-radius: 10px; font-size: 13px;
  font-weight: 500; cursor: pointer; color: #555;
  transition: background 0.15s;
}
.btn-cancel:hover:not(:disabled) { background: #ebebeb; }
.btn-cancel:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-confirm {
  flex: 2; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 11px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-confirm:hover:not(:disabled) { background: #3d35a0; }
.btn-confirm:disabled { opacity: 0.6; cursor: not-allowed; }

/* Spinner */
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Modal animation */
.modal-enter-active, .modal-leave-active { transition: all 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: scale(0.96); }
</style>