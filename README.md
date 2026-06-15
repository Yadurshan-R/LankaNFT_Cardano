# LankaNFT — Cardano NFT Platform

> A full-stack NFT minting and marketplace platform built on Cardano Preprod. Supports both custodial wallets (email login) and external wallets (Lace/CIP-30) with the same feature set.



##  Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        FRONTEND                                  │
│                   Vue 3 + Pinia (Port 5173)                     │
│                                                                  │
│   Dashboard │ Create Mint │ Batch Upload │ Browse Mints         │
│   NFT Detail │ Activity │ Certificate │ Settings                │
│                                                                  │
│   External Wallet Flow:          Custodial Wallet Flow:         │
│   CIP-30 → getUtxos()            JWT Auth → API calls           │
│   signTx(unsigned_cbor)          Backend signs with mnemonic    │
│   submitSignedTx()                                              │
└──────────────────┬──────────────────────────────────────────────┘
                   │ HTTP / REST
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                      BACKEND (Go + Gin)                          │
│                        Port 8080                                 │
│                                                                  │
│  Auth        │  NFT         │  Listing      │  Batch            │
│  ─────────── │  ─────────── │  ──────────── │  ──────────────  │
│  OTP Login   │  PrepareMint │  Create       │  Prepare          │
│  WalletVerify│  MintNFT     │  Buy          │  Mint             │
│  JWT Session │  Transfer    │  Cancel       │  MintUnsigned     │
│              │  Unsigned ✕4 │  Unsigned ✕3  │  Confirm          │
│                                                                  │
│  PostgreSQL Database                                            │
│  Tables: users, custodial_wallets, external_wallets,           │
│          nfts, listings, batch_jobs, batch_items               │
└──────────────────┬──────────────────────────────────────────────┘
                   │ HTTP / Internal Secret
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│              BLOCKCHAIN SIDECAR (Node.js + MeshSDK)             │
│                        Port 3001                                 │
│                                                                  │
│  /api/mint/single           │  /api/mint/single-unsigned        │
│  /api/mint/batch            │  /api/mint/batch-unsigned         │
│  /api/marketplace/list      │  /api/marketplace/list-unsigned   │
│  /api/marketplace/buy       │  /api/marketplace/buy-unsigned    │
│  /api/marketplace/cancel    │  /api/marketplace/cancel-unsigned │
│  /api/transfer              │  /api/transfer/unsigned           │
│  /api/submit                │  /api/wallet/generate             │
│                                                                  │
│  CSL (Cardano Serialization Lib) — assembles signed tx         │
└──────────────────┬──────────────────────────────────────────────┘
                   │ API Calls
        ┌──────────┴──────────┐
        ▼                     ▼
┌──────────────┐    ┌──────────────────┐
│  Blockfrost  │    │     Pinata       │
│  (Cardano    │    │   (IPFS image    │
│   Preprod)   │    │    storage)      │
└──────────────┘    └──────────────────┘
```

---

##  Transaction Flows

### External Wallet — Single Mint
```
User → Upload Image → PrepareMint (IPFS upload, DB record)
     → getUtxos() via CIP-30
     → POST /api/nft/mint-unsigned → Sidecar builds unsigned CBOR
     → Lace signTx() popup → witness CBOR returned
     → POST /api/submit → CSL assembles full tx → Blockfrost submits
     → POST /api/nft/confirm-mint → DB updated to minted
```

### External Wallet — Marketplace Buy
```
User → Click Buy → POST /api/listing/buy-unsigned → Sidecar builds tx
     → Lace signTx() popup → witness CBOR
     → POST /api/submit → submitted on-chain
     → POST /api/listing/confirm-buy → listing marked sold, NFT ownership transferred
```

### Custodial Wallet — Any Action
```
User → Action → Backend loads mnemonic from DB
     → Sidecar builds + signs + submits in one step
     → DB updated
```

---

##  NFT Token Architecture (CIP-68)

Every mint creates **3 tokens** under one policy:

```
Policy ID (unique per NFT / collection)
│
├── 000643b0 + hex(name)  →  Reference Token
│   Locked in ImmutableLock contract
│   Holds metadata as inline Plutus datum:
│   { name, image (IPFS), mediaType, description }
│
├── 001bc280 + hex(name)  →  User Token  ← This is the NFT you own
│   Sent to creator's wallet
│   Transferred on buy/sell/transfer
│
└── 001f4d70 + hex(name)  →  Royalty Token
    Locked in RoyaltyLock contract
    Encodes: creator address, royalty % (e.g. 5%)
    Read by marketplace on every sale
```

**CIP-25 metadata** (label 721) is also written to the transaction so wallets like Lace can display the image immediately without reading the datum.

---

##  Marketplace Architecture

```
LISTING:
  User NFT (wallet) ──mint──► Marketplace Script (locked)
  Script stores: price, seller, royalty policy

BUYING:
  Buyer ADA ──────────────────► Seller (price - royalty)
                              ► Royalty Lock (creator's cut)
                              ► NFT ──────────────────────► Buyer

CANCELLING:
  Marketplace Script ──unlock──► Original Owner's Wallet
```

Smart contracts built with **Aiken v1.1.19** (Plutus V3).

---

##  Wallet Support

| Feature | Custodial (Email) | External (Lace) |
|---------|:-----------------:|:---------------:|
| Login | OTP email | CIP-30 signature |
| Key custody | Platform (encrypted) | User's device |
| Single mint | ✅ | ✅ |
| Batch mint | ✅ | ✅ |
| List for sale | ✅ | ✅ |
| Cancel listing | ✅ | ✅ |
| Buy NFT | ✅ | ✅ |
| Transfer NFT | ✅ | ✅ |
| Pending re-mint | ✅ | ✅ |

---

##  Tech Stack

### Frontend
| Technology | Purpose |
|-----------|---------|
| Vue 3 + Composition API | UI framework |
| Pinia | State management |
| Vue Router | Client-side routing |
| Lucide Vue | Icons |
| Bech32 | Cardano address conversion |
| Vite | Build tool |

### Backend
| Technology | Purpose |
|-----------|---------|
| Go 1.22 + Gin | REST API server |
| pgx v5 | PostgreSQL driver |
| JWT | Session authentication |
| SMTP (Gmail) | OTP email delivery |

### Blockchain Sidecar
| Technology | Purpose |
|-----------|---------|
| Node.js + TypeScript | Runtime |
| MeshSDK | Cardano transaction building |
| @emurgo/cardano-serialization-lib | Signed tx assembly (CSL) |
| cbor | CIP-30 UTxO parsing |
| bech32 | Address conversion |

### Infrastructure
| Service | Purpose |
|---------|---------|
| Blockfrost (Preprod) | Cardano node API |
| Pinata | IPFS image pinning |
| PostgreSQL | Application database |

---

##  Getting Started

### Prerequisites
- Go 1.22+
- Node.js 18+
- PostgreSQL 15+
- Blockfrost Preprod API key
- Pinata API key

### Environment Variables

**Backend** (`backend/.env`):
```env
DB_URL=postgres://user:password@localhost:5432/midnight_nft
JWT_SECRET=your_jwt_secret
BLOCKFROST_PROJECT_ID=preprod...
BLOCKFROST_BASE_URL=https://cardano-preprod.blockfrost.io/api/v0
BLOCKCHAIN_SERVICE_URL=http://localhost:3001
BLOCKCHAIN_SERVICE_SECRET=your_shared_secret
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@gmail.com
SMTP_PASSWORD=your_app_password
TREASURY_ADDRESS=addr_test1...
```

**Blockchain Sidecar** (`blockchain-service/.env`):
```env
BLOCKFROST_PROJECT_ID=preprod...
BLOCKCHAIN_SERVICE_SECRET=your_shared_secret
```

### Running Locally

```bash
# 1. Database
createdb midnight_nft
psql midnight_nft < backend/schema.sql

# 2. Backend
cd backend
go run cmd/api/main.go

# 3. Blockchain Sidecar
cd blockchain-service
npm install
npm run dev

# 4. Frontend
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`

---

##  Key API Endpoints

### Auth
```
POST /auth/request-otp      — Send OTP to email
POST /auth/verify-otp       — Verify OTP, get JWT
GET  /auth/nonce            — Get signing nonce (external wallet)
POST /auth/wallet-verify    — Verify CIP-30 signature, get JWT
POST /auth/logout           — Clear session
```

### NFTs
```
POST /api/nft/prepare-mint      — Upload to IPFS, create DB record
POST /api/nft/mint              — Custodial: build + sign + submit
POST /api/nft/mint-unsigned     — External: build unsigned CBOR
POST /api/nft/confirm-mint      — Update DB after external wallet submits
POST /api/nft/transfer          — Custodial transfer
POST /api/nft/transfer-unsigned — External transfer unsigned
POST /api/nft/confirm-transfer  — Confirm external transfer
GET  /api/nft/my-nfts           — Get user's NFTs
GET  /api/nft/balance           — Get wallet ADA balance
GET  /api/nft/external-assets   — Get Lace wallet NFTs via Blockfrost
```

### Marketplace
```
POST /api/listing/create              — Custodial list
POST /api/listing/create-unsigned     — External list unsigned
POST /api/listing/confirm-create      — Confirm external listing
POST /api/listing/buy                 — Custodial buy
POST /api/listing/buy-unsigned        — External buy unsigned
POST /api/listing/confirm-buy         — Confirm external buy
POST /api/listing/cancel              — Custodial cancel
POST /api/listing/cancel-unsigned     — External cancel unsigned
POST /api/listing/confirm-cancel      — Confirm external cancel
GET  /api/listing/all                 — All active listings
GET  /api/listing/mine                — My listings
```

### Batch
```
POST /api/batch/prepare          — Upload all images to IPFS
POST /api/batch/mint             — Custodial batch mint
POST /api/batch/mint-unsigned    — External batch mint unsigned
POST /api/batch/confirm          — Confirm external batch
GET  /api/batch/:id              — Batch job status
```

### Other
```
POST /api/submit                 — Submit signed tx (external wallets)
GET  /api/certificate/:id        — Public NFT certificate (no auth)
```

---

##  Database Schema

```sql
users               — user_id, email, created_at
custodial_wallets   — user_id, wallet_address, encrypted_mnemonic
external_wallets    — user_id, wallet_address
nfts                — id, owner_id, nft_name, asset_name, policy_id,
                      user_token_name, image_ipfs, metadata_ipfs,
                      description, royalties, privacy, status,
                      tx_hash, created_at
listings            — id, nft_id, seller_id, price_lovelace, status,
                      nft_policy_id, nft_asset_name, royalty_policy_id,
                      listing_tx_hash, script_utxo
batch_jobs          — id, user_id, total, uploaded, minted, status
batch_items         — id, batch_id, nft_id, row_order, status
```

---

##  Key Technical Decisions

### Why CIP-68 not CIP-25?
CIP-68 stores metadata on-chain in a Plutus datum. It's updatable (in theory), trustless, and is the emerging standard for serious NFT projects. CIP-25 only writes to transaction metadata — immutable and limited. We use **both**: CIP-68 for on-chain data, CIP-25 for wallet display compatibility.

### Why a Node.js Sidecar?
MeshSDK (the best Cardano transaction builder) is TypeScript-only. Go has no comparable library. Rather than re-implement transaction building in Go, we run a lightweight Node.js service that handles all Cardano-specific logic. Go handles business logic, auth, and DB. The sidecar handles blockchain.

### Why CSL for Tx Assembly?
Lace's `signTx()` returns only the **witness set** (signatures), not the full transaction. We must reconstruct the full tx by combining the original unsigned CBOR + witness CBOR. The Cardano Serialization Library (CSL) is the only reliable way to do this.

### Why Blockfrost?
Running a Cardano node requires 100GB+ storage and significant maintenance. Blockfrost is a managed Cardano node API used by the entire ecosystem. For preprod testing and production at moderate scale, it's the standard choice.

### Why Native Script for Batch?
Single NFT mints use a one-shot Plutus policy (parameterized by a UTxO). For batch, this would require one policy per NFT. Instead, batch uses a **native script** with a time lock — one policy for the entire collection, signed by the creator's wallet key. This is how NMKR and jpg.store handle collections.

---

##  Project Structure

```
NFT_Minting_Platform/
├── backend/                    # Go REST API
│   ├── cmd/api/main.go         # Entry point, route registration
│   ├── internal/
│   │   ├── auth/               # OTP + wallet auth handlers
│   │   ├── nft/                # NFT mint/transfer handlers
│   │   ├── listing/            # Marketplace handlers
│   │   └── batch/              # Batch mint handlers
│   └── pkg/blockchain/         # Sidecar client types
│
├── blockchain-service/         # Node.js blockchain sidecar
│   └── src/
│       ├── app.ts              # Express server + /api/submit (CSL)
│       ├── config.ts           # Environment config
│       └── routes/
│           ├── mint.ts         # Single mint (signed + unsigned)
│           ├── batch.ts        # Batch mint (signed + unsigned)
│           ├── marketplace.ts  # List/Buy/Cancel (signed + unsigned)
│           ├── transfer.ts     # Transfer (signed + unsigned)
│           └── wallet.ts       # Wallet generation
│
├── frontend/                   # Vue 3 frontend
│   └── src/
│       ├── views/              # Page components
│       ├── components/         # Reusable UI components
│       │   ├── marketplace/    # Listing panels, buy modal
│       │   └── shared/         # TxSuccessCard, ActivityDetailModal
│       ├── stores/             # Pinia state (auth, dashboard, batch)
│       ├── services/           # API service functions
│       ├── composables/        # useWalletSession (CIP-30)
│       └── utils/cardano.ts    # Cardanoscan URL builder (hex encoding)
│
├── contracts/                  # Aiken smart contracts
│   └── plutus.json             # Compiled validator scripts
│
└── README.md
```

---

##  Testing on Preprod

Get test ADA from the [Cardano Preprod Faucet](https://docs.cardano.org/cardano-testnets/tools/faucet/).

For Lace wallet on Preprod:
1. Open Lace → Settings → Network → Switch to Preprod
2. Connect at `localhost:5173`

---

##  Known Limitations

- **Mainnet not configured** — currently Preprod only. Switching requires updating Blockfrost project ID and network IDs in the sidecar.
- **Lace image display delay** — New NFT images appear in Lace 30–60 minutes after minting (Preprod indexing delay).
- **Old NFTs pre-CIP-25** — NFTs minted before CIP-25 metadata was added will always show blank in wallets. Blockchain is immutable.
- **UTxO cooldown** — Minting multiple NFTs too quickly causes `ConwayMempoolFailure` (UTxO already spent). A 20-second cooldown is enforced between single mints.

---

##  Built By

Yadurshan Rajakumar — [github.com/Yadurshan-R](https://github.com/Yadurshan-R)

---

*LankaNFT · Cardano Preprod · v1.0.0*
