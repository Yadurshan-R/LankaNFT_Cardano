<template>
  <button
    :class="[
      'base-btn',
      `base-btn--${variant}`,
      { 'base-btn--loading': loading },
    ]"
    :disabled="disabled || loading"
    @click="$emit('click')"
  >
    <span v-if="loading" class="spinner" />
    <span v-else><slot /></span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  variant?: 'primary' | 'secondary' | 'outline'
  loading?: boolean
  disabled?: boolean
}>()

defineEmits(['click'])
</script>

<style scoped>
.base-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
  width: 100%;
}
.base-btn--primary {
  background: #534ab7;
  color: #fff;
}
.base-btn--primary:hover:not(:disabled) {
  background: #3c3489;
}
.base-btn--secondary {
  background: #f4f4f4;
  color: #111;
}
.base-btn--secondary:hover:not(:disabled) {
  background: #e8e8e8;
}
.base-btn--outline {
  background: transparent;
  color: #534ab7;
  border: 1.5px solid #534ab7;
}
.base-btn--outline:hover:not(:disabled) {
  background: #eeedfe;
}
.base-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>