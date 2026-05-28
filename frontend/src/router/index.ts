import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth',
      name: 'auth',
      component: () => import('@/views/AuthView.vue'),
      meta: { requiresGuest: true },
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/mint',
      name: 'create-mint',
      component: () => import('@/views/CreateMintView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/batch',
      name: 'batch-mint',
      component: () => import('@/views/BatchMintView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/browse',
      name: 'browse',
      component: () => import('@/views/BrowseMintsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/nft/:id',
      name: 'nft-detail',
      component: () => import('@/views/NFTDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/activity',
      name: 'activity',
      component: () => import('@/views/ActivityView.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.isAuthenticated && !auth.checked) {
    await auth.checkAuth()
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'auth' }
  }
  if (to.meta.requiresGuest && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router