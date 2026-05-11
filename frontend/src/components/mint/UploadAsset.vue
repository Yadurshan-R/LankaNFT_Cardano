<template>
  <div class="upload-section">
    <h3 class="section-num"><span>1</span> Upload Asset</h3>
    <p class="section-sub">PNG, JPG, GIF, MP4, MP3, GLB. Max size 100MB.</p>

    <!-- Drop zone -->
    <div
      class="drop-zone"
      :class="{ 'drop-zone--active': isDragging, 'drop-zone--filled': previewUrl }"
      @dragover.prevent="isDragging = true"
      @dragleave="isDragging = false"
      @drop.prevent="handleDrop"
      @click="triggerInput"
    >
      <!-- Preview -->
      <img v-if="previewUrl" :src="previewUrl" class="preview-img" alt="NFT preview" />

      <!-- Placeholder -->
      <div v-else class="drop-placeholder">
        <div class="upload-icon">↑</div>
        <p class="drop-text">Drag & drop your file here</p>
        <p class="drop-link">or browse files</p>
      </div>
    </div>

    <input
      ref="inputRef"
      type="file"
      accept="image/*,video/*,audio/*,.glb"
      style="display: none"
      @change="handleFileChange"
    />

    <!-- File info -->
    <p v-if="fileName" class="file-name">{{ fileName }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ (e: 'file-selected', file: File): void }>()

const isDragging = ref(false)
const previewUrl = ref<string | null>(null)
const fileName = ref<string | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)

function triggerInput() {
  inputRef.value?.click()
}

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
  fileName.value = file.name
  emit('file-selected', file)

  // Show image preview
  if (file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (e) => {
      previewUrl.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}
</script>

<style scoped>
.upload-section { margin-bottom: 24px; }
.section-num {
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}
.section-num span {
  width: 24px;
  height: 24px;
  background: #534AB7;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
}
.section-sub { font-size: 12px; color: #888; margin-bottom: 12px; }
.drop-zone {
  border: 2px dashed #d0d0d0;
  border-radius: 12px;
  padding: 40px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 180px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.drop-zone--active { border-color: #534AB7; background: #EEEDFE; }
.drop-zone--filled { padding: 12px; border-style: solid; border-color: #534AB7; }
.drop-zone:hover { border-color: #534AB7; }
.upload-icon { font-size: 32px; color: #534AB7; margin-bottom: 8px; }
.drop-text { font-size: 14px; color: #333; font-weight: 500; }
.drop-link { font-size: 13px; color: #534AB7; margin-top: 4px; }
.preview-img { max-height: 200px; max-width: 100%; border-radius: 8px; object-fit: contain; }
.file-name { font-size: 12px; color: #666; margin-top: 8px; }
</style>