# LankaNFT — Complete Architecture & System Design

---

## 1. System Overview — Every Service & Why

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         USER'S BROWSER                                       │
│                    Vue 3 + Pinia (localhost:5173)                            │
│                                                                              │
│   ┌─────────────────────────┐    ┌──────────────────────────────────────┐   │
│   │   Custodial User        │    │   External Wallet User               │   │
│   │   (Email + OTP login)   │    │   (Lace browser extension)           │   │
│   │                         │    │                                      │   │
│   │   Platform manages      │    │   User keeps their own keys.         │   │
│   │   their wallet keys.    │    │   window.cardano.lace (CIP-30 API)   │   │
│   │   They just click.      │    │   signs transactions locally.        │   │
│   └─────────────────────────┘    └──────────────────────────────────────┘   │
└──────────────────────────────────────────┬──────────────────────────────────┘
                                           │ HTTPS REST API
                                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      BACKEND — Go + Gin (localhost:8080)                     │
│                                                                              │
│  Why Go?                                                                     │
│  • Compiled, fast, strong type safety                                        │
│  • Excellent for REST APIs                                                   │
│  • pgx v5 is best-in-class PostgreSQL driver                                 │
│  • Low memory usage                                                          │
│                                                                              │
│  ┌─────────┐  ┌──────────┐  ┌────────────┐  ┌─────────┐  ┌──────────────┐ │
│  │  auth/  │  │   nft/   │  │  listing/  │  │ batch/  │  │ pkg/         │ │
│  │         │  │          │  │            │  │         │  │ blockchain/  │ │
│  │ OTP     │  │ Prepare  │  │ Create     │  │ Prepare │  │             │ │
│  │ Login   │  │ Mint     │  │ Buy        │  │ Mint    │  │ HTTP client │ │
│  │ Wallet  │  │ Mint     │  │ Cancel     │  │ Confirm │  │ to sidecar  │ │
│  │ Verify  │  │ Transfer │  │ (signed +  │  │         │  │             │ │
│  │ JWT     │  │ (signed +│  │  unsigned) │  │         │  │ All request │ │
│  │         │  │unsigned) │  │            │  │         │  │ & response  │ │
│  │         │  │ External │  │            │  │         │  │ types       │ │
│  │         │  │ Assets   │  │            │  │         │  │             │ │
│  └─────────┘  └──────────┘  └────────────┘  └─────────┘  └──────────────┘ │
│                                                                              │
│  PostgreSQL (localhost:5432, DB: midnight_nft)                              │
│  Why PostgreSQL?                                                             │
│  • Transactional integrity — NFT status updates are atomic                   │
│  • pgx v5 supports context cancellation and connection pooling               │
│  • Structured data fits relational model perfectly                           │
└──────────────────────────────────────────┬──────────────────────────────────┘
                                           │ HTTP + X-Service-Secret header
                                           │ (internal only, never exposed)
                                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│              BLOCKCHAIN SIDECAR — Node.js + TypeScript (localhost:3001)      │
│                                                                              │
│  Why a separate Node.js service?                                             │
│  • MeshSDK (best Cardano tx builder) is TypeScript ONLY                      │
│  • No equivalent Go library exists for Cardano                               │
│  • Keeps blockchain complexity isolated from business logic                  │
│  • Can be replaced/upgraded without touching Go code                         │
│                                                                              │
│  ┌──────────────────────┐   ┌───────────────────────┐                       │
│  │   routes/mint.ts     │   │  routes/marketplace.ts│                       │
│  │                      │   │                       │                       │
│  │  POST /single        │   │  POST /list           │                       │
│  │  POST /single-unsigned│  │  POST /list-unsigned  │                       │
│  │  POST /batch         │   │  POST /buy            │                       │
│  │  POST /batch-unsigned│   │  POST /buy-unsigned   │                       │
│  │                      │   │  POST /cancel         │                       │
│  │  Uses MeshSDK:       │   │  POST /cancel-unsigned│                       │
│  │  MintingBlueprint    │   │                       │                       │
│  │  MeshTxBuilder       │   │  Plutus V3 script     │                       │
│  │  metadataToCip68()   │   │  interactions via     │                       │
│  │  serializeData()     │   │  MeshTxBuilder        │                       │
│  └──────────────────────┘   └───────────────────────┘                       │
│                                                                              │
│  ┌──────────────────────┐   ┌───────────────────────┐                       │
│  │  routes/transfer.ts  │   │  app.ts               │                       │
│  │                      │   │                       │                       │
│  │  POST /              │   │  POST /api/submit     │                       │
│  │  POST /unsigned      │   │                       │                       │
│  │                      │   │  WHY THIS EXISTS:     │                       │
│  │  Sends NFT token     │   │  Lace's signTx()      │                       │
│  │  to any Cardano      │   │  returns WITNESS SET  │                       │
│  │  address             │   │  not full tx. CSL     │                       │
│  │                      │   │  assembles: unsigned  │                       │
│  │                      │   │  CBOR + witness CBOR  │                       │
│  │                      │   │  = full signed tx     │                       │
│  │                      │   │  then submits via     │                       │
│  │                      │   │  Blockfrost           │                       │
│  └──────────────────────┘   └───────────────────────┘                       │
│                                                                              │
│  Key libraries in sidecar:                                                   │
│  • @meshsdk/core        — Cardano transaction building                        │
│  • @emurgo/cardano-serialization-lib-nodejs — CSL for tx assembly            │
│  • cbor                 — parse CIP-30 UTxO hex format from Lace            │
│  • bech32               — convert raw address bytes to addr_test1...         │
└─────────────────────┬──────────────────────────────┬────────────────────────┘
                      │                              │
                      ▼                              ▼
┌──────────────────────────┐         ┌──────────────────────────────┐
│       BLOCKFROST          │         │           PINATA             │
│  (Cardano Preprod API)    │         │      (IPFS Pinning)          │
│                          │         │                              │
│  Why?                    │         │  Why?                        │
│  • Managed Cardano node  │         │  • Blockchain can't store    │
│  • Running own node =    │         │    images (too expensive)    │
│    100GB+ storage        │         │  • IPFS = decentralised,     │
│  • Blockfrost handles    │         │    images live forever       │
│    sync, indexing,       │         │  • Pinata = reliable         │
│    mempool, submission   │         │    gateway & pinning         │
│  • Standard in Cardano   │         │    service                   │
│    ecosystem             │         │  • Free tier covers dev      │
│                          │         │                              │
│  Used for:               │         │  Used for:                   │
│  • Submit transactions   │         │  • Upload NFT image → get    │
│  • Fetch UTxOs           │         │    ipfs://Qm... hash         │
│  • Fetch address balance │         │  • Image URL stored in       │
│  • Fetch NFT metadata    │         │    CIP-68 datum + CIP-25     │
│    for external wallets  │         │    metadata                  │
│  • Check tx confirmation │         │                              │
│                          │         │  gateway.pinata.cloud/ipfs/  │
│  preprod.blockfrost.io   │         │  Qm... serves the image      │
└──────────────────────────┘         └──────────────────────────────┘
```

---

## 2. Authentication Flows

```
CUSTODIAL WALLET — EMAIL OTP FLOW
══════════════════════════════════════════════════════════════════════

User types email
    │
    ▼
POST /auth/request-otp
    │
    ▼
Backend generates 6-digit OTP
Stores in DB with 5 min expiry
Sends via SMTP (Gmail App Password)
    │
    ▼
User receives email, types code
    │
    ▼
POST /auth/verify-otp
    │
    ├── OTP valid + user exists?
    │       └── Get existing custodial wallet from DB
    │
    └── OTP valid + new user?
            └── Call sidecar POST /api/wallet/generate
                    │
                    ▼
                MeshSDK generates new 24-word mnemonic
                Derives wallet address
                    │
                    ▼
                Store in custodial_wallets table
                (mnemonic stored encrypted at rest)
    │
    ▼
Return JWT (HttpOnly cookie, 7 day expiry)
Set wallet_type = 'custodial' in users table
    │
    ▼
Frontend auth store saves: userID, walletType, walletAddress
Vue Router allows navigation to protected routes


EXTERNAL WALLET — LACE CIP-30 FLOW
══════════════════════════════════════════════════════════════════════

User clicks "Connect Lace Wallet"
    │
    ▼
window.cardano.lace.enable()
    │  (Lace shows "Connect" popup)
    ▼
Returns CIP-30 walletApi object
    │
    ▼
GET /auth/nonce
    │  Backend generates random UUID nonce, stores temporarily
    ▼
walletApi.signData(address, nonce)
    │  (Lace shows "Sign" popup — user approves)
    ▼
Returns { signature, key } in CIP-30 format
    │
    ▼
POST /auth/wallet-verify { address, signature, key, nonce }
    │
    ▼
Backend verifies signature cryptographically
    │  (proves user owns the private key for that address)
    │
    ├── Valid + new user?
    │       └── Create user record
    │           Create external_wallets record
    │           Set wallet_type = 'external'
    │
    └── Valid + existing user?
            └── Load existing record
    │
    ▼
Return JWT (HttpOnly cookie)
    │
    ▼
Frontend:
  auth.setWallet(walletId, walletApi)
  localStorage.setItem('lankanft_wallet_id', 'lace')
  Saves walletType = 'external', walletAddress
```

---

## 3. CIP-68 Token Architecture — Complete Detail

```
WHAT IS CIP-68?
═══════════════════════════════════════════════════════════════════════════════
CIP-68 is a Cardano NFT standard where metadata lives ON-CHAIN in a
Plutus datum, not in transaction metadata. This means:
• Metadata is queryable without scanning every transaction
• Can theoretically be updated (unlike CIP-25)
• Reference token pattern separates "metadata holder" from "tradeable NFT"


EVERY MINT CREATES 3 TOKENS UNDER ONE POLICY:
═══════════════════════════════════════════════════════════════════════════════

Policy ID: unique per NFT (for single mint) or per collection (for batch)
│
├── Token 1: Reference Token  (CIP-68 label 100)
│   Name:    000643b0 + hex(assetName)
│   e.g.:    000643b0 + 436865636b696e67  =  000643b0436865636b696e67
│            (prefix)    (hex of "Checking")
│
│   Destination: ImmutableLock Plutus V3 contract (locked forever)
│   Contains inline datum (Plutus Map format):
│   {
│     "name":        "Checking",
│     "image":       "ipfs://QmXxx...",
│     "mediaType":   "image/png",
│     "description": "My first NFT",
│     "files":       []
│   }
│   Why locked? Reference tokens must never move — they're the
│   metadata source. Wallets and marketplaces read from here.
│
│
├── Token 2: User Token  (CIP-68 label 222) ← THE ACTUAL NFT YOU OWN
│   Name:    001bc280 + hex(assetName)
│   e.g.:    001bc280436865636b696e67
│
│   Destination: Creator's wallet
│   This is the token that:
│   • Shows in your wallet
│   • Gets listed on marketplace
│   • Gets transferred to buyers
│   • Appears in Browse Mints
│
│
└── Token 3: Royalty Token  (CIP-68 label 500)
    Name:    001f4d70 + hex(assetName)
    e.g.:    001f4d70436865636b696e67

    Destination: RoyaltyLock Plutus V3 contract (locked forever)
    Contains inline datum:
    {
      creator_address: "addr_test1...",
      royalty_recipients: [{
        address: "addr_test1...",
        rate:     50000   (= 5% expressed as millionths: 50000/1000000)
      }],
      min_payout: 2000000  (= 2 ADA)
    }
    Why? Marketplace reads this on every sale to calculate royalty split.
    Creator gets their cut automatically, enforced by the smart contract.


MINTING POLICY TYPES:
═══════════════════════════════════════════════════════════════════════════════

SINGLE NFT → One-Shot Plutus Policy (parameterised by UTxO)
  • Policy is parameterised by a specific UTxO (txHash + index)
  • Can only mint once (UTxO is consumed and can never be reused)
  • Unique policy per NFT = each NFT has its own policy ID
  • Script: minting_policy.cip68_mint_single.mint (Aiken V3)
  • Compiled to plutus.json

BATCH NFTs → Time-Locked Native Script
  • All NFTs in batch share ONE policy ID
  • Policy = RequireAllOf [RequireSignature(paymentKeyHash), RequireBefore(slot)]
  • Expires 2 hours from minting (security: can't mint more tokens after window)
  • Signed by creator's payment key (external wallet) or treasury (custodial)
  • This is how NMKR, jpg.store handle collections


CIP-25 (TRANSACTION METADATA) — Added alongside CIP-68:
═══════════════════════════════════════════════════════════════════════════════
Every mint also writes label 721 metadata to the transaction:
{
  "721": {
    "<policyId>": {
      "<assetName>": {
        "name":        "Checking",
        "image":       "ipfs://QmXxx...",
        "mediaType":   "image/png",
        "description": "My description"
      }
    }
  }
}

WHY BOTH?
• CIP-68 datum = authoritative on-chain source, read by smart contracts
• CIP-25 = wallets like Lace, Nami, Eternl read THIS to show images
• Without CIP-25, wallets show blank images
• CIP-25 is immutable (in transaction metadata) — CIP-68 datum is what matters
```

---

## 4. Single Mint — Complete Transaction Flow

```
CUSTODIAL WALLET SINGLE MINT
═══════════════════════════════════════════════════════════════════════════════

Browser                  Backend (Go)              Sidecar (Node.js)         Pinata         Blockfrost
   │                         │                          │                       │                │
   │── POST prepare-mint ───►│                          │                       │                │
   │   (multipart: image,    │                          │                       │                │
   │    name, description,   │                          │                       │                │
   │    royalties, privacy)  │                          │                       │                │
   │                         │                          │                       │                │
   │                         │── Upload image ─────────────────────────────────►│                │
   │                         │                          │                       │                │
   │                         │◄── ipfs://QmXxx... ─────────────────────────────│                │
   │                         │                          │                       │                │
   │                         │  Create nft record in DB                         │                │
   │                         │  status = 'pending'                              │                │
   │                         │  asset_name = name (max 28 chars)               │                │
   │                         │  user_token_name = "001bc280" + name            │                │
   │                         │                          │                       │                │
   │◄── { nft_id, ... } ─────│                          │                       │                │
   │                         │                          │                       │                │
   │── POST /api/nft/mint ──►│                          │                       │                │
   │   { nft_id }            │                          │                       │                │
   │                         │  Load mnemonic from DB   │                       │                │
   │                         │  (custodial_wallets)     │                       │                │
   │                         │── POST /api/mint/single ►│                       │                │
   │                         │   { mnemonic, asset_name,│                       │                │
   │                         │     image_ipfs,          │                       │                │
   │                         │     description,         │                       │                │
   │                         │     royalties }          │                       │                │
   │                         │                          │                       │                │
   │                         │                          │  MeshWallet(mnemonic) │                │
   │                         │                          │── getUtxos() ─────────────────────────►│
   │                         │                          │◄── [utxos] ───────────────────────────│
   │                         │                          │                       │                │
   │                         │                          │  Pick smallest pure-ADA UTxO          │
   │                         │                          │  as one-shot (minting policy param)   │
   │                         │                          │                       │                │
   │                         │                          │  MintingBlueprint.paramScript(        │
   │                         │                          │    compiledCode, [txHash, txIndex])   │
   │                         │                          │  policyId = blueprint.hash            │
   │                         │                          │                       │                │
   │                         │                          │  metadataToCip68({    │                │
   │                         │                          │    name, image, ...}) │                │
   │                         │                          │  → Plutus Map datum CBOR              │
   │                         │                          │                       │                │
   │                         │                          │  Build tx:            │                │
   │                         │                          │  .txIn(oneShotUtxo)  │                │
   │                         │                          │  .mint(refToken)     │                │
   │                         │                          │  .mint(userToken)    │                │
   │                         │                          │  .mint(royaltyToken) │                │
   │                         │                          │  .txOut(immutableLockAddr, refToken)  │
   │                         │                          │  .txOutInlineDatum(cip68DatumCbor)   │
   │                         │                          │  .txOut(creatorAddr, userToken)       │
   │                         │                          │  .txOut(royaltyLockAddr, royToken)    │
   │                         │                          │  .txOutInlineDatum(royaltyDatumCbor) │
   │                         │                          │  .metadataValue("721", cip25)         │
   │                         │                          │  .complete()          │                │
   │                         │                          │                       │                │
   │                         │                          │  wallet.signTx(tx)    │                │
   │                         │                          │── submitTx(signed) ───────────────────►│
   │                         │                          │◄── txHash ────────────────────────────│
   │                         │                          │                       │                │
   │                         │◄── { tx_hash, policy_id }│                       │                │
   │                         │                          │                       │                │
   │                         │  UPDATE nfts SET         │                       │                │
   │                         │    status = 'minted'     │                       │                │
   │                         │    tx_hash = txHash      │                       │                │
   │                         │    policy_id = policyId  │                       │                │
   │                         │                          │                       │                │
   │◄── { tx_hash, ... } ────│                          │                       │                │
   │  Show success + Cardanoscan link                   │                       │                │


EXTERNAL WALLET (LACE) SINGLE MINT — UNSIGNED FLOW
═══════════════════════════════════════════════════════════════════════════════

Browser (Lace)            Backend (Go)              Sidecar (Node.js)         Pinata         Blockfrost
   │                         │                          │                       │                │
   │  (Same prepare-mint step — IPFS upload, DB record) │                       │                │
   │                         │                          │                       │                │
   │── getUtxos() ──────────►│ (CIP-30 call to Lace)   │                       │                │
   │◄── [hex-encoded UTxOs] ─│                          │                       │                │
   │  (All UTxOs from ALL HD wallet derived addresses)  │                       │                │
   │                         │                          │                       │                │
   │── POST /api/nft/       ─┤                          │                       │                │
   │     mint-unsigned       │                          │                       │                │
   │   { nft_id,             │                          │                       │                │
   │     wallet_utxos: [...] }│                          │                       │                │
   │                         │                          │                       │                │
   │                         │  Load wallet_address from│                       │                │
   │                         │  external_wallets table  │                       │                │
   │                         │── POST /api/mint/        │                       │                │
   │                         │     single-unsigned ────►│                       │                │
   │                         │   { wallet_address,      │                       │                │
   │                         │     wallet_utxos,        │                       │                │
   │                         │     asset_name, ... }    │                       │                │
   │                         │                          │                       │                │
   │                         │                          │  parseCip30Utxos()    │                │
   │                         │                          │  (decode hex CBOR from Lace)          │
   │                         │                          │                       │                │
   │                         │                          │  resolvePaymentKeyHash(wallet_address)│
   │                         │                          │  → paymentKeyHash     │                │
   │                         │                          │                       │                │
   │                         │                          │  Build same tx as custodial...        │
   │                         │                          │  BUT: .complete() only (NO signing)   │
   │                         │                          │                       │                │
   │                         │◄── { unsigned_cbor,      │                       │                │
   │                         │      policy_id }         │                       │                │
   │                         │                          │                       │                │
   │◄── { unsigned_cbor,     │                          │                       │                │
   │      policy_id, nft_id }│                          │                       │                │
   │                         │                          │                       │                │
   │  walletApi.signTx(      │                          │                       │                │
   │    unsigned_cbor, true) │                          │                       │                │
   │  → Lace popup appears ← USER REVIEWS & APPROVES   │                       │                │
   │  → Returns witness_cbor │                          │                       │                │
   │    (ONLY the signatures,│                          │                       │                │
   │     NOT the full tx)    │                          │                       │                │
   │                         │                          │                       │                │
   │── POST /api/submit ────►│                          │                       │                │
   │   { unsigned_cbor,      │                          │                       │                │
   │     witness_cbor }      │                          │                       │                │
   │                         │── POST /api/submit ─────►│                       │                │
   │                         │                          │                       │                │
   │                         │                          │  CSL assembly:        │                │
   │                         │                          │  Transaction.from_hex(unsigned_cbor)  │
   │                         │                          │  TransactionWitnessSet.from_hex(      │
   │                         │                          │    witness_cbor)      │                │
   │                         │                          │  Transaction.new(     │                │
   │                         │                          │    tx.body(),         │                │
   │                         │                          │    witnessSet,        │                │
   │                         │                          │    tx.auxiliary_data()│                │
   │                         │                          │  )                    │                │
   │                         │                          │  → full signed CBOR   │                │
   │                         │                          │── submitTx(signed) ───────────────────►│
   │                         │                          │◄── txHash ────────────────────────────│
   │                         │◄── { tx_hash } ──────────│                       │                │
   │◄── { tx_hash } ─────────│                          │                       │                │
   │                         │                          │                       │                │
   │── POST /api/nft/        │                          │                       │                │
   │     confirm-mint ──────►│                          │                       │                │
   │   { nft_id, tx_hash,    │                          │                       │                │
   │     policy_id }         │                          │                       │                │
   │                         │  UPDATE nfts SET         │                       │                │
   │                         │    status = 'minted'     │                       │                │
   │                         │    tx_hash, policy_id    │                       │                │
   │◄── { message: "minted" }│                          │                       │                │
```

---

## 5. Marketplace Flow — Complete Detail

```
LISTING AN NFT (CUSTODIAL)
═══════════════════════════════════════════════════════════════════════════════

The NFT is LOCKED into the marketplace Plutus script.
The script holds: NFT token + price + seller address + royalty policy

Browser                  Backend (Go)              Sidecar (Node.js)         Blockfrost
   │                         │                          │                       │
   │── POST /listing/create ►│                          │                       │
   │   { nft_id, price }     │                          │                       │
   │                         │  Load NFT from DB:       │                       │
   │                         │    policy_id             │                       │
   │                         │    user_token_name       │                       │
   │                         │    royalty_policy_id     │                       │
   │                         │                          │                       │
   │                         │  Build NFT unit:         │                       │
   │                         │    policyId +            │                       │
   │                         │    "001bc280" + hex(name)│                       │
   │                         │                          │                       │
   │                         │── POST /marketplace/list►│                       │
   │                         │   { mnemonic, nft_unit,  │                       │
   │                         │     price_lovelace,      │                       │
   │                         │     royalty_policy_id }  │                       │
   │                         │                          │                       │
   │                         │                          │  Build listing tx:    │
   │                         │                          │  .txIn(NFT from wallet)              │
   │                         │                          │  .txOut(marketplaceScript, {          │
   │                         │                          │    assets: [NFT],     │               │
   │                         │                          │    datum: {           │               │
   │                         │                          │      price_lovelace,  │               │
   │                         │                          │      seller_address,  │               │
   │                         │                          │      royalty_policy_id│               │
   │                         │                          │    }                  │               │
   │                         │                          │  })                   │               │
   │                         │                          │  sign + submit ──────────────────────►│
   │                         │◄── { tx_hash,            │◄── txHash ────────────────────────────│
   │                         │      script_utxo }       │                       │                │
   │                         │                          │                       │
   │                         │  INSERT into listings:   │                       │
   │                         │    nft_id, price         │                       │
   │                         │    status = 'active'     │                       │
   │                         │    script_utxo = txHash#0│                       │
   │                         │    listing_tx_hash       │                       │
   │◄── { tx_hash } ─────────│                          │                       │


BUYING AN NFT (EXTERNAL WALLET)
═══════════════════════════════════════════════════════════════════════════════

Buyer spends the script UTxO. Script validates:
  ✓ Payment goes to seller (minus royalty)
  ✓ Royalty goes to royalty lock address
  ✓ NFT goes to buyer

Browser                  Backend (Go)              Sidecar (Node.js)         Blockfrost
   │                         │                          │                       │
   │── POST /listing/         │                          │                       │
   │     buy-unsigned ──────►│                          │                       │
   │   { listing_id }        │                          │                       │
   │                         │  Load listing from DB:   │                       │
   │                         │    script_utxo           │                       │
   │                         │    seller_address        │                       │
   │                         │    price_lovelace        │                       │
   │                         │    nft_unit              │                       │
   │                         │    royalty_policy_id     │                       │
   │                         │── POST /marketplace/     │                       │
   │                         │     buy-unsigned ───────►│                       │
   │                         │                          │                       │
   │                         │                          │  Fetch script UTxO ──────────────────►│
   │                         │                          │  (retry 5×6s if not indexed)         │
   │                         │                          │◄── UTxO details ──────────────────────│
   │                         │                          │                       │                │
   │                         │                          │  Build buy tx:        │                │
   │                         │                          │  .txIn(scriptUtxo,    │                │
   │                         │                          │    redeemer: "Buy")   │                │
   │                         │                          │  .txOut(buyer, [NFT]) │                │
   │                         │                          │  .txOut(seller, price │                │
   │                         │                          │         - royalty)    │                │
   │                         │                          │  .txOut(royaltyLock,  │                │
   │                         │                          │         royaltyAmount)│                │
   │                         │                          │  .txInCollateral(...)  │                │
   │                         │                          │  .complete()          │                │
   │                         │◄── { unsigned_cbor }     │                       │                │
   │◄── { unsigned_cbor }────│                          │                       │                │
   │                         │                          │                       │
   │  Lace signTx popup → User approves                 │                       │
   │                         │                          │                       │
   │── POST /api/submit ────►│── POST /api/submit ─────►│                       │                │
   │   { unsigned, witness } │                          │  CSL assembly +       │                │
   │                         │                          │  submitTx ────────────────────────────►│
   │                         │◄── { tx_hash } ──────────│◄── txHash ─────────────────────────────│
   │◄── { tx_hash } ─────────│                          │                       │
   │                         │                          │                       │
   │── POST /listing/         │                          │                       │
   │     confirm-buy ───────►│                          │                       │
   │   { listing_id, tx_hash}│                          │                       │
   │                         │  UPDATE listings SET     │                       │
   │                         │    status = 'sold'       │                       │
   │                         │                          │                       │
   │  BuySuccessModal shows: │                          │                       │
   │  • NFT image            │                          │                       │
   │  • Price paid           │                          │                       │
   │  • Tx hash + copy       │                          │                       │
   │  • View Transaction     │                          │                       │
   │  • View Asset           │                          │                       │
   │    (both Cardanoscan)   │                          │                       │
```

---

## 6. Batch Mint Flow

```
BATCH MINT — HOW IT DIFFERS FROM SINGLE MINT
═══════════════════════════════════════════════════════════════════════════════

Single mint: Plutus one-shot policy (parameterised by UTxO)
             → 1 tx per NFT, each with unique policy
             → Expensive for collections

Batch mint:  Native script (time-locked)
             → ALL NFTs in 1 tx, all same policy
             → Same approach as NMKR, jpg.store


BATCH FLOW:
═══════════════════════════════════════════════════════════════════════════════

STEP 1 — PREPARE (uploads happen in background while user reviews)

User fills spreadsheet-style table: name, description, royalties per row
Uploads images
Clicks "Preview & Mint"
    │
    ▼
POST /api/batch/prepare (multipart: files[], names[], descriptions[], ...)
    │
    ▼
For each file:
  • Upload to Pinata → get ipfs://Qm... hash
  • Create nft record in DB (status = 'pending')
  • Create batch_item record
  │
  ▼
Backend creates batch_job record
Returns { batch_id, items: [{ nft_id, status, image_ipfs }...] }
    │
    ▼
Frontend shows preview grid with check marks as items upload


STEP 2 — MINT (one transaction for everything)

User clicks "Mint N NFTs on Cardano"
    │
    ▼
For EXTERNAL WALLET:
  getUtxos() from Lace (CIP-30)
  POST /api/batch/mint-unsigned { batch_id, wallet_utxos }
      │
      ▼
  Backend fetches all pending NFTs for this batch
  POST /api/mint/batch-unsigned { wallet_address, wallet_utxos, items }
      │
      ▼
  Sidecar builds native script:
    paymentKeyHash = resolvePaymentKeyHash(wallet_address)
    lockTime = now + 2 hours
    lockSlot = unixTimeToEnclosingSlot(lockTime, PREPROD)
    nativeScript = {
      type: "all",
      scripts: [
        { type: "sig", keyHash: paymentKeyHash },
        { type: "before", slot: lockSlot }
      ]
    }
    policyId = resolveNativeScriptHash(nativeScript)
      │
      ▼
  For each NFT in batch:
    .mint(refToken)    → to ImmutableLock
    .mint(userToken)   → to wallet_address
  Plus royalty token
  Plus CIP-25 metadata for entire collection
  .invalidHereafter(lockSlot)
  .complete() → unsigned CBOR
      │
      ▼
  Lace signTx popup → User signs once for ALL NFTs
      │
      ▼
  POST /api/submit → CSL assembly → Blockfrost
      │
      ▼
  POST /api/batch/confirm { batch_id, tx_hash, policy_id, tokens }
      │
      ▼
  UPDATE all NFTs: status = 'minted', tx_hash, policy_id
  UPDATE batch_job: status = 'completed', minted = N
      │
      ▼
  Success screen: "2 NFTs minted successfully" + Cardanoscan link
```

---

## 7. Database Schema — Complete

```
┌──────────────────────────────────────────────────────────────────────┐
│                           DATABASE: midnight_nft                      │
│                           PostgreSQL 15                               │
└──────────────────────────────────────────────────────────────────────┘

TABLE: users
┌────────────────┬──────────────────┬───────────────────────────────────┐
│ Column         │ Type             │ Notes                             │
├────────────────┼──────────────────┼───────────────────────────────────┤
│ id             │ UUID PRIMARY KEY │ auto-generated                    │
│ email          │ TEXT UNIQUE      │ nullable for external wallets     │
│ wallet_type    │ TEXT             │ 'custodial' or 'external'         │
│ otp_code       │ TEXT             │ 6-digit, cleared after use        │
│ otp_expires_at │ TIMESTAMP        │ 5 minute expiry                   │
│ nonce          │ TEXT             │ for CIP-30 signature verification │
│ created_at     │ TIMESTAMP        │                                   │
└────────────────┴──────────────────┴───────────────────────────────────┘

TABLE: custodial_wallets
┌────────────────┬──────────────────┬───────────────────────────────────┐
│ user_id        │ UUID → users.id  │ FK                                │
│ wallet_address │ TEXT             │ addr_test1...                     │
│ mnemonic       │ TEXT[]           │ 24-word array (stored encrypted)  │
│ created_at     │ TIMESTAMP        │                                   │
└────────────────┴──────────────────┴───────────────────────────────────┘

TABLE: external_wallets
┌────────────────┬──────────────────┬───────────────────────────────────┐
│ user_id        │ UUID → users.id  │ FK                                │
│ wallet_address │ TEXT             │ addr_test1... from Lace           │
│ created_at     │ TIMESTAMP        │                                   │
└────────────────┴──────────────────┴───────────────────────────────────┘

TABLE: nfts
┌─────────────────┬──────────────────┬──────────────────────────────────┐
│ id              │ UUID PRIMARY KEY  │                                  │
│ owner_id        │ UUID → users.id   │ FK — changes on transfer/buy    │
│ nft_name        │ TEXT              │ display name                    │
│ asset_name      │ TEXT              │ raw name (max 28 chars)         │
│ description     │ TEXT              │                                  │
│ image_ipfs      │ TEXT              │ ipfs://Qm...                    │
│ metadata_ipfs   │ TEXT              │ ipfs://Qm... (JSON metadata)    │
│ policy_id       │ TEXT              │ set after minting               │
│ user_token_name │ TEXT              │ "001bc280" + rawName            │
│ tx_hash         │ TEXT              │ minting transaction             │
│ royalties       │ FLOAT8            │ e.g. 5.0                        │
│ privacy         │ TEXT              │ 'public' or 'private'           │
│ status          │ TEXT              │ pending/minted/listed/          │
│                 │                   │ transferred/cancelled           │
│ created_at      │ TIMESTAMP         │                                  │
│ updated_at      │ TIMESTAMP         │                                  │
└─────────────────┴──────────────────┴──────────────────────────────────┘

TABLE: listings
┌──────────────────┬──────────────────┬─────────────────────────────────┐
│ id               │ UUID PRIMARY KEY  │                                 │
│ nft_id           │ UUID → nfts.id    │ FK                              │
│ seller_id        │ UUID → users.id   │ FK                              │
│ price_lovelace   │ BIGINT            │ e.g. 10000000 = 10 ADA         │
│ status           │ TEXT              │ active/sold/cancelled           │
│ nft_policy_id    │ TEXT              │ for building NFT unit           │
│ nft_asset_name   │ TEXT              │ "001bc280" + rawName            │
│ royalty_policy_id│ TEXT              │ policy holding royalty token    │
│ listing_tx_hash  │ TEXT              │ tx that locked NFT              │
│ script_utxo      │ TEXT              │ "txHash#0" — UTxO at script     │
│ created_at       │ TIMESTAMP         │                                 │
│ updated_at       │ TIMESTAMP         │                                 │
└──────────────────┴──────────────────┴─────────────────────────────────┘

TABLE: batch_jobs
┌────────────────┬──────────────────┬───────────────────────────────────┐
│ id             │ UUID PRIMARY KEY  │                                   │
│ user_id        │ UUID → users.id   │ FK                                │
│ total          │ INT               │ total NFTs in batch               │
│ uploaded       │ INT               │ successfully uploaded to IPFS     │
│ minted         │ INT               │ successfully minted on-chain      │
│ failed         │ INT               │                                   │
│ status         │ TEXT              │ pending/uploading/minting/        │
│                │                   │ completed/failed                  │
│ created_at     │ TIMESTAMP         │                                   │
│ updated_at     │ TIMESTAMP         │                                   │
└────────────────┴──────────────────┴───────────────────────────────────┘

TABLE: batch_items
┌────────────────┬──────────────────┬───────────────────────────────────┐
│ id             │ UUID PRIMARY KEY  │                                   │
│ batch_id       │ UUID → batch_jobs │ FK                                │
│ nft_id         │ UUID → nfts       │ FK                                │
│ row_order      │ INT               │ order in the batch upload        │
│ status         │ TEXT              │ uploading/uploaded/minting/       │
│                │                   │ minted/failed                    │
│ error_msg      │ TEXT              │ if failed                        │
└────────────────┴──────────────────┴───────────────────────────────────┘


RELATIONSHIPS:
═══════════════════════════════════════════════════════════════════════════════

users ──1:1──► custodial_wallets   (or)
users ──1:1──► external_wallets

users ──1:N──► nfts (via owner_id)
nfts  ──1:N──► listings (an NFT can be listed, cancelled, relisted)
users ──1:N──► batch_jobs
batch_jobs ──1:N──► batch_items ──1:1──► nfts
```

---

## 8. Smart Contracts — What Each Does

```
SOURCE: contracts/ (Aiken v1.1.19, Plutus V3)
COMPILED: plutus.json (loaded by sidecar at startup)

┌──────────────────────────────────────────────────────────────────────────┐
│  VALIDATOR 1: minting_policy.cip68_mint_single.mint                      │
│                                                                          │
│  Type: Minting Policy (Plutus V3)                                        │
│  Parameterised by: OutputReference (txHash + outputIndex)                │
│                                                                          │
│  What it does:                                                           │
│  • Can only run once — when the parameterised UTxO is consumed           │
│  • Allows minting of exactly: refToken + userToken + royaltyToken        │
│  • After that, the policy is "spent" forever                             │
│                                                                          │
│  Why one-shot?                                                           │
│  • Unique policy per NFT — no one can mint more of "your" NFT           │
│  • The UTxO is on the blockchain — provably unique, not reusable        │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  VALIDATOR 2: reference_lock.immutable_lock.spend                        │
│                                                                          │
│  Type: Spending Validator (Plutus V3)                                    │
│  Address: derived from compiled code (same address always)               │
│                                                                          │
│  What it does:                                                           │
│  • Holds all CIP-68 reference tokens (000643b0...)                       │
│  • ALWAYS FAILS — tokens locked here can NEVER leave                    │
│  • The datum (metadata) is readable by anyone via Blockfrost            │
│                                                                          │
│  Why "always fail"?                                                      │
│  • Reference tokens must stay here forever — they ARE the metadata      │
│  • If someone could move them, they could corrupt the NFT metadata      │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  VALIDATOR 3: royalty_lock.royalty_lock.spend                            │
│                                                                          │
│  Type: Spending Validator (Plutus V3)                                    │
│                                                                          │
│  What it does:                                                           │
│  • Holds royalty tokens (001f4d70...)                                    │
│  • Datum encodes: creator address + royalty % + minimum payout           │
│  • Marketplace reads this datum to calculate royalty on every sale      │
│  • Tokens are locked (similar to immutable lock)                        │
│                                                                          │
│  Why a separate contract?                                                │
│  • Royalty info must be trustless and on-chain                          │
│  • Creator can't change royalty after minting (no rug pulls)            │
│  • Any marketplace can read it — not platform-specific                  │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  VALIDATOR 4: Marketplace Script (in plutus.json)                        │
│                                                                          │
│  Type: Spending Validator (Plutus V3)                                    │
│                                                                          │
│  What it does:                                                           │
│  • Holds listed NFTs (user tokens)                                       │
│  • Datum: { price_lovelace, seller_address, royalty_policy_id }         │
│  • Redeemer: "Buy" or "Cancel"                                           │
│                                                                          │
│  Buy validation:                                                         │
│  ✓ seller receives price_lovelace - royalty_amount                      │
│  ✓ royalty_lock receives royalty_amount                                 │
│  ✓ buyer receives the NFT token                                         │
│                                                                          │
│  Cancel validation:                                                      │
│  ✓ transaction signed by original seller                                │
│  ✓ NFT returns to seller                                                │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 9. External Services — Every Detail

```
┌──────────────────────────────────────────────────────────────────────────┐
│  BLOCKFROST                                                               │
│  URL: https://cardano-preprod.blockfrost.io/api/v0                       │
│  Key: preprodXopmzcOFm9lBJvtectX0bnEiv8m79PyQ                           │
│                                                                          │
│  What it is:                                                             │
│  Managed Cardano node API. Abstracts away running a full Cardano node    │
│  (which requires 100GB+ disk, 16GB RAM, constant sync).                 │
│                                                                          │
│  Used for:                                                               │
│  1. Submit transactions → POST /tx/submit                                │
│  2. Fetch UTxOs at address → GET /addresses/{addr}/utxos                │
│  3. Fetch ADA balance → GET /addresses/{addr}                           │
│  4. Fetch NFT metadata → GET /assets/{unit}                             │
│  5. Fetch assets at address → GET /addresses/{addr}/assets              │
│  6. Retry logic → fetch script UTxO after listing (5×6s retries)       │
│                                                                          │
│  Rate limits: 500 req/10s on free tier                                  │
│  Network: Preprod (testnet) — same as mainnet behavior                  │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  PINATA                                                                   │
│  URL: https://api.pinata.cloud                                           │
│  Gateway: https://gateway.pinata.cloud/ipfs/{hash}                      │
│                                                                          │
│  What it is:                                                             │
│  IPFS pinning service. IPFS is a distributed file system where files    │
│  are addressed by their content hash (not URL). "Pinning" means         │
│  keeping the file available permanently.                                 │
│                                                                          │
│  Used for:                                                               │
│  1. Upload NFT image → returns CIDv1 hash (ipfs://Qm...)               │
│  2. This hash is stored in the CIP-68 datum and CIP-25 metadata         │
│  3. Any IPFS gateway can serve it (not just Pinata)                    │
│  4. Frontend uses gateway.pinata.cloud to display images                │
│                                                                          │
│  Why IPFS over S3/normal hosting?                                        │
│  • Content-addressed: hash changes if file changes → tamper-proof      │
│  • Decentralised: survives if our platform goes down                   │
│  • Standard for NFTs globally                                           │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  CARDANOSCAN (PREPROD)                                                   │
│  URL: https://preprod.cardanoscan.io                                     │
│                                                                          │
│  What it is:                                                             │
│  Cardano blockchain explorer. Shows transactions, tokens, addresses.    │
│                                                                          │
│  Used for:                                                               │
│  • "View Transaction" links: /transaction/{txHash}                      │
│  • "View Asset" links: /token/{policyId}{assetNameHex}                  │
│  • "View Address" links: /address/{addr}                                │
│                                                                          │
│  CRITICAL BUG WE FIXED:                                                 │
│  DB stores: "001bc280" + rawName (e.g. "001bc280Checking")             │
│  Cardanoscan needs: "001bc280" + hex(rawName)                           │
│    = "001bc280" + "436865636b696e67"                                    │
│    = "001bc280436865636b696e67"                                         │
│  Solution: utils/cardano.ts → toOnChainHex() function                  │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  GMAIL SMTP                                                               │
│  Host: smtp.gmail.com:587                                                │
│                                                                          │
│  What it is:                                                             │
│  Sends OTP emails to custodial wallet users.                            │
│                                                                          │
│  Setup required:                                                         │
│  • Gmail account with 2FA enabled                                       │
│  • App Password generated (not regular password)                        │
│  • SMTP_USER and SMTP_PASSWORD in backend .env                         │
│                                                                          │
│  Why not SendGrid/Mailgun?                                              │
│  • Gmail free tier sufficient for dev/demo                              │
│  • No additional API keys needed                                        │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  LACE WALLET (BROWSER EXTENSION)                                         │
│  Extension ID: gafhhkghbfjjkeiendhlofajokpaflmk                         │
│                                                                          │
│  What it is:                                                             │
│  Cardano wallet browser extension built by IOG (Cardano's creator).    │
│  Implements CIP-30 standard for dApp communication.                    │
│                                                                          │
│  CIP-30 methods we use:                                                 │
│  • window.cardano.lace.enable() → get wallet API                       │
│  • walletApi.getUsedAddresses() → get wallet address                   │
│  • walletApi.getUtxos() → get all UTxOs (all HD wallet addresses)      │
│  • walletApi.signData(addr, nonce) → prove ownership (login)           │
│  • walletApi.signTx(cbor, partialSign) → sign transaction              │
│                                                                          │
│  Known quirks we handle:                                                │
│  • signTx returns WITNESS SET only (not full tx) → CSL assembly needed │
│  • submitTx() is broken → we submit via Blockfrost instead            │
│  • Session expires after page navigation → re-enable before each action│
│  • HD wallet: funds spread across multiple addresses → getUtxos() needed│
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Frontend Architecture — Every File

```
frontend/src/
│
├── main.ts                 Entry point, mount app
├── App.vue                 Root component, sidebar layout
│
├── router/
│   └── index.ts            Route definitions
│                           Protected routes check JWT via /api/me
│                           Auth routes redirect if already logged in
│
├── stores/                 Pinia state management
│   ├── auth.ts             userID, walletType, walletAddress
│   │                       login(), logout(), checkAuth()
│   │                       checkAuth() called on app mount and route change
│   │
│   ├── dashboard.ts        nfts[], myListings[], lovelace, walletAddress
│   │                       loadDashboard() — detects custodial vs external
│   │                       loadCustodialDashboard() — calls /api/nft/my-nfts
│   │                       loadExternalDashboard() — calls /api/nft/external-assets
│   │                       uniqueNFTs computed (deduped for activity)
│   │                       reset() — called on logout
│   │
│   └── batch.ts            batchId, progress, isLoading, error
│                           prepare() — multipart upload to /api/batch/prepare
│                           mint() — custodial /api/batch/mint
│                           reset()
│
├── composables/
│   └── useWalletSession.ts CIP-30 wallet manager
│                           walletApi — the CIP-30 API object (in memory)
│                           walletId — 'lace' etc (localStorage)
│                           setWallet(id, api) — called after login
│                           signOnly(cbor) — re-enables wallet, calls signTx
│                           getUtxos() — re-enables wallet, calls getUtxos
│                           restoreWallet() — re-enables from localStorage
│                           clearWallet() — called on logout
│
├── services/               API call wrappers (axios)
│   ├── api.ts              Axios instance with base URL, credentials: include
│   ├── nft.ts              prepareMint, mintNFT, mintNFTUnsigned, confirmMint
│   │                       transferNFT, transferNFTUnsigned, confirmTransfer
│   │                       submitSignedTx
│   └── listing.ts          createListing, createListingUnsigned, confirmCreateListing
│                           buyListing, buyListingUnsigned, confirmBuyListing
│                           cancelListing, cancelListingUnsigned, confirmCancelListing
│                           getAllListings, getMyListings
│
├── utils/
│   └── cardano.ts          toOnChainHex(tokenName) — converts DB format to Cardanoscan
│                           cardanoscanTokenUrl(policyId, tokenName)
│                           cardanoscanTxUrl(txHash)
│                           lovelaceToAda(lovelace)
│                           shortHash(hash, start, end)
│
├── views/                  Page components (one per route)
│   ├── AuthView.vue        Login page — OTP tab + Lace tab
│   ├── DashboardView.vue   Main dashboard — stats, NFT grid categories, activity
│   ├── CreateMintView.vue  Single mint — file upload, metadata, privacy, strategy
│   ├── BatchMintView.vue   Batch mint — spreadsheet UI, preview, progress
│   ├── BrowseMintsView.vue Public marketplace — grid, filters, sort
│   ├── NFTDetailView.vue   Full NFT view — image, details, list/cancel/transfer
│   ├── ActivityView.vue    Transaction history — grouped by date, modal on click
│   ├── CertificateView.vue Public shareable NFT certificate (no auth)
│   ├── SettingsView.vue    Profile, wallet address, platform info, logout
│   └── NotFoundView.vue    404 page
│
└── components/
    ├── auth/
    │   └── WalletConnect.vue  CIP-30 login — enable wallet, sign nonce, verify
    │
    ├── layout/
    │   ├── AppSidebar.vue     Collapsible sidebar — icons + labels on hover
    │   ├── WalletPopup.vue    Balance + address popup (click Wallet icon)
    │   └── AppNavbar.vue      (if applicable)
    │
    ├── dashboard/
    │   ├── NFTCard.vue        Single NFT card in grid
    │   └── WalletCard.vue     Wallet balance display
    │
    ├── marketplace/
    │   ├── ListingDetailPanel.vue  OpenSea-style side panel for buy flow
    │   ├── ListConfirmModal.vue    List price review before signing
    │   ├── BuyConfirmModal.vue     Buy price breakdown
    │   └── BuySuccessModal.vue     Post-purchase modal with Cardanoscan links
    │
    ├── mint/
    │   ├── BatchTable.vue     Spreadsheet-style batch input table
    │   └── MetadataForm.vue   NFT metadata fields
    │
    └── shared/
        ├── TxSuccessCard.vue       Reusable success card (list/cancel/transfer)
        └── ActivityDetailModal.vue Detail modal when clicking activity row
```

---

## 11. Key Architectural Decisions — Why We Did It This Way

```
DECISION 1: Why separate Go backend + Node.js sidecar?
════════════════════════════════════════════════════════
Problem: MeshSDK (best Cardano tx builder) is TypeScript only.
Options:
  A) Use Go only → need to reimplement MeshSDK from scratch (weeks of work)
  B) Use Node.js only → lose Go's type safety and ecosystem
  C) Split: Go handles business logic, Node.js handles blockchain
Answer: C — each language does what it's best at.
Communication: HTTP with a shared secret header (X-Service-Secret).


DECISION 2: Why CSL for transaction assembly?
═══════════════════════════════════════════════
Problem: Lace's signTx() returns ONLY the witness set (vkeys and signatures).
         It does NOT return the full signed transaction.
Options:
  A) Use wallet.submitTx() after signing → broken in Lace (known bug)
  B) Reconstruct full tx manually → complex binary serialization
  C) Use @emurgo/cardano-serialization-lib to combine body + witness
Answer: C — CSL is the canonical Cardano binary library.
Code:
  const tx = CSL.Transaction.from_hex(unsigned_cbor)
  const ws = CSL.TransactionWitnessSet.from_hex(witness_cbor)
  const signed = CSL.Transaction.new(tx.body(), ws, tx.auxiliary_data())


DECISION 3: Why parseCip30Utxos() instead of Blockfrost for UTxOs?
═══════════════════════════════════════════════════════════════════════
Problem: Lace HD wallet spreads funds across 20+ derived addresses.
         Blockfrost only knows about one address at a time.
Solution: CIP-30's getUtxos() returns ALL UTxOs from ALL derived addresses.
          We parse the hex-CBOR format ourselves.
Why CBOR parsing is needed:
  • CIP-30 spec says UTxOs are returned as hex-encoded CBOR
  • Lace returns them exactly like this
  • MeshSDK expects JavaScript objects not raw CBOR hex
Our parser: uses 'cbor' npm package to decode, then extract:
  txHash, outputIndex, address (bech32), lovelace, native assets


DECISION 4: Why metadataToCip68() instead of manual list-of-pairs?
═══════════════════════════════════════════════════════════════════════
Problem: Initially used manual datum construction:
  mConStr0([[
    [stringToHex("name"), stringToHex(asset_name)],
    ...
  ]])
This creates a Plutus List. Lace reads CIP-68 and expects a Plutus Map.
The two are serialized differently in CBOR.
Fix: MeshSDK's metadataToCip68() creates the correct Plutus Map format.
Result: Images now show in Lace within 30-60 minutes of minting.


DECISION 5: Why CIP-25 in addition to CIP-68?
═══════════════════════════════════════════════
CIP-68 = datum in reference token output → needs indexing
CIP-25 = label 721 in transaction metadata → immediately available

Most wallets (Lace, Nami, Eternl) read CIP-25 for display.
Only advanced tools read CIP-68 datum.
So we write BOTH to maximize compatibility.


DECISION 6: Why one-shot policy for single mint?
══════════════════════════════════════════════════
A one-shot minting policy is parameterised by a specific UTxO.
Once that UTxO is consumed (spent in the mint tx), the policy expires.
This means:
  • Each single NFT has a completely unique policy ID
  • Nobody can ever mint another token under that policy
  • Provably scarce — 1 of 1 forever
For batch: native script with time lock is used instead.


DECISION 7: Why treasury wallet for custodial fees?
════════════════════════════════════════════════════
Each custodial user has their own wallet (mnemonic). But a new wallet
has no ADA to pay fees for the first mint.
Solution: Treasury wallet pays the initial fee + provides collateral.
The treasury is funded from the faucet and covers:
  • First mint network fee (~0.17 ADA)
  • Min-ADA locked in script outputs (~2 ADA per output)
These are returned to users over time through marketplace sales.
```

---

## 12. Token Name Encoding — The Bug That Took Days to Find

```
THE PROBLEM:
════════════════════════════════════════════════════════════════════
Cardanoscan token URL: /token/{policyId}{assetNameHex}

Where assetNameHex = FULLY hex-encoded on-chain token name.

For "Checking", the on-chain token name is:
  001bc280 + hex("Checking")
= 001bc280 + 436865636b696e67
= 001bc280436865636b696e67

But our DB stores:
  user_token_name = "001bc280" + "Checking"
                 = "001bc280Checking"

When we built the Cardanoscan URL directly from DB:
  /token/{policyId}001bc280Checking
→ Cardanoscan returned "Couldn't find what you are looking for"


THE FIX: utils/cardano.ts → toOnChainHex()
════════════════════════════════════════════════════════════════════

function toOnChainHex(tokenName: string): string {
  if (tokenName.length < 8) return tokenName
  const prefix = tokenName.slice(0, 8)  // "001bc280"
  const rest   = tokenName.slice(8)     // "Checking"
  
  // If rest contains non-hex chars → it's a raw name, needs encoding
  if (/[^0-9a-fA-F]/.test(rest)) {
    const hexEncoded = Array.from(rest)
      .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
      .join('')
    return prefix + hexEncoded
    // "001bc280" + "436865636b696e67"
  }
  
  // Already hex-encoded (from Blockfrost for external wallet NFTs)
  return tokenName
}

This handles BOTH:
  • Platform-minted NFTs (DB has prefix + rawName)
  • External wallet NFTs (Blockfrost returns fully hex-encoded names)
```

---

*LankaNFT · Cardano Preprod · Complete Architecture v1.0*