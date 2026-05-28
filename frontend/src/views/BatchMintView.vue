<template>
  <div class="batch-mint">
    <router-link to="/mint" class="back-link" title="Back to Create Mint">
      <ArrowLeft :size="16" />
    </router-link>

    <div class="page-header">
      <div>
        <h1 class="page-title">Batch Upload</h1>
        <p class="page-sub">Mint multiple NFTs in one transaction.</p>
      </div>
      <div class="upload-toggle">
        <router-link to="/mint">
          <button class="toggle-btn">Single Upload</button>
        </router-link>
        <button class="toggle-btn toggle-btn--active">Batch Upload</button>
      </div>
    </div>

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

      <div class="step-panel">
        <div class="step-panel-title">Steps</div>
        <div class="steps">
          <div v-for="(step, i) in steps" :key="i" :class="['step', stepClass(i)]">
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

        <div v-if="validRows.length > 0" class="cost-box">
          <div class="cost-box-title"><Coins :size="13" /> Estimated cost</div>
          <div class="cost-row">
            <span>Min-ADA ({{ validRows.length }} × ~2 ADA)</span>
            <span>~{{ validRows.length * 2 }} ADA</span>
          </div>
          <div class="cost-row"><span>Royalty UTxO</span><span>~2 ADA</span></div>
          <div class="cost-row"><span>Network fee</span><span>~{{ networkFeeEstimate }} ADA</span></div>
          <div class="cost-row"><span>Storage</span><span class="cost-free">Included</span></div>
          <div class="cost-divider" />
          <div class="cost-row cost-row--total">
            <span>Total</span><strong>~{{ totalCostEstimate }} ADA</strong>
          </div>
          <p class="cost-note">Keep at least <strong>{{ minRequired }} ADA</strong> in wallet.</p>
        </div>
      </div>

      <div class="form-content">

        <div v-if="currentStep === 0">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">NFT Details</div>
              
              <div class="privacy-wrap">
                <span class="privacy-label-off" :class="{ 'privacy-label--active': privacy === 'public' }">Public</span>
                <button
                  :class="['privacy-toggle', `privacy-toggle--${privacy}`]"
                  @click="privacy = privacy === 'public' ? 'private' : 'public'"
                  :title="privacy === 'public' ? 'Click to make Private' : 'Click to make Public'"
                >
                  <span class="privacy-toggle-thumb" />
                </button>
                <span class="privacy-label-on" :class="{ 'privacy-label--active': privacy === 'private' }">Private</span>
              </div>
              
            </div>
            <BatchTable ref="tableRef" @change="onTableChange" />
          </div>
          <div class="form-footer">
            <button class="btn-next" :disabled="!hasValidRows" @click="goToPreview">
              Preview & Mint
              <ArrowRight :size="14" />
            </button>
          </div>
        </div>

        <div v-if="currentStep === 1">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">
                Review your {{ validRows.length }} NFT{{ validRows.length > 1 ? 's' : '' }}
              </div>
              <div v-if="isPreparing" class="preparing-badge">
                <Loader2 :size="12" class="spin" /> Preparing...
              </div>
              <div v-else-if="prepareError" class="error-badge">
                <AlertCircle :size="12" /> Preparation failed
              </div>
              <div v-else class="ready-badge">
                <CheckCircle :size="12" /> Ready to mint
              </div>
            </div>

            <div class="preview-grid">
              <div v-for="(row, i) in validRows" :key="i" class="preview-card">
                <div class="preview-img-wrap">
                  <img v-if="row.previewUrl" :src="row.previewUrl" class="preview-img" />
                  <div v-else class="preview-img-empty">
                    <ImageIcon :size="24" color="#ccc" />
                  </div>
                  <div v-if="uploadStatuses[i] === 'uploaded'" class="preview-check">
                    <CheckCircle :size="14" color="#fff" />
                  </div>
                </div>
                <div class="preview-info">
                  <div class="preview-name">{{ row.name }}</div>
                  <div class="preview-meta">
                    <span>{{ row.royalties }}% royalty</span>
                    <span class="preview-dot">·</span>
                    <span>Supply {{ row.supply }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="prepareError" class="error-msg">
              <AlertCircle :size="13" />
              {{ prepareError }}
              <button class="retry-link" @click="retryPrepare">Retry</button>
            </div>
          </div>

          <div class="form-footer">
            <button class="btn-back" @click="currentStep = 0">
              <ArrowLeft :size="13" /> Back
            </button>
            <button
              class="btn-mint"
              :disabled="isPreparing || !!prepareError || batch.isLoading"
              @click="handleMint"
            >
              <Loader2 v-if="batch.isLoading" :size="15" class="spin" />
              <Sparkles v-else :size="15" />
              {{ batch.isLoading ? 'Minting...' : `Mint ${validRows.length} NFT${validRows.length > 1 ? 's' : ''} on Cardano` }}
            </button>
          </div>
        </div>

        <div v-if="currentStep === 2">
          <div class="content-card">
            <div class="content-card-header">
              <div class="content-card-title">Minting on Cardano</div>
            </div>
            <p class="upload-desc">
              Your NFTs are being minted in one transaction on Cardano Preprod.
              This usually takes 10–30 seconds.
            </p>
            <div class="progress-wrap">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: mintProgress + '%' }" />
              </div>
              <span class="progress-label">
                {{ batch.progress.minted }} / {{ batch.progress.uploaded }} minted
              </span>
            </div>
            <p v-if="batch.error" class="error-msg">
              <AlertCircle :size="13" /> {{ batch.error }}
            </p>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// BatchMintView.vue
//
// Batch NFT minting flow. Three steps from the user's perspective:
//   1. Edit Details  — fill name, description, royalties per NFT
//   2. Preview       — see all NFTs before minting. Asset preparation
//                      happens silently in the background here so the
//                      user never waits at a technical "upload" step.
//   3. Complete      — success screen with Cardanoscan link
//
// The IPFS preparation is intentionally hidden from the user — they see
// "Preparing..." for a moment, then "Ready to mint". This keeps the UX
// focused on the creative act of minting, not the technical pipeline.
// ─────────────────────────────────────────────────────────────────────────────

import { computed, ref } from 'vue'
import {
  AlertCircle, ArrowLeft, ArrowRight,
  Check, CheckCircle, Coins, ExternalLink,
  Image as ImageIcon, Loader2, Sparkles,
} from 'lucide-vue-next'
import { useBatchStore } from '@/stores/batch'
import BatchTable from '@/components/mint/BatchTable.vue'
import type { BatchRow } from '@/components/mint/BatchTable.vue'

const batch   = useBatchStore()
const tableRef = ref<InstanceType<typeof BatchTable> | null>(null)

const currentStep    = ref(0)
const privacy        = ref<'public' | 'private'>('public')
const rows           = ref<BatchRow[]>([])
const uploadStatuses = ref<string[]>([])
const mintedTxHash   = ref('')

// Background preparation state — hidden from user but drives the UI indicators
const isPreparing  = ref(false)
const prepareError = ref('')

const steps = [
  { label: 'Edit Details', desc: 'Name, description, royalties' },
  { label: 'Preview',      desc: 'Review before minting' },
  { label: 'Minting',      desc: 'One Cardano transaction' },
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

/**
 * Called when user clicks "Preview & Mint" from Edit Details.
 *
 * Moves to the preview step immediately so the user sees their NFTs,
 * then fires the IPFS preparation in the background. By the time the
 * user reads the preview and decides to mint, preparation is usually done.
 */
async function goToPreview() {
  currentStep.value = 1
  await runPrepare()
}

/**
 * Runs the background preparation (IPFS upload) silently.
 * Updates isPreparing and prepareError — these drive the small
 * status indicator in the preview header, not a blocking UI.
 */
async function runPrepare() {
  isPreparing.value  = true
  prepareError.value = ''
  uploadStatuses.value = validRows.value.map(() => 'uploading')

  const formData = new FormData()
  formData.append('privacy', privacy.value)
  validRows.value.forEach((row) => {
    formData.append('files[]', row.file!)
    formData.append('names[]', row.name)
    formData.append('descriptions[]', row.description)
    formData.append('royalties[]', String(row.royalties))
    formData.append('total_supplies[]', String(row.supply))
  })

  const result = await batch.prepare(formData)

  if (!result) {
    prepareError.value = batch.error || 'Preparation failed. Please try again.'
    uploadStatuses.value = validRows.value.map(() => 'failed')
  } else {
    result.items?.forEach((item: any, i: number) => {
      uploadStatuses.value[i] = item.status
    })
  }

  isPreparing.value = false
}

/** Retry preparation after failure — user clicks "Retry" in error message */
async function retryPrepare() {
  prepareError.value = ''
  await runPrepare()
}

/**
 * Mints all prepared NFTs in one Cardano transaction.
 * Moves to minting step (step index 2) while transaction is in progress.
 */
async function handleMint() {
  if (!batch.batchId) return
  currentStep.value = 2
  const result = await batch.mint(batch.batchId)
  if (result) {
    mintedTxHash.value = result.tx_hash || ''
    currentStep.value = 3
  } else {
    // Mint failed — go back to preview so user can retry
    currentStep.value = 1
  }
}

function resetBatch() {
  batch.reset()
  currentStep.value  = 0
  rows.value         = []
  uploadStatuses.value = []
  mintedTxHash.value = ''
  isPreparing.value  = false
  prepareError.value = ''
}
</script>

<style scoped>
.batch-mint { padding: 28px 32px; max-width: 1100px; }

/* Back link — icon only, tooltip on hover */
.back-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  color: #888; text-decoration: none;
  margin-bottom: 20px; transition: all 0.15s;
}
.back-link:hover { background: #EEEDFE; color: #534AB7; }

/* Page header */
.page-header {
  display: flex; justify-content: space-between;
  align-items: flex-start; margin-bottom: 24px;
}
.page-title  { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub    { font-size: 13px; color: #888; }

/* Toggle */
.upload-toggle { display: flex; gap: 6px; }
.toggle-btn {
  padding: 7px 14px; border-radius: 8px;
  border: 1px solid #e8e8e8; background: #fff;
  font-size: 12px; font-weight: 500; cursor: pointer;
  color: #666; transition: all 0.15s;
}
.toggle-btn:hover          { border-color: #534AB7; color: #534AB7; }
.toggle-btn--active        { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }

/* ── Layout ── */
.form-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 24px;
  align-items: start;
}

/* ── Step panel ── */
.step-panel {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 14px; padding: 16px;
  position: sticky; top: 20px;
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

/* Cost box */
.cost-box { background: #f8f8f8; border: 1px solid #f0f0f0; border-radius: 10px; padding: 12px; }
.cost-box-title {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 600; color: #534AB7; margin-bottom: 10px;
}
.cost-row { display: flex; justify-content: space-between; font-size: 11px; color: #666; padding: 2px 0; }
.cost-row--total { font-size: 12px; font-weight: 500; color: #333; margin-top: 2px; }
.cost-row--total strong { color: #534AB7; }
.cost-free    { color: #085041; font-weight: 500; }
.cost-divider { height: 1px; background: #ebebeb; margin: 6px 0; }
.cost-note    { font-size: 10px; color: #aaa; margin-top: 8px; line-height: 1.5; }
.cost-note strong { color: #555; }

/* ── Form content ── */
.form-content { display: flex; flex-direction: column; gap: 12px; }

.content-card {
  background: #fff; border: 1px solid #f0f0f0;
  border-radius: 14px; padding: 20px;
}
.content-card-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 20px;
}
.content-card-title { font-size: 14px; font-weight: 600; color: #111; }

/* Privacy selector */
.privacy-wrap {
  display: flex; align-items: center; gap: 8px;
}
.privacy-label-off,
.privacy-label-on {
  font-size: 12px;
  font-weight: 500;
  color: #bbb;
  transition: color 0.2s;
}
.privacy-label--active {
  color: #534AB7;
  font-weight: 600;
}
.privacy-toggle {
  position: relative;
  width: 44px; height: 24px;
  border-radius: 12px;
  border: none; cursor: pointer;
  padding: 0; transition: background 0.2s ease;
}
.privacy-toggle--public  { background: #534AB7; }
.privacy-toggle--private { background: #1a1a1a; }

.privacy-toggle-thumb {
  position: absolute;
  top: 3px; left: 3px;
  width: 18px; height: 18px;
  border-radius: 50%; background: #fff;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
  transition: transform 0.2s ease;
  display: block;
}
.privacy-toggle--private .privacy-toggle-thumb {
  transform: translateX(20px);
}


/* Preparation status badges — small, unobtrusive */
.preparing-badge {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 500; color: #185FA5;
  background: #E6F1FB; padding: 4px 10px; border-radius: 20px;
}
.ready-badge {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 500; color: #085041;
  background: #E1F5EE; padding: 4px 10px; border-radius: 20px;
}
.error-badge {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 500; color: #791F1F;
  background: #FCEBEB; padding: 4px 10px; border-radius: 20px;
}

/* ── Preview grid — clean NFT cards ── */
.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.preview-card {
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  overflow: hidden;
  transition: border-color 0.15s;
}
.preview-card:hover { border-color: #534AB7; }

.preview-img-wrap {
  aspect-ratio: 1;
  background: #f8f8f8;
  position: relative;
  overflow: hidden;
}
.preview-img {
  width: 100%; height: 100%;
  object-fit: cover; display: block;
}
.preview-img-empty {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}
/* Small check icon overlay when NFT is prepared */
.preview-check {
  position: absolute; bottom: 6px; right: 6px;
  width: 22px; height: 22px; border-radius: 50%;
  background: #085041;
  display: flex; align-items: center; justify-content: center;
}

.preview-info { padding: 10px 10px 12px; }
.preview-name {
  font-size: 12px; font-weight: 600; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-bottom: 4px;
}
.preview-meta {
  display: flex; align-items: center; gap: 4px;
  font-size: 11px; color: #aaa;
}
.preview-dot { color: #ddd; }

/* Upload desc */
.upload-desc { font-size: 13px; color: #666; margin-bottom: 16px; line-height: 1.5; }

/* Progress bar */
.progress-wrap  { display: flex; align-items: center; gap: 12px; margin-top: 8px; }
.progress-bar   { flex: 1; height: 8px; background: #f0f0f0; border-radius: 4px; overflow: hidden; }
.progress-fill  { height: 100%; background: #534AB7; border-radius: 4px; transition: width 0.4s; }
.progress-label { font-size: 12px; color: #888; white-space: nowrap; }

/* Error message */
.error-msg {
  display: flex; align-items: center; gap: 6px;
  margin-top: 16px; padding: 10px 14px;
  background: #FCEBEB; border-radius: 8px;
  font-size: 12px; color: #791F1F;
}
.retry-link {
  margin-left: auto; background: none; border: none;
  color: #534AB7; font-size: 12px; font-weight: 600;
  cursor: pointer; text-decoration: underline;
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
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 11px 24px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-mint:hover:not(:disabled) { background: #3d35a0; }
.btn-mint:disabled { opacity: 0.5; cursor: not-allowed; }

/* Success card */
.success-card {
  display: flex; flex-direction: column; align-items: center; gap: 14px;
  padding: 56px 40px; text-align: center;
  background: #fff; border: 1px solid #f0f0f0; border-radius: 16px;
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
  .preview-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>