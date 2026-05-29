// ─────────────────────────────────────────────────────────────────────────────
// frontend/src/services/api.ts
//
// Axios instance used for all API calls to the Go backend.
//
// Key behaviours:
//   - withCredentials: true — sends the httpOnly JWT cookie on every request
//   - 401 interceptor — when the JWT expires, the backend returns 401.
//     We catch this globally, clear local auth state, and redirect to /auth
//     with ?reason=session_expired so the auth page can show a clear message.
//     Without this, users get silent failures all over the app.
//   - 503 interceptor — backend is down. We redirect to /auth with
//     ?reason=backend_down so the user gets a clear message instead of
//     a wall of network errors.
// ─────────────────────────────────────────────────────────────────────────────

import axios from 'axios'
import router from '@/router'

// All API calls go through this instance.
// Base URL is set in .env as VITE_API_BASE_URL (e.g. http://localhost:8080)
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// ── Response interceptor ───────────────────────────────────────────────────────
api.interceptors.response.use(
  // Pass successful responses straight through
  (response) => response,

  (error) => {
    const status  = error.response?.status
    const isAuth  = window.location.pathname.includes('/auth')

    // 401 — JWT expired or invalid. Clear state and redirect to login.
    // Guard: don't redirect if already on /auth (would cause a loop).
    if (status === 401 && !isAuth) {
      // Use Vue Router for a clean SPA redirect (preserves history)
      router.push({ name: 'auth', query: { reason: 'session_expired' } })
      return Promise.reject(error)
    }

    // 503 — backend is completely down
    if (status === 503 && !isAuth) {
      router.push({ name: 'auth', query: { reason: 'backend_down' } })
      return Promise.reject(error)
    }

    return Promise.reject(error)
  }
)

export default api