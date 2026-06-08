// ─────────────────────────────────────────────────────────────────────────────
// frontend/src/utils/cardano.ts
//
// Cardano-specific utility functions used across the frontend.
//
// THE CORE PROBLEM THIS SOLVES:
//   The DB stores token names as: prefix + rawName
//   e.g. "001bc280" + "Checking" = "001bc280Checking"
//
//   Cardanoscan needs the fully hex-encoded on-chain format:
//   "001bc280" + hex("Checking") = "001bc280436865636b696e67"
//
//   External assets from Blockfrost already return hex-encoded names, so we
//   check for non-hex characters to distinguish raw from hex-encoded names.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Converts a stored token name to the on-chain hex format required by
 * Cardanoscan and other blockchain explorers.
 *
 * DB stored:   "001bc280Checking"       → prefix + raw
 * On-chain:    "001bc280436865636b696e67" → prefix + hex(raw)
 * Blockfrost:  "001bc280436865636b696e67" → already correct
 */
export function toOnChainHex(tokenName: string): string {
  if (!tokenName || tokenName.length < 8) return tokenName

  const prefix = tokenName.slice(0, 8) // CIP-68 label: "001bc280", "000643b0" etc.
  const rest   = tokenName.slice(8)    // the asset name part

  // If rest contains any non-hex character (g-z, uppercase non-hex, etc.)
  // it's a raw name that needs to be hex-encoded
  if (/[^0-9a-fA-F]/.test(rest)) {
    const hexEncoded = Array.from(rest)
      .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
      .join('')
    return prefix + hexEncoded
  }

  // Already hex-encoded (from Blockfrost or already converted)
  return tokenName
}

/**
 * Builds the correct Cardanoscan token URL.
 * Works for both platform-minted NFTs (DB raw names) and external wallet NFTs (Blockfrost hex).
 */
export function cardanoscanTokenUrl(policyId: string, tokenName: string): string {
  if (!policyId || !tokenName) return '#'
  return `https://preprod.cardanoscan.io/token/${policyId}${toOnChainHex(tokenName)}`
}

/**
 * Builds the correct Cardanoscan transaction URL.
 */
export function cardanoscanTxUrl(txHash: string): string {
  if (!txHash) return '#'
  return `https://preprod.cardanoscan.io/transaction/${txHash}`
}

/**
 * Converts lovelace to ADA display string.
 */
export function lovelaceToAda(lovelace: number, decimals = 2): string {
  return (lovelace / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}

/**
 * Truncates a hash or address for display.
 * e.g. "a15cc08079a357...fbea06b1" (first 12, last 8)
 */
export function shortHash(hash: string, start = 12, end = 8): string {
  if (!hash || hash.length <= start + end) return hash
  return `${hash.slice(0, start)}...${hash.slice(-end)}`
}