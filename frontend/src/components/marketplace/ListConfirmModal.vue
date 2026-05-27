<!-- ─────────────────────────────────────────────────────────────────────────
  ListConfirmModal.vue — Listing confirmation modal for LankaNFT

  Shows a full cost breakdown before the seller submits a listing transaction.
  Parent passes the nft object and price. Modal emits 'confirm' or 'cancel'.

  Cost breakdown includes:
    - Listing price set by seller
    - Min-ADA locked with NFT at marketplace script (~3 ADA, returned on sale/cancel)
    - Network fee (~0.17 ADA estimated)
    - Royalty % the seller will pay on each sale
──────────────────────────────────────────────────────────────────────────── -->
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="onCancel">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">

          <!-- Header -->
          <div class="modal-header">
            <h2 id="modal-title" class="modal-title">Confirm Listing</h2>
            <button class="modal-close" :disabled="loading" @click="onCancel" aria-label="Close">
              <X :size="18" />
            </button>
          </div>

          <!-- NFT preview -->
          <div class="modal-nft">
            <div class="nft-thumb">
              <img v-if="imgUrl" :src="imgUrl" :alt="nft?.name" class="nft-img" />
              <ImageIcon v-else :size="28" color="#ddd" />
            </div>
            <div class="nft-meta">
              <div class="nft-name">{{ nft?.name }}</div>
              <div class="nft-royalty">{{ nft?.royalties }}% royalty on every resale</div>
            </div>
          </div>

          <!-- Cost breakdown -->
          <div class="breakdown">
            <div class="breakdown-label">Transaction breakdown</div>

            <div class="breakdown-row">
              <span>Listing price</span>
              <span class="breakdown-price">{{ price }} ADA</span>
            </div>
            <div class="breakdown-row">
              <span>Min-ADA locked with NFT <span class="note">(returned on sale/cancel)</span></span>
              <span>~3.00 ADA</span>
            </div>
            <div class="breakdown-row">
              <span>Network fee <span class="note">(estimated)</span></span>
              <span>~0.17 ADA</span>
            </div>

            <div class="breakdown-divider" />

            <div class="breakdown-row breakdown-row--total">
              <span>You will spend now</span>
              <strong>~{{ totalCost }} ADA</strong>
            </div>

            <div class="breakdown-row breakdown-row--receive">
              <span>You will receive on sale</span>
              <strong class="receive-amount">~{{ receiveAmount }} ADA</strong>
            </div>
          </div>

          <!-- Info note -->
          <div class="modal-info">
            <Info :size="14" />
            The NFT is locked at the marketplace smart contract. You can cancel
            anytime to get it back.
          </div>

          <!-- Actions -->
          <div class="modal-actions">
            <button class="btn-cancel" :disabled="loading" @click="onCancel">
              Cancel
            </button>
            <button class="btn-confirm" :disabled="loading" @click="onConfirm">
              <Loader2 v-if="loading" :size="14" class="spin" />
              <Tag v-else :size="14" />
              {{ loading ? 'Submitting...' : 'Confirm Listing' }}
            </button>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Image as ImageIcon, Info, Loader2, Tag, X } from 'lucide-vue-next'

// ── Props ─────────────────────────────────────────────────────────────────
const props = defineProps<{
  show:    boolean   // controls modal visibility
  nft:     any       // the NFT being listed
  price:   number    // listing price in ADA (not lovelace)
  loading: boolean   // true while transaction is in progress
}>()

// ── Emits ─────────────────────────────────────────────────────────────────
const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'):  void
}>()

// ── Computed ──────────────────────────────────────────────────────────────

// Convert IPFS URI to HTTP gateway URL
const imgUrl = computed(() => {
  const ipfs = props.nft?.image
  if (!ipfs) return null
  return ipfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
    : ipfs
})

// Total ADA spent now = min-ADA locked + network fee
// The listing price itself stays with the NFT until sold
const totalCost = computed(() => {
  return (3 + 0.17).toFixed(2)
})

// What the seller receives after royalty is deducted
const receiveAmount = computed(() => {
  const royaltyPct = (props.nft?.royalties ?? 5) / 100
  const royalty    = props.price * royaltyPct
  return (props.price - royalty).toFixed(2)
})

function onConfirm() { emit('confirm') }
function onCancel()  { if (!props.loading) emit('cancel') }
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex; align-items: center; justify-content: center;
  z-index: 500; padding: 20px;
}

.modal {
  background: #fff;
  border-radius: 16px;
  width: 100%; max-width: 420px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.15);
  overflow: hidden;
}

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

.modal-nft {
  display: flex; align-items: center; gap: 12px;
  padding: 16px 20px; border-bottom: 1px solid #f0f0f0;
}
.nft-thumb {
  width: 56px; height: 56px; border-radius: 10px;
  background: #f8f8f8; overflow: hidden; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
}
.nft-img     { width: 100%; height: 100%; object-fit: cover; }
.nft-name    { font-size: 14px; font-weight: 600; margin-bottom: 3px; }
.nft-royalty { font-size: 11px; color: #888; }

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
.breakdown-price   { font-weight: 600; color: #534AB7; }
.breakdown-row--total {
  font-size: 14px; font-weight: 600; color: #111; padding-top: 8px;
}
.breakdown-row--total strong { color: #534AB7; }
.breakdown-row--receive { padding-top: 4px; }
.receive-amount { color: #085041; }
.breakdown-divider { height: 1px; background: #f0f0f0; margin: 6px 0; }
.note { font-size: 10px; color: #aaa; }

.modal-info {
  display: flex; align-items: flex-start; gap: 6px;
  margin: 0 20px 16px;
  padding: 10px 12px;
  background: #EEEDFE; border-radius: 8px;
  font-size: 12px; color: #534AB7; line-height: 1.5;
}

.modal-actions {
  display: flex; gap: 10px;
  padding: 0 20px 20px;
}
.btn-cancel {
  flex: 1; padding: 11px;
  background: #f5f5f5; border: none;
  border-radius: 10px; font-size: 13px;
  font-weight: 500; cursor: pointer; color: #555;
}
.btn-cancel:hover:not(:disabled) { background: #ebebeb; }
.btn-cancel:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-confirm {
  flex: 2; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 11px;
  background: #1a1a1a; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-confirm:hover:not(:disabled) { background: #333; }
.btn-confirm:disabled { opacity: 0.6; cursor: not-allowed; }

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.modal-enter-active, .modal-leave-active { transition: all 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: scale(0.96); }
</style>