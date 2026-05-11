import { Router, Request, Response } from "express";
import {
  MeshWallet,
  BlockfrostProvider,
  MeshTxBuilder,
  MintingBlueprint,
  stringToHex,
  mConStr0,
  serializeData,
} from "@meshsdk/core";
import config from "../config";
import fs from "fs";
import path from "path";

const router = Router();

// Load the compiled Aiken blueprint (plutus.json)
const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

// Get the CIP-68 minting validator compiled code
const validator = blueprintJson.validators.find(
  (v: any) => v.title === "minting_policy.cip68_mint.mint"
);

if (!validator) {
  throw new Error("CIP-68 minting policy not found in plutus.json");
}

/**
 * POST /api/mint/single
 * Builds and submits a CIP-68 mint transaction
 * Body: {
 *   mnemonic: string[],
 *   asset_name: string,
 *   metadata_ipfs: string,
 *   image_ipfs: string,
 *   royalties: number,
 * }
 */
router.post("/single", async (req: Request, res: Response) => {
  try {
    const { mnemonic, asset_name, metadata_ipfs, image_ipfs, royalties } =
      req.body;

    if (!mnemonic || !asset_name || !metadata_ipfs) {
      res.status(400).json({
        error: "mnemonic, asset_name and metadata_ipfs are required",
      });
      return;
    }

    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    // Load custodial wallet from mnemonic
    const wallet = new MeshWallet({
      networkId: 0,
      fetcher: provider,
      submitter: provider,
      key: { type: "mnemonic", words: mnemonic },
    });

    // Get wallet UTxOs
    const utxos = await wallet.getUtxos();
    if (utxos.length === 0) {
      res.status(400).json({
        error: "Wallet has no UTxOs — fund the preprod wallet via the faucet",
      });
      return;
    }

    // Use first UTxO as the one-shot parameter (consumed to prove authorization)
    const utxo = utxos[0];
    const txHash = utxo.input.txHash;
    const txIndex = utxo.input.outputIndex;

    // Collateral must be ADA-only (no native tokens) and different from the one-shot UTxO.
    // Cardano rejects collateral that contains native tokens (CollateralContainsNonADA error).
    const collateralUtxo = utxos.find(
      (u) =>
        !(u.input.txHash === txHash && u.input.outputIndex === txIndex) &&
        u.output.amount.every((a) => a.unit === "lovelace")
    );
    if (!collateralUtxo) {
      res.status(400).json({
        error:
          "No pure-ADA UTxO available for collateral. " +
          "Send at least 5 tADA to a fresh address in this wallet and try again.",
      });
      return;
    }

    // OutputReference in our plutus.json:
    // { transaction_id: ByteArray, output_index: Int }
    // Use mConStr0 directly — NOT mTxOutRef (which adds an extra wrapper)
    const outputRef = mConStr0([txHash, txIndex]);
    const blueprint = new MintingBlueprint("V3");
    // Pass the Mesh data object directly — NOT JSON.stringify(outputRef).
    // "Mesh" type calls toPlutusData(param); a string would be encoded as bytes,
    // producing a wrong policy ID.
    blueprint.paramScript(validator.compiledCode, [outputRef], "Mesh");
    const policyId = blueprint.hash;
    const parameterizedScript = blueprint.cbor;

    // CIP-68 token names
    const assetNameHex = stringToHex(asset_name);
    const refTokenName = "000643b0" + assetNameHex;
    const userTokenName = "001bc280" + assetNameHex;

    const address = await wallet.getChangeAddress();

    const txBuilder = new MeshTxBuilder({
      fetcher: provider,
      submitter: provider,
    });

    // Redeemer properly CBOR encoded
    // MintRedeemer::Mint { asset_name: ByteArray } = constructor index 0
    const redeemer = mConStr0([stringToHex(asset_name)]);
    const redeemerCbor = serializeData(redeemer);

    const unsignedTx = await txBuilder
      .txIn(txHash, txIndex)
      .mintPlutusScriptV3()
      .mint("1", policyId, refTokenName)
      .mintingScript(parameterizedScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .mintPlutusScriptV3()
      .mint("1", policyId, userTokenName)
      .mintingScript(parameterizedScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")
      .txOut(address, [
        { unit: policyId + refTokenName, quantity: "1" },
        { unit: policyId + userTokenName, quantity: "1" },
      ])
      .changeAddress(address)
      .selectUtxosFrom(utxos)
      // Collateral must be a different UTxO from the one-shot input
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex
      )
      .complete();

    // Sign and submit
    const signedTx = await wallet.signTx(unsignedTx);
    const submittedTxHash = await wallet.submitTx(signedTx);

    console.log(`✅ NFT minted — tx: ${submittedTxHash}, policy: ${policyId}`);

    res.json({
      tx_hash: submittedTxHash,
      policy_id: policyId,
      asset_name,
      ref_token: policyId + refTokenName,
      user_token: policyId + userTokenName,
    });
  } catch (error: any) {
    // @meshsdk/provider's parseHttpError throws a plain string, not an Error
    const message =
      typeof error === "string"
        ? error
        : error?.message || JSON.stringify(error);
    console.error("Mint error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;