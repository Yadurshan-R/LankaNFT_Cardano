<template>
  <div id="app">

    <div v-if="!isBackendUp" class="maintenance-banner">
      <AlertCircle :size="14" />
      LankaNFT is temporarily unavailable. Please wait a moment.
      <button class="banner-retry" @click="runHealthCheck">
        <RefreshCw :size="12" /> Retry
      </button>
    </div>

    <template v-if="isAuthPage">
      <ErrorBoundary>
        <router-view />
      </ErrorBoundary>
    </template>

    <template v-else>
      <div class="app-layout">
        <AppSidebar />
        <main class="main-content">
          <ErrorBoundary>
            <router-view />
          </ErrorBoundary>
        </main>
      </div>
    </template>

  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// App.vue — LankaNFT root layout
//
// Responsibilities:
//   1. Route-based layout — auth page is full screen, everything else
//      uses AppSidebar + main content layout
//   2. ErrorBoundary — wraps router-view so any component crash shows
//      a clean error screen instead of a white page
//   3. Health check — checks /health on mount and shows a maintenance
//      banner if the backend is down (non-blocking)
//   4. Sidebar push — body.sidebar-expanded class drives the
//      main-content margin-left transition (set by AppSidebar.vue)
// ─────────────────────────────────────────────────────────────────────────────

import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { AlertCircle, RefreshCw } from 'lucide-vue-next'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import ErrorBoundary from '@/components/ui/ErrorBoundary.vue'
import { useHealthCheck } from '@/composables/useHealthCheck'

const route      = useRoute()
const isAuthPage = computed(() => route.name === 'auth' || route.name === 'not-found')

const { isBackendUp, runHealthCheck } = useHealthCheck()

// Run health check once on app load.
// If it fails, the maintenance banner appears automatically.
onMounted(() => {
  runHealthCheck()
})
</script>

<style>
/* ── Global reset ── */
* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  background: #f8f8f8;
  color: #111;
  -webkit-font-smoothing: antialiased;
}

a { text-decoration: none; color: inherit; }

/* ── Layout ── */
.app-layout { display: flex; min-height: 100vh; }

.main-content {
  flex: 1;
  margin-left: 72px;
  min-height: 100vh;
  background: #f8f8f8;
  transition: margin-left 0.2s ease;
}

/* Sidebar expanded — pushes content right smoothly */
body.sidebar-expanded .main-content {
  margin-left: 200px;
}

/* ── Maintenance banner ── */
.maintenance-banner {
  position: fixed;
  top: 0; left: 0; right: 0;
  z-index: 9999;
  display: flex; align-items: center; justify-content: center; gap: 10px;
  padding: 10px 20px;
  background: #FAEEDA; color: #854F0B;
  font-size: 13px; font-weight: 500;
  border-bottom: 1px solid #f0d9b5;
}

.banner-retry {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 12px;
  background: #854F0B; color: #fff;
  border: none; border-radius: 6px;
  font-size: 12px; font-weight: 600;
  cursor: pointer; transition: opacity 0.15s;
  margin-left: 4px;
}
.banner-retry:hover { opacity: 0.85; }
</style>