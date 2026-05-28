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
      <!-- ── Stat cards ── -->
      <div class="stats-row">
        <div :class="['stat-card', { 'stat-card--active': activeTab === 'all' }]" @click="activeTab = 'all'">
          <div class="stat-icon stat-icon--purple"><Layers :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.uniqueNFTs.length }}</div><div class="stat-lbl">Total NFTs</div></div>
        </div>
        <div :class="['stat-card', { 'stat-card--active': activeTab === 'minted' }]" @click="activeTab = 'minted'">
          <div class="stat-icon stat-icon--green"><CheckCircle :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.mintedOnly.length }}</div><div class="stat-lbl">Minted</div></div>
        </div>
        <div :class="['stat-card', { 'stat-card--active': activeTab === 'listed' }]" @click="activeTab = 'listed'">
          <div class="stat-icon stat-icon--blue"><Tag :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.listedOnly.length }}</div><div class="stat-lbl">Listed</div></div>
        </div>
        <div :class="['stat-card', { 'stat-card--active': activeTab === 'pending' }]" @click="activeTab = 'pending'">
          <div class="stat-icon stat-icon--amber"><Clock :size="18" /></div>
          <div><div class="stat-num">{{ dashboard.pendingOnly.length }}</div><div class="stat-lbl">Pending</div></div>
        </div>
      </div>

      <div class="main-grid">
        <!-- ── My NFTs panel ── -->
        <div class="panel">
          <div class="panel-header">
            <button v-if="activeTab !== null" class="btn-back" @click="activeTab = null">
              <ArrowLeft :size="16" />
            </button>
            <div v-if="activeTab === null" class="panel-title">
              <ImageIcon :size="15" /> My NFTs
            </div>
          </div>

          <!-- ── Album grid ── -->
          <div v-if="activeTab === null" class="album-grid">
            <div class="album-card" @click="activeTab = 'all'">
              <div class="album-preview">
                <div v-for="nft in dashboard.uniqueNFTs.slice(0, 4)" :key="nft.id" class="album-thumb">
                  <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="album-img" />
                  <div v-else class="album-img-empty"><ImageIcon :size="16" color="#ccc" /></div>
                </div>
                <div v-for="n in Math.max(0, 4 - dashboard.uniqueNFTs.length)" :key="`ea-${n}`" class="album-thumb album-thumb--empty" />
                <div class="album-overlay"><span class="album-name">All NFTs</span></div>
              </div>
            </div>
            <div class="album-card" @click="activeTab = 'minted'">
              <div class="album-preview">
                <div v-for="nft in dashboard.mintedOnly.slice(0, 4)" :key="nft.id" class="album-thumb">
                  <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="album-img" />
                  <div v-else class="album-img-empty"><ImageIcon :size="16" color="#ccc" /></div>
                </div>
                <div v-for="n in Math.max(0, 4 - dashboard.mintedOnly.length)" :key="`em-${n}`" class="album-thumb album-thumb--empty" />
                <div class="album-overlay"><span class="album-name">Minted</span></div>
              </div>
            </div>
            <div class="album-card" @click="activeTab = 'listed'">
              <div class="album-preview">
                <div v-for="nft in dashboard.listedOnly.slice(0, 4)" :key="nft.id" class="album-thumb">
                  <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="album-img" />
                  <div v-else class="album-img-empty"><ImageIcon :size="16" color="#ccc" /></div>
                </div>
                <div v-for="n in Math.max(0, 4 - dashboard.listedOnly.length)" :key="`el-${n}`" class="album-thumb album-thumb--empty" />
                <div class="album-overlay"><span class="album-name">Listed</span></div>
              </div>
            </div>
            <div class="album-card" @click="activeTab = 'pending'">
              <div class="album-preview">
                <div v-for="nft in dashboard.pendingOnly.slice(0, 4)" :key="nft.id" class="album-thumb">
                  <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="album-img" />
                  <div v-else class="album-img-empty"><Clock :size="16" color="#ccc" /></div>
                </div>
                <div v-for="n in Math.max(0, 4 - dashboard.pendingOnly.length)" :key="`ep-${n}`" class="album-thumb album-thumb--empty" />
                <div class="album-overlay"><span class="album-name">Pending</span></div>
              </div>
            </div>
          </div>

          <!-- ── NFT card grid (OpenSea style) ── -->
          <template v-else>
            <div v-if="activeNFTs.length === 0" class="empty-state">
              <ImageIcon :size="40" color="#ddd" />
              <p class="empty-title">{{ emptyTitle }}</p>
              <p class="empty-desc">{{ emptyDesc }}</p>
              <router-link v-if="activeTab === 'all'" to="/mint">
                <button class="btn-create"><Plus :size="14" /> Mint your first NFT</button>
              </router-link>
            </div>

            <div v-else class="nft-grid">
              <router-link
                v-for="nft in activeNFTs"
                :key="nft.id"
                :to="`/nft/${nft.id}`"
                class="nft-card"
              >
                <!-- Image -->
                <div class="nft-card-img">
                  <img v-if="nft.image" :src="ipfsToHttp(nft.image)" :alt="nft.name" class="nft-card-img-actual" />
                  <div v-else class="nft-card-img-empty">
                    <ImageIcon :size="28" color="#ddd" />
                  </div>
                  <!-- Status badge overlaid on image -->
                  <span :class="['nft-status-badge', `nft-status-badge--${nft.status}`]">
                    {{ nft.status }}
                  </span>
                </div>

                <!-- Card info below image -->
                <div class="nft-card-body">
                  <p class="nft-card-name">{{ nft.name }}</p>
                  <p class="nft-card-price">{{ listingPrice(nft) }}</p>
                </div>
              </router-link>

              <!-- Mint new card -->
              <router-link to="/mint" class="nft-card nft-card--new">
                <Plus :size="22" />
                <span>Mint new</span>
              </router-link>
            </div>
          </template>
        </div>

        <!-- ── Activity panel ── -->
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
  Activity, AlertCircle, ArrowLeft, CheckCircle, Clock,
  Image as ImageIcon, Layers, Plus,
  Send, ShoppingCart, Sparkles, Tag,
} from 'lucide-vue-next'
import { useDashboardStore } from '@/stores/dashboard'

const dashboard = useDashboardStore()
const activeTab  = ref<string | null>(null)

const activeNFTs = computed(() => {
  const map: Record<string, any[]> = {
    all:     dashboard.uniqueNFTs,
    minted:  dashboard.mintedOnly,
    listed:  dashboard.listedOnly,
    pending: dashboard.pendingOnly,
  }
  return map[activeTab.value ?? 'all'] ?? dashboard.uniqueNFTs
})

const emptyTitle = computed(() => {
  const map: Record<string, string> = {
    minted: 'No minted NFTs', listed: 'No listed NFTs', pending: 'No pending NFTs',
  }
  return map[activeTab.value ?? 'all'] ?? 'No NFTs yet'
})

const emptyDesc = computed(() => {
  const map: Record<string, string> = {
    minted:  'Minted NFTs will appear here.',
    listed:  'List an NFT for sale from its detail page.',
    pending: 'NFTs being confirmed will appear here.',
  }
  return map[activeTab.value ?? 'all'] ?? 'Start by minting your first NFT on Cardano.'
})

function ipfsToHttp(uri: string): string {
  if (!uri) return ''
  return uri.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${uri.replace('ipfs://', '')}`
    : uri
}

function listingPrice(nft: any): string {
  if (nft.status !== 'listed') return ''
  const listing = dashboard.myListings.find(
    (l: any) => l.nft_id === nft.id && l.status === 'active'
  )
  if (!listing) return 'Listed'
  const ada = (listing.price_lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 0, maximumFractionDigits: 2,
  })
  return `${ada} ₳`
}

const recentActivity = computed(() => {
  const items: any[] = []
  dashboard.uniqueNFTs.slice(0, 3).forEach((n: any) => {
    items.push({ id: `mint-${n.id}`, type: 'minted', name: n.name, time: formatTime(n.created_at) })
  })
  dashboard.myListings.filter((l: any) => l.status === 'active').slice(0, 2).forEach((l: any) => {
    const ada = (l.price_lovelace / 1_000_000).toFixed(0)
    items.push({ id: `list-${l.id}`, type: 'listed', name: `${l.nft_name} · ${ada} ₳`, time: formatTime(l.created_at) })
  })
  return items.slice(0, 5)
})

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
.stat-card {
  background: #fff; border-radius: 12px; padding: 14px 16px;
  display: flex; align-items: center; gap: 12px;
  border: 1px solid #f0f0f0; cursor: pointer; transition: all 0.15s;
}
.stat-card:hover        { border-color: #534AB7; transform: translateY(-1px); }
.stat-card--active      { border-color: #534AB7; background: #EEEDFE; }
.stat-card--active .stat-lbl { color: #534AB7; }
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
.panel-header  { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; min-height: 32px; }
.panel-title   { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; }
.panel-title svg { color: #534AB7; }
.view-all      { font-size: 11px; color: #534AB7; font-weight: 500; background: none; border: none; cursor: pointer; padding: 4px 8px; border-radius: 6px; transition: background 0.15s; text-decoration: none; }
.view-all:hover { background: #EEEDFE; }

/* Back button */
.btn-back {
  display: inline-flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  border: 1px solid #e8e8e8; background: #fff;
  color: #666; cursor: pointer; transition: all 0.15s;
}
.btn-back:hover { border-color: #534AB7; color: #534AB7; background: #EEEDFE; }

/* Album grid */
.album-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.album-card { cursor: pointer; border-radius: 12px; overflow: hidden; border: 1px solid #f0f0f0; transition: all 0.15s; }
.album-card:hover { border-color: #534AB7; transform: translateY(-2px); }

.album-preview {
  display: grid; grid-template-columns: 1fr 1fr; gap: 1px;
  aspect-ratio: 4/3; background: #e8e8e8;
  border-radius: 12px; overflow: hidden; position: relative;
}
.album-thumb       { overflow: hidden; background: #f5f5f5; }
.album-thumb--empty { background: #f0f0f0; }
.album-img         { width: 100%; height: 100%; object-fit: cover; display: block; }
.album-img-empty   { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }

/* Name overlaid on album image */
.album-overlay {
  position: absolute; bottom: 0; left: 0; right: 0;
  padding: 20px 10px 8px;
  background: linear-gradient(to top, rgba(0,0,0,0.65) 0%, transparent 100%);
}
.album-name { font-size: 12px; font-weight: 600; color: #fff; }

/* ── NFT card grid — OpenSea style ── */
.nft-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.nft-card {
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  text-decoration: none;
  color: inherit;
  display: block;
  transition: all 0.2s;
  background: #fff;
}
.nft-card:hover {
  border-color: #534AB7;
  transform: translateY(-3px);
  box-shadow: 0 8px 24px rgba(83,74,183,0.12);
}

/* Image area */
.nft-card-img {
  aspect-ratio: 1;
  background: #f5f5f5;
  position: relative;
  overflow: hidden;
}
.nft-card-img-actual {
  width: 100%; height: 100%;
  object-fit: cover; display: block;
  transition: transform 0.3s ease;
}
.nft-card:hover .nft-card-img-actual { transform: scale(1.04); }

.nft-card-img-empty {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}

/* Status badge on image */
.nft-status-badge {
  position: absolute; top: 8px; left: 8px;
  font-size: 9px; font-weight: 700;
  padding: 3px 8px; border-radius: 20px;
  text-transform: uppercase; letter-spacing: 0.3px;
}
.nft-status-badge--minted  { background: #E1F5EE; color: #085041; }
.nft-status-badge--listed  { background: #EEEDFE; color: #534AB7; }
.nft-status-badge--pending { background: #FFF8E1; color: #7a5c00; }

/* Card info below image */
.nft-card-body {
  padding: 10px 12px 12px;
  border-top: 1px solid #f5f5f5;
}
.nft-card-name {
  font-size: 12px; font-weight: 600; color: #111;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin: 0 0 3px;
}
.nft-card-price {
  font-size: 11px; font-weight: 600; color: #534AB7;
  margin: 0;
}

/* Mint new card */
.nft-card--new {
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  border: 1.5px dashed #e0e0e0;
  color: #ccc; gap: 6px;
  font-size: 11px; font-weight: 500;
  background: #fafafa; aspect-ratio: 1;
  border-radius: 12px;
}
.nft-card--new:hover {
  border-color: #534AB7; color: #534AB7;
  transform: translateY(-3px);
  box-shadow: none;
  background: #EEEDFE;
}

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
.skeleton { border-radius: 12px; background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%); background-size: 200% 100%; animation: shimmer 1.4s infinite; }
.skeleton--stat     { height: 70px; }
.skeleton--panel    { height: 400px; }
.skeleton--panel-sm { height: 300px; }
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

/* Mobile */
@media (max-width: 900px) {
  .dashboard  { padding: 16px; }
  .stats-row  { grid-template-columns: repeat(2, 1fr); }
  .main-grid  { grid-template-columns: 1fr; }
  .panel--sticky { position: static; }
  .nft-grid   { grid-template-columns: repeat(2, 1fr); }
  .album-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>