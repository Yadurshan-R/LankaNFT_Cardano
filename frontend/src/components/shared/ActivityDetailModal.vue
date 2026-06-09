<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="item" class="modal-overlay" @click.self="emit('close')">
        <div class="modal" role="dialog" aria-modal="true">

          <!-- Close -->
          <button class="modal-close" @click="emit('close')">
            <X :size="18" />
          </button>

          <!-- Header badge -->
          <div :class="['modal-type-badge', `modal-type-badge--${item.type}`]">
            <component :is="typeIcon" :size="14" />
            {{ typeLabel }}
          </div>

          <!-- NFT image + name -->
          <div class="modal-nft-row">
            <div class="modal-img-wrap">
              <img
                v-if="item.imageUrl && !imgError"
                :src="item.imageUrl"
                :alt="item.name"
                class="modal-img"
                @error="imgError = true"
              />
              <div v-else :class="['modal-img-fallback', `modal-img-fallback--${item.type}`]">
                <component :is="typeIcon" :size="24" />
              </div>
            </div>
            <div class="modal-nft-info">
              <h2 class="modal-nft-name">{{ item.name }}</h2>
              <p class="modal-nft-time">{{ item.time }}</p>
              <p v-if="item.price" class="modal-nft-price">{{ item.price }} ₳</p>
            </div>
          </div>

          <!-- Details card -->
          <div class="modal-detail-card">

            <!-- Tx hash -->
            <div v-if="item.txHash" class="detail-row">
              <span class="detail-label">Transaction</span>
              <div class="detail-val-row">
                <code class="detail-hash">{{ item.txHash.slice(0, 12) }}...{{ item.txHash.slice(-8) }}</code>
                <button class="copy-btn" @click="copyTx" :title="txCopied ? 'Copied!' : 'Copy'">
                  <Check v-if="txCopied" :size="12" color="#085041" />
                  <Copy v-else :size="12" />
                </button>
              </div>
            </div>

            <!-- Network -->
            <div class="detail-row">
              <span class="detail-label">Network</span>
              <span class="detail-val">Cardano Preprod</span>
            </div>

            <!-- Standard -->
            <div class="detail-row">
              <span class="detail-label">Token Standard</span>
              <span class="detail-val">CIP-68</span>
            </div>

            <!-- Price if applicable -->
            <div v-if="item.price" class="detail-row">
              <span class="detail-label">Price</span>
              <span class="detail-val" style="color: #534AB7; font-weight: 700;">{{ item.price }} ₳</span>
            </div>

          </div>

          <!-- Confirmation notice -->
          <div class="confirm-notice">
            <span class="confirm-dot" />
            On Cardano Preprod
          </div>

          <!-- Action buttons -->
          <div class="modal-actions">
            <a
              v-if="item.txHash"
              :href="cardanoscanTxUrl(item.txHash)"
              target="_blank"
              rel="noopener noreferrer"
              class="btn-scan"
            >
              <ExternalLink :size="13" /> View Transaction
            </a>
            <a
              v-if="item.assetUrl"
              :href="item.assetUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="btn-scan btn-scan--secondary"
            >
              <ExternalLink :size="13" /> View Asset
            </a>
          </div>

          <!-- Navigate to NFT detail if available -->
          <button
            v-if="item.nftId && item.type !== 'cancelled'"
            class="btn-view-nft"
            @click="onViewNFT"
          >
            Open NFT Detail →
          </button>

          <button class="btn-done" @click="emit('close')">Done</button>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Check, Copy, ExternalLink, Send, ShoppingCart, Sparkles, Tag, X,
} from 'lucide-vue-next'
import { cardanoscanTxUrl } from '@/utils/cardano'

const props = defineProps<{ item: any | null }>()
const emit  = defineEmits<{ (e: 'close'): void }>()
const router = useRouter()

const imgError = ref(false)
const txCopied = ref(false)

const typeIcon = computed(() => {
  const map: Record<string, any> = {
    minted: Sparkles, listed: Tag, sold: ShoppingCart,
    transferred: Send, cancelled: X, pending: Sparkles,
  }
  return map[props.item?.type] ?? Sparkles
})

const typeLabel = computed(() => {
  const map: Record<string, string> = {
    minted: 'Minted', listed: 'Listed', sold: 'Sold',
    transferred: 'Transferred', cancelled: 'Listing Cancelled', pending: 'Pending',
  }
  return map[props.item?.type] ?? props.item?.type
})

async function copyTx() {
  if (!props.item?.txHash) return
  await navigator.clipboard.writeText(props.item.txHash)
  txCopied.value = true
  setTimeout(() => (txCopied.value = false), 2000)
}

function onViewNFT() {
  emit('close')
  router.push(`/nft/${props.item.nftId}`)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.55);
  z-index: 600;
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}
.modal {
  background: #fff; border-radius: 20px;
  padding: 28px 24px; width: 100%; max-width: 400px;
  position: relative; display: flex; flex-direction: column;
  align-items: center; gap: 14px;
  box-shadow: 0 24px 80px rgba(0,0,0,0.2); text-align: center;
}
.modal-close {
  position: absolute; top: 14px; right: 14px;
  background: #f5f5f5; border: none;
  width: 32px; height: 32px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; color: #666; transition: all 0.15s;
}
.modal-close:hover { background: #ebebeb; color: #111; }

/* Type badge */
.modal-type-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 14px; border-radius: 20px;
  font-size: 12px; font-weight: 600;
}
.modal-type-badge--minted      { background: #E1F5EE; color: #085041; }
.modal-type-badge--listed      { background: #E6F1FB; color: #185FA5; }
.modal-type-badge--sold        { background: #EEEDFE; color: #534AB7; }
.modal-type-badge--transferred { background: #FAEEDA; color: #854F0B; }
.modal-type-badge--cancelled   { background: #FCEBEB; color: #791F1F; }
.modal-type-badge--pending     { background: #FFF8E1; color: #7a5c00; }

/* NFT row */
.modal-nft-row {
  display: flex; align-items: center; gap: 14px;
  background: #fafafa; border: 1px solid #f0f0f0;
  border-radius: 14px; padding: 14px; width: 100%; text-align: left;
}
.modal-img-wrap { flex-shrink: 0; }
.modal-img {
  width: 60px; height: 60px; border-radius: 10px;
  object-fit: cover; background: #f0f0f0;
}
.modal-img-fallback {
  width: 60px; height: 60px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
}
.modal-img-fallback--minted      { background: #E1F5EE; color: #085041; }
.modal-img-fallback--listed      { background: #E6F1FB; color: #185FA5; }
.modal-img-fallback--sold        { background: #EEEDFE; color: #534AB7; }
.modal-img-fallback--transferred { background: #FAEEDA; color: #854F0B; }
.modal-img-fallback--cancelled   { background: #FCEBEB; color: #d32f2f; }
.modal-img-fallback--pending     { background: #FFF8E1; color: #7a5c00; }
.modal-nft-info { flex: 1; min-width: 0; }
.modal-nft-name {
  font-size: 15px; font-weight: 700; color: #111; margin: 0 0 3px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.modal-nft-time  { font-size: 11px; color: #aaa; margin: 0 0 2px; }
.modal-nft-price { font-size: 16px; font-weight: 700; color: #534AB7; margin: 0; }

/* Detail card */
.modal-detail-card {
  background: #f8f8f8; border-radius: 12px;
  padding: 12px 14px; width: 100%;
  display: flex; flex-direction: column;
}
.detail-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 0; border-bottom: 1px solid #efefef; font-size: 12px;
}
.detail-row:last-child { border-bottom: none; }
.detail-label { color: #888; }
.detail-val   { font-weight: 500; color: #333; }
.detail-val-row { display: flex; align-items: center; gap: 6px; }
.detail-hash {
  font-family: monospace; font-size: 11px; color: #534AB7;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 2px; opacity: 0.7; display: flex; align-items: center;
}
.copy-btn:hover { opacity: 1; }

/* Notice */
.confirm-notice {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: #aaa;
}
.confirm-dot {
  width: 6px; height: 6px; border-radius: 50%; background: #085041;
}

/* Buttons */
.modal-actions { display: flex; gap: 8px; width: 100%; }
.btn-scan {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: 6px; padding: 10px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px; font-size: 12px; font-weight: 600;
  text-decoration: none; transition: background 0.15s;
}
.btn-scan:hover { background: #3d35a0; }
.btn-scan--secondary { background: #EEEDFE; color: #534AB7; }
.btn-scan--secondary:hover { background: #dddcfc; }
.btn-view-nft {
  width: 100%; padding: 10px; background: transparent;
  border: 1px solid #e8e8e8; border-radius: 10px;
  font-size: 12px; font-weight: 500; color: #534AB7;
  cursor: pointer; transition: all 0.15s;
}
.btn-view-nft:hover { border-color: #534AB7; background: #EEEDFE; }
.btn-done {
  width: 100%; padding: 10px; background: #f5f5f5; border: none;
  border-radius: 10px; font-size: 13px; font-weight: 500;
  color: #555; cursor: pointer; transition: background 0.15s;
}
.btn-done:hover { background: #ebebeb; }

/* Animation */
.modal-enter-active, .modal-leave-active { transition: all 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; transform: scale(0.95); }
</style>