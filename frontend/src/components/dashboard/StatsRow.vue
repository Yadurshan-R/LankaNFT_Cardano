<template>
  <div class="stats-row">
    <div class="stat-card">
      <div class="stat-icon">
        <Image :size="24" :stroke-width="1.5" color="#534AB7" />
      </div>
      <div class="stat-value">{{ total }}</div>
      <div class="stat-label">Total NFTs</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">
        <CheckCircle :size="24" :stroke-width="1.5" color="#085041" />
      </div>
      <div class="stat-value">{{ minted }}</div>
      <div class="stat-label">Minted</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">
        <Clock :size="24" :stroke-width="1.5" color="#7a5c00" />
      </div>
      <div class="stat-value">{{ pending }}</div>
      <div class="stat-label">Pending</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">
        <Link :size="24" :stroke-width="1.5" color="#534AB7" />
      </div>
      <div class="stat-value">Preprod</div>
      <div class="stat-label">Network</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckCircle, Clock, Image, Link } from 'lucide-vue-next'

defineProps<{
  total: number
  minted: number
  pending: number
}>()
</script>

<style scoped>
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}
.stat-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
}
.stat-icon {
  display: flex;
  justify-content: center;
  margin-bottom: 8px;
}
.stat-value { font-size: 24px; font-weight: 700; color: #1a1a1a; }
.stat-label { font-size: 12px; color: #888; margin-top: 4px; }
</style>