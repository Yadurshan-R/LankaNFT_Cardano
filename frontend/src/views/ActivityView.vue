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
        @click="activeFilter = f.key; shownCount = PAGE_SIZE"
      >
        <component :is="f.icon" :size="13" />
        {{ f.label }}
        <span v-if="filterCount(f.key) > 0" class="filter-count">{{ filterCount(f.key) }}</span>
      </button>
    </div>

    <template v-if="dashboard.isLoading">
      <div v-for="n in 5" :key="n" class="skeleton-item">
        <div class="sk-thumb" />
        <div class="sk-body">
          <div class="sk-line sk-line--title" />
          <div class="sk-line sk-line--sub" />
          <div class="sk-line sk-line--tx" />
        </div>
      </div>
    </template>

    <div v-else-if="visibleGroups.length === 0" class="empty-state">
      <Activity :size="40" color="#ddd" />
      <p class="empty-title">{{ activeFilter === 'all' ? 'No activity yet' : `No ${activeFilter} events` }}</p>
      <p class="empty-desc">{{ activeFilter === 'all' ? 'Mint your first NFT to see activity here.' : 'Try a different filter.' }}</p>
      <button v-if="activeFilter !== 'all'" class="btn-outline" @click="activeFilter = 'all'">Show all activity</button>
      <router-link v-else to="/mint"><button class="btn-primary">Mint your first NFT</button></router-link>
    </div>

    <template v-else>
      <div v-for="group in visibleGroups" :key="group.date" class="date-group">
        <div class="date-label">{{ group.date }}</div>
        <div class="group-card">
          <div
            v-for="item in group.items"
            :key="item.id"
            class="activity-item activity-item--clickable"
            @click="selectedItem = item"
          >
            <div class="thumb-wrap">
              <img
                v-if="item.imageUrl && !item.imgError"
                :src="item.imageUrl"
                :alt="item.name"
                class="nft-thumb"
                @error="item.imgError = true"
              />
              <div v-else :class="['thumb-fallback', `thumb-fallback--${item.type}`]">
                <component :is="activityIcon(item.type)" :size="16" />
              </div>
              <div :class="['type-dot', `type-dot--${item.type}`]" />
            </div>

            <div class="item-main">
              <div class="item-top">
                <span class="item-name">{{ item.name }}</span>
                <span :class="['item-badge', `item-badge--${item.type}`]">{{ activityLabel(item.type) }}</span>
                <span v-if="item.price" class="item-price">{{ item.price }} ₳</span>
              </div>

              <div v-if="item.txHash" class="item-tx-row">
                <code class="item-tx-hash">{{ item.txHash.slice(0, 10) }}...{{ item.txHash.slice(-8) }}</code>
                <button class="copy-btn" @click.stop="copyTx(item)" :title="item.copied ? 'Copied!' : 'Copy tx hash'">
                  <Check v-if="item.copied" :size="11" color="#085041" />
                  <Copy v-else :size="11" color="#aaa" />
                </button>
                <a :href="cardanoscanTxUrl(item.txHash)" target="_blank" rel="noopener noreferrer" class="tx-scan-link" @click.stop>
                  <ExternalLink :size="11" /> Cardanoscan
                </a>
              </div>

              <div v-if="item.assetUrl" class="item-asset-row">
                <a :href="item.assetUrl" target="_blank" rel="noopener noreferrer" class="asset-link" @click.stop>
                  <ExternalLink :size="10" /> View Asset on Cardanoscan
                </a>
              </div>
            </div>

            <div class="item-time">{{ item.time }}</div>
          </div>
        </div>
      </div>

      <div class="list-footer">
        <span class="results-count">Showing {{ Math.min(shownCount, filteredItems.length) }} of {{ filteredItems.length }} events</span>
        <button v-if="shownCount < filteredItems.length" class="btn-load-more" @click="loadMore">Load more</button>
      </div>
    </template>

    <ActivityDetailModal :item="selectedItem" @close="selectedItem = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Activity, ArrowLeft, Check, Copy, ExternalLink,
  Layers, Send, ShoppingCart, Sparkles, Tag, X, Clock
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'
import { cardanoscanTxUrl, cardanoscanTokenUrl } from '@/utils/cardano'
import ActivityDetailModal from '@/components/shared/ActivityDetailModal.vue'

const dashboard    = useDashboardStore()
const router       = useRouter()
const activeFilter = ref('all')
const PAGE_SIZE    = 15
const shownCount   = ref(PAGE_SIZE)
const selectedItem = ref<any | null>(null)

const filters = [
  { key: 'all',         label: 'All',         icon: Layers },
  { key: 'minted',      label: 'Minted',      icon: Sparkles },
  { key: 'listed',      label: 'Listed',      icon: Tag },
  { key: 'pending',     label: 'Pending',     icon: Clock },
  { key: 'sold',        label: 'Sold',        icon: ShoppingCart },
  { key: 'transferred', label: 'Transferred', icon: Send },
  { key: 'cancelled',   label: 'Cancelled',   icon: X },
]

function ipfsToHttp(ipfs: string | null | undefined): string | null {
  if (!ipfs) return null
  return ipfs.startsWith('ipfs://') ? `https://gateway.pinata.cloud/ipfs/${ipfs.replace('ipfs://', '')}` : ipfs
}

function buildAssetUrl(policyId: string | null, tokenName: string | null): string | null {
  if (!policyId || !tokenName) return null
  return cardanoscanTokenUrl(policyId, tokenName)
}

const allItems = computed(() => {
  const items: any[] = []

  dashboard.uniqueNFTs.forEach((n: any) => {
    const nftType = n.status === 'pending'     ? 'pending'     :
                    n.status === 'listed'      ? 'listed'      :
                    n.status === 'transferred' ? 'transferred' : 'minted'
    items.push({
      id: `mint-${n.id}`, type: nftType,
      name: n.name || n.nft_name || 'Unnamed NFT',
      rawDate: n.created_at, time: formatTime(n.created_at),
      txHash: n.tx_hash || null, nftId: n.id,
      imageUrl: ipfsToHttp(n.image || n.image_ipfs),
      assetUrl: buildAssetUrl(n.policy_id, n.user_token_name || n.asset_name),
      imgError: false, copied: false,
    })
  })

  dashboard.myListings.forEach((l: any) => {
    const type = l.status === 'active' ? 'listed' : l.status === 'sold' ? 'sold' : l.status === 'cancelled' ? 'cancelled' : null
    if (!type) return
    items.push({
      id: `listing-${l.id}`, type,
      name: l.nft_name || 'Unnamed NFT',
      price: l.price_lovelace ? (l.price_lovelace / 1_000_000).toFixed(0) : null,
      rawDate: l.created_at, time: formatTime(l.created_at),
      txHash: l.listing_tx_hash || null, nftId: l.nft_id || null,
      imageUrl: ipfsToHttp(l.image_ipfs),
      assetUrl: buildAssetUrl(l.nft_policy_id, l.nft_asset_name),
      imgError: false, copied: false,
    })
  })

  return items.sort((a, b) => parseTimestamp(b.rawDate) - parseTimestamp(a.rawDate))
})

const filteredItems = computed(() =>
  activeFilter.value === 'all' ? allItems.value : allItems.value.filter((i) => i.type === activeFilter.value)
)
const paginatedItems = computed(() => filteredItems.value.slice(0, shownCount.value))

const visibleGroups = computed(() => {
  const groups: { date: string; items: any[] }[] = []
  const groupMap = new Map<string, any[]>()
  paginatedItems.value.forEach((item) => {
    const label = dateGroupLabel(item.rawDate)
    if (!groupMap.has(label)) { groupMap.set(label, []); groups.push({ date: label, items: groupMap.get(label)! }) }
    groupMap.get(label)!.push(item)
  })
  return groups
})

function filterCount(key: string): number {
  return key === 'all' ? allItems.value.length : allItems.value.filter((i) => i.type === key).length
}
function loadMore() { shownCount.value = Math.min(shownCount.value + PAGE_SIZE, filteredItems.value.length) }
async function copyTx(item: any) {
  await navigator.clipboard.writeText(item.txHash)
  item.copied = true
  setTimeout(() => (item.copied = false), 2000)
}

function parseTimestamp(dateStr: string): number {
  if (!dateStr) return 0
  let s = dateStr.toString().replace(' ', 'T')
  if (!s.endsWith('Z') && !s.includes('+')) s += 'Z'
  return new Date(s).getTime()
}
function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  const ts = parseTimestamp(dateStr), diff = Date.now() - ts
  if (diff < 0) return 'just now'
  const m = Math.floor(diff / 60_000), h = Math.floor(diff / 3_600_000), d = Math.floor(diff / 86_400_000)
  if (m < 2) return 'just now'
  if (m < 60) return `${m}m ago`
  if (h < 24) return `${h}h ago`
  if (d < 30) return `${d}d ago`
  return new Date(ts).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}
function dateGroupLabel(dateStr: string): string {
  if (!dateStr) return 'Unknown'
  const ts = parseTimestamp(dateStr), date = new Date(ts), now = new Date()
  const yesterday = new Date(now); yesterday.setDate(now.getDate() - 1)
  const sameDay = (a: Date, b: Date) => a.getDate() === b.getDate() && a.getMonth() === b.getMonth() && a.getFullYear() === b.getFullYear()
  if (sameDay(date, now)) return 'Today'
  if (sameDay(date, yesterday)) return 'Yesterday'
  return date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', ...(date.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }) })
}
function activityLabel(type: string): string {
  return ({ pending: 'Pending', minted: 'Minted', listed: 'Listed', sold: 'Sold', transferred: 'Transferred', cancelled: 'Cancelled' } as any)[type] ?? type
}
function activityIcon(type: string): any {
  return ({ pending: Clock, minted: Sparkles, listed: Tag, sold: ShoppingCart, transferred: Send, cancelled: X } as any)[type] ?? Activity
}

onMounted(async () => { if (dashboard.nfts.length === 0) await dashboard.loadDashboard() })
</script>

<style scoped>
.activity { padding: 28px 32px; max-width: 760px; }
.page-header { margin-bottom: 20px; }
.page-title { font-size: 22px; font-weight: 600; margin-bottom: 3px; }
.page-sub   { font-size: 13px; color: #888; }
.back-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  color: #888; text-decoration: none; margin-bottom: 20px; transition: all 0.15s;
}
.back-link:hover { background: #EEEDFE; color: #534AB7; }
.filter-bar { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 24px; }
.filter-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 12px; border-radius: 20px;
  border: 1px solid #e8e8e8; background: #fff;
  font-size: 12px; font-weight: 500; color: #666; cursor: pointer; transition: all 0.15s;
}
.filter-btn:hover   { border-color: #534AB7; color: #534AB7; }
.filter-btn--active { background: #EEEDFE; border-color: #534AB7; color: #534AB7; }
.filter-count {
  background: #534AB7; color: #fff;
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: 20px; margin-left: 2px;
}
.filter-btn--active .filter-count { background: #3d35a0; }
.date-group { margin-bottom: 20px; }
.date-label {
  font-size: 11px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.6px; color: #bbb; margin-bottom: 8px; padding-left: 2px;
}
.group-card { background: #fff; border: 1px solid #f0f0f0; border-radius: 14px; overflow: hidden; }
.activity-item {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 14px 16px; border-bottom: 1px solid #f8f8f8; transition: background 0.12s;
}
.activity-item:last-child { border-bottom: none; }
.activity-item--clickable { cursor: pointer; }
.activity-item--clickable:hover { background: #fafafa; }
.thumb-wrap { position: relative; flex-shrink: 0; width: 48px; height: 48px; }
.nft-thumb { width: 48px; height: 48px; border-radius: 10px; object-fit: cover; background: #f0f0f0; }
.thumb-fallback {
  width: 48px; height: 48px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
}
.thumb-fallback--minted      { background: #E1F5EE; color: #085041; }
.thumb-fallback--listed      { background: #E6F1FB; color: #185FA5; }
.thumb-fallback--sold        { background: #EEEDFE; color: #534AB7; }
.thumb-fallback--transferred { background: #FAEEDA; color: #854F0B; }
.thumb-fallback--cancelled   { background: #FCEBEB; color: #d32f2f; }
.type-dot {
  position: absolute; bottom: -2px; right: -2px;
  width: 14px; height: 14px; border-radius: 50%; border: 2px solid #fff;
}
.type-dot--minted      { background: #085041; }
.type-dot--listed      { background: #185FA5; }
.type-dot--sold        { background: #534AB7; }
.type-dot--transferred { background: #854F0B; }
.type-dot--cancelled   { background: #d32f2f; }
.item-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 5px; }
.item-top  { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.item-name {
  font-size: 13px; font-weight: 600; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 200px;
}
.item-badge { font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: 20px; flex-shrink: 0; }
.item-badge--minted      { background: #E1F5EE; color: #085041; }
.item-badge--listed      { background: #E6F1FB; color: #185FA5; }
.item-badge--sold        { background: #EEEDFE; color: #534AB7; }
.item-badge--transferred { background: #FAEEDA; color: #854F0B; }
.item-badge--cancelled   { background: #FCEBEB; color: #791F1F; }
.item-price { font-size: 12px; font-weight: 700; color: #534AB7; flex-shrink: 0; }
.item-tx-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.item-tx-hash {
  font-family: monospace; font-size: 11px; color: #888;
  background: #f5f5f5; padding: 2px 7px; border-radius: 5px;
}
.copy-btn {
  background: none; border: none; cursor: pointer;
  padding: 2px; display: flex; align-items: center; opacity: 0.7; transition: opacity 0.15s;
}
.copy-btn:hover { opacity: 1; }
.tx-scan-link {
  display: flex; align-items: center; gap: 3px;
  font-size: 11px; font-weight: 500; color: #534AB7;
  text-decoration: none; padding: 2px 7px;
  background: #EEEDFE; border-radius: 5px; transition: background 0.15s;
}
.tx-scan-link:hover { background: #dddcfc; }
.asset-link {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 10px; color: #aaa; text-decoration: none; transition: color 0.15s;
}
.asset-link:hover { color: #534AB7; }
.item-time { font-size: 11px; color: #bbb; white-space: nowrap; flex-shrink: 0; padding-top: 2px; }
.list-footer {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 2px; margin-top: 4px;
}
.results-count { font-size: 12px; color: #bbb; }
.btn-load-more {
  padding: 7px 16px; border: 1px solid #e8e8e8; border-radius: 20px;
  background: #fff; font-size: 12px; font-weight: 500; color: #666; cursor: pointer; transition: all 0.15s;
}
.btn-load-more:hover { border-color: #534AB7; color: #534AB7; }
.empty-state {
  display: flex; flex-direction: column; align-items: center;
  gap: 8px; padding: 64px 20px;
  border: 1.5px dashed #e8e8e8; border-radius: 14px; text-align: center;
}
.empty-title { font-size: 15px; font-weight: 600; color: #444; }
.empty-desc  { font-size: 13px; color: #aaa; }
.btn-primary {
  padding: 9px 18px; background: #534AB7; color: #fff;
  border: none; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}
.btn-outline {
  padding: 9px 18px; background: transparent; color: #534AB7;
  border: 1px solid #534AB7; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}
.skeleton-item { display: flex; align-items: center; gap: 12px; padding: 14px 0; border-bottom: 1px solid #f8f8f8; }
.sk-thumb { width: 48px; height: 48px; border-radius: 10px; background: #f0f0f0; flex-shrink: 0; }
.sk-body  { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.sk-line  {
  border-radius: 6px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite;
}
.sk-line--title { height: 13px; width: 40%; }
.sk-line--sub   { height: 11px; width: 20%; }
.sk-line--tx    { height: 10px; width: 55%; }
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
@media (max-width: 640px) {
  .activity { padding: 16px; }
  .filter-bar { gap: 4px; }
  .filter-btn { padding: 5px 10px; font-size: 11px; }
  .item-name  { max-width: 140px; }
}
</style>