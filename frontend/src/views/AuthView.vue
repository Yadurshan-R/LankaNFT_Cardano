<template>
  <div class="auth-page">

    <!-- ── Left: branding ── -->
    <div class="auth-brand">
      <div class="brand-glow" />
      <div class="brand-content">

        <div class="brand-logo">
          <div class="logo-ring">
            <img src="@/assets/lanka-nft-logo.png" alt="LankaNFT" class="logo-img" />
          </div>
          <span class="logo-text">LankaNFT</span>
        </div>

        <h1 class="brand-title">
          The Future of<br />Digital Ownership<br />in Sri Lanka
        </h1>

        <p class="brand-desc">
          Mint, trade, and own NFTs on Cardano.<br />
          No crypto knowledge needed.
        </p>

        <div class="network-badge">
          <span class="network-dot" />
          Live on Cardano Preprod
        </div>

      </div>
    </div>

    <!-- ── Right: form ── -->
    <div class="auth-right">
      <div class="auth-card">

        <div class="auth-header">
          <h2 class="auth-title">Sign in</h2>
          <p class="auth-subtitle">Welcome back to LankaNFT</p>
        </div>

        <div class="auth-tabs">
          <button
            :class="['auth-tab', { 'auth-tab--active': activeTab === 'email' }]"
            @click="activeTab = 'email'; auth.error = null"
          >
            <Mail :size="14" /> Email OTP
          </button>
          <button
            :class="['auth-tab', { 'auth-tab--active': activeTab === 'wallet' }]"
            @click="activeTab = 'wallet'; auth.error = null"
          >
            <Wallet :size="14" /> Connect Wallet
          </button>
        </div>

        <div class="auth-content">
          <EmailOTPForm v-if="activeTab === 'email'" />
          <WalletConnect v-else />
        </div>

      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Mail, Wallet } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import EmailOTPForm from '@/components/auth/EmailOTPForm.vue'
import WalletConnect from '@/components/auth/WalletConnect.vue'

const auth      = useAuthStore()
const activeTab = ref<'email' | 'wallet'>('email')
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1fr 1fr;
}

/* ── Left ── */
.auth-brand {
  background: #111827;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 56px 48px;
  position: relative;
  overflow: hidden;
}

.brand-glow {
  position: absolute;
  width: 480px; height: 480px;
  background: radial-gradient(circle, rgba(83,74,183,0.3) 0%, transparent 70%);
  top: -80px; right: -80px;
  border-radius: 50%;
  pointer-events: none;
}

.brand-content {
  position: relative;
  z-index: 1;
  max-width: 380px;
  width: 100%;
}

.brand-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 48px;
}

.logo-ring {
  width: 46px; height: 46px;
  border-radius: 50%;
  padding: 2px;
  background: rgba(255,255,255,0.12);
  flex-shrink: 0;
}

.logo-img {
  width: 100%; height: 100%;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}

.logo-text {
  font-size: 20px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.3px;
}

.brand-title {
  font-size: 38px;
  font-weight: 800;
  color: #fff;
  line-height: 1.2;
  letter-spacing: -0.8px;
  margin-bottom: 20px;
}

.brand-desc {
  font-size: 15px;
  color: rgba(255,255,255,0.5);
  line-height: 1.7;
  margin-bottom: 40px;
}

.network-badge {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 20px;
  padding: 7px 16px;
  font-size: 12px;
  color: rgba(255,255,255,0.5);
}

.network-dot {
  width: 6px; height: 6px;
  background: #4ade80;
  border-radius: 50%;
  box-shadow: 0 0 6px rgba(74,222,128,0.6);
}

/* ── Right ── */
.auth-right {
  background: #f7f8fa;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 32px;
}

.auth-card {
  background: #fff;
  border-radius: 20px;
  padding: 40px;
  width: 100%;
  max-width: 400px;
  border: 1px solid #eaeaea;
  box-shadow: 0 4px 24px rgba(0,0,0,0.06);
  position: relative;
  overflow: hidden;
}

/* Purple top accent */
.auth-card::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 3px;
  background: linear-gradient(90deg, #534AB7, #7B6FD4 50%, #085041);
}

.auth-header { margin-bottom: 28px; }

.auth-title {
  font-size: 28px;
  font-weight: 700;
  color: #0d0d0d;
  letter-spacing: -0.5px;
  margin-bottom: 4px;
}

.auth-subtitle {
  font-size: 14px;
  color: #999;
}

.auth-tabs {
  display: flex;
  gap: 4px;
  background: #f3f4f6;
  border-radius: 12px;
  padding: 4px;
  margin-bottom: 28px;
}

.auth-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 9px 12px;
  border: none;
  border-radius: 9px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  color: #999;
  transition: all 0.18s ease;
}

.auth-tab:hover { color: #534AB7; }

.auth-tab--active {
  background: #fff;
  color: #534AB7;
  font-weight: 600;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08), 0 0 0 1px rgba(83,74,183,0.1);
}

@media (max-width: 768px) {
  .auth-page  { grid-template-columns: 1fr; }
  .auth-brand { display: none; }
  .auth-right { padding: 24px 16px; min-height: 100vh; }
}
</style>