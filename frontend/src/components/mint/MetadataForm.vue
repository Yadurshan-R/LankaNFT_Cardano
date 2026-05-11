<template>
  <div class="metadata-section">
    <h3 class="section-num"><span>2</span> Metadata</h3>

    <div class="form-grid">
      <BaseInput
        v-model="form.name"
        label="NFT Name *"
        placeholder="Enter name"
      />
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
</style>