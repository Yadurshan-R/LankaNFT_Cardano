<template>
  <div class="metadata-section">
    <h3 class="section-num"><span>2</span> Metadata</h3>

    <div class="form-grid">
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
      <BaseInput
        v-model="form.royalties"
        label="Royalties (%)"
        type="number"
        placeholder="5"
      />
      <div class="full-width">
        <BaseInput
          v-model="form.description"
          label="Description"
          placeholder="Enter description"
        />
      </div>
      <BaseInput
        v-model="form.totalSupply"
        label="Total Supply *"
        type="number"
        placeholder="1"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const emit = defineEmits<{
  (e: 'update', data: typeof form): void
}>()

const form = reactive({
  name: '',
  description: '',
  royalties: '5',
  totalSupply: '1',
})

// Emit updates to parent whenever form changes
watch(form, () => emit('update', form), { deep: true })
</script>

<style scoped>
.metadata-section { margin-bottom: 24px; }
.section-num {
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
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
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.full-width { grid-column: 1 / -1; }

.field-wrap { display: flex; flex-direction: column; gap: 4px; }
.field-label { font-size: 12px; font-weight: 500; color: #444; }
.field-input {
  padding: 10px 12px; border: 1.5px solid #e0e0e0;
  border-radius: 8px; font-size: 14px; outline: none;
}
.field-input:focus { border-color: #534AB7; }
.char-count { font-size: 11px; color: #999; margin: 0; }
.char-count--warn { color: #d32f2f; }
</style>