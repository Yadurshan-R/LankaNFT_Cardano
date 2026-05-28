<template>
  <div class="create-mint">

    <router-link to="/" class="back-link" title="Back to Dashboard">
      <ArrowLeft :size="16" />
    </router-link>

    <div class="page-header">
      <div>
        <h1 class="page-title">Create Mint</h1>
        <p class="page-sub">Upload your asset, set details, and mint on Cardano.</p>
      </div>
      <div class="upload-toggle">
        <button :class="['toggle-btn', { 'toggle-btn--active': uploadMode === 'single' }]"
          @click="uploadMode = 'single'">
          Single Upload
        </button>
        <router-link to="/batch">
          <button class="toggle-btn">Batch Upload</button>
        </router-link>
      </div>
    </div>

    <div v-if="mintSuccess" class="success-card">
      <div class="success-icon-wrap">
        <CheckCircle :size="40" color="#085041" />
      </div>
      <h2 class="success-title">NFT Minted Successfully!</h2>
      <p class="success-desc">Your NFT has been minted on Cardano Preprod.</p>
      <a :href="`https://preprod.cardanoscan.io/transaction/${lastTxHash}`"
        target="_blank" rel="noopener noreferrer" class="btn-scan">
        <ExternalLink :size="13" /> View on Cardanoscan
      </a>
      <button class="btn-ghost" @click="resetForm">Mint Another</button>
    </div>

    <div v-else class="form-layout">

      <div class="step-panel">
        <div class="step-panel-title">Steps</div>

        <div class="steps">
          <div :class="['step', stepClass(1)]">
            <div class="step-num">
              <Check v-if="stepDone(1)" :size="11" />
              <span v-else>1</span>
            </div>
            <div>
              <div class="step-label">Upload asset</div>
              <div class="step-desc">PNG, JPG, GIF, MP4</div>
            </div>
          </div>
          <div :class="['step', stepClass(2)]">
            <div class="step-num">
              <Check v-if="stepDone(2)" :size="11" />
              <span v-else>2</span>
            </div>
            <div>
              <div class="step-label">Metadata</div>
              <div class="step-desc">Name, description</div>
            </div>
          </div>
          <div :class="['step', stepClass(3)]">
            <div class="step-num">
              <Check v-if="stepDone(3)" :size="11" />
              <span v-else>3</span>
            </div>
            <div>
              <div class="step-label">Privacy</div>
              <div class="step-desc">Public or private</div>
            </div>
          </div>
          <div :class="['step', stepClass(4)]">
            <div class="step-num">
              <Check v-if="stepDone(4)" :size="11" />
              <span v-else>4</span>
            </div>
            <div>
              <div class="step-label">Strategy</div>
              <div class="step-desc">Standard or lazy</div>
            </div>
          </div>
        </div>

        <div v-if="selectedFile && metadata.name" class="cost-box">
          <div class="cost-box-title">
            <Coins :size="13" /> Estimated cost
          </div>
          <div class="cost-row">
            <span>Min-ADA (3 UTxOs)</span><span>~6 ADA</span>
          </div>
          <div class="cost-row">
            <span>Network fee</span><span>~0.5 ADA</span>
          </div>
          <div class="cost-row">
            <span>IPFS pinning</span><span class="cost-free">Included</span>
          </div>
          <div class="cost-divider" />
          <div class="cost-row cost-row--total">
            <span>Total</span><strong>~6.5 ADA</strong>
          </div>
          <p class="cost-note">Keep at least <strong>7 ADA</strong> in wallet.</p>
        </div>
      </div>

      <div class="form-content">
        <UploadAsset @file-selected="onFileSelected" />
        <MetadataForm @update="onMetadataUpdate" />
        <PrivacySelector v-model="privacy" />
        <MintStrategy v-model="mintStrategy" />

        <div v-if="nftStore.error" class="error-msg">
          <AlertCircle :size="14" />
          {{ friendlyError(nftStore.error) }}
        </div>

        <button
          class="btn-mint"
          :disabled="!selectedFile || !metadata.name || !metadata.description || nftStore.isLoading"
          @click="handleMint"
        >
          <Loader2 v-if="nftStore.isLoading" :size="15" class="spin" />
          <Sparkles v-else :size="15" />
          {{ nftStore.isLoading ? 'Minting...' : 'Mint NFT' }}
        </button>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  AlertCircle, ArrowLeft, Check,
  CheckCircle, Coins, ExternalLink,
  Loader2, Sparkles,
} from 'lucide-vue-next'
import { useNFTStore } from '@/stores/nft'
import UploadAsset from '@/components/mint/UploadAsset.vue'
import MetadataForm from '@/components/mint/MetadataForm.vue'
import PrivacySelector from '@/components/mint/PrivacySelector.vue'
import MintStrategy from '@/components/mint/MintStrategy.vue'

const nftStore = useNFTStore()

const selectedFile  = ref<File | null>(null)
const uploadMode    = ref<'single' | 'batch'>('single')
const privacy       = ref<'public' | 'private'>('public')
const mintStrategy  = ref<'standard' | 'lazy'>('standard')
const mintSuccess   = ref(false)
const lastTxHash    = ref('')

const metadata = reactive({
  name: '', description: '', royalties: '5', totalSupply: '1',
})

// Derive which steps are done for the progress tracker
const stepDone = (step: number) => {
  if (step === 1) return !!selectedFile.value
  if (step === 2) return !!metadata.name
  if (step === 3) return true // privacy always has a value
  if (step === 4) return true // strategy always has a value
  return false
}

// Active step = first incomplete step
const activeStep = computed(() => {
  if (!selectedFile.value) return 1
  if (!metadata.name) return 2
  return 4
})

function stepClass(step: number) {
  if (stepDone(step)) return 'step--done'
  if (step === activeStep.value) return 'step--active'
  return 'step--pending'
}

function onFileSelected(file: File)           { selectedFile.value = file }
function onMetadataUpdate(data: typeof metadata) { Object.assign(metadata, data) }

// Translate raw API errors to friendly messages
function friendlyError(raw: string): string {
  if (raw.includes('Insufficient input'))  return 'Not enough ADA in wallet. Please add funds.'
  if (raw.includes('too long'))            return raw
  if (raw.includes('UTxO'))               return 'Transaction failed. Please wait and try again.'
  if (raw.includes('collateral'))         return 'Wallet needs a 5 ADA collateral UTxO.'
  return 'Something went wrong. Please try again.'
}

async function handleMint() {
  if (!selectedFile.value) {
    nftStore.error = 'Please upload an image first.'
    return
  }
  if (!metadata.name) {
    nftStore.error = 'Please enter an NFT name.'
    return
  }
  if (!metadata.description) {
    nftStore.error = 'Please enter a description for your NFT.'
    return
  }
  const formData = new FormData()
  formData.append('file', selectedFile.value)
  formData.append('name', metadata.name)
  formData.append('description', metadata.description)
  formData.append('royalties', metadata.royalties)
  formData.append('total_supply', metadata.totalSupply)
  formData.append('privacy', privacy.value)

  const prepared = await nftStore.prepare(formData)
  if (!prepared) return
  const minted = await nftStore.mint(prepared.nft_id)
  if (!minted) return
  lastTxHash.value = minted.tx_hash
  mintSuccess.value = true
}

function resetForm() {
  mintSuccess.value  = false
  lastTxHash.value   = ''
  selectedFile.value = null
  nftStore.error     = null
}
</script>

<style scoped>
.create-mint { padding: 28px 32px; max-width: 1100px; }

/* Back link */
.back-link {
  display: inline-flex;
  align-items: center;
  width: 32px;
  height: 32px;
  justify-content: center;
  border-radius: 8px;
  color: #888;
  text-decoration: none;
  margin-bottom: 20px;
  transition: all 0.15s;
}
.back-link:hover {
  background: #EEEDFE;
  color: #534AB7;
}

/* Page header */
.page-header {
  display: flex; justify-content: space-between;
  align-items: flex-start; margin-bottom: 24px;
}
.page-title { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub   { font-size: 13px; color: #888; }

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

/* ── Main layout: step panel + form ── */
.form-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 24px;
  align-items: start;
}

/* Step panel */
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

/* Steps */
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
.step--active .step-num  { background: #534AB7; color: #fff; }
.step--done   .step-num  { background: #085041; color: #fff; }

.step-label { font-size: 12px; font-weight: 500; color: #333; }
.step-desc  { font-size: 10px; color: #aaa; margin-top: 1px; }

/* Cost box */
.cost-box {
  background: #f8f8f8;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  padding: 12px;
}
.cost-box-title {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 600; color: #534AB7;
  margin-bottom: 10px;
}
.cost-row {
  display: flex; justify-content: space-between;
  font-size: 11px; color: #666; padding: 2px 0;
}
.cost-row--total { font-size: 12px; font-weight: 500; color: #333; margin-top: 2px; }
.cost-row--total strong { color: #534AB7; }
.cost-free    { color: #085041; font-weight: 500; }
.cost-divider { height: 1px; background: #ebebeb; margin: 6px 0; }
.cost-note    { font-size: 10px; color: #aaa; margin-top: 8px; line-height: 1.5; }
.cost-note strong { color: #555; }

/* Form content */
.form-content { display: flex; flex-direction: column; gap: 12px; }

/* Error */
.error-msg {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 14px; background: #FCEBEB;
  border-radius: 8px; font-size: 12px; color: #791F1F;
}

/* Mint button */
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

/* Success card */
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

/* Spinner */
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Mobile */
@media (max-width: 768px) {
  .create-mint  { padding: 16px; }
  .form-layout  { grid-template-columns: 1fr; }
  .step-panel   { position: static; }
  .page-header  { flex-direction: column; gap: 12px; }
}
</style>