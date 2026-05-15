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

export default router;