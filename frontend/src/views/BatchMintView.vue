<template>
  <div class="batch-mint">
    <router-link to="/mint" class="back-link">
      <ArrowLeft :size="14" /> Back to Create Mint
    </router-link>

    <div class="page-header">
      <div>
        <h1 class="page-title">Batch Upload</h1>
        <p class="page-sub">Mint multiple NFTs in one transaction.</p>
      </div>
    </div>

    <!-- ── Step 4: Complete ── -->
    <div v-if="currentStep === 3" class="success-card">
      <div class="success-icon-wrap">
        <CheckCircle :size="40" color="#085041" />
      </div>
      <h2 class="success-title">Batch Complete!</h2>
      <p class="success-desc">
        <strong>{{ batch.progress.minted }}</strong> NFTs minted successfully on Cardano Preprod.
        <span v-if="batch.progress.failed > 0" class="failed-note">
          · {{ batch.progress.failed }} failed.
        </span>
      </p>
      <a
        v-if="mintedTxHash"
        :href="`https://preprod.cardanoscan.io/transaction/${mintedTxHash}`"
        target="_blank" rel="noopener noreferrer"
        class="btn-scan"
      >
        <ExternalLink :size="13" /> View on Cardanoscan
      </a>
      <button class="btn-ghost" @click="resetBatch">Mint Another Batch</button>
    </div>

    <div v-else class="form-layout">
      <!-- ── Step sidebar ── -->
      <div class="step-panel">
        <div class="step-panel-title">Steps</div>
        <div class="steps">
          <div
            v-for="(step, i) in steps"
            :key="i"
            :class="['step', stepClass(i)]"
          >
            <div class="step-num">
              <Check v-if="currentStep > i" :size="11" />
              <span v-else>{{ i + 1 }}</span>
            </div>
            <div>
              <div class="step-label">{{ step.label }}</div>
              <div class="step-desc">{{ step.desc }}</div>
            </div>
          </div>
        </div>

        <!-- Cost estimate — shown when rows have valid data -->
        <div v-if="validRows.length > 0" class="cost-box">
          <div class="cost-box-title"><Coins :size="13" /> Estimated cost</div>
          <div class="cost-row">
            <span>Min-ADA ({{ validRows.length }} × ~2 ADA)</span>
            <span>~{{ validRows.length * 2 }} ADA</span>
          </div>
          <div class="cost-row">
            <span>Royalty UTxO</span>
            <span>~2 ADA</span>
          </div>
          <div class="cost-row">
            <span>Network fee</span>
            <span>~{{ networkFeeEstimate }} ADA</span>
          </div>
          <div class="cost-row">
            <span>IPFS pinning</span>
            <span class="cost-free">Included</span>
          </div>
          <div class="cost-divider" />
          <div class="cost-row cost-row--total">
            <span>Total</span>
            <strong>~{{ totalCostEstimate }} ADA</strong>
          </div>
          <p class="cost-note">Keep at least <strong>{{ minRequired }} ADA</strong> in wallet.</p>
        </div>
      </div>

      <!-- ── Main content ── -->
      <div class="form-content">

        <!-- Step 1: Edit Details -->
        <div v-if="currentStep === 0">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">NFT Details</div>
              <div class="privacy-wrap">
                <label class="privacy-label">Privacy</label>
                <select v-model="privacy" class="privacy-select">
                  <option value="public">Public</option>
                  <option value="private">Private</option>
                </select>
              </div>
            </div>
            <BatchTable ref="tableRef" @change="onTableChange" />
          </div>
          <div class="form-footer">
            <button
              class="btn-next"
              :disabled="!hasValidRows"
              @click="currentStep = 1"
            >
              Next: Upload Assets
              <ArrowRight :size="14" />
            </button>
          </div>
        </div>

        <!-- Step 2: Upload to IPFS -->
        <div v-if="currentStep === 1">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">Upload {{ validRows.length }} NFT{{ validRows.length > 1 ? 's' : '' }} to IPFS</div>
            </div>
            <p class="upload-desc">Images and metadata will be pinned to IPFS via Pinata before minting.</p>
            <div class="summary-list">
              <div v-for="(row, i) in validRows" :key="i" class="summary-item">
                <div class="summary-img-wrap">
                  <img v-if="row.previewUrl" :src="row.previewUrl" class="summary-img" />
                  <div v-else class="summary-no-img"><ImageIcon :size="18" color="#ccc" /></div>
                </div>
                <div class="summary-info">
                  <div class="summary-name">{{ row.name }}</div>
                  <div class="summary-meta">{{ row.royalties }}% royalty · Supply {{ row.supply }}</div>
                </div>
                <span :class="['summary-status', `summary-status--${uploadStatuses[i] || 'pending'}`]">
                  {{ uploadStatuses[i] || 'pending' }}
                </span>
              </div>
            </div>
            <p v-if="batch.error" class="error-msg">
              <AlertCircle :size="13" /> {{ batch.error }}
            </p>
          </div>
          <div class="form-footer">
            <button class="btn-back" @click="currentStep = 0">
              <ArrowLeft :size="13" /> Back
            </button>
            <button
              class="btn-next"
              :disabled="batch.isLoading"
              @click="handleUpload"
            >
              <Loader2 v-if="batch.isLoading" :size="14" class="spin" />
              <CloudUpload v-else :size="14" />
              {{ batch.isLoading ? 'Uploading...' : 'Upload to IPFS' }}
            </button>
          </div>
        </div>

        <!-- Step 3: Mint on Chain -->
        <div v-if="currentStep === 2">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">Mint on Cardano</div>
            </div>
            <p class="upload-desc">
              {{ batch.progress.uploaded }} NFTs uploaded. Ready to mint in one transaction on Cardano Preprod.
            </p>
            <div class="progress-wrap">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: mintProgress + '%' }" />
              </div>
              <span class="progress-label">{{ batch.progress.minted }} / {{ batch.progress.uploaded }} minted</span>
            </div>
            <p v-if="batch.error" class="error-msg">
              <AlertCircle :size="13" /> {{ batch.error }}
            </p>
          </div>
          <div class="form-footer">
            <button
              class="btn-mint"
              :disabled="batch.isLoading"
              @click="handleMint"
            >
              <Loader2 v-if="batch.isLoading" :size="15" class="spin" />
              <Sparkles v-else :size="15" />
              {{ batch.isLoading ? 'Minting...' : 'Mint All on Cardano' }}
            </button>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  AlertCircle, ArrowLeft, ArrowRight,
  Check, CheckCircle, CloudUpload,
  Coins, ExternalLink, Image as ImageIcon,
  Loader2, Sparkles,
} from 'lucide-vue-next'
import { useBatchStore } from '@/stores/batch'
import BatchTable from '@/components/mint/BatchTable.vue'
import type { BatchRow } from '@/components/mint/BatchTable.vue'

const batch = useBatchStore()
const tableRef = ref<InstanceType<typeof BatchTable> | null>(null)

const currentStep     = ref(0)
const privacy         = ref<'public' | 'private'>('public')
const rows            = ref<BatchRow[]>([])
const uploadStatuses  = ref<string[]>([])
const mintedTxHash    = ref('')

const steps = [
  { label: 'Edit Details',   desc: 'Name, description, royalties' },
  { label: 'Upload Assets',  desc: 'Images + metadata to IPFS' },
  { label: 'Mint on Chain',  desc: 'One Cardano transaction' },
]

function stepClass(i: number) {
  if (i < currentStep.value) return 'step--done'
  if (i === currentStep.value) return 'step--active'
  return 'step--pending'
}

function onTableChange(updatedRows: BatchRow[]) {
  rows.value = updatedRows
}

const validRows    = computed(() => rows.value.filter((r) => r.name.trim() && r.file))
const hasValidRows = computed(() => validRows.value.length > 0)

const networkFeeEstimate = computed(() =>
  (0.5 + validRows.value.length * 0.1).toFixed(1)
)
const totalCostEstimate = computed(() => {
  const minAda = validRows.value.length * 2 + 2
  return (minAda + parseFloat(networkFeeEstimate.value)).toFixed(1)
})
const minRequired = computed(() =>
  (parseFloat(totalCostEstimate.value) + 1).toFixed(0)
)
const mintProgress = computed(() => {
  if (batch.progress.uploaded === 0) return 0
  return Math.round((batch.progress.minted / batch.progress.uploaded) * 100)
})

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
  result.items?.forEach((item: any, i: number) => {
    uploadStatuses.value[i] = item.status
  })
  if (batch.progress.uploaded > 0) currentStep.value = 2
}

async function handleMint() {
  if (!batch.batchId) return
  const result = await batch.mint(batch.batchId)
  if (result) {
    mintedTxHash.value = result.tx_hash || ''
    currentStep.value = 3
  }
}

function resetBatch() {
  batch.reset()
  currentStep.value = 0
  rows.value = []
  uploadStatuses.value = []
  mintedTxHash.value = ''
}
</script>

<style scoped>
.batch-mint { padding: 28px 32px; max-width: 1100px; }

.back-link {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 13px; color: #888; text-decoration: none;
  margin-bottom: 20px; transition: color 0.15s;
}
.back-link:hover { color: #534AB7; }

.page-header { margin-bottom: 24px; }
.page-title  { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub    { font-size: 13px; color: #888; }

/* ── Layout ── */
.form-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 24px;
  align-items: start;
}

/* ── Step panel (mirrors CreateMintView) ── */
.step-panel {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  padding: 16px;
  position: sticky;
  top: 20px;
}
.step-panel-title {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: #aaa; margin-bottom: 12px;
}
.steps { display: flex; flex-direction: column; gap: 4px; margin-bottom: 20px; }

.step {
  display: flex; align-items: flex-start; gap: 10px;
  padding: 8px; border-radius: 8px; transition: background 0.15s;
}
.step--active  { background: #EEEDFE; }
.step--done    { opacity: 0.7; }
.step--pending { opacity: 0.45; }

.step-num {
  width: 20px; height: 20px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 10px; font-weight: 700; flex-shrink: 0;
  background: #e0e0e0; color: #666;
}
.step--active .step-num { background: #534AB7; color: #fff; }
.step--done   .step-num { background: #085041; color: #fff; }

.step-label { font-size: 12px; font-weight: 500; color: #333; }
.step-desc  { font-size: 10px; color: #aaa; margin-top: 1px; }

/* Cost box (mirrors CreateMintView) */
.cost-box { background: #f8f8f8; border: 1px solid #f0f0f0; border-radius: 10px; padding: 12px; }
.cost-box-title {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 600; color: #534AB7; margin-bottom: 10px;
}
.cost-row {
  display: flex; justify-content: space-between;
  font-size: 11px; color: #666; padding: 2px 0;
}
.cost-row--total { font-size: 12px; font-weight: 500; color: #333; margin-top: 2px; }
.cost-row--total strong { color: #534AB7; }
.cost-free   { color: #085041; font-weight: 500; }
.cost-divider { height: 1px; background: #ebebeb; margin: 6px 0; }
.cost-note   { font-size: 10px; color: #aaa; margin-top: 8px; line-height: 1.5; }
.cost-note strong { color: #555; }

/* ── Form content ── */
.form-content { display: flex; flex-direction: column; gap: 12px; }

.content-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  padding: 20px;
}
.content-card-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 16px;
}
.content-card-title { font-size: 14px; font-weight: 600; color: #111; }

.privacy-wrap { display: flex; align-items: center; gap: 8px; }
.privacy-label { font-size: 12px; color: #666; }
.privacy-select {
  padding: 5px 10px; border: 1px solid #e8e8e8;
  border-radius: 8px; font-size: 12px; outline: none; cursor: pointer;
}
.privacy-select:focus { border-color: #534AB7; }

.upload-desc { font-size: 13px; color: #666; margin-bottom: 16px; line-height: 1.5; }

/* Upload summary list */
.summary-list { display: flex; flex-direction: column; gap: 8px; }
.summary-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px;
  border: 1px solid #f0f0f0; border-radius: 10px;
}
.summary-img-wrap { flex-shrink: 0; }
.summary-img {
  width: 44px; height: 44px;
  border-radius: 8px; object-fit: cover;
}
.summary-no-img {
  width: 44px; height: 44px; border-radius: 8px;
  background: #f5f5f5;
  display: flex; align-items: center; justify-content: center;
}
.summary-info { flex: 1; min-width: 0; }
.summary-name { font-size: 13px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.summary-meta { font-size: 11px; color: #aaa; margin-top: 2px; }
.summary-status {
  font-size: 11px; font-weight: 500;
  padding: 3px 10px; border-radius: 20px;
  flex-shrink: 0;
}
.summary-status--pending   { background: #f0f0f0; color: #888; }
.summary-status--uploading { background: #E6F1FB; color: #185FA5; }
.summary-status--uploaded  { background: #E1F5EE; color: #085041; }
.summary-status--failed    { background: #FCEBEB; color: #791F1F; }

/* Progress bar */
.progress-wrap  { display: flex; align-items: center; gap: 12px; margin-top: 8px; }
.progress-bar   { flex: 1; height: 8px; background: #f0f0f0; border-radius: 4px; overflow: hidden; }
.progress-fill  { height: 100%; background: #534AB7; border-radius: 4px; transition: width 0.4s; }
.progress-label { font-size: 12px; color: #888; white-space: nowrap; }

/* Error */
.error-msg {
  display: flex; align-items: center; gap: 6px;
  margin-top: 12px; padding: 10px 14px;
  background: #FCEBEB; border-radius: 8px;
  font-size: 12px; color: #791F1F;
}

/* Footer actions */
.form-footer {
  display: flex; align-items: center; justify-content: flex-end;
  gap: 10px; padding-top: 4px;
}

.btn-back {
  display: flex; align-items: center; gap: 5px;
  padding: 9px 16px; background: #fff;
  border: 1px solid #e8e8e8; border-radius: 10px;
  font-size: 13px; color: #666; cursor: pointer; transition: all 0.15s;
}
.btn-back:hover { border-color: #534AB7; color: #534AB7; }

.btn-next {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 20px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-next:hover:not(:disabled) { background: #3d35a0; }
.btn-next:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-mint {
  display: flex; align-items: center; justify-content: center;
  gap: 8px; width: 100%; padding: 13px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 14px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-mint:hover:not(:disabled) { background: #3d35a0; }
.btn-mint:disabled { opacity: 0.5; cursor: not-allowed; }

/* ── Success card (mirrors CreateMintView) ── */
.success-card {
  display: flex; flex-direction: column;
  align-items: center; gap: 14px;
  padding: 56px 40px; text-align: center;
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 16px;
}
.success-icon-wrap {
  width: 72px; height: 72px; border-radius: 50%;
  background: #E1F5EE;
  display: flex; align-items: center; justify-content: center;
}
.success-title { font-size: 22px; font-weight: 600; color: #111; }
.success-desc  { font-size: 14px; color: #888; }
.failed-note   { color: #d32f2f; }

.btn-scan {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 20px; background: #534AB7; color: #fff;
  border-radius: 10px; text-decoration: none;
  font-size: 13px; font-weight: 600; transition: background 0.15s;
}
.btn-scan:hover { background: #3d35a0; }

.btn-ghost {
  padding: 10px 20px; background: transparent;
  border: 1px solid #e8e8e8; border-radius: 10px;
  font-size: 13px; cursor: pointer; color: #666;
}
.btn-ghost:hover { border-color: #534AB7; color: #534AB7; }

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 768px) {
  .batch-mint  { padding: 16px; }
  .form-layout { grid-template-columns: 1fr; }
  .step-panel  { position: static; }
}
</style>