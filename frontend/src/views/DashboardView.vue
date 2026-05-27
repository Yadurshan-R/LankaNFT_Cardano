<template>
  <div class="dashboard">

    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-sub">Welcome back. Here is your NFT portfolio.</p>
      </div>
      <router-link to="/mint">
        <button class="btn-create">
          <Plus :size="15" />
          Create NFT
        </button>
      </router-link>
    </div>

    <template v-if="dashboard.isLoading">
      <div class="stats-row">
        <div v-for="n in 4" :key="n" class="skeleton skeleton--stat" />
      </div>
      <div class="main-grid">
        <div class="skeleton skeleton--panel" />
        <div class="skeleton skeleton--panel-sm" />
      </div>
    </template>

    <div v-else-if="dashboard.error" class="error-state">
      <AlertCircle :size="32" />
      <p>{{ dashboard.error }}</p>
      <button class="btn-retry" @click="dashboard.loadDashboard()">Try again</button>
    </div>

    <template v-else>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-icon stat-icon--purple"><Layers :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.uniqueNFTs.length }}</div><div class="stat-lbl">Total NFTs</div></div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--green"><CheckCircle :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.mintedOnly.length }}</div><div class="stat-lbl">Minted</div></div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--blue"><Tag :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.listedOnly.length }}</div><div class="stat-lbl">Listed</div></div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--amber"><Clock :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.pendingOnly.length }}</div><div class="stat-lbl">Pending</div></div>
        </div>
      </div>

      <div class="main-grid">

        <div class="panel">
          <div class="panel-header">
            <div class="panel-title"><ImageIcon :size="15" /> My NFTs</div>
            <div class="filter-tabs">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                :class="['filter-tab', { 'filter-tab--active': activeTab === tab.key }]"
                @click="activeTab = tab.key"
              >
                {{ tab.label }}
                <span class="tab-count">{{ tab.count }}</span>
              </button>
            </div>
          </div>

          <div v-if="activeNFTs.length === 0" class="empty-state">
            <ImageIcon :size="40" color="#ddd" />
            <p class="empty-title">{{ emptyTitle }}</p>
            <p class="empty-desc">{{ emptyDesc }}</p>
            <router-link v-if="activeTab === 'all'" to="/mint">
              <button class="btn-create">
                <Plus :size="14" /> Mint your first NFT
              </button>
            </router-link>
          </div>

          <div v-else class="nft-grid">
            <router-link
              v-for="nft in activeNFTs"
              :key="nft.id"
              :to="`/nft/${nft.id}`"
              class="nft-card"
            >
              <div class="nft-img">
                <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="nft-img-actual" />
                <ImageIcon v-else :size="28" color="#ddd" />
                <div class="nft-hover-overlay">
                  <span class="nft-hover-label">View details</span>
                </div>
              </div>
              <div class="nft-info">
                <div class="nft-name-row">
                  <span class="nft-name">{{ nft.name }}</span>
                  <span :class="['nft-badge', `nft-badge--${nft.status}`]">{{ nft.status }}</span>
                </div>
                <div class="nft-royalty">{{ nft.royalties }}% royalty</div>
                <div class="nft-price">{{ listingPrice(nft) }}</div>
              </div>
            </router-link>

            <router-link to="/mint" class="nft-card nft-card--new">
              <Plus :size="22" />
              <span>Mint new</span>
            </router-link>
          </div>
        </div>

        <div class="panel panel--sticky">
          <div class="panel-header">
            <div class="panel-title"><Activity :size="15" /> Activity</div>
            <router-link to="/activity" class="view-all">View all</router-link>
          </div>

          <div v-if="recentActivity.length === 0" class="empty-state empty-state--sm">
            <Activity :size="28" color="#ddd" />
            <p class="empty-desc">No activity yet</p>
          </div>

          <div v-else class="activity-list">
            <div v-for="item in recentActivity" :key="item.id" class="activity-item">
              <div :class="['activity-dot', `activity-dot--${item.type}`]" />
              <div class="activity-thumb">
                <component :is="activityIcon(item.type)" :size="13" />
              </div>
              <div class="activity-info">
                <div class="activity-type">{{ activityLabel(item.type) }}</div>
                <div class="activity-name">{{ item.name }}</div>
              </div>
              <div class="activity-time">{{ item.time }}</div>
            </div>
          </div>
        </div>

      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  Activity, AlertCircle, CheckCircle, Clock,
  Image as ImageIcon, Layers, Plus,
  Send, ShoppingCart, Sparkles, Tag,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'

const dashboard = useDashboardStore()
const activeTab  = ref('all')

// ── Tabs ──────────────────────────────────────────────────────────────────
const tabs = computed(() => [
  { key: 'all',     label: 'All',     count: dashboard.uniqueNFTs.length },
  { key: 'minted',  label: 'Minted',  count: dashboard.mintedOnly.length },
  { key: 'listed',  label: 'Listed',  count: dashboard.listedOnly.length },
  { key: 'pending', label: 'Pending', count: dashboard.pendingOnly.length },
])

/**
 * Computes the list of NFTs to display based on the currently selected tab.
 * Falls back to the entire unique NFT collection if the tab key is not found.
 */
const activeNFTs = computed(() => {
  const map: Record<string, any[]> = {
    minted:  dashboard.mintedOnly,
    listed:  dashboard.listedOnly,
    pending: dashboard.pendingOnly,
  }
  return map[activeTab.value] ?? dashboard.uniqueNFTs
})

// ── Empty state text ──────────────────────────────────────────────────────
const emptyTitle = computed(() => {
  const map: Record<string, string> = {
    minted: 'No minted NFTs', listed: 'No listed NFTs', pending: 'No pending NFTs',
  }
  return map[activeTab.value] ?? 'No NFTs yet'
})

const emptyDesc = computed(() => {
  const map: Record<string, string> = {
    minted:  'Minted NFTs will appear here.',
    listed:  'List an NFT for sale from its detail page.',
    pending: 'NFTs being confirmed will appear here.',
  }
  return map[activeTab.value] ?? 'Start by minting your first NFT on Cardano.'
})

// ── Helpers ───────────────────────────────────────────────────────────────

/**
 * Converts a native IPFS URI (ipfs://...) to a public HTTP gateway URL for rendering in the browser.
 * @param uri - The raw image URI from the NFT metadata.
 */
function ipfsToHttp(uri: string): string {
  if (!uri) return ''
  return uri.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${uri.replace('ipfs://', '')}`
    : uri
}

/**
 * Determines the display price for an NFT.
 * Searches the active listings store array for a match. If found, formats the lovelace value to ADA.
 * @param nft - The NFT object being evaluated.
 */
function listingPrice(nft: any): string {
  if (nft.status !== 'listed') return 'Not listed'
  
  const listing = dashboard.myListings.find(
    (l: any) => l.nft_id === nft.id && l.status === 'active'
  )
  
  if (!listing) return 'Listed'
  
  // Convert lovelace to ADA and format with 2 decimal places
  const ada = (listing.price_lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 0, maximumFractionDigits: 2,
  })
  return `${ada} ₳`
}

// ── Activity feed — derived from store data ───────────────────────────────

/**
 * Aggregates recent mints and listings into a single chronological activity feed.
 * Takes the top 3 mints and top 2 active listings to create a dynamic sidebar preview.
 */
const recentActivity = computed(() => {
  const items: any[] = []

  // Gather recent mints
  dashboard.uniqueNFTs.slice(0, 3).forEach((n: any) => {
    // DEBUG: Output raw timestamp to console for debugging timezone issues
    if (items.length === 0) {
      console.log('created_at raw:', JSON.stringify(n.created_at), typeof n.created_at)
    }
    
    items.push({
      id: `mint-${n.id}`, type: 'minted',
      name: n.name, time: formatTime(n.created_at),
    })
  })

  // Gather recent listings
  dashboard.myListings.filter((l: any) => l.status === 'active').slice(0, 2).forEach((l: any) => {
    const ada = (l.price_lovelace / 1_000_000).toFixed(0)
    items.push({
      id: `list-${l.id}`, type: 'listed',
      name: `${l.nft_name} · ${ada} ₳`, time: formatTime(l.created_at),
    })
  })

  return items.slice(0, 5)
})

/**
 * Formats a local database timestamp string into a relative time (e.g., "5m ago").
 * The DB string is in local time, so replacing the space with 'T' ensures it parses 
 * correctly as an ISO string without mistakenly treating it as UTC.
 * @param dateStr - The timestamp string from the database.
 */
function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  
  // DB timestamps are in local time — parse as local, not UTC
  const localStr = dateStr.toString().replace(' ', 'T')
  const diff = Date.now() - new Date(localStr).getTime()
  
  if (diff < 0) return 'just now'
  
  const minutes = Math.floor(diff / 60_000)
  const hours   = Math.floor(diff / 3_600_000)
  const days    = Math.floor(diff / 86_400_000)
  
  if (minutes < 2)  return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24)   return `${hours}h ago`
  return `${days}d ago`
}

function activityLabel(type: string): string {
  const map: Record<string, string> = {
    minted: 'Minted', listed: 'Listed', sold: 'Sold',
    transferred: 'Transferred', cancelled: 'Cancelled',
  }
  return map[type] ?? type
}

function activityIcon(type: string): any {
  const map: Record<string, any> = {
    minted: Sparkles, listed: Tag, sold: ShoppingCart, transferred: Send, cancelled: Activity,
  }
  return map[type] ?? Activity
}

onMounted(() => dashboard.loadDashboard())
</script>

<style scoped>
.dashboard { padding: 28px 32px; max-width: 1200px; }

/* Header */
.page-header    { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-title     { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub       { font-size: 13px; color: #888; }
.btn-create     { display: flex; align-items: center; gap: 6px; background: #534AB7; color: #fff; border: none; border-radius: 10px; padding: 9px 16px; font-size: 13px; font-weight: 600; cursor: pointer; transition: background 0.15s; }
.btn-create:hover { background: #3d35a0; }

/* Stats */
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 20px; }
.stat-card { background: #fff; border-radius: 12px; padding: 14px 16px; display: flex; align-items: center; gap: 12px; border: 1px solid #f0f0f0; }
.stat-icon { width: 38px; height: 38px; border-radius: 10px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.stat-icon--purple { background: #EEEDFE; color: #534AB7; }
.stat-icon--green  { background: #E1F5EE; color: #085041; }
.stat-icon--blue   { background: #E6F1FB; color: #185FA5; }
.stat-icon--amber  { background: #FAEEDA; color: #854F0B; }
.stat-num { font-size: 22px; font-weight: 600; line-height: 1.2; }
.stat-lbl { font-size: 11px; color: #888; }

/* Main grid */
.main-grid { display: grid; grid-template-columns: 1fr 280px; gap: 16px; align-items: start; }

/* Panel */
.panel         { background: #fff; border-radius: 14px; padding: 18px; border: 1px solid #f0f0f0; }
.panel--sticky { position: sticky; top: 20px; }
.panel-header  { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.panel-title   { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; }
.panel-title svg { color: #534AB7; }
.view-all      { font-size: 11px; color: #534AB7; font-weight: 500; }

/* Filter tabs */
.filter-tabs       { display: flex; gap: 4px; }
.filter-tab        { display: flex; align-items: center; gap: 5px; padding: 4px 10px; border-radius: 20px; font-size: 11px; font-weight: 500; border: 1px solid #e8e8e8; background: transparent; color: #888; cursor: pointer; transition: all 0.15s; }
.filter-tab:hover  { border-color: #534AB7; color: #534AB7; }
.filter-tab--active { background: #534AB7; color: #fff; border-color: #534AB7; }
.tab-count         { font-size: 10px; background: #f0f0f0; color: #666; padding: 0 4px; border-radius: 8px; }
.filter-tab--active .tab-count { background: rgba(255,255,255,0.25); color: #fff; }

/* NFT grid */
.nft-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(148px, 1fr)); gap: 12px; }

.nft-card { border: 1px solid #f0f0f0; border-radius: 12px; overflow: hidden; cursor: pointer; text-decoration: none; color: inherit; display: block; transition: all 0.15s; }
.nft-card:hover { border-color: #534AB7; transform: translateY(-2px); }

/* Image — hover overlay is hidden by default, shown on card hover via parent selector */
.nft-img              { aspect-ratio: 1; background: #f8f8f8; display: flex; align-items: center; justify-content: center; position: relative; overflow: hidden; }
.nft-img-actual       { width: 100%; height: 100%; object-fit: cover; }
.nft-hover-overlay    { position: absolute; inset: 0; background: rgba(83,74,183,0.88); display: flex; align-items: center; justify-content: center; opacity: 0; transition: opacity 0.15s; }
.nft-card:hover .nft-hover-overlay { opacity: 1; }
.nft-hover-label      { background: #fff; color: #534AB7; padding: 6px 12px; border-radius: 8px; font-size: 11px; font-weight: 600; }

.nft-info    { padding: 10px 12px; }
.nft-name-row { display: flex; align-items: center; justify-content: space-between; gap: 4px; margin-bottom: 3px; }
.nft-name    { font-size: 12px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.nft-badge   { font-size: 9px; font-weight: 600; padding: 2px 6px; border-radius: 20px; white-space: nowrap; flex-shrink: 0; }
.nft-badge--minted  { background: #E1F5EE; color: #085041; }
.nft-badge--listed  { background: #EEEDFE; color: #534AB7; }
.nft-badge--pending { background: #FFF8E1; color: #7a5c00; }
.nft-royalty { font-size: 10px; color: #aaa; margin-bottom: 3px; }
.nft-price   { font-size: 12px; font-weight: 600; color: #534AB7; }

/* Mint new card */
.nft-card--new { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 130px; border: 1.5px dashed #e0e0e0; color: #ccc; gap: 6px; font-size: 11px; font-weight: 500; }
.nft-card--new:hover { border-color: #534AB7; color: #534AB7; transform: none; }

/* Activity */
.activity-list  { display: flex; flex-direction: column; }
.activity-item  { display: flex; align-items: center; gap: 10px; padding: 10px 0; border-bottom: 1px solid #f5f5f5; }
.activity-item:last-child { border-bottom: none; }
.activity-dot   { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.activity-dot--minted      { background: #085041; }
.activity-dot--listed      { background: #185FA5; }
.activity-dot--sold        { background: #534AB7; }
.activity-dot--transferred { background: #854F0B; }
.activity-dot--cancelled   { background: #d32f2f; }
.activity-thumb { width: 30px; height: 30px; background: #f8f8f8; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: #534AB7; }
.activity-info  { flex: 1; min-width: 0; }
.activity-type  { font-size: 11px; font-weight: 600; }
.activity-name  { font-size: 10px; color: #888; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-time  { font-size: 10px; color: #aaa; white-space: nowrap; }

/* Empty states */
.empty-state     { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 48px 20px; border: 1.5px dashed #e8e8e8; border-radius: 12px; text-align: center; }
.empty-state--sm { padding: 24px 16px; }
.empty-title     { font-size: 14px; font-weight: 600; color: #444; }
.empty-desc      { font-size: 12px; color: #aaa; }

/* Error */
.error-state { display: flex; flex-direction: column; align-items: center; gap: 12px; padding: 64px; text-align: center; color: #d32f2f; font-size: 14px; }
.btn-retry   { padding: 8px 20px; border: 1px solid #d32f2f; border-radius: 8px; background: transparent; color: #d32f2f; font-size: 13px; cursor: pointer; }

/* Skeleton */
.skeleton         { border-radius: 12px; background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%); background-size: 200% 100%; animation: shimmer 1.4s infinite; }
.skeleton--stat   { height: 70px; }
.skeleton--panel  { height: 400px; }
.skeleton--panel-sm { height: 300px; }
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

/* Mobile */
@media (max-width: 900px) {
  .dashboard  { padding: 16px; }
  .stats-row  { grid-template-columns: repeat(2, 1fr); }
  .main-grid  { grid-template-columns: 1fr; }
  .panel--sticky { position: static; }
}
</style>