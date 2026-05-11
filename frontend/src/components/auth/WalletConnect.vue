<template>
  <div class="wallet-connect">
    <h2 class="form-title">Connect a Wallet</h2>
    <p class="form-subtitle">Use your existing Cardano wallet to sign in</p>

    <!-- No wallets detected -->
    <div v-if="wallets.length === 0" class="no-wallets">
      <p>No wallet extensions detected.</p>
      <p>
        Install
        <a href="https://lace.io" target="_blank">Lace</a>,
        <a href="https://namiwallet.io" target="_blank">Nami</a>, or
        <a href="https://eternl.io" target="_blank">Eternl</a>.
      </p>
    </div>

    <!-- Wallet list -->
    <div v-else class="wallet-list">
      <button
        v-for="wallet in wallets"
        :key="wallet.id"
        class="wallet-btn"
        :disabled="auth.isLoading"
        @click="handleConnect(wallet.id, wallet.name)"
      >
        <img :src="wallet.icon" :alt="wallet.name" class="wallet-icon" />
        <span>{{ wallet.name }}</span>
      </button>
    </div>

    <p v-if="auth.error" class="error-msg">{{ auth.error }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'

const auth = useAuthStore()
const router = useRouter()

interface WalletInfo {
  id: string
  name: string
  icon: string
}

const wallets = ref<WalletInfo[]>([])

// CIP-30: all wallets inject into window.cardano
// We detect which ones are installed
onMounted(() => {
  const cardano = (window as any).cardano
  if (!cardano) return

  // Loop through known wallet IDs
  const knownWallets = ['lace', 'nami', 'eternl', 'typhon', 'flint', 'gerowallet']
  wallets.value = knownWallets
    .filter((id) => cardano[id])
    .map((id) => ({
      id,
      name: cardano[id].name || id,
      icon: cardano[id].icon || '',
    }))
})

async function handleConnect(walletId: string, walletName: string) {
  try {
    auth.isLoading = true
    auth.error = null

    const cardano = (window as any).cardano

    // Step 1 — enable wallet (triggers browser popup for permission)
    const walletApi = await cardano[walletId].enable()

    // Step 2 — get wallet address (CIP-30 returns hex, decode to bech32)
    const addresses = await walletApi.getUsedAddresses()
    const address = addresses[0] || (await walletApi.getChangeAddress())

    // Step 3 — get nonce from Go backend
    const nonceRes = await api.get(`/auth/nonce?wallet=${address}`)
    const nonce = nonceRes.data.nonce

    // Step 4 — sign nonce with wallet (CIP-30 signData)
    const signature = await walletApi.signData(address, toHex(nonce))

    // Step 5 — send to Go backend to verify + receive JWT cookie
    const success = await auth.connectWallet(
      address,
      walletName.toLowerCase(),
      JSON.stringify(signature),
    )

    if (success) router.push('/')
  } catch (err: any) {
    auth.error = err.message || 'Wallet connection failed'
  } finally {
    auth.isLoading = false
  }
}

// Convert string to hex for CIP-30 signData
function toHex(str: string): string {
  return Array.from(str)
    .map((c) => c.charCodeAt(0).toString(16).padStart(2, '0'))
    .join('')
}
</script>

<style scoped>
.wallet-connect {
  width: 100%;
}
.form-title {
  font-size: 22px;
  font-weight: 600;
  color: #111;
  margin-bottom: 6px;
}
.form-subtitle {
  font-size: 14px;
  color: #666;
  margin-bottom: 24px;
}
.wallet-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.wallet-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1.5px solid #e0e0e0;
  border-radius: 10px;
  background: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: border 0.2s;
  width: 100%;
}
.wallet-btn:hover:not(:disabled) {
  border-color: #534ab7;
  background: #fafafa;
}
.wallet-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.wallet-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
}
.no-wallets {
  font-size: 14px;
  color: #666;
  line-height: 1.8;
}
.no-wallets a {
  color: #534ab7;
  text-decoration: underline;
}
.error-msg {
  margin-top: 12px;
  font-size: 13px;
  color: #d32f2f;
}
</style>