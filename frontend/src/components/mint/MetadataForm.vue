<template>
  <div class="metadata-section">
    <h3 class="section-num"><span>2</span> Metadata</h3>

    <div class="form-grid">
      <!-- NFT Name -->
      <div class="field-wrap">
        <label class="field-label">NFT Name *</label>
        <input
          v-model="form.name"
          class="field-input"
          placeholder="Enter name"
          maxlength="28"
        />
        <p class="char-count" :class="{ 'char-count--warn': form.name.length > 24 }">
          {{ form.name.length }}/28 characters
        </p>
      </div>

      <!-- Royalties -->
      <div class="field-wrap">
        <label class="field-label">Royalties (%)</label>
        <input
          v-model="form.royalties"
          class="field-input"
          type="number"
          placeholder="5"
          min="0"
          max="100"
        />
        <p class="field-hint">Paid to you on every resale</p>
      </div>

      <!-- Description — full width -->
      <div class="field-wrap field-wrap--full">
        <label class="field-label">Description *</label>
        <textarea
          v-model="form.description"
          class="field-input field-input--textarea"
          placeholder="Describe your NFT..."
          maxlength="200"
          rows="3"
        />
        <p class="char-count" :class="{ 'char-count--warn': form.description.length > 180 }">
          {{ form.description.length }}/200 characters
        </p>
      </div>
    </div>

    <!-- Advanced options — collapsed by default -->
    <!-- Total Supply hidden here — 99% of users mint supply 1 and don't need this -->
    <button class="advanced-toggle" @click="showAdvanced = !showAdvanced">
      <ChevronDown
        :size="14"
        :class="['toggle-icon', { 'toggle-icon--open': showAdvanced }]"
      />
      Advanced options
    </button>

    <div v-if="showAdvanced" class="advanced-content">
      <div class="field-wrap">
        <label class="field-label">Total Supply</label>
        <input
          v-model="form.totalSupply"
          class="field-input field-input--sm"
          type="number"
          placeholder="1"
          min="1"
        />
        <p class="field-hint">Number of editions to mint. Leave as 1 for a unique NFT.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// MetadataForm.vue
//
// NFT metadata input form. Used in the single mint flow (CreateMintView).
//
// Design decision: Total Supply is hidden behind "Advanced options" because
// 99% of users mint supply 1. Surfacing it by default confuses new users
// and clutters the form. Power users can expand it when needed.
// ─────────────────────────────────────────────────────────────────────────────

import { reactive, ref, watch } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

const emit = defineEmits<{ (e: 'update', data: typeof form): void }>()

const showAdvanced = ref(false)

const form = reactive({
  name:        '',
  description: '',
  royalties:   '5',
  totalSupply: '1',
})

watch(form, () => emit('update', form), { deep: true })
</script>

<style scoped>
.metadata-section { margin-bottom: 20px; }

.section-num {
  font-size: 15px; font-weight: 600;
  display: flex; align-items: center; gap: 10px; margin-bottom: 16px;
}
.section-num span {
  width: 24px; height: 24px; background: #534AB7; color: #fff;
  border-radius: 50%; display: flex; align-items: center;
  justify-content: center; font-size: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.field-wrap { display: flex; flex-direction: column; gap: 4px; }
.field-wrap--full { grid-column: 1 / -1; }

.field-label { font-size: 12px; font-weight: 500; color: #444; }

.field-input {
  padding: 9px 12px;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
  font-family: inherit;
}
.field-input:focus   { border-color: #534AB7; }
.field-input--textarea { resize: none; line-height: 1.5; }
.field-input--sm     { max-width: 120px; }

.char-count      { font-size: 11px; color: #999; margin: 0; }
.char-count--warn { color: #d32f2f; }
.field-hint      { font-size: 11px; color: #aaa; margin: 0; }

/* Advanced toggle */
.advanced-toggle {
  display: flex; align-items: center; gap: 6px;
  margin-top: 14px; padding: 0;
  background: none; border: none;
  font-size: 12px; font-weight: 500; color: #888;
  cursor: pointer; transition: color 0.15s;
}
.advanced-toggle:hover { color: #534AB7; }

.toggle-icon { transition: transform 0.2s ease; flex-shrink: 0; }
.toggle-icon--open { transform: rotate(180deg); }

.advanced-content {
  margin-top: 12px;
  padding: 14px;
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
}
</style>