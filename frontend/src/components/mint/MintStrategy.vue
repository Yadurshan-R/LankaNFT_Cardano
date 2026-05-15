<template>
  <div class="strategy-section">
    <h3 class="section-num"><span>4</span> Minting Strategy</h3>
    <p class="section-sub">Choose how you want to mint this NFT.</p>

    <div class="options">
      <div
        :class="['option', { active: modelValue === 'standard' }]"
        @click="$emit('update:modelValue', 'standard')"
      >
        <div class="option-icon">
          <Rocket :size="24" :color="modelValue === 'standard' ? '#534AB7' : '#888'" />
        </div>
        <div>
          <div class="option-title">Standard Mint</div>
          <div class="option-desc">Creator pays gas and mints the NFT on-chain.</div>
        </div>
        <div class="radio" :class="{ selected: modelValue === 'standard' }" />
      </div>

      <div
        :class="['option', { active: modelValue === 'lazy' }]"
        @click="$emit('update:modelValue', 'lazy')"
      >
        <div class="option-icon">
          <PenLine :size="24" :color="modelValue === 'lazy' ? '#534AB7' : '#888'" />
        </div>
        <div>
          <div class="option-title">Lazy Minting</div>
          <div class="option-desc">Sign metadata only. NFT is minted when the first buyer purchases it.</div>
        </div>
        <div class="radio" :class="{ selected: modelValue === 'lazy' }" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { PenLine, Rocket } from 'lucide-vue-next'

defineProps<{ modelValue: 'standard' | 'lazy' }>()
defineEmits(['update:modelValue'])
</script>

<style scoped>
.strategy-section { margin-bottom: 24px; }
.section-num {
  font-size: 15px; font-weight: 600;
  display: flex; align-items: center; gap: 10px; margin-bottom: 4px;
}
.section-num span {
  width: 24px; height: 24px; background: #534AB7; color: #fff;
  border-radius: 50%; display: flex; align-items: center;
  justify-content: center; font-size: 12px;
}
.section-sub { font-size: 12px; color: #888; margin-bottom: 12px; }
.options { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.option {
  border: 1.5px solid #e0e0e0; border-radius: 10px;
  padding: 14px; cursor: pointer;
  display: flex; align-items: center; gap: 12px; transition: all 0.2s;
}
.option.active { border-color: #534AB7; background: #EEEDFE; }
.option-icon { display: flex; align-items: center; flex-shrink: 0; }
.option-title { font-size: 14px; font-weight: 500; }
.option-desc { font-size: 12px; color: #666; margin-top: 2px; }
.radio {
  width: 18px; height: 18px; border-radius: 50%;
  border: 2px solid #ccc; margin-left: auto; flex-shrink: 0;
}
.radio.selected { border-color: #534AB7; background: #534AB7; }
</style>