import { defineStore } from 'pinia'
import { ref } from 'vue'
import { prepareBatch, mintBatch, getBatchStatus } from '@/services/batch'

export const useBatchStore = defineStore('batch', () => {
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const batchId = ref<string | null>(null)
  const progress = ref({ total: 0, uploaded: 0, minted: 0, failed: 0 })

  // Step 1 — upload all files to IPFS
  async function prepare(formData: FormData) {
    isLoading.value = true
    error.value = null
    try {
      const result = await prepareBatch(formData)
      batchId.value = result.batch_id
      progress.value.total = result.total
      progress.value.uploaded = result.uploaded
      return result
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to prepare batch'
      return null
    } finally {
      isLoading.value = false
    }
  }

  // Step 2 — mint all on blockchain
  async function mint(id: string) {
    isLoading.value = true
    error.value = null
    try {
      const result = await mintBatch(id)
      progress.value.minted = result.minted
      progress.value.failed = result.failed
      return result
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to mint batch'
      return null
    } finally {
      isLoading.value = false
    }
  }

  function reset() {
    batchId.value = null
    error.value = null
    progress.value = { total: 0, uploaded: 0, minted: 0, failed: 0 }
  }

  return { isLoading, error, batchId, progress, prepare, mint, reset }
})