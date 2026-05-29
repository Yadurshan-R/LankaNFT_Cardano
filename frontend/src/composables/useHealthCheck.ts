// ─────────────────────────────────────────────────────────────────────────────
// frontend/src/composables/useHealthCheck.ts
//
// Composable that checks whether the Go backend is reachable.
// Called once on app mount in App.vue.
//
// Why this exists:
//   If the backend is down, every API call fails silently with a network error.
//   Users see loading spinners forever or cryptic errors. By checking /health
//   upfront, we can show a clear "service unavailable" banner immediately.
//
// Behaviour:
//   - Checks GET /health (no auth required)
//   - If first check fails, retries once after 3 seconds
//   - Sets isBackendUp = false only if both attempts fail
//   - Uses a short 5s timeout so the check doesn't hang
// ─────────────────────────────────────────────────────────────────────────────

import { ref } from 'vue'

const HEALTH_URL    = `${import.meta.env.VITE_API_BASE_URL}/health`
const TIMEOUT_MS    = 5_000  // 5 second timeout per attempt
const RETRY_DELAY   = 3_000  // Wait 3 seconds before retrying

export function useHealthCheck() {
  const isBackendUp   = ref(true)  // Optimistic default — assume up
  const isChecking    = ref(false)

  async function checkOnce(): Promise<boolean> {
    try {
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), TIMEOUT_MS)

      const response = await fetch(HEALTH_URL, {
        signal: controller.signal,
        credentials: 'include',
      })
      clearTimeout(timer)
      return response.ok
    } catch {
      return false
    }
  }

  /**
   * Runs the health check with one automatic retry.
   * Updates isBackendUp reactively so App.vue can show/hide the banner.
   */
  async function runHealthCheck() {
    isChecking.value = true

    const firstAttempt = await checkOnce()

    if (firstAttempt) {
      isBackendUp.value = true
      isChecking.value  = false
      return
    }

    // First attempt failed — wait and retry once before declaring backend down
    await new Promise((resolve) => setTimeout(resolve, RETRY_DELAY))
    const secondAttempt = await checkOnce()

    isBackendUp.value = secondAttempt
    isChecking.value  = false
  }

  return { isBackendUp, isChecking, runHealthCheck }
}