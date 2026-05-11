import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  // State
  const userID = ref<string | null>(null)
  const isAuthenticated = ref(false)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Request OTP — sends email to Go backend
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

  // Verify OTP — Go backend sets JWT cookie on success
  async function verifyOTP(email: string, code: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await api.post('/auth/verify-otp', { email, code })
      userID.value = res.data.user_id
      isAuthenticated.value = true
      return true
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Invalid OTP'
      return false
    } finally {
      isLoading.value = false
    }
  }

  // Connect external wallet (Lace/Nami/Eternl)
  async function connectWallet(walletAddress: string, walletName: string, signature: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await api.post('/auth/wallet-verify', {
        wallet_address: walletAddress,
        wallet_name: walletName,
        signature,
      })
      userID.value = res.data.user_id
      isAuthenticated.value = true
      return true
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Wallet connection failed'
      return false
    } finally {
      isLoading.value = false
    }
  }

  // Check if user is already logged in (cookie still valid)
  async function checkAuth() {
    try {
      const res = await api.get('/api/me')
      userID.value = res.data.user_id
      isAuthenticated.value = true
    } catch {
      isAuthenticated.value = false
      userID.value = null
    }
  }

  // Logout
  async function logout() {
    try {
      await api.post('/auth/logout')
    } finally {
      userID.value = null
      isAuthenticated.value = false
    }
  }

  return {
    userID,
    isAuthenticated,
    isLoading,
    error,
    requestOTP,
    verifyOTP,
    connectWallet,
    checkAuth,
    logout,
  }
})