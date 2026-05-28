<template>
  <div class="strategy-section">
    <h3 class="section-num"><span>4</span> Minting Strategy</h3>
    <p class="section-sub">Choose how you want to mint this NFT.</p>

    <div class="options">
      <!-- Standard Mint — fully active -->
      <div
        :class="['option', { 'option--active': modelValue === 'standard' }]"
        @click="$emit('update:modelValue', 'standard')"
      >
        <div class="option-icon">
          <Rocket :size="20" :color="modelValue === 'standard' ? '#534AB7' : '#aaa'" />
        </div>
        <div class="option-body">
          <div class="option-title">Standard Mint</div>
          <div class="option-desc">Creator pays gas and mints on-chain.</div>
        </div>
        <div class="radio" :class="{ 'radio--selected': modelValue === 'standard' }" />
      </div>

      <!-- Lazy Minting — disabled, not yet built -->
      <!-- Kept visible so users know it's coming, but not selectable -->
      <div class="option option--disabled">
        <div class="option-icon">
          <PenLine :size="20" color="#ccc" />
        </div>
        <div class="option-body">
          <div class="option-title-row">
            <span class="option-title">Lazy Minting</span>
            <span class="coming-soon-badge">Coming Soon</span>
          </div>
          <div class="option-desc">NFT minted when first buyer purchases.</div>
        </div>
        <div class="radio" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// MintStrategy.vue
//
// Minting strategy selector for single NFT minting.
//
// Lazy Minting is intentionally disabled — the feature is on the roadmap
// but not yet implemented. Keeping it visible (with "Coming Soon") is
// better UX than hiding it, as it sets user expectations about future
// platform capabilities without allowing selection of a broken feature.
// ─────────────────────────────────────────────────────────────────────────────

import { PenLine, Rocket } from 'lucide-vue-next'
defineProps<{ modelValue: 'standard' | 'lazy' }>()
defineEmits(['update:modelValue'])
</script>

<style scoped>
.strategy-section { margin-bottom: 20px; }

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

/* Disabled state — not clickable, visually muted */
.option--disabled {
  cursor: not-allowed;
  opacity: 0.55;
  background: #fafafa;
}
.option--disabled:hover { border-color: #e8e8e8; }

.option-icon { flex-shrink: 0; display: flex; align-items: center; }

.option-body { flex: 1; min-width: 0; }

.option-title-row {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
}
.option-title { font-size: 13px; font-weight: 500; color: #111; }
.option-desc  { font-size: 11px; color: #888; margin-top: 1px; }

/* Coming Soon badge */
.coming-soon-badge {
  font-size: 9px; font-weight: 600;
  padding: 2px 7px; border-radius: 20px;
  background: #FAEEDA; color: #854F0B;
  white-space: nowrap;
}

.radio {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid #ddd; flex-shrink: 0;
  transition: all 0.15s;
}
.radio--selected { border-color: #534AB7; background: #534AB7; }
</style>