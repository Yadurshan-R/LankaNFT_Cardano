<template>
  <div class="cert-page">

    <div v-if="loading" class="cert-center">
      <div class="cert-card">
        <div class="skeleton skeleton--logo" />
        <div class="skeleton skeleton--image" />
        <div class="skeleton skeleton--title" />
        <div class="skeleton skeleton--line" />
        <div class="skeleton skeleton--line" style="width: 60%" />
      </div>
    </div>

    <div v-else-if="error" class="cert-center">
      <div class="cert-card cert-card--error">
        <AlertCircle :size="40" color="#d32f2f" />
        <h2 class="error-title">Certificate Not Found</h2>
        <p class="error-desc">{{ error }}</p>
        <router-link to="/browse">
          <button class="btn-outline">Browse Mints</button>
        </router-link>
      </div>
    </div>

    <div v-else-if="cert" class="cert-center">

      <div class="action-bar no-print">
        <router-link to="/" class="back-link" title="Back to LankaNFT">
          <ArrowLeft :size="16" />
        </router-link>
        <div class="action-buttons">
          <button class="btn-share" @click="copyLink">
            <component :is="linkCopied ? Check : Link" :size="14" />
            {{ linkCopied ? 'Link copied!' : 'Share link' }}
          </button>
          <button class="btn-download" @click="downloadPDF">
            <Download :size="14" /> Download PDF
          </button>
        </div>
      </div>

      <div class="cert-card" id="certificate">

        <div class="cert-header">
          <img src="@/assets/lanka-nft-logo.png" alt="LankaNFT" class="cert-logo" />
          <div class="cert-header-text">
            <span class="cert-platform">LankaNFT</span>
            <span class="cert-subtitle">Certificate of Authenticity</span>
          </div>
          <div class="cert-verified">
            <ShieldCheck :size="18" color="#085041" />
            <span>Verified</span>
          </div>
        </div>

        <div class="cert-divider" />

        <div class="cert-image-wrap">
          <img
            v-if="imageUrl"
            :src="imageUrl"
            :alt="cert.nft_name"
            class="cert-image"
          />
          <div v-else class="cert-image-placeholder">
            <ImageIcon :size="48" color="#ddd" />
          </div>
        </div>

        <div class="cert-name-section">
          <h1 class="cert-nft-name">{{ cert.nft_name }}</h1>
          <p v-if="cert.description" class="cert-description">{{ cert.description }}</p>
        </div>

        <div class="cert-body-text">
          This certifies that the above NFT was minted on the
          <strong>Cardano blockchain</strong> via LankaNFT on
          <strong>{{ formattedDate }}</strong> and is authenticated
          by the following on-chain proof.
        </div>

        <div class="cert-divider" />

        <div class="cert-details">

          <div class="cert-detail-row">
            <span class="cert-detail-label">Owner Address</span>
            <span class="cert-detail-value cert-detail-value--mono cert-detail-value--truncate">
              {{ cert.owner_address || '—' }}
            </span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Policy ID</span>
            <span class="cert-detail-value cert-detail-value--mono cert-detail-value--truncate">
              {{ cert.policy_id }}
            </span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Asset Name</span>
            <span class="cert-detail-value cert-detail-value--mono">
              {{ cert.asset_name }}
            </span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Transaction Hash</span>
            <span class="cert-detail-value cert-detail-value--mono cert-detail-value--truncate">
              {{ cert.tx_hash }}
            </span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Network</span>
            <span class="cert-detail-value">
              <span class="network-badge">
                <span class="network-dot" /> {{ cert.network }}
              </span>
            </span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Royalties</span>
            <span class="cert-detail-value">{{ cert.royalties }}%</span>
          </div>

          <div class="cert-detail-row">
            <span class="cert-detail-label">Minted On</span>
            <span class="cert-detail-value">{{ formattedDate }}</span>
          </div>

        </div>

        <div class="cert-divider" />

        <div class="cert-links no-print">
          <a
            v-if="cert.cardanoscan_tx"
            :href="cert.cardanoscan_tx"
            target="_blank"
            rel="noopener noreferrer"
            class="cert-link"
          >
            <ExternalLink :size="13" /> View Transaction
          </a>
          <a
            v-if="cert.cardanoscan_asset"
            :href="cert.cardanoscan_asset"
            target="_blank"
            rel="noopener noreferrer"
            class="cert-link"
          >
            <ExternalLink :size="13" /> View Asset
          </a>
        </div>

        <div class="cert-print-urls print-only">
          <p v-if="cert.cardanoscan_tx">
            <strong>Transaction:</strong> {{ cert.cardanoscan_tx }}
          </p>
          <p v-if="cert.cardanoscan_asset">
            <strong>Asset:</strong> {{ cert.cardanoscan_asset }}
          </p>
          <p><strong>Certificate URL:</strong> {{ currentUrl }}</p>
        </div>

        <div class="cert-footer">
          <p class="cert-footer-text">
            Verify authenticity at
            <strong>preprod.cardanoscan.io</strong>
            using the Policy ID and Transaction Hash above.
          </p>
          <p class="cert-footer-brand">LankaNFT · Cardano NFT Platform · Sri Lanka</p>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// CertificateView.vue
//
// Public NFT ownership certificate page.
// Route: /certificate/:id (no auth required)
//
// Features:
//   - Fetches certificate data from GET /api/certificate/:id
//   - Displays a professional certificate card with NFT image and on-chain proof
//   - Share link — copies the certificate URL to clipboard
//   - Download PDF — uses window.print() with @media print CSS
//     Works on ALL operating systems and browsers (Windows, macOS, Linux)
//     without any third-party library. The browser's native print dialog
//     allows saving as PDF on every platform.
//
// PDF approach: We use window.print() instead of a PDF library because:
//   1. Works on every OS and browser without extra dependencies
//   2. No library version drift or security vulnerabilities
//   3. The browser renders it identically to what the user sees
//   4. Users can save to PDF, print to paper, or send to a printer
// ─────────────────────────────────────────────────────────────────────────────

import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  AlertCircle, ArrowLeft, Check, Download,
  ExternalLink, Image as ImageIcon,
  Link, ShieldCheck,
} from 'lucide-vue-next'

const route = useRoute()

// ── State ─────────────────────────────────────────────────────────────────────
const cert       = ref<any>(null)
const loading    = ref(true)
const error      = ref('')
const linkCopied = ref(false)
const currentUrl = window.location.href

// ── Computed ──────────────────────────────────────────────────────────────────

/**
 * Converts ipfs:// URI to a public HTTP gateway URL for image display.
 * Uses Pinata's gateway — the same gateway used throughout the platform.
 */
const imageUrl = computed(() => {
  if (!cert.value?.image_ipfs) return null
  const uri = cert.value.image_ipfs
  return uri.startsWith('ipfs://')
    ? `https://gateway.pinata.cloud/ipfs/${uri.replace('ipfs://', '')}`
    : uri
})

/**
 * Formats the minted_at ISO timestamp into a human-readable date.
 * Example: "May 28, 2026"
 */
const formattedDate = computed(() => {
  if (!cert.value?.minted_at) return '—'
  return new Date(cert.value.minted_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'long', day: 'numeric',
  })
})

// ── Data fetching ─────────────────────────────────────────────────────────────

onMounted(async () => {
  const nftId = route.params.id as string
  try {
    const response = await fetch(
      `${import.meta.env.VITE_API_BASE_URL}/api/certificate/${nftId}`
    )
    if (!response.ok) {
      const data = await response.json()
      error.value = data.error || 'Certificate not found.'
      return
    }
    cert.value = await response.json()
  } catch {
    error.value = 'Failed to load certificate. Please check your connection and try again.'
  } finally {
    loading.value = false
  }
})

// ── Actions ───────────────────────────────────────────────────────────────────

/**
 * Copies the current certificate URL to clipboard.
 * Shows a "Copied!" confirmation for 2 seconds.
 */
async function copyLink() {
  try {
    await navigator.clipboard.writeText(currentUrl)
    linkCopied.value = true
    setTimeout(() => { linkCopied.value = false }, 2000)
  } catch {
    // Fallback for browsers that block clipboard access
    window.prompt('Copy this link:', currentUrl)
  }
}

/**
 * Opens the browser print dialog which allows saving as PDF.
 *
 * This works on ALL operating systems:
 * - Windows: "Microsoft Print to PDF" printer
 * - macOS:   "Save as PDF" button in print dialog
 * - Linux:   "Print to File" option
 *
 * The @media print CSS below hides action buttons and shows print-only
 * content (full URLs instead of clickable links).
 */
function downloadPDF() {
  window.print()
}
</script>

<style scoped>
/* ── Page shell ── */
.cert-page {
  min-height: 100vh;
  background: #f0f0f0;
  padding: 32px 16px;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}

.cert-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

/* ── Action bar ── */
.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: 680px;
  margin-bottom: 4px;
}

.back-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 34px; height: 34px; border-radius: 8px;
  color: #666; text-decoration: none;
  background: #fff; border: 1px solid #e8e8e8;
  transition: all 0.15s;
}
.back-link:hover { border-color: #534AB7; color: #534AB7; }

.action-buttons { display: flex; gap: 8px; }

.btn-share {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px;
  border: 1px solid #e8e8e8; border-radius: 8px;
  background: #fff; font-size: 13px; font-weight: 500;
  color: #555; cursor: pointer; transition: all 0.15s;
}
.btn-share:hover { border-color: #534AB7; color: #534AB7; }

.btn-download {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px;
  background: #534AB7; color: #fff;
  border: none; border-radius: 8px;
  font-size: 13px; font-weight: 600;
  cursor: pointer; transition: background 0.15s;
}
.btn-download:hover { background: #3d35a0; }

/* ── Certificate card ── */
.cert-card {
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 20px;
  padding: 40px;
  width: 100%;
  max-width: 680px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  box-shadow: 0 4px 32px rgba(0,0,0,0.08);
}

.cert-card--error {
  align-items: center;
  text-align: center;
  gap: 16px;
  padding: 64px 40px;
}

/* ── Header ── */
.cert-header {
  display: flex;
  align-items: center;
  gap: 12px;
}
.cert-logo {
  width: 44px; height: 44px;
  border-radius: 50%; object-fit: cover;
  flex-shrink: 0;
}
.cert-header-text {
  flex: 1;
  display: flex; flex-direction: column; gap: 2px;
}
.cert-platform {
  font-size: 16px; font-weight: 700; color: #534AB7;
}
.cert-subtitle {
  font-size: 12px; color: #888; font-weight: 400;
}
.cert-verified {
  display: flex; align-items: center; gap: 5px;
  font-size: 12px; font-weight: 600; color: #085041;
  background: #E1F5EE; padding: 5px 12px;
  border-radius: 20px; flex-shrink: 0;
}

/* ── Divider ── */
.cert-divider {
  height: 1px;
  background: linear-gradient(to right, transparent, #e8e8e8, transparent);
}

/* ── NFT Image ── */
.cert-image-wrap {
  display: flex;
  justify-content: center;
}
.cert-image {
  width: 100%;
  max-width: 320px;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid #f0f0f0;
}
.cert-image-placeholder {
  width: 320px;
  aspect-ratio: 1;
  background: #f8f8f8;
  border-radius: 16px;
  display: flex; align-items: center; justify-content: center;
  border: 1px solid #f0f0f0;
}

/* ── Name section ── */
.cert-name-section { text-align: center; }
.cert-nft-name    { font-size: 26px; font-weight: 700; color: #111; margin: 0 0 8px; }
.cert-description { font-size: 14px; color: #666; line-height: 1.6; margin: 0; }

/* ── Body text ── */
.cert-body-text {
  font-size: 13px;
  color: #666;
  line-height: 1.7;
  text-align: center;
  padding: 0 20px;
}

/* ── On-chain details ── */
.cert-details {
  display: flex;
  flex-direction: column;
  gap: 0;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  overflow: hidden;
}

.cert-detail-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid #f8f8f8;
}
.cert-detail-row:last-child { border-bottom: none; }
.cert-detail-row:nth-child(even) { background: #fafafa; }

.cert-detail-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: #aaa;
  min-width: 130px;
  flex-shrink: 0;
  padding-top: 2px;
}
.cert-detail-value {
  font-size: 13px;
  color: #333;
  font-weight: 500;
  word-break: break-all;
}
.cert-detail-value--mono {
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
}
.cert-detail-value--truncate {
  /* On screen: show full value, let it wrap */
  word-break: break-all;
}

/* Network badge */
.network-badge {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 12px; font-weight: 500; color: #7a5c00;
  background: #FFF8E1; padding: 3px 10px; border-radius: 20px;
}
.network-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #085041; display: inline-block;
}

/* ── Verification links ── */
.cert-links {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}
.cert-link {
  display: flex; align-items: center; gap: 6px;
  padding: 9px 18px;
  border: 1px solid #534AB7; border-radius: 8px;
  color: #534AB7; text-decoration: none;
  font-size: 13px; font-weight: 500;
  transition: all 0.15s;
}
.cert-link:hover { background: #EEEDFE; }

/* Print-only full URLs */
.cert-print-urls {
  display: none;
  font-size: 11px;
  color: #666;
  line-height: 1.8;
  word-break: break-all;
}

/* ── Footer ── */
.cert-footer { text-align: center; }
.cert-footer-text  { font-size: 12px; color: #888; line-height: 1.6; margin: 0 0 4px; }
.cert-footer-brand { font-size: 11px; color: #bbb; margin: 0; }

/* ── Error state ── */
.error-title { font-size: 20px; font-weight: 600; color: #111; margin: 0; }
.error-desc  { font-size: 13px; color: #888; margin: 0; max-width: 320px; }
.btn-outline {
  padding: 9px 20px; background: transparent;
  color: #534AB7; border: 1px solid #534AB7;
  border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer;
}

/* ── Skeleton loading ── */
.skeleton {
  border-radius: 8px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
.skeleton--logo  { width: 44px; height: 44px; border-radius: 50%; }
.skeleton--image { width: 280px; height: 280px; border-radius: 16px; align-self: center; }
.skeleton--title { height: 28px; width: 60%; align-self: center; }
.skeleton--line  { height: 14px; }
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>

<style>
@media print {
  /* Hide sidebar, action bar, and everything outside the certificate */
  .no-print,
  .sidebar,
  aside,
  nav { display: none !important; }

  .print-only { display: block !important; }

  body, html {
    background: #fff !important;
    margin: 0; padding: 0;
  }

  .cert-page {
    background: #fff !important;
    padding: 0 !important;
    min-height: unset !important;
  }

  .cert-center { padding: 0 !important; }

  .cert-card {
    box-shadow: none !important;
    border: 1px solid #e0e0e0 !important;
    border-radius: 0 !important;
    max-width: 100% !important;
    width: 100% !important;
    margin: 0 !important;
    padding: 20px !important;
    gap: 16px !important;
  }

  /* Shrink image for single page fit */
  .cert-image {
    max-width: 180px !important;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .cert-image-wrap { margin: 0 !important; }

  /* Reduce font sizes slightly for single page */
  .cert-nft-name    { font-size: 20px !important; }
  .cert-body-text   { font-size: 12px !important; padding: 0 !important; }
  .cert-detail-row  { padding: 8px 12px !important; }
  .cert-footer-text { font-size: 11px !important; }

  /* Force single page */
  .cert-details { page-break-inside: avoid; }
  .cert-card    { page-break-after: avoid; }

  /* Preserve badge colors */
  .cert-verified,
  .network-badge,
  .cert-detail-row:nth-child(even) {
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }
}
</style>