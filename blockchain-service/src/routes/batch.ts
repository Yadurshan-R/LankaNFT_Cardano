import { Router, Request, Response } from "express";
import * as cborLib from 'cbor';
import { bech32 } from 'bech32';
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

const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

function getValidator(title: string) {
  const v = blueprintJson.validators.find((v: any) => v.title === title);
  if (!v) throw new Error(`Validator "${title}" not found in plutus.json`);
  return v;
}

const immutableLockValidator = getValidator("reference_lock.immutable_lock.spend");
const royaltyLockValidator = getValidator("royalty_lock.royalty_lock.spend");

const immutableLockAddress = serializePlutusScript(
  { code: applyCborEncoding(immutableLockValidator.compiledCode), version: "V3" },
  undefined,
  0
).address;

const royaltyLockAddress = serializePlutusScript(
  { code: applyCborEncoding(royaltyLockValidator.compiledCode), version: "V3" },
  undefined,
  0
).address;

interface BatchNFTItem {
  nft_id: string;
  asset_name: string;
  metadata_ipfs: string;
  image_ipfs: string;
  royalties: number;
}

router.post("/batch", async (req: Request, res: Response) => {
  try {
    const { mnemonic, items }: { mnemonic: string[]; items: BatchNFTItem[] } = req.body;

    if (!mnemonic || !Array.isArray(items) || items.length === 0) {
      res.status(400).json({ error: "mnemonic and items array are required" });
      return;
    }

    if (items.length > 100) {
      res.status(400).json({ error: "Maximum 100 NFTs per batch" });
      return;
    }

    const provider = new BlockfrostProvider(config.blockfrost.projectId);
    const wallet = new MeshWallet({
      networkId: 0,
      fetcher: provider,
      submitter: provider,
      key: { type: "mnemonic", words: mnemonic },
    });

    const utxos = await wallet.getUtxos();
    if (utxos.length === 0) {
      res.status(400).json({ error: "Wallet has no UTxOs — fund the preprod wallet via the faucet" });
      return;
    }

    const creatorAddress = await wallet.getChangeAddress();
    const paymentKeyHash = resolvePaymentKeyHash(creatorAddress);

    const LOCK_WINDOW_MS = 2 * 60 * 60 * 1000;
    const lockTime = Date.now() + LOCK_WINDOW_MS;
    const lockSlot = unixTimeToEnclosingSlot(lockTime, SLOT_CONFIG_NETWORK["preprod"]);

    const nativeScript: NativeScript = {
      type: "all",
      scripts: [
        { type: "sig", keyHash: paymentKeyHash },
        { type: "before", slot: lockSlot.toString() },
      ],
    };

    const serialized = serializeNativeScript(nativeScript);
    const forgingScript = serialized.scriptCbor!;
    const policyId = resolveNativeScriptHash(nativeScript);

    const firstAssetName = items[0]?.asset_name ?? "Collection";
    const royaltyTokenName = "001f4d70" + stringToHex(firstAssetName);

    const avgRoyalties = items.reduce((sum, item) => sum + item.royalties, 0) / items.length;
    const royaltyRate = Math.floor((avgRoyalties / 100) * 1_000_000);

    const royaltyDatum = mConStr0([
      stringToHex(creatorAddress),
      mConStr0([
        [mConStr0([stringToHex(creatorAddress), royaltyRate])],
        2_000_000,
      ]),
      policyId,
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    txBuilder
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(forgingScript)
      .txOut(royaltyLockAddress, [{ unit: policyId + royaltyTokenName, quantity: "1" }])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR");

    const tokens: object[] = [];

    for (const item of items) {
      const assetNameHex = stringToHex(item.asset_name);
      const refTokenName = "000643b0" + assetNameHex;
      const userTokenName = "001bc280" + assetNameHex;

      const cip68Datum = metadataToCip68({
        name: item.asset_name,
        image: item.image_ipfs,
        mediaType: "image/png",
        description: "",
        files: [],
      });
      const cip68DatumCbor = serializeData(cip68Datum);

      txBuilder
        .mint("1", policyId, refTokenName)
        .mintingScript(forgingScript)
        .txOut(immutableLockAddress, [{ unit: policyId + refTokenName, quantity: "1" }])
        .txOutInlineDatumValue(cip68DatumCbor, "CBOR");

      txBuilder
        .mint("1", policyId, userTokenName)
        .mintingScript(forgingScript)
        .txOut(creatorAddress, [{ unit: policyId + userTokenName, quantity: "1" }]);

      tokens.push({
        nft_id: item.nft_id,
        asset_name: item.asset_name,
        ref_token: policyId + refTokenName,
        user_token: policyId + userTokenName,
      });
    }

    const unsignedTx = await txBuilder
      .changeAddress(creatorAddress)
      .selectUtxosFrom(utxos)
      .invalidHereafter(lockSlot)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    res.json({
      tx_hash: txHash,
      policy_id: policyId,
      lock_slot: lockSlot,
      minted: items.length,
      tokens,
    });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("Batch mint error:", message);
    res.status(500).json({ error: message });
  }
});

// ── parseCip30Utxos ───────────────────────────────────────────────────────────
// Parses UTxOs from CIP-30 hex CBOR format into MeshSDK-compatible objects.
// Same function used in mint.ts for single mint external wallet flow.
function parseCip30Utxos(cborUtxos: string[]): any[] {
  return cborUtxos.flatMap(hex => {
    try {
      const decoded = cborLib.decodeAllSync(Buffer.from(hex, 'hex'))[0];
      const utxoData = decoded?.value ?? decoded;
      const [txInput, txOutput] = Array.isArray(utxoData) ? utxoData : [utxoData[0], utxoData[1]];
      const txHash = Buffer.from(txInput[0]).toString('hex');
      const outputIndex = Number(txInput[1]);
      const addrBytes = Buffer.from(Array.isArray(txOutput) ? txOutput[0] : txOutput.get(0));
      const headerByte = addrBytes[0];
      const prefix = (headerByte & 0x0f) === 1 ? 'addr' : 'addr_test';
      const address = bech32.encode(prefix, bech32.toWords(addrBytes), 1000);
      const rawAmount = Array.isArray(txOutput) ? txOutput[1] : txOutput.get(1);
      let lovelace = '0';
      const assets: Array<{unit: string, quantity: string}> = [];
      if (typeof rawAmount === 'bigint' || typeof rawAmount === 'number') {
        lovelace = rawAmount.toString();
      } else if (Array.isArray(rawAmount)) {
        lovelace = rawAmount[0].toString();
        if (rawAmount[1] instanceof Map) {
          for (const [policyId, assetMap] of rawAmount[1]) {
            const policyHex = Buffer.from(policyId).toString('hex');
            for (const [name, qty] of assetMap) {
              assets.push({ unit: policyHex + Buffer.from(name).toString('hex'), quantity: qty.toString() });
            }
          }
        }
      }
      return [{ input: { txHash, outputIndex }, output: { address, amount: [{ unit: 'lovelace', quantity: lovelace }, ...assets] } }];
    } catch { return []; }
  });
}

// ── POST /batch-unsigned ──────────────────────────────────────────────────────
// Builds an unsigned batch mint tx for external wallet users.
// Accepts wallet_address + wallet_utxos (CIP-30 hex format).
// Returns unsigned CBOR — frontend signs with Lace then calls /api/submit.
//
// Body: { wallet_address, wallet_utxos, items }
router.post("/batch-unsigned", async (req: Request, res: Response) => {
  try {
    const {
      wallet_address,
      wallet_utxos = [],
      items,
    }: { wallet_address: string; wallet_utxos: string[]; items: BatchNFTItem[] } = req.body;

    if (!wallet_address || !Array.isArray(items) || items.length === 0) {
      res.status(400).json({ error: "wallet_address and items array are required" });
      return;
    }
    if (items.length > 100) {
      res.status(400).json({ error: "Maximum 100 NFTs per batch" });
      return;
    }

    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    // Get UTxOs — from CIP-30 hex if provided, otherwise fetch from Blockfrost
    let utxos: any[];
    if (wallet_utxos && wallet_utxos.length > 0) {
      utxos = parseCip30Utxos(wallet_utxos);
      console.log(`[BATCH-UNSIGNED] Parsed ${utxos.length} UTxOs from CIP-30`);
    } else {
      utxos = await provider.fetchAddressUTxOs(wallet_address);
      console.log(`[BATCH-UNSIGNED] Fetched ${utxos.length} UTxOs from Blockfrost`);
    }

    if (utxos.length === 0) {
      res.status(400).json({ error: "Wallet has no UTxOs — fund via the Cardano faucet" });
      return;
    }

    // Derive payment key hash from wallet address — no mnemonic needed
    const paymentKeyHash = resolvePaymentKeyHash(wallet_address);

    // Build time-locked native script (2 hour window)
    const LOCK_WINDOW_MS = 2 * 60 * 60 * 1000;
    const lockTime = Date.now() + LOCK_WINDOW_MS;
    const lockSlot = unixTimeToEnclosingSlot(lockTime, SLOT_CONFIG_NETWORK["preprod"]);

    const nativeScript: NativeScript = {
      type: "all",
      scripts: [
        { type: "sig", keyHash: paymentKeyHash },
        { type: "before", slot: lockSlot.toString() },
      ],
    };

    const serialized  = serializeNativeScript(nativeScript);
    const forgingScript = serialized.scriptCbor!;
    const policyId    = resolveNativeScriptHash(nativeScript);

    // Build royalty token (one per collection)
    const firstAssetName  = items[0]?.asset_name ?? "Collection";
    const royaltyTokenName = "001f4d70" + stringToHex(firstAssetName);
    const avgRoyalties     = items.reduce((sum, item) => sum + item.royalties, 0) / items.length;
    const royaltyRate      = Math.floor((avgRoyalties / 100) * 1_000_000);

    const royaltyDatum = mConStr0([
      stringToHex(wallet_address),
      mConStr0([
        [mConStr0([stringToHex(wallet_address), royaltyRate])],
        2_000_000,
      ]),
      policyId,
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    // Add royalty token
    txBuilder
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(forgingScript)
      .txOut(royaltyLockAddress, [{ unit: policyId + royaltyTokenName, quantity: "1" }])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR");

    // Add all NFT tokens
    const tokens: object[] = [];
    for (const item of items) {
      const assetNameHex = stringToHex(item.asset_name);
      const refTokenName  = "000643b0" + assetNameHex;
      const userTokenName = "001bc280" + assetNameHex;

      const cip68Datum     = metadataToCip68({
        name: item.asset_name, image: item.image_ipfs,
        mediaType: "image/png", description: "", files: [],
      });
      const cip68DatumCbor = serializeData(cip68Datum);

      txBuilder
        .mint("1", policyId, refTokenName)
        .mintingScript(forgingScript)
        .txOut(immutableLockAddress, [{ unit: policyId + refTokenName, quantity: "1" }])
        .txOutInlineDatumValue(cip68DatumCbor, "CBOR");

      txBuilder
        .mint("1", policyId, userTokenName)
        .mintingScript(forgingScript)
        .txOut(wallet_address, [{ unit: policyId + userTokenName, quantity: "1" }]);

      tokens.push({
        nft_id:     item.nft_id,
        asset_name: item.asset_name,
        ref_token:  policyId + refTokenName,
        user_token: policyId + userTokenName,
      });
    }

    const unsignedTx = await txBuilder
      .changeAddress(wallet_address)
      .selectUtxosFrom(utxos)
      .invalidHereafter(lockSlot)
      .complete();

    res.json({
      unsigned_cbor: unsignedTx,
      policy_id:     policyId,
      lock_slot:     lockSlot,
      minted:        items.length,
      tokens,
    });

  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[BATCH-UNSIGNED] Error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;