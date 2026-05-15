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

    <!-- Success state -->
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

    <!-- Mint form -->
    <div v-else class="mint-form">
      <UploadAsset @file-selected="onFileSelected" />
      <MetadataForm @update="onMetadataUpdate" />
      <PrivacySelector v-model="privacy" />
      <MintStrategy v-model="mintStrategy" />

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
import { CheckCircle, Tag } from 'lucide-vue-next'
import { useNFTStore } from '@/stores/nft'
import UploadAsset from '@/components/mint/UploadAsset.vue'
import MetadataForm from '@/components/mint/MetadataForm.vue'
import PrivacySelector from '@/components/mint/PrivacySelector.vue'
import MintStrategy from '@/components/mint/MintStrategy.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const nftStore = useNFTStore()
const selectedFile = ref<File | null>(null)
const uploadMode = ref<'single' | 'batch'>('single')
const privacy = ref<'public' | 'private'>('public')
const mintStrategy = ref<'standard' | 'lazy'>('standard')
const mintSuccess = ref(false)
const lastTxHash = ref('')

const metadata = reactive({
  name: '',
  description: '',
  royalties: '5',
  totalSupply: '1',
})

function onFileSelected(file: File) { selectedFile.value = file }
function onMetadataUpdate(data: typeof metadata) { Object.assign(metadata, data) }

async function handleMint() {
  if (!selectedFile.value || !metadata.name) return
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
  mintSuccess.value = false
  lastTxHash.value = ''
  selectedFile.value = null
  nftStore.error = null
}
</script>

<style scoped>
.create-mint { max-width: 760px; margin: 0 auto; padding: 32px; }
.back-link { font-size: 13px; color: #666; text-decoration: none; display: block; margin-bottom: 16px; }
.back-link:hover { color: #534AB7; }
.header-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 28px; }
.page-title { font-size: 28px; font-weight: 700; }
.page-sub { font-size: 13px; color: #666; margin-top: 4px; }
.upload-toggle { display: flex; gap: 8px; }
.toggle-btn {
  padding: 8px 16px; border-radius: 8px; border: 1.5px solid #e0e0e0;
  background: #fff; font-size: 13px; cursor: pointer; font-weight: 500;
}
.toggle-btn.active { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }
.mint-form { display: flex; flex-direction: column; gap: 8px; }
.error-msg { font-size: 13px; color: #d32f2f; margin-bottom: 8px; }
.success-card {
  text-align: center; padding: 48px; border: 1px solid #e0e0e0;
  border-radius: 16px; display: flex; flex-direction: column;
  align-items: center; gap: 16px;
}
.success-icon { display: flex; justify-content: center; }
.tx-link { color: #534AB7; font-size: 13px; text-decoration: underline; }
</style>