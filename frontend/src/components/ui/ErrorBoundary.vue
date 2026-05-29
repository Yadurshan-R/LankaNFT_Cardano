<template>
  <div v-if="hasError" class="error-boundary">
    <div class="error-card">
      <div class="error-icon">
        <AlertCircle :size="32" color="#d32f2f" />
      </div>
      <h2 class="error-title">Something went wrong</h2>
      <p class="error-desc">
        An unexpected error occurred. Please refresh the page.
        If the problem persists, try signing out and back in.
      </p>

      <!-- Show raw error in development only — never in production -->
      <pre v-if="isDev && errorMessage" class="error-detail">{{ errorMessage }}</pre>

      <div class="error-actions">
        <button class="btn-primary" @click="refresh">
          <RefreshCw :size="14" /> Refresh page
        </button>
        <router-link to="/">
          <button class="btn-outline" @click="reset">
            Back to Dashboard
          </button>
        </router-link>
      </div>
    </div>
  </div>

  <!-- Render children normally when no error -->
  <slot v-else />
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// ErrorBoundary.vue
//
// Wraps <router-view> in App.vue. Catches Vue component render errors
// via onErrorCaptured and shows a clean error screen instead of a
// white/blank page.
//
// Raw error details are shown only in development (import.meta.env.DEV)
// to avoid leaking internal info to production users.
//
// Usage in App.vue:
//   <ErrorBoundary>
//     <router-view />
//   </ErrorBoundary>
// ─────────────────────────────────────────────────────────────────────────────

import { onErrorCaptured, ref } from 'vue'
import { AlertCircle, RefreshCw } from 'lucide-vue-next'

const hasError    = ref(false)
const errorMessage = ref('')
const isDev       = import.meta.env.DEV

// Vue's error boundary hook — catches errors from child components
onErrorCaptured((error: Error) => {
  hasError.value     = true
  errorMessage.value = error?.message || String(error)

  // Return false to stop error propagation up the component tree
  return false
})

function refresh() {
  window.location.reload()
}

function reset() {
  hasError.value     = false
  errorMessage.value = ''
}
</script>

<style scoped>
.error-boundary {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  padding: 24px; background: #f8f8f8;
}

.error-card {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 16px; padding: 48px 40px;
  max-width: 440px; width: 100%;
  text-align: center;
  display: flex; flex-direction: column;
  align-items: center; gap: 14px;
}

.error-icon {
  width: 64px; height: 64px; border-radius: 50%;
  background: #FCEBEB;
  display: flex; align-items: center; justify-content: center;
}

.error-title { font-size: 20px; font-weight: 600; color: #111; margin: 0; }
.error-desc  { font-size: 13px; color: #888; line-height: 1.6; margin: 0; max-width: 320px; }

.error-detail {
  background: #f8f8f8; border: 1px solid #e8e8e8;
  border-radius: 8px; padding: 12px;
  font-size: 11px; color: #666;
  text-align: left; width: 100%;
  overflow-x: auto; white-space: pre-wrap;
  word-break: break-word;
  max-height: 120px; overflow-y: auto;
}

.error-actions {
  display: flex; gap: 10px; flex-wrap: wrap;
  justify-content: center; margin-top: 4px;
}

.btn-primary {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 20px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-primary:hover { background: #3d35a0; }

.btn-outline {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 20px;
  background: transparent; color: #534AB7;
  border: 1px solid #534AB7; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: all 0.15s;
}
.btn-outline:hover { background: #EEEDFE; }
</style>