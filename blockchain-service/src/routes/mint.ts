import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  MeshTxBuilder,
  MeshWallet,
  MintingBlueprint,
  applyCborEncoding,
  mConStr0,
  mConStr1,
  serializeData,
  serializePlutusScript,
  stringToHex,
} from "@meshsdk/core";
import config from "../config";
import fs from "fs";
import path from "path";
import * as cborLib from 'cbor';
import { bech32 } from 'bech32';

const router = Router();

const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

function getValidator(title: string) {
  const v = blueprintJson.validators.find((v: any) => v.title === title);
  if (!v) throw new Error(`Validator "${title}" not found in plutus.json`);
  return v;
}

const mintSingleValidator = getValidator("minting_policy.cip68_mint_single.mint");
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

// ─── UTxO Parser for CIP-30 Wallets (HD Wallet Fix) ──────────────────────────
function parseCip30Utxos(cborUtxos: string[]): any[] {
  return cborUtxos.map(hex => {
    const decoded = cborLib.decodeAllSync(Buffer.from(hex, 'hex'))[0];
    const utxoData = decoded?.value ?? decoded;
    const [txInput, txOutput] = Array.isArray(utxoData) ? utxoData : [utxoData[0], utxoData[1]];

    const txHash = Buffer.from(txInput[0]).toString('hex');
    const outputIndex = Number(txInput[1]);

    // Parse address bytes to bech32
    const addrBytes = Buffer.from(Array.isArray(txOutput) ? txOutput[0] : txOutput.get(0));
    const headerByte = addrBytes[0];
    const prefix = (headerByte & 0x0f) === 1 ? 'addr' : 'addr_test';
    const address = bech32.encode(prefix, bech32.toWords(addrBytes), 1000);

    // Parse ADA amount
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
            assets.push({
              unit: policyHex + Buffer.from(name).toString('hex'),
              quantity: qty.toString()
            });
          }
        }
      }
    }

    return {
      input: { txHash, outputIndex },
      output: { address, amount: [{ unit: 'lovelace', quantity: lovelace }, ...assets] }
    };
  });
}

router.post("/single", async (req: Request, res: Response) => {
  try {
    const { mnemonic, asset_name, metadata_ipfs, image_ipfs, royalties = 0 } = req.body;

    if (!mnemonic || !asset_name || !metadata_ipfs) {
      res.status(400).json({ error: "mnemonic, asset_name and metadata_ipfs are required" });
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

    const pureAdaUtxos = utxos.filter((u) =>
      u.output.amount.every((a) => a.unit === "lovelace")
    );

    if (pureAdaUtxos.length === 0) {
      res.status(400).json({ error: "No pure-ADA UTxO available. Send at least 5 tADA to the wallet." });
      return;
    }

    const sortedPureAda = pureAdaUtxos.sort((a, b) => {
      const aAda = parseInt(a.output.amount.find((x) => x.unit === "lovelace")?.quantity || "0");
      const bAda = parseInt(b.output.amount.find((x) => x.unit === "lovelace")?.quantity || "0");
      return aAda - bAda;
    });

    const oneShotUtxo = sortedPureAda[0]!;
    const txHash = oneShotUtxo.input.txHash;
    const txIndex = oneShotUtxo.input.outputIndex;

    if (pureAdaUtxos.length < 2) {
      res.status(400).json({
        error: "Need at least 2 pure-ADA UTxOs (one for minting, one for collateral). Send at least 5 tADA to the wallet address.",
      });
      return;
    }

    const collateralUtxo = sortedPureAda[sortedPureAda.length - 1]!;

    const outputRef = mConStr0([txHash, txIndex]);
    const mintBlueprint = new MintingBlueprint("V3");
    mintBlueprint.paramScript(mintSingleValidator.compiledCode, [outputRef], "Mesh");

    const policyId = mintBlueprint.hash;
    const mintScript = mintBlueprint.cbor;

    const assetNameHex = stringToHex(asset_name);
    const refTokenName = "000643b0" + assetNameHex;
    const userTokenName = "001bc280" + assetNameHex;
    const royaltyTokenName = "001f4d70" + assetNameHex;

    const creatorAddress = await wallet.getChangeAddress();

    const cip68Datum = mConStr0([
      [
        [stringToHex("name"),        stringToHex(asset_name)],
        [stringToHex("image"),       stringToHex(image_ipfs)],
        [stringToHex("mediaType"),   stringToHex("image/png")],
        [stringToHex("description"), stringToHex("")],
        [stringToHex("files"),       []],
      ],
      1,
    ]);
    const cip68DatumCbor = serializeData(cip68Datum);

    const royaltyRate = Math.floor((royalties / 100) * 1_000_000);
    const royaltyDatum = mConStr0([
      stringToHex(creatorAddress),
      mConStr0([
        [mConStr0([stringToHex(creatorAddress), royaltyRate])],
        2_000_000,
      ]),
      policyId,
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    const redeemer = mConStr0([assetNameHex, mConStr1([])]);
    const redeemerCbor = serializeData(redeemer);

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      .txIn(txHash, txIndex)
      .mintPlutusScriptV3()
      .mint("1", policyId, refTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .mintPlutusScriptV3()
      .mint("1", policyId, userTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .mintPlutusScriptV3()
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .txOut(creatorAddress, [{ unit: policyId + userTokenName, quantity: "1" }])
      .txOut(immutableLockAddress, [{ unit: policyId + refTokenName, quantity: "1" }])
      .txOutInlineDatumValue(cip68DatumCbor, "CBOR")
      .txOut(royaltyLockAddress, [{ unit: policyId + royaltyTokenName, quantity: "1" }])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR")
      // CIP-25 metadata — makes NFT images visible in all wallets (Lace, Nami, Eternl)
      .metadataValue("721", {
        [policyId]: {
          [asset_name]: {
            name:        asset_name,
            image:       image_ipfs,
            mediaType:   "image/png",
            description: "",
          }
        }
      })
      .changeAddress(creatorAddress)
      .selectUtxosFrom(utxos)
      .txInCollateral(collateralUtxo.input.txHash, collateralUtxo.input.outputIndex)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const submittedTxHash = await wallet.submitTx(signedTx);

    res.json({
      tx_hash: submittedTxHash,
      policy_id: policyId,
      asset_name,
      ref_token: policyId + refTokenName,
      user_token: policyId + userTokenName,
      royalty_token: policyId + royaltyTokenName,
    });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("Mint error:", message);
    res.status(500).json({ error: message });
  }
});

router.post("/single-unsigned", async (req: Request, res: Response) => {
  try {
    const { wallet_address, wallet_utxos, asset_name, metadata_ipfs, image_ipfs, royalties = 0 } = req.body;

    if (!wallet_address || !asset_name || !metadata_ipfs) {
      res.status(400).json({ error: "wallet_address, asset_name and metadata_ipfs are required" });
      return;
    }

    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    let utxos: any[];
    if (wallet_utxos && wallet_utxos.length > 0) {
      utxos = parseCip30Utxos(wallet_utxos);
    } else {
      utxos = await provider.fetchAddressUTxOs(wallet_address);
    }
    
    if (utxos.length === 0) {
      res.status(400).json({ error: "Wallet has no UTxOs — fund the wallet first" });
      return;
    }

    // Pure-ADA UTxOs for one-shot minting policy and collateral
    const pureAdaUtxos = utxos.filter((u) =>
      u.output.amount.every((a: any) => a.unit === "lovelace")
    );
    if (pureAdaUtxos.length < 2) {
      res.status(400).json({
        error: "Need at least 2 pure-ADA UTxOs. Send at least 10 tADA to your wallet.",
      });
      return;
    }

    const sortedPureAda = pureAdaUtxos.sort((a, b) => {
      const aAda = parseInt(a.output.amount.find((x: any) => x.unit === "lovelace")?.quantity || "0");
      const bAda = parseInt(b.output.amount.find((x: any) => x.unit === "lovelace")?.quantity || "0");
      return aAda - bAda;
    });

    const oneShotUtxo = sortedPureAda[0]!;
    const txHash = oneShotUtxo.input.txHash;
    const txIndex = oneShotUtxo.input.outputIndex;
    const collateralUtxo = sortedPureAda[sortedPureAda.length - 1]!;

    // Build minting policy (same as signed route — policy depends on one-shot UTxO)
    const outputRef = mConStr0([txHash, txIndex]);
    const mintBlueprint = new MintingBlueprint("V3");
    mintBlueprint.paramScript(mintSingleValidator.compiledCode, [outputRef], "Mesh");
    const policyId = mintBlueprint.hash;
    const mintScript = mintBlueprint.cbor;

    const assetNameHex = stringToHex(asset_name);
    const refTokenName = "000643b0" + assetNameHex;
    const userTokenName = "001bc280" + assetNameHex;
    const royaltyTokenName = "001f4d70" + assetNameHex;

    const royaltyRate = Math.floor((royalties / 100) * 1_000_000);
    const cip68Datum = mConStr0([
      [
        [stringToHex("name"),        stringToHex(asset_name)],
        [stringToHex("image"),       stringToHex(image_ipfs)],
        [stringToHex("mediaType"),   stringToHex("image/png")],
        [stringToHex("description"), stringToHex("")],
        [stringToHex("files"),       []],
      ],
      1,
    ]);
    const cip68DatumCbor = serializeData(cip68Datum);

    const royaltyDatum = mConStr0([
      stringToHex(wallet_address),
      mConStr0([
        [mConStr0([stringToHex(wallet_address), royaltyRate])],
        2_000_000,
      ]),
      policyId,
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    const redeemer = mConStr0([assetNameHex, mConStr1([])]);
    const redeemerCbor = serializeData(redeemer);

    // Build tx — identical to signed route
    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });
    const unsignedTx = await txBuilder
      .txIn(txHash, txIndex)
      .mintPlutusScriptV3()
      .mint("1", policyId, refTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .mintPlutusScriptV3()
      .mint("1", policyId, userTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .mintPlutusScriptV3()
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .txOut(wallet_address, [{ unit: policyId + userTokenName, quantity: "1" }])
      .txOut(immutableLockAddress, [{ unit: policyId + refTokenName, quantity: "1" }])
      .txOutInlineDatumValue(cip68DatumCbor, "CBOR")
      .txOut(royaltyLockAddress, [{ unit: policyId + royaltyTokenName, quantity: "1" }])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR")
      // CIP-25 metadata — makes NFT images visible in all wallets (Lace, Nami, Eternl)
      .metadataValue("721", {
        [policyId]: {
          [asset_name]: {
            name:        asset_name,
            image:       image_ipfs,
            mediaType:   "image/png",
            description: "",
          }
        }
      })
      .changeAddress(wallet_address)
      .selectUtxosFrom(utxos)
      .txInCollateral(collateralUtxo.input.txHash, collateralUtxo.input.outputIndex)
      .complete();

    // Return unsigned CBOR + policy info needed for confirm-mint call
    res.json({
      unsigned_cbor:    unsignedTx,
      policy_id:        policyId,
      asset_name,
      ref_token_name:   refTokenName,
      user_token_name:  userTokenName,
    });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[MINT-UNSIGNED] Error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;