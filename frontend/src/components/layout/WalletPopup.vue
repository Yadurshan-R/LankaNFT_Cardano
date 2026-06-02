<template>
  <div class="popup" @click.stop>
    <div class="popup-header">
      <Wallet :size="13" class="popup-header-icon" />
      <span>{{ walletType === 'external' ? 'External Wallet' : 'Custodial Wallet' }}</span>
    </div>

    <div class="popup-row">
      <span class="popup-label">Address</span>
      <span class="popup-addr">{{ shortAddress }}</span>
    </div>
    <div class="popup-row">
      <span class="popup-label">Network</span>
      <span class="popup-net">Preprod</span>
    </div>

    <div class="popup-divider" />

    <div class="popup-balance-label">Balance</div>
    <div class="popup-balance">{{ adaBalance }} ₳</div>

    <div class="popup-divider" />

    <a
      :href="`https://preprod.cardanoscan.io/address/${address}`"
      target="_blank"
      rel="noopener noreferrer"
      class="popup-btn"
    >
      <ExternalLink :size="12" />
      View on Cardanoscan
    </a>

    <div class="popup-security">
      <Lock :size="11" />
      {{ walletType === 'external' ? 'Self-custodied · You hold your own keys' : 'AES-256 encrypted · Only you can access' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ExternalLink, Lock, Wallet } from 'lucide-vue-next'

const props = defineProps<{
  address: string
  lovelace: string
  walletType?: string
}>()

defineEmits<{ (e: 'close'): void }>()

// Show first 8 and last 6 characters of address
const shortAddress = computed(() => {
  if (!props.address || props.address.length < 16) return props.address
  return `${props.address.slice(0, 8)}...${props.address.slice(-6)}`
})

// Convert lovelace (string) to ADA with 2 decimal places
const adaBalance = computed(() => {
  const lovelace = parseInt(props.lovelace || '0')
  return (lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
})
</script>

<style scoped>
.popup {
  position: fixed;
  left: 210px;
  bottom: 120px;
  width: 240px;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 14px;
  padding: 16px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.1);
  z-index: 200;
  animation: fadeIn 0.15s ease;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateX(-8px); }
  to   { opacity: 1; transform: translateX(0); }
}

.popup-header       { display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 600; color: #534AB7; margin-bottom: 12px; }
.popup-header-icon  { color: #534AB7; }
.popup-row          { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; font-size: 12px; }
.popup-label        { color: #888; }
.popup-addr         { font-family: monospace; font-size: 11px; color: #444; }
.popup-net          { font-size: 10px; font-weight: 600; color: #7a5c00; background: #FFF8E1; padding: 2px 8px; border-radius: 20px; }
.popup-divider      { height: 1px; background: #f0f0f0; margin: 10px 0; }
.popup-balance-label{ font-size: 11px; color: #888; text-align: center; margin-bottom: 2px; }
.popup-balance      { font-size: 24px; font-weight: 700; color: #534AB7; text-align: center; }
.popup-btn          { display: flex; align-items: center; justify-content: center; gap: 6px; width: 100%; padding: 9px; background: #534AB7; color: #fff; border: none; border-radius: 10px; font-size: 12px; font-weight: 600; cursor: pointer; text-decoration: none; margin-top: 4px; }
.popup-btn:hover    { background: #3d35a0; }
.popup-security     { display: flex; align-items: center; justify-content: center; gap: 4px; font-size: 10px; color: #888; margin-top: 8px; padding: 6px; background: #fafafa; border-radius: 8px; }
</style>