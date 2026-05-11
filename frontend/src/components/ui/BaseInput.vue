<template>
  <div class="input-wrap">
    <label v-if="label" class="input-label">{{ label }}</label>
    <input
      :type="type"
      :placeholder="placeholder"
      :value="modelValue"
      :disabled="disabled"
      class="input-field"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <span v-if="error" class="input-error">{{ error }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string
  label?: string
  placeholder?: string
  type?: string
  error?: string
  disabled?: boolean
}>()

defineEmits(['update:modelValue'])
</script>

<style scoped>
.input-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.input-label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}
.input-field {
  padding: 10px 14px;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: border 0.2s;
  width: 100%;
  box-sizing: border-box;
}
.input-field:focus {
  border-color: #534ab7;
}
.input-field:disabled {
  background: #f4f4f4;
  cursor: not-allowed;
}
.input-error {
  font-size: 12px;
  color: #d32f2f;
}
</style>