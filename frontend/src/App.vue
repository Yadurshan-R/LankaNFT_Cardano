<!-- ─────────────────────────────────────────────────────────────────────────
  App.vue — LankaNFT root layout
  
  Auth page:  full screen, no sidebar
  All others: collapsible sidebar (AppSidebar) + main content area
──────────────────────────────────────────────────────────────────────────── -->
<template>
  <div id="app">
    <template v-if="isAuthPage">
      <router-view />
    </template>
    <template v-else>
      <div class="app-layout">
        <AppSidebar />
        <main class="main-content">
          <router-view />
        </main>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from '@/components/layout/AppSidebar.vue'

const route      = useRoute()
const isAuthPage = computed(() => route.name === 'auth')
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  background: #f8f8f8;
  color: #111;
  -webkit-font-smoothing: antialiased;
}

a { text-decoration: none; color: inherit; }

.app-layout   { display: flex; min-height: 100vh; }
.main-content { flex: 1; margin-left: 60px; min-height: 100vh; background: #f8f8f8; }
</style>