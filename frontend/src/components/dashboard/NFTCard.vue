<template>
  <div class="nft-card" @click="goToDetail" style="cursor: pointer;">
    <div class="nft-image-wrap">
      <img
        v-if="imageUrl"
        :src="imageUrl"
        :alt="nft.name"
        class="nft-image"
        @error="imageError = true"
      />
      <div v-else class="nft-image-placeholder">
        <ImageIcon :size="40" color="#ccc" />
      </div>

      <div :class="['status-badge', `status-badge--${nft.status}`]">
        {{ nft.status }}
      </div>

      <div v-if="nft.privacy === 'private'" class="privacy-badge">
        <Lock :size="12" color="#fff" />
      </div>
    </div>

    <div class="nft-info">
      <h3 class="nft-name">{{ nft.name }}</h3>
      <p v-if="nft.description" class="nft-desc">{{ nft.description }}</p>

      <div class="nft-meta">
        <span class="meta-tag">{{ nft.royalties }}% royalty</span>
        <span v-if="nft.policy_id" class="meta-tag meta-tag--policy">
          {{ nft.policy_id.slice(0, 8) }}...
        </span>
      </div>

      <a
        v-if="nft.tx_hash"
        :href="`https://preprod.cardanoscan.io/transaction/${nft.tx_hash}`"
        target="_blank"
        rel="noopener noreferrer"
        class="tx-link"
        @click.stop
      >
        View on Cardanoscan →
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Image as ImageIcon, Lock } from 'lucide-vue-next'

const props = defineProps<{
  nft: {
    id: string
    name: string
    description: string
    image: string
    policy_id: string
    asset_name: string
    status: string
    tx_hash: string | null
    royalties: number
    privacy: string
    created_at: string
  }
}>()

const router = useRouter()
const imageError = ref(false)

function goToDetail() {
  router.push({ name: 'nft-detail', params: { id: props.nft.id } })
}

const imageUrl = computed(() => {
  if (imageError.value) return null
  if (!props.nft.image) return null
  if (props.nft.image.startsWith('ipfs://')) {
    return `https://gateway.pinata.cloud/ipfs/${props.nft.image.replace('ipfs://', '')}`
  }
  return props.nft.image
})
</script>

<style scoped>
.nft-card {
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  overflow: hidden;
  transition: box-shadow 0.2s, transform 0.2s;
}
.nft-card:hover {
  box-shadow: 0 4px 20px rgba(0,0,0,0.08);
  transform: translateY(-2px);
}
.nft-image-wrap {
  position: relative;
  aspect-ratio: 1;
  background: #f8f8f8;
  overflow: hidden;
}
.nft-image { width: 100%; height: 100%; object-fit: cover; }
.nft-image-placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}
.status-badge {
  position: absolute; top: 8px; left: 8px;
  font-size: 10px; font-weight: 600;
  padding: 2px 8px; border-radius: 20px;
  text-transform: uppercase;
  background: #f0f0f0; color: #666;
}
.status-badge--minted { background: #E1F5EE; color: #085041; }
.status-badge--pending { background: #FFF8E1; color: #7a5c00; }
.status-badge--failed { background: #FCEBEB; color: #791F1F; }
.privacy-badge {
  position: absolute; top: 8px; right: 8px;
  background: rgba(0,0,0,0.5);
  border-radius: 50%;
  width: 22px; height: 22px;
  display: flex; align-items: center; justify-content: center;
}
.nft-info { padding: 12px; }
.nft-name {
  font-size: 14px; font-weight: 600; margin: 0 0 4px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.nft-desc {
  font-size: 12px; color: #888; margin: 0 0 8px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.nft-meta { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
.meta-tag {
  font-size: 11px; background: #f5f5f5; color: #666;
  padding: 2px 6px; border-radius: 4px;
}
.meta-tag--policy { font-family: monospace; background: #EEEDFE; color: #534AB7; }
.tx-link { font-size: 11px; color: #534AB7; text-decoration: none; }
.tx-link:hover { text-decoration: underline; }
</style>