import api from './api'

// Create batch job and upload all files to IPFS
export async function prepareBatch(formData: FormData) {
  const res = await api.post('/api/batch/prepare', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data
}

// Mint all uploaded NFTs in the batch
export async function mintBatch(batchId: string) {
  const res = await api.post('/api/batch/mint', { batch_id: batchId })
  return res.data
}

// Get batch job status
export async function getBatchStatus(batchId: string) {
  const res = await api.get(`/api/batch/${batchId}`)
  return res.data
}