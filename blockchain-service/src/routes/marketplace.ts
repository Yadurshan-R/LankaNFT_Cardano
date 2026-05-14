// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/marketplace.ts
// ─────────────────────────────────────────────────────────────────────────────

import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  MeshTxBuilder,
  MeshWallet,
  applyCborEncoding,
  deserializeAddress,
  mConStr0,
  mConStr1,
  resolvePaymentKeyHash,
  serializeData,
  serializePlutusScript,
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

const marketplaceValidator = getValidator("marketplace.marketplace.spend");

const marketplaceAddress = serializePlutusScript(
  { code: applyCborEncoding(marketplaceValidator.compiledCode), version: "V3" },
  undefined,
  0
).address;

console.log("✅ Marketplace address:", marketplaceAddress);

function buildListingDatum(
  sellerAddress: string,
  priceLovelace: number,
  nftPolicyId: string,
  nftAssetName: string,
  royaltyPolicyId: string
): string {
  const sellerPkh = resolvePaymentKeyHash(sellerAddress);
  
  // Deserialize address to get staking credential
  const addrObj = deserializeAddress(sellerAddress);
  
  // Build correct Plutus Address type including staking credential
  // Cardano Address = Constr 0 [paymentCred, Option<StakingCred>]
  // With staking:    Constr 0 [Constr 0 [pkh], Constr 0 [Constr 0 [Constr 0 [stakePkh]]]]
  // Without staking: Constr 0 [Constr 0 [pkh], Constr 1 []]
  let plutusSellerAddress;
  
  if (addrObj.stakeCredentialHash) {
    plutusSellerAddress = mConStr0([
      mConStr0([sellerPkh]),
      mConStr0([
        mConStr0([
          mConStr0([addrObj.stakeCredentialHash])
        ])
      ])
    ]);
  } else {
    plutusSellerAddress = mConStr0([
      mConStr0([sellerPkh]),
      mConStr1([])
    ]);
  }
  
  const datum = mConStr0([
    sellerPkh,
    plutusSellerAddress,
    priceLovelace,
    nftPolicyId,
    nftAssetName,
    royaltyPolicyId,
  ]);
  
  return serializeData(datum);
}

// ─── List ─────────────────────────────────────────────────────────────────────
router.post("/list", async (req: Request, res: Response) => {
  try {
    const { mnemonic, nft_unit, price_lovelace, royalty_policy_id = "" } = req.body;

    if (!mnemonic || !nft_unit || !price_lovelace) {
      res.status(400).json({ error: "mnemonic, nft_unit, price_lovelace required" });
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
      res.status(400).json({ error: "Wallet has no UTxOs" });
      return;
    }

    const sellerAddress = await wallet.getChangeAddress();
    const policyId = nft_unit.slice(0, 56);
    const assetNameHex = nft_unit.slice(56);

    const listingDatumCbor = buildListingDatum(
      sellerAddress,
      price_lovelace,
      policyId,
      assetNameHex,
      royalty_policy_id
    );

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      .txOut(marketplaceAddress, [
        { unit: nft_unit, quantity: "1" },
        { unit: "lovelace", quantity: "2000000" },
      ])
      .txOutInlineDatumValue(listingDatumCbor, "CBOR")
      .changeAddress(sellerAddress)
      .selectUtxosFrom(utxos)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    console.log(`✅ NFT listed — tx: ${txHash}, price: ${price_lovelace} lovelace`);

    res.json({
      tx_hash: txHash,
      script_utxo: `${txHash}#0`,
      marketplace_address: marketplaceAddress,
      price_lovelace,
    });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("List error:", message);
    res.status(500).json({ error: message });
  }
});

// ─── Buy ──────────────────────────────────────────────────────────────────────
router.post("/buy", async (req: Request, res: Response) => {
  try {
    const {
      mnemonic,
      listing_utxo_hash,
      listing_utxo_index,
      seller_address,
      price_lovelace,
      nft_unit,
      royalty_amount = 0,
      royalty_address = "",
      royalty_utxo_hash = "",
      royalty_utxo_index = 0,
    } = req.body;

    if (!mnemonic || !listing_utxo_hash || !seller_address || !price_lovelace || !nft_unit) {
      res.status(400).json({ error: "Missing required fields" });
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
    const buyerAddress = await wallet.getChangeAddress();

    // Find collateral UTxO — pure ADA, at least 5 ADA
    // Required by Cardano protocol when spending Plutus V3 scripts
    const collateralUtxo = utxos.find(
      (u) =>
        u.output.amount.length === 1 &&
        u.output.amount[0].unit === "lovelace" &&
        parseInt(u.output.amount[0].quantity) >= 5_000_000
    );

    if (!collateralUtxo) {
      res.status(400).json({
        error:
          "Buyer wallet needs a pure ADA UTxO of at least 5 ADA for collateral. " +
          "Fund your wallet at https://docs.cardano.org/cardano-testnet/tools/faucet",
      });
      return;
    }

    // Fetch listing UTxO from marketplace script address
    const scriptUtxos = await provider.fetchAddressUTxOs(marketplaceAddress);
    const listingUtxo = scriptUtxos.find(
      (u) =>
        u.input.txHash === listing_utxo_hash &&
        u.input.outputIndex === parseInt(listing_utxo_index.toString())
    );

    if (!listingUtxo) {
      res.status(404).json({
        error: `Listing UTxO not found. TX: ${listing_utxo_hash}#${listing_utxo_index}`,
      });
      return;
    }

    const sellerAmount = price_lovelace - royalty_amount;
    const marketplaceScript = applyCborEncoding(marketplaceValidator.compiledCode);

    console.log("Buy — listing UTxO found:", {
      hash: listingUtxo.input.txHash,
      index: listingUtxo.input.outputIndex,
      nft_unit,
      sellerAmount,
      collateral: collateralUtxo.input.txHash,
    });

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    txBuilder
      .spendingPlutusScriptV3()
      .txIn(
        listingUtxo.input.txHash,
        listingUtxo.input.outputIndex,
        listingUtxo.output.amount,
        listingUtxo.output.address
      )
      .txInInlineDatumPresent()
      .txInRedeemerValue("d87980", "CBOR")
      .txInScript(marketplaceScript)
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex,
        collateralUtxo.output.amount,
        collateralUtxo.output.address
      )
      .requiredSignerHash(resolvePaymentKeyHash(buyerAddress))
      .txOut(buyerAddress, [{ unit: nft_unit, quantity: "1" }])
      .txOut(seller_address, [{ unit: "lovelace", quantity: sellerAmount.toString() }]);

    if (royalty_amount > 0 && royalty_address) {
      txBuilder.txOut(royalty_address, [
        { unit: "lovelace", quantity: royalty_amount.toString() },
      ]);
    }

    if (royalty_utxo_hash) {
      txBuilder.readOnlyTxInReference(royalty_utxo_hash, royalty_utxo_index);
    }

    const unsignedTx = await txBuilder
      .changeAddress(buyerAddress)
      .selectUtxosFrom(utxos)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    console.log(`✅ NFT purchased — tx: ${txHash}`);
    res.json({ tx_hash: txHash });

  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("Buy error:", message);
    res.status(500).json({ error: message });
  }
});

// ─── Cancel ───────────────────────────────────────────────────────────────────
router.post("/cancel", async (req: Request, res: Response) => {
  try {
    const { mnemonic, listing_utxo_hash, listing_utxo_index, nft_unit } = req.body;

    if (!mnemonic || !listing_utxo_hash || !nft_unit) {
      res.status(400).json({ error: "Missing required fields" });
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
    const sellerAddress = await wallet.getChangeAddress();

    // Find collateral UTxO — pure ADA, at least 5 ADA
    const collateralUtxo = utxos.find(
      (u) =>
        u.output.amount.length === 1 &&
        u.output.amount[0].unit === "lovelace" &&
        parseInt(u.output.amount[0].quantity) >= 5_000_000
    );

    if (!collateralUtxo) {
      res.status(400).json({
        error: "Wallet needs a pure ADA UTxO of at least 5 ADA for collateral",
      });
      return;
    }

    const scriptUtxos = await provider.fetchAddressUTxOs(marketplaceAddress);
    const listingUtxo = scriptUtxos.find(
      (u) =>
        u.input.txHash === listing_utxo_hash &&
        u.input.outputIndex === parseInt(listing_utxo_index.toString())
    );

    if (!listingUtxo) {
      res.status(404).json({ error: "Listing UTxO not found — already sold or cancelled" });
      return;
    }

    const marketplaceScript = applyCborEncoding(marketplaceValidator.compiledCode);
    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      .spendingPlutusScriptV3()
      .txIn(
        listingUtxo.input.txHash,
        listingUtxo.input.outputIndex,
        listingUtxo.output.amount,
        listingUtxo.output.address
      )
      .txInInlineDatumPresent()
      .txInRedeemerValue("d87a80", "CBOR")
      .txInScript(marketplaceScript)
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex,
        collateralUtxo.output.amount,
        collateralUtxo.output.address
      )
      .txOut(sellerAddress, [{ unit: nft_unit, quantity: "1" }])
      .changeAddress(sellerAddress)
      .selectUtxosFrom(utxos)
      .requiredSignerHash(resolvePaymentKeyHash(sellerAddress))
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    console.log(`✅ Listing cancelled — tx: ${txHash}`);
    res.json({ tx_hash: txHash });

  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("Cancel error:", message);
    res.status(500).json({ error: message });
  }
});

export { marketplaceAddress };
export default router;