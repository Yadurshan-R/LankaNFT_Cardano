<template>
  <div class="not-found">
    <div class="not-found-card">

      <!-- Logo -->
      <img src="@/assets/lanka-nft-logo.png" alt="LankaNFT" class="logo" />

      <!-- Error code -->
      <div class="error-code">404</div>

      <h1 class="error-title">Page not found</h1>
      <p class="error-desc">
        The page you're looking for doesn't exist or has been moved.
      </p>

      <!-- Actions -->
      <div class="actions">
        <router-link to="/">
          <button class="btn-primary">
            <LayoutDashboard :size="14" /> Go to Dashboard
          </button>
        </router-link>
        <router-link to="/browse">
          <button class="btn-outline">
            <LayoutGrid :size="14" /> Browse Mints
          </button>
        </router-link>
      </div>

      <!-- Network badge -->
      <div class="network-badge">
        <span class="network-dot" />
        Cardano Preprod
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// NotFoundView.vue
//
// Shown when a user navigates to a URL that doesn't exist.
// Registered as the catch-all route in router/index.ts:
//   { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView }
//
// Does NOT require auth — anyone can land on a broken URL.
// ─────────────────────────────────────────────────────────────────────────────

import { LayoutDashboard, LayoutGrid } from 'lucide-vue-next'
</script>

<style scoped>
.not-found {
  min-height: 100vh;
  background: #f8f8f8;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.not-found-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 20px;
  padding: 48px 40px;
  text-align: center;
  max-width: 420px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.logo {
  width: 56px; height: 56px;
  object-fit: contain;
  border-radius: 12px;
}

.error-code {
  font-size: 72px;
  font-weight: 800;
  color: #EEEDFE;
  line-height: 1;
  letter-spacing: -2px;
  /* Subtle purple outline effect */
  -webkit-text-stroke: 2px #534AB7;
}

.error-title {
  font-size: 22px;
  font-weight: 600;
  color: #111;
  margin: 0;
}

.error-desc {
  font-size: 14px;
  color: #888;
  line-height: 1.6;
  margin: 0;
  max-width: 300px;
}

.actions {
  display: flex;
  gap: 10px;
  margin-top: 8px;
  flex-wrap: wrap;
  justify-content: center;
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

.network-badge {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: #aaa;
  margin-top: 8px;
}
.network-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #085041;
  display: inline-block;
}
</style>