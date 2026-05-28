<template>
  <div class="privacy-section">
    <h3 class="section-num"><span>3</span> Privacy & Visibility</h3>
    <p class="section-sub">Choose who can view and access this NFT.</p>

    <div class="options">
      <div
        :class="['option', { 'option--active': modelValue === 'public' }]"
        @click="$emit('update:modelValue', 'public')"
      >
        <div class="option-icon">
          <Globe :size="20" :color="modelValue === 'public' ? '#534AB7' : '#aaa'" />
        </div>
        <div class="option-body">
          <div class="option-title">Public</div>
          <div class="option-desc">Anyone can view and interact.</div>
        </div>
        <div class="radio" :class="{ 'radio--selected': modelValue === 'public' }" />
      </div>

      <div
        :class="['option', { 'option--active': modelValue === 'private' }]"
        @click="$emit('update:modelValue', 'private')"
      >
        <div class="option-icon">
          <Lock :size="20" :color="modelValue === 'private' ? '#534AB7' : '#aaa'" />
        </div>
        <div class="option-body">
          <div class="option-title">Private</div>
          <div class="option-desc">Access restricted by ZK rules.</div>
        </div>
        <div class="radio" :class="{ 'radio--selected': modelValue === 'private' }" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Globe, Lock } from 'lucide-vue-next'
defineProps<{ modelValue: 'public' | 'private' }>()
defineEmits(['update:modelValue'])
</script>

<style scoped>
.privacy-section { margin-bottom: 20px; }

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

.options { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

.option {
  border: 1.5px solid #e8e8e8;
  border-radius: 10px;
  padding: 11px 12px;
  cursor: pointer;
  display: flex; align-items: center; gap: 10px;
  transition: all 0.15s;
}
.option:hover       { border-color: #534AB7; }
.option--active     { border-color: #534AB7; background: #EEEDFE; }

.option-icon { flex-shrink: 0; display: flex; align-items: center; }

.option-body { flex: 1; min-width: 0; }
.option-title { font-size: 13px; font-weight: 500; color: #111; }
.option-desc  { font-size: 11px; color: #888; margin-top: 1px; }

.radio {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid #ddd; flex-shrink: 0;
  transition: all 0.15s;
}
.radio--selected { border-color: #534AB7; background: #534AB7; }
</style>