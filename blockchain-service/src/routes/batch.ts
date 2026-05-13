// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/batch.ts
//
// CIP-68 Compliant Batch NFT Minting Route — Native Script Collection Policy
//
// This route mints ALL NFTs in a batch in ONE single Cardano transaction.
// This is how NMKR and professional platforms handle batch minting.
//
// How it works:
//   - Uses a Native Script policy (ForgeScript.withOneSignature)
//   - The policy is locked to the creator's wallet address
//   - All NFTs in the batch share ONE policy ID
//   - All (100) ref NFTs + (222) user NFTs minted in one transaction
//   - One (500) royalty NFT minted for the whole collection
//   - No UTxO conflicts — no one-shot UTxO needed
//   - No collateral needed — native scripts don't require Plutus collateral
//   - Fast — no waiting between mints
//
// Token structure per NFT (CIP-68):
//   (100) Reference NFT → locked at immutable_lock script with CIP-68 datum
//   (222) User NFT      → sent to creator wallet
//
// Collection tokens (once per batch):
//   (500) Royalty NFT   → locked at royalty_lock script with CIP-102 datum
//
// References:
//   CIP-68: https://cips.cardano.org/cip/CIP-68
//   CIP-102: https://cips.cardano.org/cip/CIP-102
// ─────────────────────────────────────────────────────────────────────────────

import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  ForgeScript,
  MeshTxBuilder,
  MeshWallet,
  applyCborEncoding,
  mConStr0,
  resolveScriptHash,
  serializeData,
  serializePlutusScript,
  stringToHex,
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
const immutableLockAddress = serializePlutusScript(
  {
    code: applyCborEncoding(immutableLockValidator.compiledCode),
    version: "V3",
  },
  undefined,
  0
).address;

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
 * Mints ALL NFTs in a batch in ONE Cardano transaction.
 * Uses a native script collection policy tied to the creator's wallet.
 *
 * Request body:
 *   mnemonic  string[]       — 24 word mnemonic of the custodial wallet
 *   items     BatchNFTItem[] — list of NFTs to mint
 *
 * Response:
 *   tx_hash    string   — single Cardano transaction hash for all NFTs
 *   policy_id  string   — shared policy ID for the whole collection
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

    // ── Native Script Collection Policy ───────────────────────────────────
    // ForgeScript.withOneSignature creates a native script policy
    // that requires the creator's wallet signature to mint
    // This is how NMKR handles collections — simple, reliable, no Plutus overhead
    const forgingScript = ForgeScript.withOneSignature(creatorAddress);
    const policyId = resolveScriptHash(forgingScript);

    console.log(`🎨 Batch mint — policy: ${policyId}, items: ${items.length}`);

    // ── Build CIP-102 royalty datum ───────────────────────────────────────
    // Use royalty from the first item for the whole collection
    const collectionRoyalties = items[0]?.royalties ?? 0;
    const royaltyRate = Math.floor((collectionRoyalties / 100) * 1_000_000);

    const royaltyTokenName = "001f4d7043006f006c006c656374696f6e"; // (500)Collection

    const royaltyDatum = mConStr0([
      stringToHex(creatorAddress),
      mConStr0([
        [
          mConStr0([
            stringToHex(creatorAddress),
            royaltyRate,
          ]),
        ],
        2_000_000,
      ]),
      policyId,
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    // ── Build transaction ─────────────────────────────────────────────────
    const txBuilder = new MeshTxBuilder({
      fetcher: provider,
      submitter: provider,
    });

    // Start building — add royalty NFT first
    txBuilder
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(forgingScript)
      .txOut(royaltyLockAddress, [
        { unit: policyId + royaltyTokenName, quantity: "1" },
      ])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR");

    // Track all minted tokens for the response
    const tokens: object[] = [];

    // Add all NFTs to the same transaction
    for (const item of items) {
      const assetNameHex = stringToHex(item.asset_name);
      const refTokenName = "000643b0" + assetNameHex;   // (100) Reference NFT
      const userTokenName = "001bc280" + assetNameHex;  // (222) User NFT

      // CIP-68 metadata datum for this NFT's reference token
      const cip68Datum = mConStr0([
        [
          [stringToHex("name"),        stringToHex(item.asset_name)],
          [stringToHex("image"),       stringToHex(item.image_ipfs)],
          [stringToHex("mediaType"),   stringToHex("image/png")],
          [stringToHex("description"), stringToHex("")],
          [stringToHex("files"),       []],
        ],
        1, // CIP-68 version
      ]);
      const cip68DatumCbor = serializeData(cip68Datum);

      // Mint (100) Reference NFT — goes to immutable_lock with CIP-68 datum
      txBuilder
        .mint("1", policyId, refTokenName)
        .mintingScript(forgingScript)
        .txOut(immutableLockAddress, [
          { unit: policyId + refTokenName, quantity: "1" },
        ])
        .txOutInlineDatumValue(cip68DatumCbor, "CBOR");

      // Mint (222) User NFT — goes to creator wallet
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

    // Finalize transaction
    const unsignedTx = await txBuilder
      .changeAddress(creatorAddress)
      .selectUtxosFrom(utxos)
      .complete();

    // Sign and submit — native script only needs wallet signature
    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    console.log(`✅ Batch minted — tx: ${txHash}`);
    console.log(`   Policy ID: ${policyId}`);
    console.log(`   NFTs minted: ${items.length}`);

    res.json({
      tx_hash:   txHash,
      policy_id: policyId,
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