<template>
  <div class="wallet-card">
    <div class="wallet-header">
      <Wallet :size="20" color="#fff" />
      <div class="wallet-label">{{ walletType === 'external' ? 'External Wallet' : 'Custodial Wallet' }}</div>
      <div class="wallet-network">Preprod</div>
    </div>

    <div class="wallet-balance">
      <span class="balance-amount">{{ adaBalance }}</span>
      <span class="balance-unit">tADA</span>
    </div>

    <div class="wallet-address-row">
      <span class="wallet-address">{{ shortAddress }}</span>
      <button
        class="copy-btn"
        @click="copyAddress"
        :title="copied ? 'Copied!' : 'Copy address'"
      >
        <Check v-if="copied" :size="14" color="#fff" />
        <Copy v-else :size="14" color="#fff" />
      </button>
    </div>

    <a
      v-if="address"
      :href="`https://preprod.cardanoscan.io/address/${address}`"
      target="_blank"
      rel="noopener noreferrer"
      class="view-link"
    >
      View on Cardanoscan →
    </a>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, Copy, Wallet } from 'lucide-vue-next'

const props = defineProps<{
  address: string
  lovelace: string
  walletType?: string
}>()

const copied = ref(false)

const adaBalance = computed(() => {
  const ada = parseInt(props.lovelace) / 1_000_000
  return ada.toLocaleString('en-US', { maximumFractionDigits: 2 })
})

const shortAddress = computed(() => {
  if (!props.address) return 'No wallet found'
  return props.address.slice(0, 20) + '...' + props.address.slice(-8)
})

async function copyAddress() {
  if (!props.address) return
  await navigator.clipboard.writeText(props.address)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}
</script>

<style scoped>
.wallet-card {
  background: linear-gradient(135deg, #534AB7 0%, #3d35a0 100%);
  border-radius: 16px;
  padding: 24px;
  color: #fff;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.wallet-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.wallet-label { font-size: 14px; font-weight: 500; flex: 1; }
.wallet-network {
  font-size: 11px;
  background: rgba(255,255,255,0.2);
  padding: 2px 8px;
  border-radius: 20px;
}
.wallet-balance { display: flex; align-items: baseline; gap: 6px; }
.balance-amount { font-size: 36px; font-weight: 700; letter-spacing: -1px; }
.balance-unit { font-size: 16px; opacity: 0.8; }
.wallet-address-row {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(255,255,255,0.1);
  border-radius: 8px;
  padding: 8px 12px;
}
.wallet-address {
  font-size: 12px;
  font-family: monospace;
  flex: 1;
  opacity: 0.9;
}
.copy-btn {
  background: none; border: none;
  cursor: pointer; padding: 0;
  opacity: 0.8;
  display: flex; align-items: center;
}
.copy-btn:hover { opacity: 1; }
.view-link { font-size: 12px; color: rgba(255,255,255,0.7); text-decoration: none; }
.view-link:hover { color: #fff; }
</style>