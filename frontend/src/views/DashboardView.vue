<template>
  <div class="dashboard">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-sub">Welcome back. Here is your NFT portfolio.</p>
      </div>
      <router-link to="/mint">
        <BaseButton variant="primary">+ Create NFT</BaseButton>
      </router-link>
    </div>

    <!-- Loading -->
    <div v-if="dashboard.isLoading" class="loading-state">
      <div class="spinner" />
      <p>Loading your dashboard...</p>
    </div>

    <!-- Error -->
    <div v-else-if="dashboard.error" class="error-state">
      <p>{{ dashboard.error }}</p>
      <BaseButton variant="outline" @click="dashboard.loadDashboard()">Retry</BaseButton>
    </div>

    <!-- Content -->
    <div v-else class="dashboard-content">
      <!-- Top row: wallet + stats -->
      <div class="top-row">
        <!-- Wallet card -->
        <WalletCard
          :address="dashboard.walletAddress"
          :lovelace="dashboard.lovelace"
        />

        <!-- Stats -->
        <StatsRow
          :total="dashboard.totalNFTs"
          :minted="dashboard.mintedNFTs"
          :pending="dashboard.pendingNFTs"
        />
      </div>

      <!-- My NFTs -->
      <div class="section">
        <div class="section-header">
          <h2 class="section-title">My NFTs</h2>
          <router-link to="/mint" class="section-action">+ Mint New</router-link>
        </div>

        <!-- Empty state -->
        <div v-if="dashboard.nfts.length === 0" class="empty-nfts">
          <div class="empty-icon">🎨</div>
          <h3>No NFTs yet</h3>
          <p>Start minting your first NFT on Cardano Preprod.</p>
          <router-link to="/mint">
            <BaseButton variant="primary">Create Your First NFT</BaseButton>
          </router-link>
        </div>

        <!-- NFT Grid -->
        <div v-else class="nft-grid">
          <NFTCard
            v-for="nft in dashboard.nfts"
            :key="nft.id"
            :nft="nft"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useDashboardStore } from '@/stores/dashboard'
import WalletCard from '@/components/dashboard/WalletCard.vue'
import StatsRow from '@/components/dashboard/StatsRow.vue'
import NFTCard from '@/components/dashboard/NFTCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const dashboard = useDashboardStore()

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

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 80px;
  color: #888;
}
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #f0f0f0;
  border-top-color: #534AB7;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.error-state {
  text-align: center;
  padding: 48px;
  color: #d32f2f;
}

.dashboard-content { display: flex; flex-direction: column; gap: 32px; }

.top-row {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 24px;
  align-items: start;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-title { font-size: 18px; font-weight: 600; margin: 0; }
.section-action { font-size: 13px; color: #534AB7; text-decoration: none; }
.section-action:hover { text-decoration: underline; }

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
.empty-icon { font-size: 48px; }
.empty-nfts h3 { font-size: 18px; font-weight: 600; margin: 0; }
.empty-nfts p { font-size: 14px; color: #888; margin: 0; }

.nft-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}
</style>