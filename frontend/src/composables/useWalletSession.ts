// ─────────────────────────────────────────────────────────────────────────────
// composables/useWalletSession.ts
//
// External wallet session manager for CIP-30 wallets (Nami/Eternl/Lace).
//
// Transaction flow for external wallets:
//   Backend returns unsigned CBOR
//   → signOnly(cbor) — wallet extension signs, returns signed CBOR
//   → backend /api/nft/submit-tx submits via Blockfrost
//   → tx hash returned
//
// WHY we don't use wallet.submitTx():
//   Lace's submitTx() has a known bug — it rejects its own signed CBOR with
//   "DeserialiseFailure: expected list len or indef".
//   Submitting via Blockfrost (through our backend) is reliable and also
//   gives us better error messages from the node.
// ─────────────────────────────────────────────────────────────────────────────
import { ref } from 'vue'

// Module-level refs — shared across all component instances
const walletApi  = ref<any>(null)
const walletId   = ref<string | null>(localStorage.getItem('lankanft_wallet_id'))
const isRestored = ref(false)

export function useWalletSession() {

  // ── Restore wallet after page refresh ────────────────────────────────────────
  async function restoreWallet(): Promise<boolean> {
    if (isRestored.value && walletApi.value) return true
    if (!walletId.value) return false

    const cardano = (window as any).cardano
    console.log('Restoring wallet:', walletId.value, 'Available:', Object.keys(cardano || {}))

    if (!cardano?.[walletId.value]) {
      clearWallet()
      return false
    }

    try {
      walletApi.value  = await cardano[walletId.value].enable()
      isRestored.value = true
      return true
    } catch {
      clearWallet()
      return false
    }
  }

  // ── Store wallet after login ──────────────────────────────────────────────────
  function setWallet(id: string, api: any) {
    walletId.value   = id
    walletApi.value  = api
    isRestored.value = true
    localStorage.setItem('lankanft_wallet_id', id)
  }

  // ── Sign only — returns signed CBOR without submitting ───────────────────────
  //
  // We separate signing from submission because Lace's submitTx() is broken.
  // After signing, the caller sends the signed CBOR to /api/submit on the
  // backend, which submits via Blockfrost.
  //
  // partialSign = true because Plutus script txs have multiple witnesses:
  // the user's key + the script witness. Wallet only provides its own.
  async function signOnly(unsignedCbor: string): Promise<string> {
    // Re-enable wallet to get fresh session — Lace loses session after navigation
    if (walletId.value) {
      const cardano = (window as any).cardano
      if (cardano?.[walletId.value]) {
        try {
          walletApi.value  = await cardano[walletId.value].enable()
          isRestored.value = true
        } catch (e) {
          console.error('Failed to re-enable wallet:', e)
        }
      }
    }

    if (!walletApi.value) {
      throw new Error('Wallet not connected. Please connect your wallet again.')
    }

    try {
      // partialSign = true: wallet adds its witness, script witness already present
      const signedCbor = await walletApi.value.signTx(unsignedCbor, true)
      return signedCbor
    } catch (e: any) {
      // CIP-30 errors are objects like { code: 2, info: "User declined" }
      const msg = e?.info || e?.message || JSON.stringify(e) || 'Wallet signing failed'
      throw new Error(msg)
    }
  }

  // ── Get all UTxOs from wallet (HD wallet aware) ───────────────────────────────
  // CIP-30 getUtxos() returns UTxOs from ALL derived addresses in the HD wallet.
  // This is needed because Lace spreads funds across multiple derived addresses.
  async function getUtxos(): Promise<string[]> {
    // Re-enable wallet to get fresh session — same as signOnly
    // Lace loses its internal session after page navigation
    if (walletId.value) {
      const cardano = (window as any).cardano
      if (cardano?.[walletId.value]) {
        try {
          walletApi.value  = await cardano[walletId.value].enable()
          isRestored.value = true
        } catch (e) {
          console.error('Failed to re-enable wallet for getUtxos:', e)
        }
      }
    }

    if (!walletApi.value) {
      throw new Error('Wallet not connected. Please connect your wallet again.')
    }

    try {
      const utxos = await walletApi.value.getUtxos()
      return utxos || []
    } catch (e: any) {
      const msg = e?.info || e?.message || JSON.stringify(e) || 'Failed to get wallet UTxOs'
      throw new Error(msg)
    }
  }

  // ── Get wallet address ────────────────────────────────────────────────────────
  async function getAddress(): Promise<string | null> {
    if (!walletApi.value) {
      const restored = await restoreWallet()
      if (!restored) return null
    }
    try {
      const addresses = await walletApi.value.getUsedAddresses()
      return addresses[0] || await walletApi.value.getChangeAddress()
    } catch {
      return null
    }
  }

  // ── Clear on logout ───────────────────────────────────────────────────────────
  function clearWallet() {
    walletApi.value  = null
    walletId.value   = null
    isRestored.value = false
    localStorage.removeItem('lankanft_wallet_id')
  }

  return {
    walletApi,
    walletId,
    isRestored,
    setWallet,
    restoreWallet,
    signOnly,
    getUtxos,
    getAddress,
    clearWallet,
  }
}