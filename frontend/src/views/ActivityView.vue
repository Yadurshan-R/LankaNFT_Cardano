<template>
  <div class="activity">

    <router-link to="/" class="back-link" title="Back to Dashboard">
      <ArrowLeft :size="16" />
    </router-link>

    <div class="page-header">
      <div>
        <h1 class="page-title">Activity</h1>
        <p class="page-sub">Your complete NFT history on Cardano Preprod.</p>
      </div>
    </div>

    <div class="filter-bar">
      <button
        v-for="f in filters"
        :key="f.key"
        :class="['filter-btn', { 'filter-btn--active': activeFilter === f.key }]"
        @click="activeFilter = f.key"
      >
        <component :is="f.icon" :size="13" />
        {{ f.label }}
        <span v-if="filterCount(f.key) > 0" class="filter-count">
          {{ filterCount(f.key) }}
        </span>
      </button>
    </div>

    <template v-if="dashboard.isLoading">
      <div v-for="n in 5" :key="n" class="skeleton-item">
        <div class="sk-thumb" />
        <div class="sk-body">
          <div class="sk-line sk-line--title" />
          <div class="sk-line sk-line--sub" />
        </div>
        <div class="sk-line sk-line--time" />
      </div>
    </template>

    <div v-else-if="visibleGroups.length === 0" class="empty-state">
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

    <template v-else>
      <div v-for="group in visibleGroups" :key="group.date" class="date-group">

        <div class="date-label">{{ group.date }}</div>

        <div class="group-items">
          <div
            v-for="item in group.items"
            :key="item.id"
            class="activity-item"
          >
            <div :class="['activity-thumb', `activity-thumb--${item.type}`]">
              <component :is="activityIcon(item.type)" :size="15" />
            </div>

            <div class="activity-info">
              <div class="activity-name">{{ item.name }}</div>
              <div class="activity-sub">
                <span :class="['activity-badge', `activity-badge--${item.type}`]">
                  {{ activityLabel(item.type) }}
                </span>
                <span v-if="item.price" class="activity-price">{{ item.price }} ₳</span>
              </div>
            </div>

            <div class="activity-right">
              <span class="activity-time">{{ item.time }}</span>
              <a
                v-if="item.txHash"
                :href="`https://preprod.cardanoscan.io/transaction/${item.txHash}`"
                target="_blank"
                rel="noopener noreferrer"
                class="activity-tx"
                title="View transaction on Cardanoscan"
              >
                <ExternalLink :size="11" />
              </a>
            </div>
          </div>
        </div>
      </div>

      <div class="list-footer">
        <span class="results-count">
          Showing {{ shownCount }} of {{ filteredItems.length }} events
        </span>
        <button
          v-if="shownCount < filteredItems.length"
          class="btn-load-more"
          @click="loadMore"
        >
          Load more
        </button>
      </div>
    </template>

  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// ActivityView.vue
//
// Full activity history for the authenticated user.
// Events are sourced from the dashboard store (NFTs + listings).
//
// Key design decisions:
//   - Events grouped by date so the user can scan by day, not scroll forever
//   - Paginated with "Load more" — 15 items initially to keep page fast
//   - Single colored icon thumb per item (no redundant dot + icon pair)
//   - Tx hash shown as a small icon link, not a long string
// ─────────────────────────────────────────────────────────────────────────────

import { computed, onMounted, ref } from 'vue'
import {
  Activity, ArrowLeft, ExternalLink, Layers,
  Send, ShoppingCart, Sparkles, Tag, X,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'

const dashboard    = useDashboardStore()
const activeFilter = ref('all')

// How many items to show before "Load more"
const PAGE_SIZE    = 15
const shownCount   = ref(PAGE_SIZE)

const filters = [
  { key: 'all',         label: 'All',         icon: Layers },
  { key: 'minted',      label: 'Minted',      icon: Sparkles },
  { key: 'listed',      label: 'Listed',      icon: Tag },
  { key: 'sold',        label: 'Sold',        icon: ShoppingCart },
  { key: 'transferred', label: 'Transferred', icon: Send },
  { key: 'cancelled',   label: 'Cancelled',   icon: X },
]

// ── Data ─────────────────────────────────────────────────────────────────────

/**
 * Builds a unified, sorted activity list from NFTs and listings.
 * Sorted newest first.
 */
const allItems = computed(() => {
  const items: any[] = []

  // One minted event per NFT
  dashboard.uniqueNFTs.forEach((n: any) => {
    items.push({
      id:        `mint-${n.id}`,
      type:      'minted',
      name:      n.name,
      rawDate:   n.created_at,
      time:      formatTime(n.created_at),
      txHash:    n.tx_hash || null,
    })
  })

  // Listing events — one per listing record
  dashboard.myListings.forEach((l: any) => {
    const type =
      l.status === 'active'    ? 'listed'    :
      l.status === 'sold'      ? 'sold'      :
      l.status === 'cancelled' ? 'cancelled' : null

    if (!type) return

    items.push({
      id:      `listing-${l.id}`,
      type,
      name:    l.nft_name,
      price:   l.price_lovelace ? (l.price_lovelace / 1_000_000).toFixed(0) : null,
      rawDate: l.created_at,
      time:    formatTime(l.created_at),
      txHash:  null,
    })
  })

  // Sort newest first — items with no date fall to the bottom
  return items.sort((a, b) => {
    const ta = parseTimestamp(a.rawDate)
    const tb = parseTimestamp(b.rawDate)
    return tb - ta
  })
})

/** Filter by active tab */
const filteredItems = computed(() => {
  if (activeFilter.value === 'all') return allItems.value
  return allItems.value.filter((i) => i.type === activeFilter.value)
})

/** Items limited to shownCount for pagination */
const paginatedItems = computed(() =>
  filteredItems.value.slice(0, shownCount.value)
)

/**
 * Groups paginated items by human-readable date label.
 * e.g. "Today", "Yesterday", "May 25", "Apr 12, 2025"
 */
const visibleGroups = computed(() => {
  const groups: { date: string; items: any[] }[] = []
  const groupMap = new Map<string, any[]>()

  paginatedItems.value.forEach((item) => {
    const label = dateGroupLabel(item.rawDate)
    if (!groupMap.has(label)) {
      groupMap.set(label, [])
      groups.push({ date: label, items: groupMap.get(label)! })
    }
    groupMap.get(label)!.push(item)
  })

  return groups
})

/** Count for each filter tab badge */
function filterCount(key: string): number {
  if (key === 'all') return allItems.value.length
  return allItems.value.filter((i) => i.type === key).length
}

function loadMore() {
  shownCount.value = Math.min(
    shownCount.value + PAGE_SIZE,
    filteredItems.value.length
  )
}

// ── Time helpers ──────────────────────────────────────────────────────────────

/** Parse a timestamp string safely, always treating it as UTC */
function parseTimestamp(dateStr: string): number {
  if (!dateStr) return 0
  let s = dateStr.toString().replace(' ', 'T')
  if (!s.endsWith('Z') && !s.includes('+')) s += 'Z'
  return new Date(s).getTime()
}

/** Relative time string — "just now", "5m ago", "3h ago", "2d ago" */
function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  const ts   = parseTimestamp(dateStr)
  const diff = Date.now() - ts
  if (diff < 0)           return 'just now'
  const minutes = Math.floor(diff / 60_000)
  const hours   = Math.floor(diff / 3_600_000)
  const days    = Math.floor(diff / 86_400_000)
  if (minutes < 2)  return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24)   return `${hours}h ago`
  if (days < 30)    return `${days}d ago`
  return new Date(parseTimestamp(dateStr)).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric',
  })
}

/**
 * Returns a human-readable date group label for an event.
 * Groups: "Today", "Yesterday", or "May 25" / "Apr 12, 2025"
 */
function dateGroupLabel(dateStr: string): string {
  if (!dateStr) return 'Unknown'
  const ts    = parseTimestamp(dateStr)
  const date  = new Date(ts)
  const now   = new Date()

  const isToday =
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear()

  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  const isYesterday =
    date.getDate() === yesterday.getDate() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getFullYear() === yesterday.getFullYear()

  if (isToday)      return 'Today'
  if (isYesterday) return 'Yesterday'

  // Show year only if different from current year
  const sameYear = date.getFullYear() === now.getFullYear()
  return date.toLocaleDateString('en-US', {
    month: 'long',
    day:   'numeric',
    ...(sameYear ? {} : { year: 'numeric' }),
  })
}

// ── Labels & icons ────────────────────────────────────────────────────────────

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
.activity { padding: 28px 32px; max-width: 760px; }

.page-header  { margin-bottom: 20px; }
.page-title   { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub     { font-size: 13px; color: #888; }

.back-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  color: #888;
  text-decoration: none;
  margin-bottom: 20px;
  transition: all 0.15s;
}
.back-link:hover {
  background: #EEEDFE;
  color: #534AB7;
}

/* ── Filter bar ── */
.filter-bar {
  display: flex; gap: 6px; flex-wrap: wrap;
  margin-bottom: 24px;
}
.filter-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 12px; border-radius: 20px;
  border: 1px solid #e8e8e8; background: #fff;
  font-size: 12px; font-weight: 500; color: #666;
  cursor: pointer; transition: all 0.15s;
}
.filter-btn:hover       { border-color: #534AB7; color: #534AB7; }
.filter-btn--active     { background: #EEEDFE; border-color: #534AB7; color: #534AB7; }
.filter-count {
  background: #534AB7; color: #fff;
  font-size: 10px; font-weight: 600;
  padding: 1px 6px; border-radius: 20px;
  margin-left: 2px;
}
.filter-btn--active .filter-count { background: #3d35a0; }

/* ── Date group ── */
.date-group { margin-bottom: 24px; }

.date-label {
  font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.6px;
  color: #bbb; margin-bottom: 8px;
  padding-left: 2px;
}

.group-items {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  overflow: hidden;
}

/* ── Activity item ── */
.activity-item {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #f8f8f8;
  transition: background 0.1s;
}
.activity-item:last-child { border-bottom: none; }
.activity-item:hover      { background: #fafafa; }

/* Single colored icon — replaces redundant dot + thumb combo */
.activity-thumb {
  width: 38px; height: 38px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.activity-thumb--minted      { background: #E1F5EE; color: #085041; }
.activity-thumb--listed      { background: #E6F1FB; color: #185FA5; }
.activity-thumb--sold        { background: #EEEDFE; color: #534AB7; }
.activity-thumb--transferred { background: #FAEEDA; color: #854F0B; }
.activity-thumb--cancelled   { background: #FCEBEB; color: #d32f2f; }

.activity-info { flex: 1; min-width: 0; }

.activity-name {
  font-size: 13px; font-weight: 500; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-bottom: 3px;
}

.activity-sub { display: flex; align-items: center; gap: 8px; }

.activity-badge {
  font-size: 10px; font-weight: 600;
  padding: 2px 8px; border-radius: 20px;
}
.activity-badge--minted      { background: #E1F5EE; color: #085041; }
.activity-badge--listed      { background: #E6F1FB; color: #185FA5; }
.activity-badge--sold        { background: #EEEDFE; color: #534AB7; }
.activity-badge--transferred { background: #FAEEDA; color: #854F0B; }
.activity-badge--cancelled   { background: #FCEBEB; color: #791F1F; }

.activity-price {
  font-size: 11px; font-weight: 600; color: #534AB7;
}

/* Right side — time + tx icon */
.activity-right {
  display: flex; align-items: center; gap: 8px;
  flex-shrink: 0;
}
.activity-time { font-size: 11px; color: #bbb; white-space: nowrap; }
.activity-tx {
  display: flex; align-items: center;
  color: #ccc; text-decoration: none; transition: color 0.15s;
}
.activity-tx:hover { color: #534AB7; }

/* ── List footer ── */
.list-footer {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 2px; margin-top: 4px;
}
.results-count { font-size: 12px; color: #bbb; }
.btn-load-more {
  padding: 7px 16px;
  border: 1px solid #e8e8e8; border-radius: 20px;
  background: #fff; font-size: 12px; font-weight: 500;
  color: #666; cursor: pointer; transition: all 0.15s;
}
.btn-load-more:hover { border-color: #534AB7; color: #534AB7; }

/* ── Empty state ── */
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

/* ── Skeleton ── */
.skeleton-item {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 0; border-bottom: 1px solid #f8f8f8;
}
.sk-thumb { width: 38px; height: 38px; border-radius: 10px; background: #f0f0f0; flex-shrink: 0; }
.sk-body  { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.sk-line  {
  border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.sk-line--title { height: 13px; width: 45%; }
.sk-line--sub   { height: 11px; width: 25%; }
.sk-line--time  { height: 11px; width: 44px; flex-shrink: 0; }

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@media (max-width: 640px) {
  .activity   { padding: 16px; }
  .filter-bar { gap: 4px; }
  .filter-btn { padding: 5px 10px; font-size: 11px; }
}
</style>