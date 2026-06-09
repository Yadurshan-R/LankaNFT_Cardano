// ─────────────────────────────────────────────────────────────────────────────
// frontend/src/router/index.ts
//
// Vue Router configuration for LankaNFT.
//
// Key behaviours:
//   - Auth guard: unauthenticated users are redirected to /auth
//   - Guest guard: authenticated users cannot access /auth
//   - Page titles: each route sets document.title for browser tabs
//     and bookmarks (format: "Page — LankaNFT")
//   - 404 catch-all: any unknown URL shows NotFoundView instead of
//     a blank page or console error
// ─────────────────────────────────────────────────────────────────────────────

import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth',
      name: 'auth',
      component: () => import('@/views/AuthView.vue'),
      meta: { requiresGuest: true, title: 'Sign In' },
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true, title: 'Dashboard' },
    },
    {
      path: '/mint',
      name: 'create-mint',
      component: () => import('@/views/CreateMintView.vue'),
      meta: { requiresAuth: true, title: 'Create Mint' },
    },
    {
      path: '/batch',
      name: 'batch-mint',
      component: () => import('@/views/BatchMintView.vue'),
      meta: { requiresAuth: true, title: 'Batch Upload' },
    },
    {
      path: '/browse',
      name: 'browse',
      component: () => import('@/views/BrowseMintsView.vue'),
      meta: { requiresAuth: true, title: 'Browse Mints' },
    },
    {
      path: '/nft/:id',
      name: 'nft-detail',
      component: () => import('@/views/NFTDetailView.vue'),
      meta: { requiresAuth: true, title: 'NFT Detail' },
    },
    {
      path: '/activity',
      name: 'activity',
      component: () => import('@/views/ActivityView.vue'),
      meta: { requiresAuth: true, title: 'Activity' },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/SettingsView.vue'),
      meta: { requiresAuth: true, title: 'Settings' },
    },
    {
      path: '/certificate/:id',
      name: 'certificate',
      component: () => import('@/views/CertificateView.vue'),
      meta: { title: 'NFT Certificate' },
      // No requiresAuth or requiresGuest — fully public page
    },
    // ── Catch-all 404 ──────────────────────────────────────────────────────
    // Must be last — matches any URL not matched above.
    // No auth requirement — anyone can land on a broken URL.
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { title: 'Page Not Found' },
    },
  ],
})

// ── Navigation guards ─────────────────────────────────────────────────────────
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // Check auth state once per session (not on every navigation)
  if (!auth.isAuthenticated && !auth.checked) {
    await auth.checkAuth()
  }

  // Set page title — format: "Dashboard — LankaNFT"
  const pageTitle = to.meta.title as string | undefined
  document.title = pageTitle ? `${pageTitle} — LankaNFT` : 'LankaNFT'

  // Protected route — redirect to sign in if not authenticated
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'auth' }
  }

  // Guest-only route — redirect to dashboard if already signed in
  if (to.meta.requiresGuest && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router