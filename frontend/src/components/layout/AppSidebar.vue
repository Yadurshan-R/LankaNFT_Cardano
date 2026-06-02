<template>
  <aside
    class="sidebar"
    :class="{ 'sidebar--expanded': isExpanded }"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <router-link to="/" class="sidebar-logo">
      <img src="@/assets/lanka-nft-logo.png" alt="LankaNFT" class="logo-img" />
      <span class="logo-text">LankaNFT</span>
    </router-link>

    <div class="sidebar-divider" />

    <nav class="sidebar-nav">
      <router-link to="/"      exact-active-class="sidebar-item--active" class="sidebar-item"><LayoutDashboard :size="20" /><span class="sidebar-label">Dashboard</span></router-link>
      <router-link to="/mint"   active-class="sidebar-item--active"       class="sidebar-item"><Sparkles         :size="20" /><span class="sidebar-label">Create Mint</span></router-link>
      <router-link to="/browse" active-class="sidebar-item--active"       class="sidebar-item"><LayoutGrid       :size="20" /><span class="sidebar-label">Browse Mints</span></router-link>
    </nav>

    <div class="sidebar-divider" />

    <nav class="sidebar-nav">
      <router-link to="/activity" active-class="sidebar-item--active" class="sidebar-item"><Activity :size="20" /><span class="sidebar-label">Activity</span></router-link>
    </nav>

    <div class="sidebar-bottom">
      <div class="sidebar-divider" />

      <button
        class="sidebar-item sidebar-item--btn"
        :class="{ 'sidebar-item--active': isWalletOpen }"
        @click.stop="isWalletOpen = !isWalletOpen"
      >
        <Wallet :size="20" />
        <span class="sidebar-label">Wallet</span>
      </button>

      <WalletPopup
        v-if="isWalletOpen && isExpanded"
        :address="dashboard.walletAddress"
        :lovelace="dashboard.lovelace"
        :walletType="auth.walletType ?? undefined"
        @close="isWalletOpen = false"
      />

      <router-link to="/settings" active-class="sidebar-item--active" class="sidebar-item"><Settings :size="20" /><span class="sidebar-label">Settings</span></router-link>

      <button class="sidebar-item sidebar-item--btn sidebar-item--logout" @click="handleLogout">
        <LogOut :size="20" />
        <span class="sidebar-label">Logout</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, LayoutDashboard, LayoutGrid, LogOut, Settings, Sparkles, Wallet } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useDashboardStore } from '@/stores/dashboard'
import { useWalletSession } from '@/composables/useWalletSession'
import WalletPopup from '@/components/layout/WalletPopup.vue'

const router        = useRouter()
const auth          = useAuthStore()
const dashboard     = useDashboardStore()
const walletSession = useWalletSession()

const isExpanded   = ref(false)
const isWalletOpen = ref(false)

function onMouseEnter() {
  isExpanded.value = true
  document.body.classList.add('sidebar-expanded')
}

function onMouseLeave() {
  isExpanded.value = false
  isWalletOpen.value = false
  document.body.classList.remove('sidebar-expanded')
}

async function handleLogout() {
  walletSession.clearWallet()
  auth.logout()
  dashboard.reset()
  router.push('/auth')
}
</script>

<style scoped>
.sidebar {
  width: 64px;
  min-height: 100vh;
  background: #fff;
  border-right: 1px solid #ebebeb;
  display: flex;
  flex-direction: column;
  padding: 12px 10px;
  position: fixed;
  top: 0; left: 0; bottom: 0;
  z-index: 100;
  overflow: hidden;
  transition: width 0.2s ease;
}
.sidebar--expanded {
  width: 200px;
  box-shadow: 2px 0 16px rgba(0,0,0,0.06);
}

/* Logo */
.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 4px;
  margin-bottom: 8px;
  text-decoration: none;
  overflow: hidden;
  min-height: 44px;
}

.logo-img {
  width: 36px;
  height: 36px;
  object-fit: cover;
  flex-shrink: 0;
  border-radius: 50%;
  display: block;
}

.logo-text {
  font-size: 15px;
  font-weight: 700;
  color: #1B2A6B;
  white-space: nowrap;
  opacity: 0;
  width: 0;
  overflow: hidden;
  transition: opacity 0.2s ease, width 0.2s ease;
}

.sidebar--expanded .logo-text {
  opacity: 1;
  width: auto;
}

/* Divider */
.sidebar-divider { height: 1px; background: #f0f0f0; margin: 6px 0; }

/* Nav */
.sidebar-nav { display: flex; flex-direction: column; gap: 2px; }

/* Item — shared by router-link and button */
.sidebar-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: 10px;
  color: #444;
  cursor: pointer;
  transition: all 0.15s;
  border: none;
  background: transparent;
  width: 100%;
  text-align: left;
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  justify-content: flex-start;
}

.sidebar-item:hover { background: #f0effd; color: #534AB7; }

.sidebar-item--active {
  background: #EEEDFE;
  color: #534AB7;
  border-left: 3px solid #534AB7;
  padding-left: 9px;
}
.sidebar-item--logout:hover { background: #fff0f0; color: #d32f2f; }

/* Label hidden by default, slides in when expanded */
.sidebar-label {
  font-size: 13px;
  font-weight: 500;
  color: inherit;
  opacity: 0;
  width: 0;
  overflow: hidden;
  transition: opacity 0.2s ease, width 0.2s ease;
  white-space: nowrap;
}

.sidebar--expanded .sidebar-label {
  opacity: 1;
  width: auto;
}

/* Bottom sticks to bottom */
.sidebar-bottom { margin-top: auto; display: flex; flex-direction: column; gap: 2px; position: relative; }
</style>