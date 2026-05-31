<template>
  <Teleport to="body">
    <Transition name="panel">
      <div v-if="listing" class="panel-overlay" @click.self="onClose">
        <div class="panel" role="dialog" aria-modal="true">

          <button class="panel-close" @click="onClose" title="Close">
            <X :size="20" />
          </button>

          <div class="panel-inner">

            <div class="panel-left">
              <div class="panel-img-wrap">
                <img
                  v-if="imgUrl"
                  :src="imgUrl"
                  :alt="listing.nft_name"
                  class="panel-img"
                />
                <div v-else class="panel-img-placeholder">
                  <ImageIcon :size="64" color="#ddd" />
                </div>
              </div>

              <div class="panel-img-meta">
                <div class="img-meta-row">
                  <span class="img-meta-label">Network</span>
                  <span class="network-badge">
                    <span class="network-dot" /> Cardano Preprod
                  </span>
                </div>
                <div class="img-meta-row">
                  <span class="img-meta-label">Royalties</span>
                  <span class="img-meta-value">{{ listing.royalties ?? 5 }}%</span>
                </div>
              </div>
            </div>

            <div class="panel-right">

              <div class="panel-name-section">
                <h2 class="panel-nft-name">{{ listing.nft_name }}</h2>
                <div class="panel-collection">
                  <ShieldCheck :size="14" color="#534AB7" />
                  <span>LankaNFT · Verified on Cardano</span>
                </div>
              </div>

              <div class="panel-buy-section">
                <div class="buy-label">Current Price</div>
                <div class="buy-price">
                  {{ adaAmount(listing.price_lovelace) }}
                  <span class="buy-currency">₳</span>
                </div>

                <template v-if="listing.is_own">
                  <div class="own-listing-notice">
                    <Tag :size="16" />
                    This is your listing
                  </div>
                </template>
                <template v-else-if="!showConfirm">
                  <button class="btn-buy" @click="showConfirm = true">
                    <ShoppingCart :size="16" /> Buy Now
                  </button>
                </template>

                <template v-else>
                  <div class="confirm-box">
                    <div class="confirm-title">Price Breakdown</div>

                    <div class="confirm-row">
                      <span>NFT price</span>
                      <span>{{ adaAmount(listing.price_lovelace) }} ₳</span>
                    </div>
                    <div class="confirm-row">
                      <span>Royalty ({{ listing.royalties ?? 5 }}%)</span>
                      <span>{{ royaltyAda }} ₳</span>
                    </div>
                    <div class="confirm-row">
                      <span>Network fee <span class="note">(est.)</span></span>
                      <span>~0.17 ₳</span>
                    </div>
                    <div class="confirm-row">
                      <span>Min-ADA <span class="note">(returned to you)</span></span>
                      <span>~2.00 ₳</span>
                    </div>
                    <div class="confirm-divider" />
                    <div class="confirm-row confirm-row--total">
                      <span>Total you pay</span>
                      <strong>~{{ totalAda }} ₳</strong>
                    </div>

                    <div class="confirm-warning">
                      <AlertCircle :size="13" />
                      Make sure your wallet has at least
                      <strong>{{ minRequired }} ₳</strong> before confirming.
                    </div>

                    <div class="confirm-actions">
                      <button
                        class="btn-cancel"
                        :disabled="loading"
                        @click="showConfirm = false"
                      >
                        Cancel
                      </button>
                      <button
                        class="btn-confirm"
                        :disabled="loading"
                        @click="onConfirm"
                      >
                        <Loader2 v-if="loading" :size="14" class="spin" />
                        <ShoppingCart v-else :size="14" />
                        {{ loading ? 'Processing...' : 'Confirm Purchase' }}
                      </button>
                    </div>
                  </div>
                </template>
              </div>

              <div class="panel-tabs">
                <button
                  v-for="tab in tabs"
                  :key="tab"
                  :class="['tab-btn', { 'tab-btn--active': activeTab === tab }]"
                  @click="activeTab = tab"
                >
                  {{ tab }}
                </button>
              </div>

              <div v-if="activeTab === 'Details'" class="tab-content">
                <div class="detail-row">
                  <span class="detail-label">Policy ID</span>
                  <div class="detail-copy">
                    <span class="detail-mono">{{ shortPolicyId }}</span>
                    <button class="copy-btn" @click="copy(listing.nft_policy_id, 'policy')" title="Copy Policy ID">
                      <Check v-if="copied === 'policy'" :size="12" color="#085041" />
                      <Copy v-else :size="12" color="#534AB7" />
                    </button>
                  </div>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Asset Name</span>
                  <span class="detail-mono">{{ listing.nft_asset_name }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Token Standard</span>
                  <span class="detail-value">CIP-68</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Blockchain</span>
                  <span class="detail-value">Cardano</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Royalties</span>
                  <span class="detail-value">{{ listing.royalties ?? 5 }}% on every resale</span>
                </div>
              </div>

              <div v-if="activeTab === 'Activity'" class="tab-content">
                <div class="activity-item">
                  <Tag :size="14" color="#534AB7" />
                  <div class="activity-info">
                    <span class="activity-type">Listed</span>
                    <span class="activity-price">{{ adaAmount(listing.price_lovelace) }} ₳</span>
                  </div>
                </div>
              </div>

            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// ListingDetailPanel.vue
//
// OpenSea-style slide-over panel shown when a buyer clicks a listing.
// Replaces the old BuyConfirmModal with a richer, more informative experience.
//
// Layout:
//   Left  — large NFT image + network/royalty metadata
//   Right — name, price, inline buy confirmation, tabs (Details / Activity)
//
// Tabs:
//   Details  — policy ID (copyable), asset name, token standard, royalties
//   Activity — listing history (future: sold history from API)
// ─────────────────────────────────────────────────────────────────────────────

import { computed, ref, watch } from 'vue'
import {
  AlertCircle, Check, Copy, Image as ImageIcon,
  Loader2, ShoppingCart, ShieldCheck, Tag, X,
} from 'lucide-vue-next'

// ── Props & emits ─────────────────────────────────────────────────────────────
const props = defineProps<{
  listing: any | null  // null = panel hidden
  loading: boolean     // true while buy tx is in progress
}>()

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'close'):   void
}>()

// ── Local state ───────────────────────────────────────────────────────────────
const showConfirm = ref(false)
const activeTab   = ref('Details')
const copied      = ref<string | null>(null)
const tabs        = ['Details', 'Activity']

// Reset confirm state when a new listing is opened
watch(() => props.listing, () => {
  showConfirm.value = false
  activeTab.value   = 'Details'
})

// ── Computed ──────────────────────────────────────────────────────────────────

const imgUrl = computed(() => {
  const ipfs = props.listing?.image_ipfs
  if (!ipfs) return null
  return ipfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}`
    : ipfs
})

/** Truncate policy ID for display — show first 16 + last 6 chars */
const shortPolicyId = computed(() => {
  const id = props.listing?.nft_policy_id || ''
  if (id.length < 24) return id
  return `${id.slice(0, 16)}...${id.slice(-6)}`
})

// Royalty amount: paid from sale price by the smart contract
const royaltyAda = computed(() => {
  if (!props.listing) return '0.00'
  const pct    = (props.listing.royalties ?? 5) / 100
  const amount = (props.listing.price_lovelace / 1_000_000) * pct
  return amount.toFixed(2)
})

// Total buyer pays: price + network fee + min-ADA
// Note: royalty is deducted from seller's proceeds, not added to buyer's cost
const totalAda = computed(() => {
  if (!props.listing) return '0.00'
  const price = props.listing.price_lovelace / 1_000_000
  return (price + 0.17 + 2).toFixed(2)
})

const minRequired = computed(() =>
  Math.ceil(parseFloat(totalAda.value)) + 1
)

// ── Methods ───────────────────────────────────────────────────────────────────

function adaAmount(lovelace: number): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 2, maximumFractionDigits: 2,
  })
}

async function copy(text: string, key: string) {
  await navigator.clipboard.writeText(text)
  copied.value = key
  setTimeout(() => (copied.value = null), 2000)
}

function onConfirm() { emit('confirm') }
function onClose()   {
  if (!props.loading) {
    showConfirm.value = false
    emit('close')
  }
}
</script>

<style scoped>
/* ── Overlay ── */
.panel-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.5);
  z-index: 500;
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}

/* ── Panel ── */
.panel {
  background: #fff;
  border-radius: 20px;
  width: 100%;
  max-width: 900px;
  max-height: 90vh;
  overflow: hidden;
  box-shadow: 0 24px 80px rgba(0,0,0,0.2);
  position: relative;
  display: flex;
  flex-direction: column;
}

.panel-close {
  position: absolute; top: 16px; right: 16px;
  z-index: 10;
  background: #f5f5f5; border: none;
  width: 36px; height: 36px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; color: #666;
  transition: all 0.15s;
}
.panel-close:hover { background: #ebebeb; color: #111; }

.panel-inner {
  display: grid;
  grid-template-columns: 1fr 1fr;
  height: 100%;
  overflow: hidden;
}

/* ── Left: Image ── */
.panel-left {
  background: #f8f8f8;
  border-right: 1px solid #f0f0f0;
  display: flex; flex-direction: column;
  padding: 32px;
  gap: 16px;
  overflow-y: auto;
}

.panel-img-wrap {
  aspect-ratio: 1;
  border-radius: 16px;
  overflow: hidden;
  background: #f0f0f0;
}
.panel-img {
  width: 100%; height: 100%;
  object-fit: cover; display: block;
}
.panel-img-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}

.panel-img-meta {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  padding: 12px 14px;
  display: flex; flex-direction: column; gap: 8px;
}
.img-meta-row {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 12px;
}
.img-meta-label { color: #888; }
.img-meta-value { font-weight: 500; color: #333; }

.network-badge {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 500; color: #7a5c00;
  background: #FFF8E1; padding: 3px 8px; border-radius: 20px;
}
.network-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #085041; display: inline-block;
}

/* ── Right: Details ── */
.panel-right {
  display: flex; flex-direction: column;
  padding: 32px;
  gap: 20px;
  overflow-y: auto;
}

.panel-name-section {}
.panel-nft-name {
  font-size: 24px; font-weight: 700; color: #111;
  margin: 0 0 6px;
  padding-right: 40px; /* avoid close button overlap */
}
.panel-collection {
  display: flex; align-items: center; gap: 5px;
  font-size: 12px; color: #534AB7; font-weight: 500;
}

/* Buy section */
.panel-buy-section {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  padding: 16px;
  display: flex; flex-direction: column; gap: 12px;
}
.buy-label { font-size: 11px; color: #888; font-weight: 500; text-transform: uppercase; letter-spacing: 0.4px; }
.buy-price {
  font-size: 32px; font-weight: 800; color: #534AB7;
  line-height: 1;
}
.buy-currency { font-size: 20px; font-weight: 600; }

.own-listing-notice {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 13px; background: #EEEDFE; color: #534AB7;
  border-radius: 10px; font-size: 14px; font-weight: 600;
}

.btn-buy {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  width: 100%; padding: 13px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 14px; font-weight: 700;
  cursor: pointer; transition: background 0.15s;
}
.btn-buy:hover { background: #3d35a0; }

/* Inline confirmation */
.confirm-box {
  display: flex; flex-direction: column; gap: 6px;
}
.confirm-title {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.4px;
  color: #aaa; margin-bottom: 4px;
}
.confirm-row {
  display: flex; justify-content: space-between;
  font-size: 13px; color: #555; padding: 3px 0;
}
.confirm-row--total {
  font-size: 15px; font-weight: 600; color: #111;
  padding-top: 4px;
}
.confirm-row--total strong { color: #534AB7; }
.confirm-divider { height: 1px; background: #ebebeb; margin: 4px 0; }
.note { font-size: 10px; color: #aaa; }

.confirm-warning {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  padding: 8px 10px; background: #FFF8E1; border-radius: 8px;
  font-size: 11px; color: #7a5c00; line-height: 1.5;
}
.confirm-warning strong { color: #5a3e00; }

.confirm-actions { display: flex; gap: 8px; margin-top: 4px; }
.btn-cancel {
  flex: 1; padding: 10px;
  background: #f5f5f5; border: none;
  border-radius: 8px; font-size: 13px;
  font-weight: 500; cursor: pointer; color: #555;
  transition: background 0.15s;
}
.btn-cancel:hover:not(:disabled) { background: #ebebeb; }
.btn-cancel:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-confirm {
  flex: 2; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 10px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-confirm:hover:not(:disabled) { background: #3d35a0; }
.btn-confirm:disabled { opacity: 0.6; cursor: not-allowed; }

/* Tabs */
.panel-tabs {
  display: flex; gap: 0;
  border-bottom: 1px solid #f0f0f0;
}
.tab-btn {
  padding: 8px 16px;
  background: none; border: none;
  font-size: 13px; font-weight: 500; color: #888;
  cursor: pointer; transition: all 0.15s;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab-btn:hover        { color: #534AB7; }
.tab-btn--active      { color: #534AB7; border-bottom-color: #534AB7; }

/* Tab content */
.tab-content { display: flex; flex-direction: column; gap: 0; }

.detail-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 0; border-bottom: 1px solid #f8f8f8;
  gap: 12px;
}
.detail-row:last-child { border-bottom: none; }
.detail-label { font-size: 12px; color: #888; flex-shrink: 0; }
.detail-value { font-size: 12px; font-weight: 500; color: #333; text-align: right; }
.detail-mono  { font-family: monospace; font-size: 11px; color: #534AB7; }
.detail-copy  { display: flex; align-items: center; gap: 6px; }

.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 0; display: flex; align-items: center;
  opacity: 0.7; transition: opacity 0.15s;
}
.copy-btn:hover { opacity: 1; }

/* Activity tab */
.activity-item {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 0; border-bottom: 1px solid #f8f8f8;
}
.activity-info  { display: flex; flex-direction: column; gap: 2px; }
.activity-type  { font-size: 12px; font-weight: 500; color: #534AB7; }
.activity-price { font-size: 11px; color: #888; }

/* Spinner */
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Panel animation */
.panel-enter-active, .panel-leave-active { transition: all 0.25s ease; }
.panel-enter-from, .panel-leave-to { opacity: 0; transform: scale(0.97); }

/* Mobile */
@media (max-width: 768px) {
  .panel-inner { grid-template-columns: 1fr; }
  .panel-left  { border-right: none; border-bottom: 1px solid #f0f0f0; padding: 20px; }
  .panel-right { padding: 20px; }
  .panel-img-wrap { max-width: 240px; margin: 0 auto; }
}
</style>