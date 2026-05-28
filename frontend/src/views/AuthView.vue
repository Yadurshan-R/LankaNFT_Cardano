<!-- ─────────────────────────────────────────────────────────────────────────
  AuthView.vue — LankaNFT login page

  Two-column layout on desktop:
    Left:  LankaNFT branding, tagline, feature highlights
    Right: auth card with Email OTP and Wallet Connect tabs

  Child components (unchanged):
    EmailOTPForm.vue  — handles email → OTP → JWT flow
    WalletConnect.vue — handles CIP-30 wallet connection
──────────────────────────────────────────────────────────────────────────── -->
<template>
  <div class="auth-page">

    <!-- Left: branding panel -->
    <div class="auth-brand">
      <div class="brand-content">

        <!-- Logo -->
        <div class="brand-logo">
          <img src="@/assets/lanka-nft-logo.png" alt="LankaNFT" class="logo-img" />
          <span class="logo-text">LankaNFT</span>
        </div>

        <!-- Tagline -->
        <h1 class="brand-title">
          The Future of<br />Digital Ownership<br />in Sri Lanka
        </h1>
        <p class="brand-desc">
          Mint, trade, and own NFTs on Cardano with zero crypto knowledge required.
          Your wallet is created automatically — just sign in with email.
        </p>

        <!-- Feature highlights -->
        <div class="brand-features">
          <div class="feature-item">
            <div class="feature-icon">
              <ShieldCheck :size="16" />
            </div>
            <div>
              <div class="feature-title">Custodial wallet</div>
              <div class="feature-desc">AES-256 encrypted, only you can access</div>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <Zap :size="16" />
            </div>
            <div>
              <div class="feature-title">One-click minting</div>
              <div class="feature-desc">CIP-68 compliant NFTs on Cardano</div>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <Coins :size="16" />
            </div>
            <div>
              <div class="feature-title">On-chain royalties</div>
              <div class="feature-desc">Enforced by smart contracts forever</div>
            </div>
          </div>
        </div>

        <!-- Network badge -->
        <div class="network-badge">
          <span class="network-dot" />
          Running on Cardano Preprod
        </div>

      </div>
    </div>

    <!-- Right: auth card -->
    <div class="auth-right">
      <div class="auth-card">

        <!-- Header -->
        <div class="auth-header">
          <h2 class="auth-title">Welcome back</h2>
          <p class="auth-subtitle">Sign in to your LankaNFT account</p>
        </div>

        <!-- Tab switcher -->
        <div class="auth-tabs">
          <button
            :class="['auth-tab', { 'auth-tab--active': activeTab === 'email' }]"
            @click="activeTab = 'email'; auth.error = null"
          >
            <Mail :size="14" />
            Email OTP
          </button>
          <button
            :class="['auth-tab', { 'auth-tab--active': activeTab === 'wallet' }]"
            @click="activeTab = 'wallet'; auth.error = null"
          >
            <Wallet :size="14" />
            Connect Wallet
          </button>
        </div>

        <!-- Tab content — child components unchanged -->
        <div class="auth-content">
          <EmailOTPForm v-if="activeTab === 'email'" />
          <WalletConnect v-else />
        </div>

        <!-- Footer note -->
        <p class="auth-footer">
          By signing in you agree to our
          <a href="#" class="auth-link">Terms of Service</a>
        </p>

      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Coins, Mail, ShieldCheck, Wallet, Zap } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import EmailOTPForm from '@/components/auth/EmailOTPForm.vue'
import WalletConnect from '@/components/auth/WalletConnect.vue'

const auth      = useAuthStore()
const activeTab = ref<'email' | 'wallet'>('email')
</script>

<style scoped>
/* ── Full page layout ─────────────────────────────────────────────────────── */
.auth-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1fr 1fr;
}

/* ── Left branding panel ──────────────────────────────────────────────────── */
.auth-brand {
  background: #1B2A6B;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  position: relative;
  overflow: hidden;
}

/* Subtle background pattern */
.auth-brand::before {
  content: '';
  position: absolute;
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(83,74,183,0.3) 0%, transparent 70%);
  top: -100px; right: -100px;
  border-radius: 50%;
}

.auth-brand::after {
  content: '';
  position: absolute;
  width: 400px; height: 400px;
  background: radial-gradient(circle, rgba(83,74,183,0.2) 0%, transparent 70%);
  bottom: -50px; left: -50px;
  border-radius: 50%;
}

.brand-content {
  position: relative;
  z-index: 1;
  max-width: 420px;
}

/* Logo */
.brand-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 40px;
}
.logo-img  { width: 48px; height: 48px; object-fit: contain; border-radius: 10px; }
.logo-text { font-size: 22px; font-weight: 700; color: #fff; letter-spacing: -0.3px; }

/* Tagline */
.brand-title {
  font-size: 36px;
  font-weight: 700;
  color: #fff;
  line-height: 1.25;
  margin-bottom: 16px;
  letter-spacing: -0.5px;
}

.brand-desc {
  font-size: 15px;
  color: rgba(255,255,255,0.65);
  line-height: 1.7;
  margin-bottom: 36px;
}

/* Features */
.brand-features {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 36px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.feature-icon {
  width: 32px; height: 32px;
  background: rgba(255,255,255,0.1);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #a8b4ff;
}

.feature-title { font-size: 13px; font-weight: 600; color: #fff; margin-bottom: 2px; }
.feature-desc  { font-size: 12px; color: rgba(255,255,255,0.5); }

/* Network badge */
.network-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 12px;
  color: rgba(255,255,255,0.6);
}
.network-dot {
  width: 6px; height: 6px;
  background: #4ade80;
  border-radius: 50%;
}

/* ── Right auth panel ─────────────────────────────────────────────────────── */
.auth-right {
  background: #f8f8f8;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 32px;
}

.auth-card {
  background: #fff;
  border-radius: 16px;
  padding: 36px;
  width: 100%;
  max-width: 400px;
  border: 1px solid #f0f0f0;
  box-shadow: 0 4px 24px rgba(0,0,0,0.06);
}

/* Auth header */
.auth-header  { margin-bottom: 24px; }
.auth-title   { font-size: 20px; font-weight: 600; margin-bottom: 4px; }
.auth-subtitle { font-size: 13px; color: #888; }

/* Tab switcher */
.auth-tabs {
  display: flex;
  gap: 6px;
  background: #f5f5f5;
  border-radius: 10px;
  padding: 4px;
  margin-bottom: 24px;
}

.auth-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px;
  border: none;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  color: #888;
  transition: all 0.15s;
}
.auth-tab:hover          { color: #534AB7; }
.auth-tab--active        {
  background: #fff;
  color: #534AB7;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
}

/* Content area */
.auth-content { margin-bottom: 20px; }

/* Footer */
.auth-footer { font-size: 11px; color: #aaa; text-align: center; }
.auth-link   { color: #534AB7; text-decoration: none; }
.auth-link:hover { text-decoration: underline; }

/* ── Mobile ───────────────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .auth-page   { grid-template-columns: 1fr; }
  .auth-brand  { display: none; }
  .auth-right  { padding: 24px 16px; min-height: 100vh; }
}
</style>