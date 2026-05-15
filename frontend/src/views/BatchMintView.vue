<template>
  <div class="batch-mint">
    <div class="page-header">
      <router-link to="/mint" class="back-link">← Back to Create Mint</router-link>
      <h1 class="page-title">Batch Upload</h1>
      <p class="page-sub">Add or edit your NFT details below. Each row is one NFT.</p>
    </div>

    <!-- ── Step Indicator ─────────────────────────────────────────── -->
    <div class="steps">
      <div
        v-for="(step, i) in steps"
        :key="i"
        :class="['step', { active: currentStep === i, done: currentStep > i }]"
      >
        <div class="step-circle">
          <CheckCircle v-if="currentStep > i" :size="16" color="#fff" />
          <span v-else>{{ i + 1 }}</span>
        </div>
        <div class="step-label">{{ step.label }}</div>
        <div v-if="i < steps.length - 1" class="step-line" />
      </div>
    </div>

    <!-- ── Step 1: Edit Details ───────────────────────────────────── -->
    <div v-if="currentStep === 0" class="step-content">
      <BatchTable ref="tableRef" @change="onTableChange" />

      <!-- Cost estimate — shown dynamically as user adds NFTs to the table -->
      <!-- Updates in real time so users know how much ADA they need before minting -->
      <div v-if="validRows.length > 0" class="cost-banner">
        <div class="cost-banner-header">
          <Coins :size="16" color="#534AB7" />
          <span>Estimated Minting Cost for {{ validRows.length }} NFT{{ validRows.length > 1 ? 's' : '' }}</span>
        </div>
        <div class="cost-rows">
          <!-- Each NFT creates 2 UTxOs on-chain: (100) reference + (222) user token -->
          <!-- Cardano requires minimum ADA locked in each UTxO (~2 ADA per NFT pair) -->
          <div class="cost-row">
            <span class="cost-label">Min-ADA ({{ validRows.length }} × ~2 ADA)</span>
            <span class="cost-val">~{{ validRows.length * 2 }} ADA</span>
          </div>
          <!-- Plus 1 royalty (500) token UTxO shared across the whole collection -->
          <div class="cost-row">
            <span class="cost-label">Royalty token UTxO</span>
            <span class="cost-val">~2 ADA</span>
          </div>
          <!-- Network fee grows slightly with each NFT due to tx size -->
          <div class="cost-row">
            <span class="cost-label">Network fee</span>
            <span class="cost-val">~{{ networkFeeEstimate }} ADA</span>
          </div>
          <!-- IPFS pinning via Pinata is handled server-side, no cost to user -->
          <div class="cost-row">
            <span class="cost-label">IPFS pinning (Pinata)</span>
            <span class="cost-val cost-free">Included</span>
          </div>
          <div class="cost-divider" />
          <div class="cost-row cost-row--total">
            <span class="cost-label">Total estimate</span>
            <span class="cost-total">~{{ totalCostEstimate }} ADA</span>
          </div>
        </div>
        <p class="cost-note">
          Ensure your wallet has at least <strong>{{ minRequired }} ADA</strong> before minting.
          Unused ADA is returned after the transaction.
        </p>
      </div>

      <div class="step-footer">
        <div class="privacy-row">
          <label class="privacy-label">Privacy:</label>
          <select v-model="privacy" class="privacy-select">
            <option value="public">Public</option>
            <option value="private">Private</option>
          </select>
        </div>
        <BaseButton variant="primary" :disabled="!hasValidRows" @click="currentStep = 1">
          Next: Upload Assets →
        </BaseButton>
      </div>
    </div>

    <!-- ── Step 2: Upload to IPFS ─────────────────────────────────── -->
    <div v-if="currentStep === 1" class="step-content">
      <div class="upload-summary">
        <h3>Ready to upload {{ validRows.length }} NFTs to IPFS</h3>
        <p>Images and metadata will be pinned to IPFS via Pinata.</p>
        <div class="summary-list">
          <div v-for="(row, i) in validRows" :key="i" class="summary-item">
            <img v-if="row.previewUrl" :src="row.previewUrl" class="summary-img" />
            <div class="summary-no-img" v-else>
              <ImageIcon :size="18" color="#ccc" />
            </div>
            <div class="summary-info">
              <div class="summary-name">{{ row.name }}</div>
              <div class="summary-meta">{{ row.royalties }}% royalty · Supply: {{ row.supply }}</div>
            </div>
            <div :class="['summary-status', uploadStatuses[i]]">
              {{ uploadStatuses[i] || 'pending' }}
            </div>
          </div>
        </div>
      </div>
      <p v-if="batch.error" class="error-msg">{{ batch.error }}</p>
      <div class="step-footer">
        <button class="back-btn" @click="currentStep = 0">← Back</button>
        <BaseButton variant="primary" :loading="batch.isLoading" @click="handleUpload">
          Upload to IPFS
        </BaseButton>
      </div>
    </div>

    <!-- ── Step 3: Mint on Blockchain ────────────────────────────── -->
    <div v-if="currentStep === 2" class="step-content">
      <div class="mint-summary">
        <h3>{{ batch.progress.uploaded }} NFTs uploaded successfully</h3>
        <p>Ready to mint on Cardano Preprod. All NFTs mint in one single transaction.</p>
        <div class="progress-bar-wrap">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: mintProgress + '%' }" />
          </div>
          <span class="progress-label">
            {{ batch.progress.minted }} / {{ batch.progress.uploaded }} minted
          </span>
        </div>
      </div>
      <p v-if="batch.error" class="error-msg">{{ batch.error }}</p>
      <div class="step-footer">
        <BaseButton variant="primary" :loading="batch.isLoading" @click="handleMint">
          <Rocket :size="14" /> Mint All on Cardano
        </BaseButton>
      </div>
    </div>

    <!-- ── Step 4: Complete ───────────────────────────────────────── -->
    <div v-if="currentStep === 3" class="step-content">
      <div class="success-card">
        <div class="success-icon">
          <PartyPopper :size="48" color="#534AB7" />
        </div>
        <h2>Batch Complete!</h2>
        <p>
          <strong>{{ batch.progress.minted }}</strong> NFTs minted successfully.
          <span v-if="batch.progress.failed > 0" class="failed-note">
            {{ batch.progress.failed }} failed.
          </span>
        </p>
        <a
          v-if="mintedTxHash"
          :href="`https://preprod.cardanoscan.io/transaction/${mintedTxHash}`"
          target="_blank"
          rel="noopener noreferrer"
          class="tx-link"
        >
          View on Cardanoscan →
        </a>
        <div class="success-actions">
          <router-link to="/">
            <BaseButton variant="primary">Go to Dashboard</BaseButton>
          </router-link>
          <BaseButton variant="outline" @click="resetBatch">Mint Another Batch</BaseButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  CheckCircle,
  Coins,
  Image as ImageIcon,
  PartyPopper,
  Rocket,
} from 'lucide-vue-next'
import { useBatchStore } from '@/stores/batch'
import BatchTable from '@/components/mint/BatchTable.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import type { BatchRow } from '@/components/mint/BatchTable.vue'

const batch = useBatchStore()
const tableRef = ref<InstanceType<typeof BatchTable> | null>(null)
const currentStep = ref(0)
const privacy = ref<'public' | 'private'>('public')
const rows = ref<BatchRow[]>([])
const uploadStatuses = ref<string[]>([])
const mintedTxHash = ref('')

const steps = [
  { label: 'Edit Details' },
  { label: 'Upload Assets' },
  { label: 'Mint on Chain' },
  { label: 'Complete' },
]

function onTableChange(updatedRows: BatchRow[]) { rows.value = updatedRows }

// Only rows with both a name AND a file are valid for minting
const validRows = computed(() => rows.value.filter((r) => r.name.trim() && r.file))
const hasValidRows = computed(() => validRows.value.length > 0)

// ── Cost estimate calculations ────────────────────────────────────────────
// Network fee grows with tx size: base 0.5 ADA + 0.1 ADA per NFT (approx)
const networkFeeEstimate = computed(() =>
  (0.5 + validRows.value.length * 0.1).toFixed(1)
)

// Total = (2 ADA per NFT × count) + 2 ADA royalty UTxO + network fee
const totalCostEstimate = computed(() => {
  const minAda = validRows.value.length * 2 + 2
  const fee = parseFloat(networkFeeEstimate.value)
  return (minAda + fee).toFixed(1)
})

// Add 1 ADA buffer so users don't run out mid-transaction
const minRequired = computed(() =>
  (parseFloat(totalCostEstimate.value) + 1).toFixed(0)
)

const mintProgress = computed(() => {
  if (batch.progress.uploaded === 0) return 0
  return Math.round((batch.progress.minted / batch.progress.uploaded) * 100)
})

// Upload all NFT images and metadata to IPFS via the backend
async function handleUpload() {
  const formData = new FormData()
  formData.append('privacy', privacy.value)

  validRows.value.forEach((row) => {
    formData.append('files[]', row.file!)
    formData.append('names[]', row.name)
    formData.append('descriptions[]', row.description)
    formData.append('royalties[]', String(row.royalties))
    formData.append('total_supplies[]', String(row.supply))
  })

  uploadStatuses.value = validRows.value.map(() => 'uploading')

  const result = await batch.prepare(formData)
  if (!result) return

  // Update per-NFT upload statuses from the server response
  result.items?.forEach((item: any, i: number) => {
    uploadStatuses.value[i] = item.status
  })

  if (batch.progress.uploaded > 0) currentStep.value = 2
}

// Submit all NFTs in one Cardano transaction
async function handleMint() {
  if (!batch.batchId) return
  const result = await batch.mint(batch.batchId)
  if (result) {
    mintedTxHash.value = result.tx_hash || ''
    currentStep.value = 3
  }
}

// Reset everything for another batch
function resetBatch() {
  batch.reset()
  currentStep.value = 0
  rows.value = []
  uploadStatuses.value = []
  mintedTxHash.value = ''
}
</script>

<style scoped>
.batch-mint { max-width: 900px; margin: 0 auto; padding: 32px; }

.back-link {
  font-size: 13px; color: #666; display: block;
  margin-bottom: 16px; text-decoration: none;
}
.back-link:hover { color: #534AB7; }
.page-title { font-size: 28px; font-weight: 700; margin-bottom: 4px; }
.page-sub { font-size: 13px; color: #666; margin-bottom: 28px; }

/* ── Steps ───────────────────────────────────────────────── */
.steps { display: flex; align-items: center; margin-bottom: 32px; }
.step { display: flex; align-items: center; gap: 8px; flex: 1; }
.step-circle {
  width: 32px; height: 32px; border-radius: 50%;
  border: 2px solid #e0e0e0; display: flex;
  align-items: center; justify-content: center;
  font-size: 13px; font-weight: 500; color: #aaa;
  background: #fff; flex-shrink: 0;
}
.step.active .step-circle { border-color: #534AB7; background: #534AB7; color: #fff; }
.step.done .step-circle { border-color: #1D9E75; background: #1D9E75; color: #fff; }
.step-label { font-size: 13px; color: #666; white-space: nowrap; }
.step.active .step-label { color: #534AB7; font-weight: 500; }
.step.done .step-label { color: #1D9E75; }
.step-line { flex: 1; height: 1px; background: #e0e0e0; margin: 0 8px; }

/* ── Step Content ────────────────────────────────────────── */
.step-content { margin-top: 8px; }
.step-footer {
  display: flex; justify-content: flex-end; align-items: center;
  gap: 16px; margin-top: 24px; padding-top: 16px; border-top: 1px solid #f0f0f0;
}
.back-btn { background: none; border: none; font-size: 13px; color: #666; cursor: pointer; }
.back-btn:hover { color: #534AB7; }
.privacy-row { display: flex; align-items: center; gap: 8px; margin-right: auto; }
.privacy-label { font-size: 13px; color: #444; }
.privacy-select {
  padding: 6px 10px; border: 1.5px solid #e0e0e0;
  border-radius: 8px; font-size: 13px;
}

/* ── Cost Banner ─────────────────────────────────────────── */
.cost-banner {
  background: #FAFAFA; border: 1.5px solid #e8e8e8;
  border-radius: 12px; padding: 16px; margin-top: 16px;
}
.cost-banner-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600; color: #534AB7; margin-bottom: 12px;
}
.cost-rows { display: flex; flex-direction: column; gap: 6px; }
.cost-row {
  display: flex; justify-content: space-between;
  font-size: 13px; color: #555;
}
.cost-row--total { margin-top: 4px; }
.cost-label { color: #666; }
.cost-val { font-weight: 500; color: #333; }
.cost-free { color: #085041; font-weight: 500; }
.cost-divider { height: 1px; background: #e0e0e0; margin: 6px 0; }
.cost-total { font-size: 15px; font-weight: 700; color: #534AB7; }
.cost-note { font-size: 12px; color: #888; margin: 12px 0 0; line-height: 1.5; }
.cost-note strong { color: #333; }

.error-msg { font-size: 13px; color: #d32f2f; margin-top: 12px; }

/* ── Upload Summary ──────────────────────────────────────── */
.upload-summary h3 { font-size: 16px; font-weight: 600; margin-bottom: 4px; }
.upload-summary p { font-size: 13px; color: #666; margin-bottom: 16px; }
.summary-list { display: flex; flex-direction: column; gap: 8px; }
.summary-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; border: 1px solid #f0f0f0; border-radius: 8px;
}
.summary-img { width: 40px; height: 40px; border-radius: 6px; object-fit: cover; }
.summary-no-img {
  width: 40px; height: 40px; border-radius: 6px; background: #f0f0f0;
  display: flex; align-items: center; justify-content: center;
}
.summary-info { flex: 1; }
.summary-name { font-size: 14px; font-weight: 500; }
.summary-meta { font-size: 12px; color: #888; }
.summary-status {
  font-size: 12px; padding: 2px 8px;
  border-radius: 20px; background: #f0f0f0; color: #666;
}
.summary-status.uploaded { background: #E1F5EE; color: #085041; }
.summary-status.failed { background: #FCEBEB; color: #791F1F; }

/* ── Mint Progress ───────────────────────────────────────── */
.mint-summary h3 { font-size: 16px; font-weight: 600; margin-bottom: 4px; }
.mint-summary p { font-size: 13px; color: #666; margin-bottom: 20px; }
.progress-bar-wrap { display: flex; align-items: center; gap: 12px; }
.progress-bar {
  flex: 1; height: 8px; background: #f0f0f0;
  border-radius: 4px; overflow: hidden;
}
.progress-fill {
  height: 100%; background: #534AB7;
  border-radius: 4px; transition: width 0.3s;
}
.progress-label { font-size: 13px; color: #666; white-space: nowrap; }

/* ── Success Card ────────────────────────────────────────── */
.success-card {
  text-align: center; padding: 48px;
  border: 1px solid #e0e0e0; border-radius: 16px;
}
.success-icon { margin-bottom: 16px; display: flex; justify-content: center; }
.success-card h2 { font-size: 24px; font-weight: 700; margin-bottom: 8px; }
.success-card p { font-size: 14px; color: #666; margin-bottom: 24px; }
.failed-note { color: #d32f2f; }
.tx-link {
  color: #534AB7; font-size: 13px; text-decoration: underline;
  display: inline-block; margin-bottom: 24px;
}
.success-actions { display: flex; gap: 12px; justify-content: center; }
</style>