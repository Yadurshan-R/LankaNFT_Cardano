<template>
  <div class="upload-section">
    <h3 class="section-num"><span>1</span> Upload Asset</h3>
    <p class="section-sub">PNG, JPG, GIF, MP4, MP3, GLB. Max size 10MB.</p>

    <!-- ── Filled state — compact strip with thumbnail ── -->
    <div v-if="previewUrl || fileName" class="file-strip" @click="triggerInput">
      <div class="strip-thumb">
        <img v-if="previewUrl" :src="previewUrl" class="strip-img" alt="Preview" />
        <div v-else class="strip-icon"><FileIcon :size="20" color="#534AB7" /></div>
      </div>
      <div class="strip-info">
        <p class="strip-name">{{ fileName }}</p>
        <p class="strip-change">Click to change file</p>
      </div>
      <div class="strip-check"><CheckCircle :size="18" color="#085041" /></div>
    </div>

    <!-- ── Empty state — compact drop zone ── -->
    <div
      v-else
      class="drop-zone"
      :class="{ 'drop-zone--active': isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave="isDragging = false"
      @drop.prevent="handleDrop"
      @click="triggerInput"
    >
      <UploadCloud :size="28" class="drop-icon" />
      <p class="drop-text">Drag & drop your file here</p>
      <p class="drop-link">or browse files</p>
    </div>

    <input
      ref="inputRef"
      type="file"
      accept="image/*,video/*,audio/*,.glb"
      style="display: none"
      @change="handleFileChange"
    />

    <p v-if="fileError"   class="file-msg file-msg--error"><AlertCircle :size="12" /> {{ fileError }}</p>
    <p v-if="fileWarning" class="file-msg file-msg--warn"><AlertCircle :size="12" /> {{ fileWarning }}</p>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// UploadAsset.vue
//
// File upload component for single NFT minting.
// Shows a compact drop zone when empty, and a thumbnail strip when a file
// is selected — keeps the form tight and focused.
//
// Validation:
//   Hard limit: 10MB — matches backend MaxFileSizeBytes in pinata.go
//   Soft warning: 2MB — Pinata free tier is slow above this
// ─────────────────────────────────────────────────────────────────────────────

import { ref } from 'vue'
import { AlertCircle, CheckCircle, UploadCloud } from 'lucide-vue-next'
import { File as FileIcon } from 'lucide-vue-next'

const emit = defineEmits<{ (e: 'file-selected', file: File): void }>()

const isDragging = ref(false)
const previewUrl = ref<string | null>(null)
const fileName   = ref<string | null>(null)
const inputRef   = ref<HTMLInputElement | null>(null)
const fileError   = ref('')
const fileWarning = ref('')

const MAX_FILE_SIZE_MB  = 10
const WARN_FILE_SIZE_MB = 2

function triggerInput() { inputRef.value?.click() }

function handleFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) processFile(file)
}

function handleDrop(e: DragEvent) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) processFile(file)
}

function processFile(file: File) {
  const sizeMB = file.size / 1024 / 1024

  if (sizeMB > MAX_FILE_SIZE_MB) {
    fileError.value   = `File too large (${sizeMB.toFixed(1)}MB). Maximum is ${MAX_FILE_SIZE_MB}MB. Please compress your image.`
    fileWarning.value = ''
    previewUrl.value  = null
    fileName.value    = null
    return
  }

  fileError.value = ''
  fileWarning.value = sizeMB > WARN_FILE_SIZE_MB
    ? `Large file (${sizeMB.toFixed(1)}MB). Upload may take up to 2 minutes.`
    : ''

  fileName.value = file.name
  emit('file-selected', file)

  if (file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (e) => { previewUrl.value = e.target?.result as string }
    reader.readAsDataURL(file)
  } else {
    previewUrl.value = null
  }
}
</script>

<style scoped>
.upload-section { margin-bottom: 20px; }

.section-num {
  font-size: 15px; font-weight: 600;
  display: flex; align-items: center; gap: 10px; margin-bottom: 4px;
}
.section-num span {
  width: 24px; height: 24px; background: #534AB7; color: #fff;
  border-radius: 50%; display: flex; align-items: center;
  justify-content: center; font-size: 12px;
}
.section-sub { font-size: 12px; color: #888; margin-bottom: 10px; }

/* ── Compact drop zone ── */
.drop-zone {
  border: 2px dashed #d8d8d8;
  border-radius: 12px;
  padding: 28px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.drop-zone:hover,
.drop-zone--active {
  border-color: #534AB7;
  background: #EEEDFE;
}
.drop-icon  { color: #534AB7; }
.drop-text  { font-size: 13px; font-weight: 500; color: #444; margin: 0; }
.drop-link  { font-size: 12px; color: #534AB7; margin: 0; }

/* ── File strip — shown after file selected ── */
.file-strip {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 14px;
  border: 1.5px solid #534AB7;
  border-radius: 12px;
  background: #EEEDFE;
  cursor: pointer;
  transition: all 0.15s;
}
.file-strip:hover { background: #e5e3fc; }

.strip-thumb {
  width: 48px; height: 48px; border-radius: 8px;
  background: #fff; overflow: hidden; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  border: 1px solid #e0e0e0;
}
.strip-img { width: 100%; height: 100%; object-fit: cover; }

.strip-info { flex: 1; min-width: 0; }
.strip-name {
  font-size: 13px; font-weight: 500; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin: 0;
}
.strip-change { font-size: 11px; color: #534AB7; margin: 2px 0 0; }

.strip-check { flex-shrink: 0; }

/* Validation messages */
.file-msg {
  display: flex; align-items: center; gap: 5px;
  font-size: 12px; margin-top: 6px;
}
.file-msg--error { color: #d32f2f; }
.file-msg--warn  { color: #854F0B; }
</style>