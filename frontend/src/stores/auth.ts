// ─────────────────────────────────────────────────────────────────────────────
// stores/auth.ts
//
// Auth store — manages user session state across the app
//
// State:
//   userID        — UUID of the logged-in user
//   walletType    — 'custodial' | 'external' | null
//   walletAddress — on-chain Cardano address
//   isAuthenticated, isLoading, error, checked
//
// wallet_type drives the entire UI split:
//   custodial → backend signs all transactions (mnemonic-based)
//   external  → frontend signs with CIP-30 (unsigned CBOR flow)
//
// checkAuth() is called on every page load. It hits /api/me which now
// returns wallet_type and wallet_address, so the rest of the app always
// knows which user type it's dealing with without an extra DB query.
// ─────────────────────────────────────────────────────────────────────────────
import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', () => {

  // ── State ────────────────────────────────────────────────────────────────────
  const userID        = ref<string | null>(null)
  const walletType    = ref<'custodial' | 'external' | null>(null)
  const walletAddress = ref<string | null>(null)
  const isAuthenticated = ref(false)
  const isLoading     = ref(false)
  const error         = ref<string | null>(null)
  const checked       = ref(false) // true after first /api/me call completes

  // ── Actions ──────────────────────────────────────────────────────────────────

  // Request OTP — sends 6-digit code to user's email
  async function requestOTP(email: string) {
    isLoading.value = true
    error.value = null
    try {
      await api.post('/auth/request-otp', { email })
      return true
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to send OTP'
      return false
    } finally {
      isLoading.value = false
    }
  }

  // Verify OTP — backend sets JWT cookie on success
  async function verifyOTP(email: string, code: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await api.post('/auth/verify-otp', { email, code })
      userID.value          = res.data.user_id
      walletType.value      = 'custodial'
      walletAddress.value   = null // will be populated by checkAuth
      isAuthenticated.value = true
      return true
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Invalid OTP'
      return false
    } finally {
      isLoading.value = false
    }
  }

  // Connect external CIP-30 wallet (Nami/Eternl/Lace)
  // walletAddress is the bech32 address from getUsedAddresses()/getChangeAddress()
  async function connectWallet(walletAddr: string, walletName: string, signature: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await api.post('/auth/wallet-verify', {
        wallet_address: walletAddr,
        wallet_name:    walletName,
        signature,
      })
      userID.value          = res.data.user_id
      walletType.value      = 'external'
      walletAddress.value   = walletAddr
      isAuthenticated.value = true
      return true
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Wallet connection failed'
      return false
    } finally {
      isLoading.value = false
    }
  }

  // Check if user is already logged in (JWT cookie still valid).
  // Called on every page load. Now also loads wallet_type + wallet_address
  // from /api/me so the rest of the app can react to the user type.
  async function checkAuth() {
    try {
      const res = await api.get('/api/me')
      userID.value          = res.data.user_id
      walletType.value      = res.data.wallet_type    || 'custodial'
      walletAddress.value   = res.data.wallet_address || null
      isAuthenticated.value = true
    } catch {
      isAuthenticated.value = false
      userID.value          = null
      walletType.value      = null
      walletAddress.value   = null
    } finally {
      checked.value = true
    }
  }

  // Logout — clears JWT cookie server-side and resets all local state
  async function logout() {
    try {
      await api.post('/auth/logout')
    } finally {
      userID.value          = null
      walletType.value      = null
      walletAddress.value   = null
      isAuthenticated.value = false
    }
  }

  return {
    userID,
    walletType,
    walletAddress,
    isAuthenticated,
    isLoading,
    error,
    checked,
    requestOTP,
    verifyOTP,
    connectWallet,
    checkAuth,
    logout,
  }
})