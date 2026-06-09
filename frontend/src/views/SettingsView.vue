<template>
  <div class="settings">
    <router-link to="/" class="back-link" title="Back to Dashboard">
      <ArrowLeft :size="16" />
    </router-link>

    <div class="page-header">
      <h1 class="page-title">Settings</h1>
      <p class="page-sub">Manage your account and wallet preferences.</p>
    </div>

    <div class="settings-card">
      <div class="card-title">Profile</div>
      <div class="profile-row">
        <div class="avatar">
          {{ avatarLetter }}
        </div>
        <div class="profile-info">
          <div class="profile-name">{{ displayName }}</div>
          <div class="profile-email">
            {{ auth.walletType === 'external' ? 'External Wallet User' : (auth.userID || 'Custodial User') }}
          </div>
          <span :class="['wallet-type-badge', auth.walletType === 'external' ? 'badge--external' : 'badge--custodial']">
            <component :is="auth.walletType === 'external' ? Wallet : Shield" :size="11" />
            {{ auth.walletType === 'external' ? 'External Wallet (Lace)' : 'Custodial Wallet' }}
          </span>
        </div>
      </div>
    </div>

    <div class="settings-card">
      <div class="card-title">Wallet</div>

      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Network</div>
          <div class="setting-desc">Connected blockchain network</div>
        </div>
        <span class="network-badge">
          <span class="network-dot" /> Cardano Preprod
        </span>
      </div>

      <div class="setting-divider" />

      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Wallet Type</div>
          <div class="setting-desc">How your keys are managed</div>
        </div>
        <span class="setting-value">
          {{ auth.walletType === 'external' ? 'Self-custody (Lace)' : 'Platform-managed' }}
        </span>
      </div>

      <div v-if="walletAddress" class="setting-divider" />

      <div v-if="walletAddress" class="setting-row setting-row--col">
        <div class="setting-label">
          <div class="setting-name">Wallet Address</div>
          <div class="setting-desc">Your Cardano Preprod address</div>
        </div>
        <div class="address-box">
          <code class="address-text">{{ walletAddress }}</code>
          <button class="copy-btn" @click="copyAddress" :title="addressCopied ? 'Copied!' : 'Copy address'">
            <Check v-if="addressCopied" :size="13" color="#085041" />
            <Copy v-else :size="13" />
          </button>
        </div>
        <a
          :href="`https://preprod.cardanoscan.io/address/${walletAddress}`"
          target="_blank"
          rel="noopener noreferrer"
          class="scan-link"
        >
          <ExternalLink :size="11" /> View on Cardanoscan
        </a>
      </div>
    </div>

    <div class="settings-card">
      <div class="card-title">Platform</div>

      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Token Standard</div>
          <div class="setting-desc">CIP-68 on-chain metadata with reference tokens</div>
        </div>
        <span class="setting-value">CIP-68</span>
      </div>

      <div class="setting-divider" />

      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Default Royalties</div>
          <div class="setting-desc">Applied to every NFT you mint</div>
        </div>
        <span class="setting-value">5%</span>
      </div>

      <div class="setting-divider" />

      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Image Storage</div>
          <div class="setting-desc">Decentralised — images live on IPFS forever</div>
        </div>
        <span class="setting-value">IPFS (Pinata)</span>
      </div>
    </div>

    <div class="settings-card settings-card--danger">
      <div class="card-title card-title--danger">Account</div>
      <div class="setting-row">
        <div class="setting-label">
          <div class="setting-name">Sign Out</div>
          <div class="setting-desc">You'll need to log in again to access your NFTs</div>
        </div>
        <button class="btn-logout" @click="handleLogout" :disabled="loggingOut">
          <Loader2 v-if="loggingOut" :size="13" class="spin" />
          <LogOut v-else :size="13" />
          {{ loggingOut ? 'Signing out...' : 'Sign Out' }}
        </button>
      </div>
    </div>

    <p class="version-note">LankaNFT · Cardano Preprod · v1.0.0</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowLeft, Check, Copy, ExternalLink,
  Loader2, LogOut, Shield, Wallet,
} from 'lucide-vue-next'
import { useAuthStore }      from '@/stores/auth'
import { useDashboardStore } from '@/stores/dashboard'
import { useWalletSession }  from '@/composables/useWalletSession'

const auth          = useAuthStore()
const dashboard     = useDashboardStore()
const walletSession = useWalletSession()
const router        = useRouter()

const addressCopied = ref(false)
const loggingOut    = ref(false)

const walletAddress = computed(() => auth.walletAddress || '')

const displayName = computed(() => {
  if (auth.walletType === 'external') {
    const addr = walletAddress.value
    return addr ? `${addr.slice(0, 12)}...${addr.slice(-6)}` : 'External Wallet'
  }
  return auth.userID || 'User'
})

const avatarLetter = computed(() => {
  if (auth.walletType === 'external') return '◈'
  const identifier = auth.userID || ''
  return identifier.charAt(0).toUpperCase() || 'U'
})

async function copyAddress() {
  if (!walletAddress.value) return
  await navigator.clipboard.writeText(walletAddress.value)
  addressCopied.value = true
  setTimeout(() => (addressCopied.value = false), 2000)
}

async function handleLogout() {
  loggingOut.value = true
  try {
    await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
  } catch {}
  walletSession.clearWallet?.()
  dashboard.reset()
  auth.$reset?.()
  router.push('/auth')
}
</script>

<style scoped>
.settings { padding: 28px 32px; max-width: 640px; }

.back-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  color: #888; text-decoration: none;
  margin-bottom: 20px; transition: all 0.15s;
}
.back-link:hover { background: #EEEDFE; color: #534AB7; }

.page-header { margin-bottom: 24px; }
.page-title  { font-size: 22px; font-weight: 600; margin: 0 0 4px; }
.page-sub    { font-size: 13px; color: #888; margin: 0; }

/* ── Cards ── */
.settings-card {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 14px; padding: 20px;
  margin-bottom: 14px;
}
.settings-card--danger { border-color: #fde8e8; }

.card-title {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #aaa; margin-bottom: 16px;
}
.card-title--danger { color: #d32f2f; }

/* ── Profile ── */
.profile-row {
  display: flex; align-items: center; gap: 14px;
}
.avatar {
  width: 52px; height: 52px; border-radius: 50%;
  background: #EEEDFE; color: #534AB7;
  font-size: 20px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.profile-info     { display: flex; flex-direction: column; gap: 3px; }
.profile-name     { font-size: 15px; font-weight: 600; color: #111; }
.profile-email    { font-size: 12px; color: #888; }
.wallet-type-badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 9px; border-radius: 20px;
  font-size: 10px; font-weight: 600; margin-top: 2px;
  width: fit-content;
}
.badge--external  { background: #EEEDFE; color: #534AB7; }
.badge--custodial { background: #E1F5EE; color: #085041; }

/* ── Setting rows ── */
.setting-row {
  display: flex; align-items: center;
  justify-content: space-between; gap: 16px;
}
.setting-row--col { flex-direction: column; align-items: flex-start; gap: 10px; }
.setting-divider  { height: 1px; background: #f5f5f5; margin: 12px 0; }

.setting-label { flex: 1; min-width: 0; }
.setting-name  { font-size: 13px; font-weight: 500; color: #111; margin-bottom: 2px; }
.setting-desc  { font-size: 11px; color: #aaa; line-height: 1.4; }
.setting-value { font-size: 12px; font-weight: 500; color: #555; flex-shrink: 0; }

/* Network badge */
.network-badge {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 500; color: #7a5c00;
  background: #FFF8E1; padding: 4px 10px; border-radius: 20px;
  flex-shrink: 0;
}
.network-dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: #085041; display: inline-block;
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; } 50% { opacity: 0.4; }
}

/* Address */
.address-box {
  display: flex; align-items: center; gap: 8px;
  background: #f8f8f8; border-radius: 8px; padding: 10px 12px;
  width: 100%;
}
.address-text {
  font-family: monospace; font-size: 11px; color: #444;
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 0; flex-shrink: 0; display: flex; align-items: center;
  opacity: 0.7; transition: opacity 0.15s;
}
.copy-btn:hover { opacity: 1; }
.scan-link {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 500; color: #534AB7;
  text-decoration: none; transition: opacity 0.15s;
}
.scan-link:hover { opacity: 0.75; }

/* Logout */
.btn-logout {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; background: #FCEBEB; color: #d32f2f;
  border: 1px solid #f5c6c6; border-radius: 8px;
  font-size: 12px; font-weight: 600; cursor: pointer;
  transition: all 0.15s; flex-shrink: 0;
}
.btn-logout:hover:not(:disabled) { background: #f5c6c6; }
.btn-logout:disabled { opacity: 0.6; cursor: not-allowed; }

.version-note {
  font-size: 11px; color: #ccc; text-align: center;
  margin-top: 8px;
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 640px) {
  .settings { padding: 16px; }
  .setting-row { flex-direction: column; align-items: flex-start; }
}
</style>