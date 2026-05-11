<template>
  <div class="auth-page">
    <div class="auth-card">
      <!-- Tab switcher -->
      <div class="auth-tabs">
        <button
  :class="['tab', { active: activeTab === 'email' }]"
  @click="activeTab = 'email'; auth.error = null"
>
  Email OTP
</button>
<button
  :class="['tab', { active: activeTab === 'wallet' }]"
  @click="activeTab = 'wallet'; auth.error = null"
>
  Connect Wallet
</button>
      </div>

      <!-- Tab content -->
      <div class="auth-content">
        <EmailOTPForm v-if="activeTab === 'email'" />
        <WalletConnect v-else />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import EmailOTPForm from '@/components/auth/EmailOTPForm.vue'
import WalletConnect from '@/components/auth/WalletConnect.vue'

const auth = useAuthStore()
const activeTab = ref<'email' | 'wallet'>('email')
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f7f8;
}
.auth-card {
  background: #fff;
  border-radius: 16px;
  padding: 40px;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
}
.auth-tabs {
  display: flex;
  gap: 4px;
  background: #f4f4f4;
  border-radius: 8px;
  padding: 4px;
  margin-bottom: 28px;
}
.tab {
  flex: 1;
  padding: 8px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  color: #666;
  transition: all 0.2s;
}
.tab.active {
  background: #fff;
  color: #111;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}
</style>