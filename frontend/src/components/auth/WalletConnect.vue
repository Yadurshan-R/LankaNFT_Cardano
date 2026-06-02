<template>
  <div class="wallet-connect">

    <div v-if="wallets.length === 0" class="no-wallets">
      <div class="no-wallet-icon">
        <Wallet :size="28" color="#bbb" />
      </div>
      <p class="no-wallet-title">No wallet detected</p>
      <p class="no-wallet-desc">
        Install a Cardano wallet extension to connect.
      </p>
      <div class="wallet-links">
        <a href="https://namiwallet.io" target="_blank" class="wallet-link">Nami</a>
        <a href="https://eternl.io" target="_blank" class="wallet-link">Eternl</a>
        <a href="https://lace.io" target="_blank" class="wallet-link">Lace</a>
      </div>
    </div>

    <div v-else class="wallet-list">
      <p class="step-label">
        <Wallet :size="13" />
        Choose your wallet
      </p>
      <button
        v-for="wallet in wallets"
        :key="wallet.id"
        class="wallet-btn"
        :disabled="auth.isLoading"
        @click="handleConnect(wallet.id, wallet.name)"
      >
        <img :src="wallet.icon" :alt="wallet.name" class="wallet-icon" />
        <span class="wallet-name">{{ wallet.name }}</span>
        <ChevronRight :size="14" color="#ccc" class="wallet-chevron" />
      </button>
    </div>

    <p v-if="auth.error" class="error-msg">{{ auth.error }}</p>

  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// WalletConnect.vue
//
// CIP-30 wallet login tab. Detects installed Cardano wallet extensions
// and handles the sign-in flow:
//
//   1. Detect installed wallets via window.cardano (CIP-30 standard)
//   2. User clicks their wallet → wallet.enable() opens browser permission popup
//   3. Get wallet address via getUsedAddresses() or getChangeAddress()
//   4. Get nonce from backend (/auth/nonce)
//   5. Sign nonce with wallet.signData() — proves key ownership
//   6. Send signature to backend (/auth/wallet-verify) → JWT cookie set
//   7. Store wallet ID in useWalletSession so we can sign future transactions
//   8. Redirect to dashboard
//
// The walletApi object is stored in useWalletSession (module-level ref).
// It persists across page navigation and is restored from localStorage on refresh.
// ─────────────────────────────────────────────────────────────────────────────
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronRight, Wallet } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useWalletSession } from '@/composables/useWalletSession'
import api from '@/services/api'
import { bech32 } from 'bech32'

const auth          = useAuthStore()
const router        = useRouter()
const walletSession = useWalletSession()

interface WalletInfo {
  id:   string
  name: string
  icon: string
}

const wallets = ref<WalletInfo[]>([])

// CIP-30 wallets return addresses as hex-encoded raw bytes, not bech32.
// Blockfrost requires bech32. This converts hex → bech32.
// If the address is already bech32 (starts with 'addr'), returns as-is.
function toCardanoBech32(hex: string): string {
  if (hex.startsWith('addr')) return hex
  const bytes = new Uint8Array(
    hex.match(/.{1,2}/g)!.map(b => parseInt(b, 16))
  )
  const words = bech32.toWords(bytes)
  return bech32.encode('addr_test', words, 1000)
}

// CIP-30: all wallets inject into window.cardano
onMounted(() => {
  const cardano = (window as any).cardano
  if (!cardano) return

  const knownWallets = ['nami', 'eternl', 'lace', 'typhon', 'flint', 'gerowallet']
  wallets.value = knownWallets
    .filter(id => cardano[id])
    .map(id => ({
      id,
      name: cardano[id].name || id,
      icon: cardano[id].icon || '',
    }))
})

async function handleConnect(walletId: string, walletName: string) {
  try {
    auth.isLoading = true
    auth.error     = null

    const cardano = (window as any).cardano

    // Step 1 — enable wallet (opens browser permission popup on first use)
    const walletApi = await cardano[walletId].enable()

    // Step 2 — get wallet address and convert from hex to bech32
    const rawAddresses = await walletApi.getUsedAddresses()
    const rawAddress   = rawAddresses[0] || (await walletApi.getChangeAddress())
    const address      = toCardanoBech32(rawAddress)

    // Step 3 — get nonce from Go backend
    const nonceRes = await api.get(`/auth/nonce?wallet=${address}`)
    const nonce    = nonceRes.data.nonce

    // Step 4 — sign nonce to prove ownership (CIP-30 signData)
    const signature = await walletApi.signData(address, toHex(nonce))

    // Step 5 — verify with backend → JWT cookie set
    const success = await auth.connectWallet(
      address,
      walletName.toLowerCase(),
      JSON.stringify(signature),
    )

    if (success) {
      // Step 6 — store wallet API for future transaction signing
      // This is crucial: signAndSubmit() needs this API object to sign
      // unsigned CBOR returned by backend unsigned endpoints
      walletSession.setWallet(walletId, walletApi)

      router.push('/')
    }

  } catch (err: any) {
    auth.error = err.message || 'Wallet connection failed'
  } finally {
    auth.isLoading = false
  }
}

// Convert string to hex for CIP-30 signData
function toHex(str: string): string {
  return Array.from(str)
    .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
    .join('')
}
</script>

<style scoped>
.wallet-connect { width: 100%; }

/* ── No wallets ─────────────────────────────────────────────────────────────── */
.no-wallets {
  text-align: center;
  padding: 8px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.no-wallet-icon {
  width: 52px; height: 52px;
  background: #f5f5f5; border-radius: 14px;
  display: flex; align-items: center; justify-content: center;
  margin-bottom: 4px;
}
.no-wallet-title { font-size: 15px; font-weight: 600; color: #333; margin: 0; }
.no-wallet-desc  { font-size: 13px; color: #999; margin: 0; line-height: 1.5; }
.wallet-links {
  display: flex; gap: 8px; margin-top: 4px;
}
.wallet-link {
  font-size: 12px; font-weight: 500;
  color: #534AB7; text-decoration: none;
  padding: 4px 10px;
  border: 1px solid rgba(83,74,183,0.25);
  border-radius: 6px;
  transition: all 0.15s;
}
.wallet-link:hover { background: #EEEDFE; }

/* ── Step label ─────────────────────────────────────────────────────────────── */
.step-label {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #534AB7; margin-bottom: 10px;
}

/* ── Wallet list ────────────────────────────────────────────────────────────── */
.wallet-list { display: flex; flex-direction: column; gap: 8px; }

.wallet-btn {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 14px;
  border: 1.5px solid #eaeaea;
  border-radius: 12px;
  background: #fff;
  cursor: pointer;
  width: 100%;
  transition: all 0.15s;
}
.wallet-btn:hover:not(:disabled) {
  border-color: #534AB7;
  background: #FAFAFE;
}
.wallet-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.wallet-icon    { width: 28px; height: 28px; border-radius: 7px; flex-shrink: 0; }
.wallet-name    { font-size: 14px; font-weight: 500; color: #333; flex: 1; text-align: left; }
.wallet-chevron { flex-shrink: 0; }

/* ── Error ──────────────────────────────────────────────────────────────────── */
.error-msg {
  margin-top: 12px; font-size: 13px; color: #d32f2f;
}
</style>