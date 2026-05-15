<template>
  <div class="dashboard">
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-sub">Welcome back. Here is your NFT portfolio.</p>
      </div>
      <router-link to="/mint">
        <BaseButton variant="primary">+ Create NFT</BaseButton>
      </router-link>
    </div>

    <!-- Loading skeleton -->
    <div v-if="dashboard.isLoading" class="dashboard-content">
      <div class="top-row">
        <div class="skeleton skeleton-wallet" />
        <div class="skeleton-stats">
          <div class="skeleton skeleton-stat" v-for="n in 4" :key="n" />
        </div>
      </div>
      <div class="nft-grid">
        <div class="skeleton skeleton-card" v-for="n in 8" :key="n" />
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="dashboard.error" class="error-state">
      <p>{{ dashboard.error }}</p>
      <BaseButton variant="outline" @click="dashboard.loadDashboard()">Retry</BaseButton>
    </div>

    <!-- Content -->
    <div v-else class="dashboard-content">
      <div class="top-row">
        <WalletCard :address="dashboard.walletAddress" :lovelace="dashboard.lovelace" />
        <StatsRow
          :total="dashboard.totalNFTs"
          :minted="dashboard.mintedNFTs"
          :pending="dashboard.pendingNFTs"
          @filter="activeTab = $event"
        />
      </div>

      <!-- Tabs -->
      <div class="section">
        <div class="section-header">
          <div class="tabs">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              :class="['tab', { active: activeTab === tab.key }]"
              @click="activeTab = tab.key"
            >
              {{ tab.label }}
              <span class="tab-count">{{ tab.count }}</span>
            </button>
          </div>
          <router-link to="/mint" class="section-action">+ Mint New</router-link>
        </div>

        <!-- Empty state -->
        <div v-if="activeNFTs.length === 0" class="empty-nfts">
          <Palette :size="48" color="#ccc" />
          <h3>{{ emptyTitle }}</h3>
          <p>{{ emptyDesc }}</p>
          <router-link v-if="activeTab === 'all'" to="/mint">
            <BaseButton variant="primary">Create Your First NFT</BaseButton>
          </router-link>
        </div>

        <!-- NFT Grid -->
        <div v-else class="nft-grid">
          <NFTCard
            v-for="nft in activeNFTs"
            :key="nft.id"
            :nft="nft"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Palette } from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'
import WalletCard from '@/components/dashboard/WalletCard.vue'
import StatsRow from '@/components/dashboard/StatsRow.vue'
import NFTCard from '@/components/dashboard/NFTCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const dashboard = useDashboardStore()
const activeTab = ref('all')

const tabs = computed(() => [
  { key: 'all',     label: 'All NFTs',  count: dashboard.uniqueNFTs.length },
  { key: 'minted',  label: 'Minted',    count: dashboard.mintedOnly.length },
  { key: 'listed',  label: 'Listed',    count: dashboard.listedOnly.length },
  { key: 'pending', label: 'Pending',   count: dashboard.pendingOnly.length },
])

const activeNFTs = computed(() => {
  switch (activeTab.value) {
    case 'minted':  return dashboard.mintedOnly
    case 'listed':  return dashboard.listedOnly
    case 'pending': return dashboard.pendingOnly
    default:        return dashboard.uniqueNFTs
  }
})

const emptyTitle = computed(() => {
  switch (activeTab.value) {
    case 'minted':  return 'No minted NFTs'
    case 'listed':  return 'No listed NFTs'
    case 'pending': return 'No pending NFTs'
    default:        return 'No NFTs yet'
  }
})

const emptyDesc = computed(() => {
  switch (activeTab.value) {
    case 'minted':  return 'Minted NFTs will appear here.'
    case 'listed':  return 'List an NFT for sale from its detail page.'
    case 'pending': return 'NFTs being minted will appear here.'
    default:        return 'Start minting your first NFT on Cardano Preprod.'
  }
})

onMounted(() => {
  dashboard.loadDashboard()
})
</script>

<style scoped>
.dashboard { max-width: 1100px; margin: 0 auto; padding: 32px; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}
.page-title { font-size: 28px; font-weight: 700; margin: 0 0 4px; }
.page-sub { font-size: 13px; color: #666; margin: 0; }

.error-state { text-align: center; padding: 48px; color: #d32f2f; }

.dashboard-content { display: flex; flex-direction: column; gap: 32px; }

.top-row {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 24px;
  align-items: start;
}

/* Skeleton */
.skeleton {
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
  border-radius: 12px;
}
.skeleton-wallet { height: 180px; }
.skeleton-stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.skeleton-stat { height: 100px; }
.skeleton-card { height: 260px; }
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Tabs */
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.tabs { display: flex; gap: 4px; }
.tab {
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s;
}
.tab:hover { background: #f5f5f5; color: #333; }
.tab.active { background: #EEEDFE; color: #534AB7; }
.tab-count {
  background: #e0e0e0;
  color: #666;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}
.tab.active .tab-count { background: #534AB7; color: #fff; }

.section-action { font-size: 13px; color: #534AB7; text-decoration: none; }
.section-action:hover { text-decoration: underline; }

/* Empty */
.empty-nfts {
  text-align: center;
  padding: 64px;
  border: 1.5px dashed #e0e0e0;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.empty-nfts h3 { font-size: 18px; font-weight: 600; margin: 0; }
.empty-nfts p { font-size: 14px; color: #888; margin: 0; }

/* Grid */
.nft-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

/* Mobile */
@media (max-width: 768px) {
  .dashboard { padding: 16px; }
  .top-row { grid-template-columns: 1fr; }
  .tabs { flex-wrap: wrap; }
  .nft-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .page-title { font-size: 22px; }
}
</style>