<template>
  <div class="create-mint">
    <div class="page-header">
      <router-link to="/" class="back-link">← Back to Dashboard</router-link>
      <div class="header-row">
        <div>
          <h1 class="page-title">Create Mint</h1>
          <p class="page-sub">Upload your asset, set details, and choose how you want to mint.</p>
        </div>
        <div class="upload-toggle">
          <button :class="['toggle-btn', { active: uploadMode === 'single' }]" @click="uploadMode = 'single'">
            Single Upload
          </button>
          <router-link to="/batch">
            <button :class="['toggle-btn', { active: uploadMode === 'batch' }]">
              Batch Upload
            </button>
          </router-link>
        </div>
      </div>
    </div>

    <!-- ── Success State ─────────────────────────────────────────── -->
    <div v-if="mintSuccess" class="success-card">
      <div class="success-icon">
        <CheckCircle :size="48" color="#1D9E75" />
      </div>
      <h2>NFT Minted Successfully!</h2>
      <p>Your NFT has been minted on Cardano Preprod.</p>
      <a
        :href="`https://preprod.cardanoscan.io/transaction/${lastTxHash}`"
        target="_blank"
        rel="noopener noreferrer"
        class="tx-link"
      >
        View on Cardanoscan →
      </a>
      <BaseButton variant="outline" @click="resetForm">Mint Another</BaseButton>
    </div>

    <!-- ── Mint Form ──────────────────────────────────────────────── -->
    <div v-else class="mint-form">
      <UploadAsset @file-selected="onFileSelected" />
      <MetadataForm @update="onMetadataUpdate" />
      <PrivacySelector v-model="privacy" />
      <MintStrategy v-model="mintStrategy" />

      <!-- ── Cost Estimate Banner ─────────────────────────────────── -->
      <!-- Shown when the user has filled in enough details to mint   -->
      <!-- Helps users know how much ADA they need before submitting  -->
      <div v-if="selectedFile && metadata.name" class="cost-banner">
        <div class="cost-banner-header">
          <Coins :size="16" color="#534AB7" />
          <span>Estimated Minting Cost</span>
        </div>
        <div class="cost-rows">
          <!-- Three UTxOs are created: user NFT, reference NFT, royalty NFT -->
          <!-- Each requires minimum ADA to be stored on-chain per Cardano rules -->
          <div class="cost-row">
            <span class="cost-label">Min-ADA (3 UTxOs)</span>
            <span class="cost-val">~6 ADA</span>
          </div>
          <!-- Network fee varies based on tx size, ~0.2–0.5 ADA for single mint -->
          <div class="cost-row">
            <span class="cost-label">Network fee</span>
            <span class="cost-val">~0.5 ADA</span>
          </div>
          <!-- IPFS pinning via Pinata — included in platform, not charged to user -->
          <div class="cost-row">
            <span class="cost-label">IPFS pinning (Pinata)</span>
            <span class="cost-val cost-free">Included</span>
          </div>
          <div class="cost-divider" />
          <div class="cost-row cost-row--total">
            <span class="cost-label">Total estimate</span>
            <span class="cost-total">~6.5 ADA</span>
          </div>
        </div>
        <p class="cost-note">
          Make sure your wallet has at least <strong>7 ADA</strong> before minting.
          Unused ADA is returned to your wallet after the transaction.
        </p>
      </div>

      <p v-if="nftStore.error" class="error-msg">{{ nftStore.error }}</p>

      <BaseButton
        variant="primary"
        :loading="nftStore.isLoading"
        :disabled="!selectedFile || !metadata.name"
        @click="handleMint"
      >
        <Tag :size="14" /> Mint NFT
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { CheckCircle, Coins, Tag } from 'lucide-vue-next'
import { useNFTStore } from '@/stores/nft'
import UploadAsset from '@/components/mint/UploadAsset.vue'
import MetadataForm from '@/components/mint/MetadataForm.vue'
import PrivacySelector from '@/components/mint/PrivacySelector.vue'
import MintStrategy from '@/components/mint/MintStrategy.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const nftStore = useNFTStore()

// UI state
const selectedFile = ref<File | null>(null)
const uploadMode = ref<'single' | 'batch'>('single')
const privacy = ref<'public' | 'private'>('public')
const mintStrategy = ref<'standard' | 'lazy'>('standard')
const mintSuccess = ref(false)
const lastTxHash = ref('')

// Form metadata — synced from MetadataForm component
const metadata = reactive({
  name: '',
  description: '',
  royalties: '5',
  totalSupply: '1',
})

function onFileSelected(file: File) { selectedFile.value = file }
function onMetadataUpdate(data: typeof metadata) { Object.assign(metadata, data) }

// Two-step mint: prepare (IPFS upload + DB) then mint (on-chain)
async function handleMint() {
  if (!selectedFile.value || !metadata.name) return

  const formData = new FormData()
  formData.append('file', selectedFile.value)
  formData.append('name', metadata.name)
  formData.append('description', metadata.description)
  formData.append('royalties', metadata.royalties)
  formData.append('total_supply', metadata.totalSupply)
  formData.append('privacy', privacy.value)

  // Step 1 — upload image to IPFS and store NFT record in DB
  const prepared = await nftStore.prepare(formData)
  if (!prepared) return

  // Step 2 — submit minting transaction to Cardano via the sidecar
  const minted = await nftStore.mint(prepared.nft_id)
  if (!minted) return

  lastTxHash.value = minted.tx_hash
  mintSuccess.value = true
}

// Reset form to allow minting another NFT
function resetForm() {
  mintSuccess.value = false
  lastTxHash.value = ''
  selectedFile.value = null
  nftStore.error = null
}
</script>

<style scoped>
.create-mint { max-width: 760px; margin: 0 auto; padding: 32px; }

.back-link {
  font-size: 13px; color: #666; text-decoration: none;
  display: block; margin-bottom: 16px;
}
.back-link:hover { color: #534AB7; }

.header-row {
  display: flex; justify-content: space-between;
  align-items: flex-start; margin-bottom: 28px;
}
.page-title { font-size: 28px; font-weight: 700; }
.page-sub { font-size: 13px; color: #666; margin-top: 4px; }

.upload-toggle { display: flex; gap: 8px; }
.toggle-btn {
  padding: 8px 16px; border-radius: 8px; border: 1.5px solid #e0e0e0;
  background: #fff; font-size: 13px; cursor: pointer; font-weight: 500;
}
.toggle-btn.active { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }

.mint-form { display: flex; flex-direction: column; gap: 8px; }

/* ── Cost Banner ─────────────────────────────────────────── */
.cost-banner {
  background: #FAFAFA;
  border: 1.5px solid #e8e8e8;
  border-radius: 12px;
  padding: 16px;
  margin-top: 8px;
}
.cost-banner-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600; color: #534AB7;
  margin-bottom: 12px;
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
.cost-divider {
  height: 1px; background: #e0e0e0; margin: 6px 0;
}
.cost-total {
  font-size: 15px; font-weight: 700; color: #534AB7;
}
.cost-note {
  font-size: 12px; color: #888;
  margin: 12px 0 0; line-height: 1.5;
}
.cost-note strong { color: #333; }

.error-msg { font-size: 13px; color: #d32f2f; margin-bottom: 8px; }

/* ── Success Card ─────────────────────────────────────────── */
.success-card {
  text-align: center; padding: 48px;
  border: 1px solid #e0e0e0; border-radius: 16px;
  display: flex; flex-direction: column;
  align-items: center; gap: 16px;
}
.success-icon { display: flex; justify-content: center; }
.tx-link { color: #534AB7; font-size: 13px; text-decoration: underline; }
</style>