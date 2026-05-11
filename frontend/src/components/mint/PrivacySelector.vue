<template>
  <div class="privacy-section">
    <h3 class="section-num"><span>3</span> Privacy & Visibility</h3>
    <p class="section-sub">Choose who can view and access this NFT.</p>

    <div class="options">
      <div
        :class="['option', { active: modelValue === 'public' }]"
        @click="$emit('update:modelValue', 'public')"
      >
        <div class="option-icon">🌐</div>
        <div>
          <div class="option-title">Public</div>
          <div class="option-desc">Anyone can view and interact.</div>
        </div>
        <div class="radio" :class="{ selected: modelValue === 'public' }" />
      </div>

      <div
        :class="['option', { active: modelValue === 'private' }]"
        @click="$emit('update:modelValue', 'private')"
      >
        <div class="option-icon">🔒</div>
        <div>
          <div class="option-title">Private</div>
          <div class="option-desc">Access is restricted and controlled by ZK rules.</div>
        </div>
        <div class="radio" :class="{ selected: modelValue === 'private' }" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ modelValue: 'public' | 'private' }>()
defineEmits(['update:modelValue'])
</script>

<style scoped>
.privacy-section { margin-bottom: 24px; }
.section-num {
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}
.section-num span {
  width: 24px; height: 24px;
  background: #534AB7; color: #fff;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px;
}
.section-sub { font-size: 12px; color: #888; margin-bottom: 12px; }
.options { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.option {
  border: 1.5px solid #e0e0e0;
  border-radius: 10px;
  padding: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: all 0.2s;
}
.option.active { border-color: #534AB7; background: #EEEDFE; }
.option-icon { font-size: 24px; }
.option-title { font-size: 14px; font-weight: 500; }
.option-desc { font-size: 12px; color: #666; margin-top: 2px; }
.radio {
  width: 18px; height: 18px;
  border-radius: 50%;
  border: 2px solid #ccc;
  margin-left: auto;
  flex-shrink: 0;
}
.radio.selected { border-color: #534AB7; background: #534AB7; }
</style>