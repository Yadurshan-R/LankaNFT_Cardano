<template>
  <div :class="['tx-success-card', `tx-success-card--${color}`]">
    <div class="tx-success-header">
      <CheckCircle :size="16" />
      {{ title }}
    </div>

    <div v-if="nftName" class="tx-success-row">
      <span class="tx-label">NFT</span>
      <strong class="tx-val">{{ nftName }}</strong>
    </div>

    <div v-if="detail" class="tx-success-row">
      <span class="tx-label">{{ detail.label }}</span>
      <strong class="tx-val">{{ detail.value }}</strong>
    </div>

    <div class="tx-success-row">
      <span class="tx-label">Tx Hash</span>
      <div class="tx-hash-row">
        <code class="tx-hash">{{ txHash.slice(0, 14) }}...{{ txHash.slice(-8) }}</code>
        <button class="tx-copy-btn" @click="copyHash" :title="copied ? 'Copied!' : 'Copy hash'">
          <Check v-if="copied" :size="12" />
          <Copy v-else :size="12" />
        </button>
      </div>
    </div>

    <div class="tx-success-links">
      <a
        :href="`https://preprod.cardanoscan.io/transaction/${txHash}`"
        target="_blank"
        rel="noopener noreferrer"
        class="tx-link"
      >
        <ExternalLink :size="11" /> View Transaction
      </a>
      <a
        v-if="policyId && assetName"
        :href="`https://preprod.cardanoscan.io/token/${policyId}${assetName}`"
        target="_blank"
        rel="noopener noreferrer"
        class="tx-link"
      >
        <ExternalLink :size="11" /> View Asset
      </a>
    </div>

    <p class="tx-confirm-note">⚡ Usually confirms on-chain in ~20 seconds</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Check, CheckCircle, Copy, ExternalLink } from 'lucide-vue-next'

const props = defineProps<{
  title:     string
  txHash:    string
  nftName?:  string
  policyId?: string
  assetName?: string
  color?:    'green' | 'purple' | 'blue'
  detail?:   { label: string; value: string }
}>()

const copied = ref(false)

async function copyHash() {
  await navigator.clipboard.writeText(props.txHash)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}
</script>

<style scoped>
.tx-success-card {
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tx-success-card--green  { background: #E1F5EE; border: 1px solid #b8e0cc; }
.tx-success-card--purple { background: #EEEDFE; border: 1px solid #c5c1f0; }
.tx-success-card--blue   { background: #EEF4FF; border: 1px solid #c0d0f0; }

.tx-success-header {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600;
}
.tx-success-card--green  .tx-success-header { color: #085041; }
.tx-success-card--purple .tx-success-header { color: #534AB7; }
.tx-success-card--blue   .tx-success-header { color: #1a4bbd; }

.tx-success-row {
  display: flex; justify-content: space-between;
  align-items: center; font-size: 12px; color: #444;
}
.tx-label { color: #888; }
.tx-val   { font-weight: 600; color: #111; }

.tx-hash-row {
  display: flex; align-items: center; gap: 6px;
}
.tx-hash {
  font-family: monospace; font-size: 11px; color: #534AB7;
}
.tx-copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 2px; opacity: 0.7; display: flex; align-items: center;
}
.tx-copy-btn:hover { opacity: 1; }

.tx-success-links {
  display: flex; gap: 8px; flex-wrap: wrap; margin-top: 2px;
}
.tx-link {
  display: flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 600; color: #534AB7;
  text-decoration: none; padding: 4px 8px;
  background: rgba(83,74,183,0.08); border-radius: 6px;
  transition: background 0.15s;
}
.tx-link:hover { background: rgba(83,74,183,0.15); }

.tx-confirm-note {
  font-size: 10px; color: #888; margin: 0;
}
</style>