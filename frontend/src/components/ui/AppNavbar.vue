<template>
  <nav class="navbar">
    <!-- Logo placeholder -->
    <div class="navbar-logo" />

    <!-- Nav links -->
    <div class="navbar-links">
      <router-link to="/" class="nav-link">Dashboard</router-link>
      <router-link to="/mint" class="nav-link">Create Mint</router-link>
      <router-link to="/browse" class="nav-link">Browse Mints</router-link>
    </div>

    <!-- Right side — wallet status or connect button -->
    <div class="navbar-right">
      <template v-if="auth.isAuthenticated">
        <span class="wallet-badge">
          <span class="dot" /> Wallet Connected
        </span>
        <button class="avatar-btn" @click="auth.logout()">Logout</button>
      </template>
      <template v-else>
        <router-link to="/auth">
          <BaseButton variant="primary">Connect Wallet</BaseButton>
        </router-link>
      </template>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import BaseButton from './BaseButton.vue'

const auth = useAuthStore()
</script>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  height: 64px;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
  position: sticky;
  top: 0;
  z-index: 100;
}
.navbar-logo {
  width: 32px;
  height: 32px;
  background: #e0e0e0;
  border-radius: 6px;
}
.navbar-links {
  display: flex;
  gap: 32px;
}
.nav-link {
  font-size: 14px;
  color: #444;
  text-decoration: none;
  font-weight: 500;
}
.nav-link.router-link-active {
  color: #111;
  font-weight: 600;
  border-bottom: 2px solid #111;
  padding-bottom: 2px;
}
.navbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.wallet-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  background: #d4f7d4;
  color: #1a6b1a;
  padding: 6px 12px;
  border-radius: 20px;
}
.dot {
  width: 8px;
  height: 8px;
  background: #1d9e1d;
  border-radius: 50%;
}
.avatar-btn {
  font-size: 13px;
  background: none;
  border: none;
  cursor: pointer;
  color: #666;
}
</style>