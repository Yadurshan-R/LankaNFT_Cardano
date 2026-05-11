<template>
  <div class="spinner-wrap">
    <div class="spinner" />
    <p v-if="message" class="spinner-msg">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  message?: string
}>()
</script>

<style scoped>
.spinner-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px;
}
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #eeedfe;
  border-top-color: #534ab7;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.spinner-msg {
  font-size: 14px;
  color: #666;
}
</style>