<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="emit('close')">
        <div class="modal" role="dialog" aria-modal="true">

          <button class="modal-close" @click="emit('close')">
            <X :size="18" />
          </button>

          <div class="success-icon-wrap">
            <CheckCircle :size="44" color="#085041" />
          </div>

          <h2 class="modal-title">Purchase Confirmed!</h2>
          <p class="modal-sub">Your NFT has been submitted to the Cardano blockchain.</p>

          <div class="nft-row">
            <img
              v-if="imageUrl && !imgError"
              :src="imageUrl"
              :alt="nftName"
              class="nft-thumb"
              @error="imgError = true"
            />
            <div v-else class="nft-thumb-placeholder">
              <ImageIcon :size="22" color="#ccc" />
            </div>
            <div class="nft-info">
              <p class="nft-name">{{ nftName }}</p>
              <p class="nft-price">{{ adaAmount }} ₳</p>
            </div>
          </div>

          <div class="detail-card">
            <div class="detail-row">
              <span class="detail-label">Transaction</span>
              <div class="hash-row">
                <code class="hash-val">{{ txHash.slice(0, 12) }}...{{ txHash.slice(-8) }}</code>
                <button class="copy-btn" @click="copyHash" :title="copied ? 'Copied!' : 'Copy'">
                  <Check v-if="copied" :size="12" color="#085041" />
                  <Copy v-else :size="12" />
                </button>
              </div>
            </div>
            <div class="detail-row">
              <span class="detail-label">Network</span>
              <span class="detail-val">Cardano Preprod</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Standard</span>
              <span class="detail-val">CIP-68</span>
            </div>
          </div>

          <div class="confirm-notice">
            <span class="confirm-dot" />
            Usually confirms on-chain within ~20 seconds
          </div>

          <div class="modal-actions">
            <a
              :href="cardanoscanTxUrl(txHash)"
              target="_blank"
              rel="noopener noreferrer"
              class="btn-scan"
            >
              <ExternalLink :size="13" /> View Transaction
            </a>
            <a
              v-if="policyId && assetName"
              :href="cardanoscanTokenUrl(policyId, assetName)"
              target="_blank"
              rel="noopener noreferrer"
              class="btn-scan btn-scan--secondary"
            >
              <ExternalLink :size="13" /> View Asset
            </a>
          </div>

          <button class="btn-done" @click="emit('close')">
            Done
          </button>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, CheckCircle, Copy, ExternalLink, X, Image as ImageIcon } from 'lucide-vue-next'
import { cardanoscanTokenUrl, cardanoscanTxUrl } from '@/utils/cardano'

const props = defineProps<{
  show:       boolean
  txHash:     string
  nftName:    string
  imageIpfs?: string
  priceLovelace?: number
  policyId?:  string
  assetName?: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const copied = ref(false)
const imgError = ref(false)

const imageUrl = computed(() => {
  if (!props.imageIpfs) return null
  return props.imageIpfs.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${props.imageIpfs.replace('ipfs://', '')}`
    : props.imageIpfs
})

const adaAmount = computed(() => {
  if (!props.priceLovelace) return '—'
  return (props.priceLovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 2, maximumFractionDigits: 2,
  })
})

async function copyHash() {
  await navigator.clipboard.writeText(props.txHash)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.55);
  z-index: 600;
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}

.modal {
  background: #fff;
  border-radius: 20px;
  padding: 32px 28px;
  width: 100%;
  max-width: 420px;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.2);
  text-align: center;
}

.modal-close {
  position: absolute; top: 14px; right: 14px;
  background: #f5f5f5; border: none;
  width: 32px; height: 32px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; color: #666; transition: all 0.15s;
}
.modal-close:hover { background: #ebebeb; color: #111; }

.success-icon-wrap {
  width: 72px; height: 72px;
  background: #E1F5EE;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
}

.modal-title { font-size: 20px; font-weight: 700; color: #111; margin: 0; }
.modal-sub   { font-size: 13px; color: #888; margin: 0; }

/* NFT row */
.nft-row {
  display: flex; align-items: center; gap: 12px;
  background: #fafafa; border: 1px solid #f0f0f0;
  border-radius: 12px; padding: 12px;
  width: 100%;
}
.nft-thumb {
  width: 52px; height: 52px;
  border-radius: 8px; object-fit: cover;
  background: #f0f0f0; flex-shrink: 0;
}
.nft-thumb-placeholder {
  width: 52px; height: 52px;
  border-radius: 8px; background: #f0f0f0;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.nft-info   { text-align: left; }
.nft-name   { font-size: 14px; font-weight: 600; color: #111; margin: 0 0 3px; }
.nft-price  { font-size: 16px; font-weight: 700; color: #534AB7; margin: 0; }

/* Detail card */
.detail-card {
  background: #f8f8f8; border-radius: 12px;
  padding: 12px 14px; width: 100%;
  display: flex; flex-direction: column; gap: 0;
}
.detail-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 0; border-bottom: 1px solid #efefef;
  font-size: 12px;
}
.detail-row:last-child { border-bottom: none; }
.detail-label { color: #888; }
.detail-val   { font-weight: 500; color: #333; }

.hash-row {
  display: flex; align-items: center; gap: 6px;
}
.hash-val {
  font-family: monospace; font-size: 11px; color: #534AB7;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 2px; opacity: 0.7; display: flex; align-items: center;
}
.copy-btn:hover { opacity: 1; }

/* Confirm notice */
.confirm-notice {
  display: flex; align-items: center; gap: 7px;
  font-size: 11px; color: #888;
}
.confirm-dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: #085041; display: inline-block;
  animation: pulse 1.5s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%       { opacity: 0.5; transform: scale(0.85); }
}

/* Buttons */
.modal-actions {
  display: flex; gap: 8px; width: 100%;
}
.btn-scan {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 10px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 12px; font-weight: 600;
  text-decoration: none; transition: background 0.15s;
}
.btn-scan:hover { background: #3d35a0; }
.btn-scan--secondary {
  background: #EEEDFE; color: #534AB7;
}
.btn-scan--secondary:hover { background: #dddcfc; }

.btn-done {
  width: 100%; padding: 11px;
  background: #f5f5f5; border: none;
  border-radius: 10px; font-size: 13px;
  font-weight: 500; color: #555; cursor: pointer;
  transition: background 0.15s;
}
.btn-done:hover { background: #ebebeb; }

/* Animation */
.modal-enter-active, .modal-leave-active { transition: all 0.25s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: scale(0.95); }
</style>