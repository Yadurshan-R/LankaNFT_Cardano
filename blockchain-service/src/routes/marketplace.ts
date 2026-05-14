// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/marketplace.ts
//
// Marketplace Transaction Builder — CIP-102 Compliant
//
// Handles three marketplace actions:
//   List   — locks NFT at marketplace script address with ListingDatum
//   Buy    — pays seller + royalty, sends NFT to buyer
//   Cancel — returns NFT to seller
//
// The marketplace validator reads the CIP-102 royalty datum on-chain
// from a reference input. Off-chain we calculate the royalty amount
// and include the royalty UTxO as a reference input in the buy tx.
// ─────────────────────────────────────────────────────────────────────────────

import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  MeshTxBuilder,
  MeshWallet,
  SpendingBlueprint,
  applyCborEncoding,
  mConStr0,
  mConStr1,
  resolvePaymentKeyHash,
  serializeAddressObj,
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

const marketplaceValidator = getValidator("marketplace.marketplace.spend");

// ─── Marketplace Script Address ───────────────────────────────────────────────
const marketplaceAddress = serializePlutusScript(
  {
    code: applyCborEncoding(marketplaceValidator.compiledCode),
    version: "V3",
  },
  undefined,
  0
).address;

console.log("✅ Marketplace address:", marketplaceAddress);

// ─── Build ListingDatum ───────────────────────────────────────────────────────
// Mirrors the Aiken ListingDatum type exactly:
// ListingDatum { seller, seller_address, price, nft_policy_id,
//                nft_asset_name, royalty_policy_id }
function buildListingDatum(
  sellerAddress: string,
  priceLovelace: number,
  nftPolicyId: string,
  nftAssetName: string,
  royaltyPolicyId: string
): string {
  const sellerPkh = resolvePaymentKeyHash(sellerAddress);

  // Build seller address as Plutus Address type
  // ConStr0([ConStr0([pubKeyHash]), ConStr1([])])  = PubKey address no staking
  const plutusSellerAddress = mConStr0([
    mConStr0([sellerPkh]),
    mConStr1([]),
  ]);

  const datum = mConStr0([
    sellerPkh,            // seller: VerificationKeyHash
    plutusSellerAddress,  // seller_address: Address
    priceLovelace,        // price: Int
    nftPolicyId,          // nft_policy_id: PolicyId
    nftAssetName,         // nft_asset_name: ByteArray
    royaltyPolicyId,      // royalty_policy_id: PolicyId
  ]);

  return serializeData(datum);
}

// ─── Routes ───────────────────────────────────────────────────────────────────

/**
 * POST /api/marketplace/list
 * Lock an NFT at the marketplace script address for sale
 *
 * Body:
 *   mnemonic          string[]  — seller's custodial wallet mnemonic
 *   nft_unit          string    — full asset unit (policyId + assetNameHex)
 *   price_lovelace    number    — asking price in lovelace
 *   royalty_policy_id string    — policy ID of (500) royalty token
 */
router.post("/list", async (req: Request, res: Response) => {
  try {
    const {
      mnemonic,
      nft_unit,
      price_lovelace,
      royalty_policy_id = "",
    } = req.body;

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

    // Parse nft_unit into policyId + assetName
    const policyId = nft_unit.slice(0, 56);
    const assetNameHex = nft_unit.slice(56);

    // Build the listing datum
    const listingDatumCbor = buildListingDatum(
      sellerAddress,
      price_lovelace,
      policyId,
      assetNameHex,
      royalty_policy_id
    );

    // Build transaction — lock NFT at marketplace with datum
    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      // Send NFT to marketplace script address with inline listing datum
      .txOut(marketplaceAddress, [
        { unit: nft_unit, quantity: "1" },
        { unit: "lovelace", quantity: "2000000" }, // min ADA
      ])
      .txOutInlineDatumValue(listingDatumCbor, "CBOR")
      .changeAddress(sellerAddress)
      .selectUtxosFrom(utxos)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    // Return the UTxO reference for the listing
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

/**
 * POST /api/marketplace/buy
 * Purchase a listed NFT — pays seller + royalty
 *
 * Body:
 *   mnemonic          string[]  — buyer's mnemonic
 *   listing_utxo_hash string    — tx hash of the listing
 *   listing_utxo_index number   — output index of the listing
 *   seller_address    string    — seller's address (from listing datum)
 *   price_lovelace    number    — listing price
 *   nft_unit          string    — full asset unit
 *   royalty_amount    number    — calculated royalty in lovelace
 *   royalty_address   string    — royalty recipient address
 *   royalty_utxo_hash string    — tx hash of (500) royalty token UTxO
 *   royalty_utxo_index number   — output index of royalty UTxO
 */
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

    // Parse NFT unit
    const policyId = nft_unit.slice(0, 56);
    const assetNameHex = nft_unit.slice(56);

    // Seller receives price minus royalty
    const sellerAmount = price_lovelace - royalty_amount;

    // Buy redeemer = ConStr1([]) = Buy constructor (index 1, no fields)
    const buyRedeemer = serializeData(mConStr1([]));

    // Build the marketplace script for spending
    const marketplaceScript = applyCborEncoding(marketplaceValidator.compiledCode);

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    // Start building buy transaction
    txBuilder
      // Spend the listing UTxO from marketplace script
      .spendingPlutusScriptV3()
      .txIn(listing_utxo_hash, listing_utxo_index)
      .txInInlineDatumPresent()
      .txInRedeemerValue(buyRedeemer, "CBOR")
      .spendingTxInReference(listing_utxo_hash, listing_utxo_index)

      // NFT goes to buyer
      .txOut(buyerAddress, [{ unit: nft_unit, quantity: "1" }])

      // Seller receives their net amount
      .txOut(seller_address, [{ unit: "lovelace", quantity: sellerAmount.toString() }]);

    // Pay royalty if applicable
    if (royalty_amount > 0 && royalty_address) {
      txBuilder.txOut(royalty_address, [
        { unit: "lovelace", quantity: royalty_amount.toString() },
      ]);
    }

    // Include royalty UTxO as reference input for on-chain CIP-102 verification
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

/**
 * POST /api/marketplace/cancel
 * Cancel a listing — returns NFT to seller
 *
 * Body:
 *   mnemonic          string[]  — seller's mnemonic
 *   listing_utxo_hash string    — tx hash of the listing
 *   listing_utxo_index number   — output index
 *   nft_unit          string    — full asset unit
 */
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

    // Cancel redeemer = ConStr0([]) = Cancel constructor (index 0, no fields)
    const cancelRedeemer = serializeData(mConStr0([]));

    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      // Spend listing UTxO with Cancel redeemer
      .spendingPlutusScriptV3()
      .txIn(listing_utxo_hash, listing_utxo_index)
      .txInInlineDatumPresent()
      .txInRedeemerValue(cancelRedeemer, "CBOR")
      .spendingTxInReference(listing_utxo_hash, listing_utxo_index)

      // NFT returns to seller
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