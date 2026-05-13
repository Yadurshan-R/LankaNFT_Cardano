// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/batch.ts
//
// CIP-68 + CIP-102 Compliant Batch NFT Minting — Time-Locked Native Script
//
// Professional batch minting — same architecture as NMKR and jpg.store.
//
// FIXES APPLIED:
//
//   Fix 1 — CIP-68 datum uses proper Plutus Map format via metadataToCip68()
//            Old: list of tuples (wrong CBOR encoding, marketplaces reject it)
//            New: Plutus Map (correct CIP-68 encoding, all marketplaces accept)
//
//   Fix 2 — Royalty token name follows CIP-102 correctly
//            Old: hardcoded wrong hex
//            New: "001f4d70" (label 500) + hex(first_nft_asset_name)
//
//   Fix 3 — Policy is time-locked after 2 hours
//            Old: open forever, creator can mint more tokens anytime
//            New: NativeScript { all: [sig, before(slot)] } — locks after 2h
//
//   Fix 4 — Native script IS the correct professional standard for batch
//            Our Aiken cip68_mint_collection accepts one asset_name per
//            redeemer — cannot validate multiple NFTs in one tx without a
//            contract rewrite. Native script is what NMKR and jpg.store use.
//
// References:
//   CIP-68: https://cips.cardano.org/cip/CIP-68
//   CIP-102: https://cips.cardano.org/cip/CIP-102
// ─────────────────────────────────────────────────────────────────────────────

import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  MeshTxBuilder,
  MeshWallet,
  NativeScript,
  SLOT_CONFIG_NETWORK,
  applyCborEncoding,
  mConStr0,
  metadataToCip68,
  resolveNativeScriptHash,
  resolvePaymentKeyHash,
  serializeData,
  serializeNativeScript,
  serializePlutusScript,
  stringToHex,
  unixTimeToEnclosingSlot,
} from "@meshsdk/core";
import config from "../config";
import fs from "fs";
import path from "path";

const router = Router();

// ─── Load Blueprint ───────────────────────────────────────────────────────────
const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

function getValidator(title: string) {
  const v = blueprintJson.validators.find((v: any) => v.title === title);
  if (!v) throw new Error(`Validator "${title}" not found in plutus.json`);
  return v;
}

const immutableLockValidator = getValidator(
  "reference_lock.immutable_lock.spend"
);
const royaltyLockValidator = getValidator("royalty_lock.royalty_lock.spend");

// ─── Script Addresses ─────────────────────────────────────────────────────────
// Reference NFTs locked here permanently (CIP-68)
const immutableLockAddress = serializePlutusScript(
  {
    code: applyCborEncoding(immutableLockValidator.compiledCode),
    version: "V3",
  },
  undefined,
  0
).address;

// Royalty NFT locked here (CIP-102)
const royaltyLockAddress = serializePlutusScript(
  {
    code: applyCborEncoding(royaltyLockValidator.compiledCode),
    version: "V3",
  },
  undefined,
  0
).address;

// ─── Types ────────────────────────────────────────────────────────────────────

interface BatchNFTItem {
  nft_id: string
  asset_name: string
  metadata_ipfs: string
  image_ipfs: string
  royalties: number
}

// ─── Route ────────────────────────────────────────────────────────────────────

/**
 * POST /api/mint/batch
 *
 * Mints ALL NFTs in ONE Cardano transaction.
 * Uses a time-locked native script collection policy.
 *
 * Request body:
 *   mnemonic  string[]       — 24 word mnemonic of the custodial wallet
 *   items     BatchNFTItem[] — list of NFTs to mint (max 100)
 *
 * Response:
 *   tx_hash    string   — single Cardano transaction hash for all NFTs
 *   policy_id  string   — shared policy ID for the whole collection
 *   lock_slot  number   — slot after which no more minting is possible
 *   minted     number   — number of NFTs minted
 *   tokens     object[] — list of minted token details
 */
router.post("/batch", async (req: Request, res: Response) => {
  try {
    const { mnemonic, items }: { mnemonic: string[]; items: BatchNFTItem[] } =
      req.body;

    // ── Validation ────────────────────────────────────────────────────────
    if (!mnemonic || !Array.isArray(items) || items.length === 0) {
      res.status(400).json({
        error: "mnemonic and items array are required",
      });
      return;
    }

    if (items.length > 100) {
      res.status(400).json({
        error: "Maximum 100 NFTs per batch",
      });
      return;
    }

    // ── Setup provider and wallet ─────────────────────────────────────────
    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    const wallet = new MeshWallet({
      networkId: 0,
      fetcher: provider,
      submitter: provider,
      key: { type: "mnemonic", words: mnemonic },
    });

    const utxos = await wallet.getUtxos();
    if (utxos.length === 0) {
      res.status(400).json({
        error: "Wallet has no UTxOs — fund the preprod wallet via the faucet",
      });
      return;
    }

    const creatorAddress = await wallet.getChangeAddress();

    // ── Fix 3: Time-locked native script policy ───────────────────────────
    // Extract payment key hash from creator address
    const paymentKeyHash = resolvePaymentKeyHash(creatorAddress);

    // Lock policy 2 hours from now on Cardano Preprod
    // After lockSlot: nobody — not even the creator — can mint more tokens
    // This makes the collection supply provably fixed
    const LOCK_WINDOW_MS = 2 * 60 * 60 * 1000; // 2 hours in ms
    const lockTime = Date.now() + LOCK_WINDOW_MS;
    const lockSlot = unixTimeToEnclosingSlot(
      lockTime,
      SLOT_CONFIG_NETWORK["preprod"]
    );

    // Native script: ALL conditions must be satisfied:
    //   1. Transaction signed by creator's payment key
    //   2. Transaction submitted before lockSlot
    const nativeScript: NativeScript = {
      type: "all",
      scripts: [
        {
          type: "sig",
          keyHash: paymentKeyHash,
        },
        {
          type: "before",
          slot: lockSlot.toString(),
        },
      ],
    };

    // serializeNativeScript returns { address, scriptCbor }
    // We need scriptCbor for mintingScript()
    const serialized = serializeNativeScript(nativeScript);
    const forgingScript = serialized.scriptCbor!;

    // Derive policy ID from the native script
    const policyId = resolveNativeScriptHash(nativeScript);

    console.log(`🎨 Batch mint — policy: ${policyId}, items: ${items.length}`);
    console.log(`   Lock slot: ${lockSlot} (~2 hours from now)`);

    // ── Fix 2: Correct CIP-102 royalty token name ─────────────────────────
    // Royalty token = label 500 prefix + same base name as first NFT
    // Per CIP-102: (500) token shares the collection's base asset name
    const firstAssetName = items[0]?.asset_name ?? "Collection";
    const royaltyTokenName = "001f4d70" + stringToHex(firstAssetName);

    // Average royalty rate across all items in the collection
    const avgRoyalties =
      items.reduce((sum, item) => sum + item.royalties, 0) / items.length;
    const royaltyRate = Math.floor((avgRoyalties / 100) * 1_000_000);

    // CIP-102 royalty datum
    // RoyaltyLockDatum { owner, royalty: RoyaltyDatum { recipients, min_ada }, collection_policy }
    const royaltyDatum = mConStr0([
      stringToHex(creatorAddress),  // owner — creator can update terms
      mConStr0([                    // RoyaltyDatum
        [
          mConStr0([                // RoyaltyRecipient
            stringToHex(creatorAddress), // creator receives royalties
            royaltyRate,                 // e.g. 50000 for 5%
          ]),
        ],
        2_000_000,                  // min_ada — minimum 2 ADA per payment
      ]),
      policyId,                     // collection_policy
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    // ── Build transaction ─────────────────────────────────────────────────
    const txBuilder = new MeshTxBuilder({
      fetcher: provider,
      submitter: provider,
    });

    // Mint royalty NFT first — one per collection
    txBuilder
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(forgingScript)
      .txOut(royaltyLockAddress, [
        { unit: policyId + royaltyTokenName, quantity: "1" },
      ])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR");

    // Track all minted tokens for the response
    const tokens: object[] = [];

    // ── Fix 1: Correct CIP-68 datum per NFT ──────────────────────────────
    // metadataToCip68() builds proper Plutus Map datum
    // Marketplaces (jpg.store, CNFT.io) read this to display NFT metadata
    for (const item of items) {
      const assetNameHex = stringToHex(item.asset_name);
      const refTokenName = "000643b0" + assetNameHex;  // (100) Reference NFT
      const userTokenName = "001bc280" + assetNameHex; // (222) User NFT

      // Proper CIP-68 metadata map — ConStr0([Map, version])
      const cip68Datum = metadataToCip68({
        name: item.asset_name,
        image: item.image_ipfs,
        mediaType: "image/png",
        description: "",
        files: [],
      });
      const cip68DatumCbor = serializeData(cip68Datum);

      // Mint (100) Reference NFT → immutable_lock with CIP-68 datum
      txBuilder
        .mint("1", policyId, refTokenName)
        .mintingScript(forgingScript)
        .txOut(immutableLockAddress, [
          { unit: policyId + refTokenName, quantity: "1" },
        ])
        .txOutInlineDatumValue(cip68DatumCbor, "CBOR");

      // Mint (222) User NFT → creator wallet
      txBuilder
        .mint("1", policyId, userTokenName)
        .mintingScript(forgingScript)
        .txOut(creatorAddress, [
          { unit: policyId + userTokenName, quantity: "1" },
        ]);

      tokens.push({
        nft_id:     item.nft_id,
        asset_name: item.asset_name,
        ref_token:  policyId + refTokenName,
        user_token: policyId + userTokenName,
      });
    }

    // Finalize — invalidHereafter required when script has "before" constraint
    const unsignedTx = await txBuilder
      .changeAddress(creatorAddress)
      .selectUtxosFrom(utxos)
      .invalidHereafter(lockSlot)
      .complete();

    // Sign and submit — native script only needs wallet signature
    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    console.log(`✅ Batch minted — tx: ${txHash}`);
    console.log(`   Policy ID:   ${policyId}`);
    console.log(`   Lock slot:   ${lockSlot}`);
    console.log(`   NFTs minted: ${items.length}`);

    res.json({
      tx_hash:   txHash,
      policy_id: policyId,
      lock_slot: lockSlot,
      minted:    items.length,
      tokens,
    });
  } catch (error: any) {
    const message =
      typeof error === "string"
        ? error
        : error?.message || JSON.stringify(error);
    console.error("Batch mint error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;