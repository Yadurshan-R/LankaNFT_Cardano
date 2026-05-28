<template>
  <div class="activity">
    <div class="page-header">
      <div>
        <h1 class="page-title">Activity</h1>
        <p class="page-sub">Your complete NFT history on Cardano Preprod.</p>
      </div>
    </div>

    <!-- Filter bar -->
    <div class="filter-bar">
      <button
        v-for="f in filters"
        :key="f.key"
        :class="['filter-btn', { 'filter-btn--active': activeFilter === f.key }]"
        @click="activeFilter = f.key"
      >
        <component :is="f.icon" :size="13" />
        {{ f.label }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="dashboard.isLoading" class="activity-list">
      <div v-for="n in 6" :key="n" class="activity-item activity-item--skeleton">
        <div class="sk-dot" />
        <div class="sk-thumb" />
        <div class="sk-body">
          <div class="sk-line sk-line--title" />
          <div class="sk-line sk-line--sub" />
        </div>
        <div class="sk-line sk-line--time" />
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredItems.length === 0" class="empty-state">
      <Activity :size="40" color="#ddd" />
      <p class="empty-title">
        {{ activeFilter === 'all' ? 'No activity yet' : `No ${activeFilter} events` }}
      </p>
      <p class="empty-desc">
        {{ activeFilter === 'all'
          ? 'Mint your first NFT to see activity here.'
          : 'Try a different filter.' }}
      </p>
      <button v-if="activeFilter !== 'all'" class="btn-outline" @click="activeFilter = 'all'">
        Show all activity
      </button>
      <router-link v-else to="/mint">
        <button class="btn-primary">Mint your first NFT</button>
      </router-link>
    </div>

    <!-- Activity list -->
    <div v-else class="activity-list">
      <div
        v-for="item in filteredItems"
        :key="item.id"
        class="activity-item"
      >
        <div :class="['activity-dot', `activity-dot--${item.type}`]" />
        <div :class="['activity-thumb', `activity-thumb--${item.type}`]">
          <component :is="activityIcon(item.type)" :size="14" />
        </div>
        <div class="activity-info">
          <div class="activity-top">
            <span class="activity-name">{{ item.name }}</span>
            <span :class="['activity-badge', `activity-badge--${item.type}`]">
              {{ activityLabel(item.type) }}
            </span>
          </div>
          <div class="activity-sub">
            <span v-if="item.price" class="activity-price">{{ item.price }} ₳</span>
            <span v-if="item.address" class="activity-addr">→ {{ item.address }}</span>
            <a
              v-if="item.txHash"
              :href="`https://preprod.cardanoscan.io/transaction/${item.txHash}`"
              target="_blank"
              rel="noopener noreferrer"
              class="activity-tx"
            >
              <ExternalLink :size="10" /> View tx
            </a>
          </div>
        </div>
        <div class="activity-time">{{ item.time }}</div>
      </div>
    </div>

    <div v-if="!dashboard.isLoading && filteredItems.length > 0" class="results-count">
      {{ filteredItems.length }} event{{ filteredItems.length !== 1 ? 's' : '' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  Activity, ExternalLink, Layers,
  Send, ShoppingCart, Sparkles, Tag, X,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'

const dashboard = useDashboardStore()
const activeFilter = ref('all')

const filters = [
  { key: 'all',         label: 'All',         icon: Layers },
  { key: 'minted',      label: 'Minted',      icon: Sparkles },
  { key: 'listed',      label: 'Listed',      icon: Tag },
  { key: 'sold',        label: 'Sold',        icon: ShoppingCart },
  { key: 'transferred', label: 'Transferred', icon: Send },
  { key: 'cancelled',   label: 'Cancelled',   icon: X },
]

// Build a unified activity list from NFTs + listings
const allItems = computed(() => {
  const items: any[] = []

  // Minted events — one per NFT
  dashboard.uniqueNFTs.forEach((n: any) => {
    items.push({
      id:     `mint-${n.id}`,
      type:   'minted',
      name:   n.name,
      time:   formatTime(n.created_at),
      txHash: n.tx_hash,
    })
  })

  // Listing events
  dashboard.myListings.forEach((l: any) => {
    if (l.status === 'active' || l.status === 'sold' || l.status === 'cancelled') {
      items.push({
        id:     `list-${l.id}`,
        type:   l.status === 'active' ? 'listed' : l.status,
        name:   l.nft_name,
        price:  l.status !== 'cancelled' ? (l.price_lovelace / 1_000_000).toFixed(0) : null,
        time:   formatTime(l.created_at),
        txHash: null,
      })
    }
  })

  // Sort newest first — items with no parseable time go to bottom
  return items.sort((a, b) => {
    const ta = parseTime(a.rawTime)
    const tb = parseTime(b.rawTime)
    return tb - ta
  })
})

const filteredItems = computed(() => {
  if (activeFilter.value === 'all') return allItems.value
  return allItems.value.filter((i) => i.type === activeFilter.value)
})

function parseTime(dateStr: string): number {
  if (!dateStr) return 0
  let s = dateStr.toString().replace(' ', 'T')
  if (!s.endsWith('Z') && !s.includes('+')) s += 'Z'
  return new Date(s).getTime()
}

function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  let s = dateStr.toString().replace(' ', 'T')
  if (!s.endsWith('Z') && !s.includes('+')) s += 'Z'
  const diff = Date.now() - new Date(s).getTime()
  if (diff < 0) return 'just now'
  const minutes = Math.floor(diff / 60_000)
  const hours   = Math.floor(diff / 3_600_000)
  const days    = Math.floor(diff / 86_400_000)
  if (minutes < 2)  return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24)   return `${hours}h ago`
  if (days < 30)    return `${days}d ago`
  return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
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
    minted: Sparkles, listed: Tag, sold: ShoppingCart,
    transferred: Send, cancelled: X,
  }
  return map[type] ?? Activity
}

onMounted(async () => {
  if (dashboard.nfts.length === 0) await dashboard.loadDashboard()
})
</script>

<style scoped>
.activity { padding: 28px 32px; max-width: 800px; }

.page-header  { margin-bottom: 20px; }
.page-title   { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub     { font-size: 13px; color: #888; }

/* Filter bar */
.filter-bar {
  display: flex; gap: 6px; flex-wrap: wrap;
  margin-bottom: 20px;
}
.filter-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 13px; border-radius: 20px;
  border: 1px solid #e8e8e8; background: #fff;
  font-size: 12px; font-weight: 500; color: #666;
  cursor: pointer; transition: all 0.15s;
}
.filter-btn:hover { border-color: #534AB7; color: #534AB7; }
.filter-btn--active { background: #EEEDFE; border-color: #534AB7; color: #534AB7; }

/* Activity list */
.activity-list { display: flex; flex-direction: column; gap: 2px; }

.activity-item {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  transition: border-color 0.15s;
}
.activity-item:hover { border-color: #e0e0e0; }

.activity-dot {
  width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0;
}
.activity-dot--minted      { background: #085041; }
.activity-dot--listed      { background: #185FA5; }
.activity-dot--sold        { background: #534AB7; }
.activity-dot--transferred { background: #854F0B; }
.activity-dot--cancelled   { background: #d32f2f; }

.activity-thumb {
  width: 36px; height: 36px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.activity-thumb--minted      { background: #E1F5EE; color: #085041; }
.activity-thumb--listed      { background: #E6F1FB; color: #185FA5; }
.activity-thumb--sold        { background: #EEEDFE; color: #534AB7; }
.activity-thumb--transferred { background: #FAEEDA; color: #854F0B; }
.activity-thumb--cancelled   { background: #FCEBEB; color: #d32f2f; }

.activity-info { flex: 1; min-width: 0; }

.activity-top {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 3px;
}
.activity-name {
  font-size: 13px; font-weight: 500; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.activity-badge {
  font-size: 10px; font-weight: 600;
  padding: 2px 8px; border-radius: 20px; flex-shrink: 0;
}
.activity-badge--minted      { background: #E1F5EE; color: #085041; }
.activity-badge--listed      { background: #E6F1FB; color: #185FA5; }
.activity-badge--sold        { background: #EEEDFE; color: #534AB7; }
.activity-badge--transferred { background: #FAEEDA; color: #854F0B; }
.activity-badge--cancelled   { background: #FCEBEB; color: #791F1F; }

.activity-sub {
  display: flex; align-items: center; gap: 8px;
  font-size: 11px; color: #aaa;
}
.activity-price { color: #534AB7; font-weight: 600; }
.activity-addr  { font-family: monospace; font-size: 10px; }
.activity-tx {
  display: inline-flex; align-items: center; gap: 3px;
  color: #534AB7; text-decoration: none; font-size: 11px;
  transition: opacity 0.15s;
}
.activity-tx:hover { opacity: 0.7; }

.activity-time {
  font-size: 11px; color: #bbb;
  white-space: nowrap; flex-shrink: 0;
}

/* Results count */
.results-count { margin-top: 16px; font-size: 12px; color: #aaa; text-align: center; }

/* Empty state */
.empty-state {
  display: flex; flex-direction: column; align-items: center;
  gap: 8px; padding: 64px 20px;
  border: 1.5px dashed #e8e8e8; border-radius: 14px;
  text-align: center;
}
.empty-title { font-size: 15px; font-weight: 600; color: #444; }
.empty-desc  { font-size: 13px; color: #aaa; }
.btn-primary {
  padding: 9px 18px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px;
  font-size: 13px; font-weight: 600; cursor: pointer;
}
.btn-outline {
  padding: 9px 18px; background: transparent;
  color: #534AB7; border: 1px solid #534AB7;
  border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}

/* Skeleton */
.activity-item--skeleton { pointer-events: none; }
.sk-dot   { width: 7px; height: 7px; border-radius: 50%; background: #f0f0f0; flex-shrink: 0; }
.sk-thumb { width: 36px; height: 36px; border-radius: 10px; background: #f0f0f0; flex-shrink: 0; }
.sk-body  { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.sk-line  {
  border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.sk-line--title { height: 13px; width: 55%; }
.sk-line--sub   { height: 11px; width: 35%; }
.sk-line--time  { height: 11px; width: 48px; flex-shrink: 0; }
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@media (max-width: 640px) {
  .activity    { padding: 16px; }
  .filter-bar  { gap: 4px; }
  .filter-btn  { padding: 5px 10px; font-size: 11px; }
}
</style>