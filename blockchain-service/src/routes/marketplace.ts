// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/marketplace.ts
//
// Marketplace routes: list, buy, cancel
//
// Key Cardano reality this file handles:
//   Blockfrost indexing delay — after a tx is submitted, the new UTxOs
//   take ~20-40 seconds to appear in Blockfrost queries (one block time).
//   Cancel and Buy both need the listing UTxO to be visible at the script
//   address. If we query too soon, Blockfrost returns nothing and the tx
//   fails. We handle this with retry logic (up to 5 attempts, 6s apart).
//
// Stale listing protection is handled in the Go backend before this sidecar
// is called — see internal/listing/handler.go CreateListing.
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

// ─── Constants ────────────────────────────────────────────────────────────────

// Cardano Preprod block time is ~20 seconds. Blockfrost indexes new UTxOs
// within 1-2 blocks. We retry up to 5 times with 6 second gaps = 30 seconds
// max wait, covering one full block confirmation window.
const UTXO_FETCH_MAX_RETRIES = 5;
const UTXO_FETCH_RETRY_DELAY_MS = 6_000;

// Min-ADA locked alongside the NFT at the marketplace script.
// Must be >= Cardano's min-ADA requirement for a multi-asset UTxO.
// Set to 3 ADA to be safely above the protocol minimum.
const LISTING_MIN_ADA_LOVELACE = "3000000";

// Minimum pure-ADA UTxO required in wallet for Plutus collateral.
// Cardano requires collateral for script-spending transactions.
const COLLATERAL_MIN_LOVELACE = 5_000_000;

// ─── Blueprint loading ────────────────────────────────────────────────────────

const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

function getValidator(title: string) {
  const v = blueprintJson.validators.find((v: any) => v.title === title);
  if (!v) throw new Error(`Validator "${title}" not found in plutus.json`);
  return v;
}

const marketplaceValidator = getValidator("marketplace.marketplace.spend");

// Derive the script address once at startup — it never changes
const marketplaceAddress = serializePlutusScript(
  { code: applyCborEncoding(marketplaceValidator.compiledCode), version: "V3" },
  undefined,
  0
).address;

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Constructs the CBOR-encoded datum required by the marketplace smart contract.
 *
 * The datum encodes the seller's payment/stake credentials and the NFT details
 * so the Aiken validator can verify:
 * - The correct seller receives the sale price
 * - The correct NFT is being sold
 * - The correct royalty policy is referenced
 *
 * Aiken datum structure (marketplace.ak):
 * { seller_pkh, seller_address, price, nft_policy, nft_name, royalty_policy }
 */
function buildListingDatum(
  sellerAddress: string,
  priceLovelace: number,
  nftPolicyId: string,
  nftAssetName: string,
  royaltyPolicyId: string
): string {
  const sellerPkh = resolvePaymentKeyHash(sellerAddress);
  const addrObj = deserializeAddress(sellerAddress);

  // Plutus addresses encode payment + optional stake credential separately
  let plutusSellerAddress;
  if (addrObj.stakeCredentialHash) {
    // Base address — has both payment and stake credentials
    plutusSellerAddress = mConStr0([
      mConStr0([sellerPkh]),
      mConStr0([mConStr0([mConStr0([addrObj.stakeCredentialHash])])]),
    ]);
  } else {
    // Enterprise address — payment credential only, no staking
    plutusSellerAddress = mConStr0([mConStr0([sellerPkh]), mConStr1([])]);
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

/**
 * Fetches the listing UTxO from the marketplace script address, with retries.
 *
 * Why retries? After a listing tx is submitted, Blockfrost takes one block
 * (~20s on Preprod) to index the new UTxO. If cancel/buy is attempted too
 * soon, fetchAddressUTxOs returns an empty set even though the UTxO exists.
 *
 * Retrying up to 5 times with 6s delays covers a full block window.
 * This makes cancel/buy reliable without requiring the user to wait manually.
 *
 * @returns The matching UTxO, or null if not found after all retries
 */
async function fetchListingUtxoWithRetry(
  provider: BlockfrostProvider,
  txHash: string,
  outputIndex: number
): Promise<any | null> {
  for (let attempt = 1; attempt <= UTXO_FETCH_MAX_RETRIES; attempt++) {
    const scriptUtxos = await provider.fetchAddressUTxOs(marketplaceAddress);
    const match = scriptUtxos.find(
      (u) =>
        u.input.txHash === txHash &&
        u.input.outputIndex === outputIndex
    );

    if (match) {
      if (attempt > 1) {
        console.log(`[MARKETPLACE] UTxO found after ${attempt} attempts (Blockfrost indexing delay)`);
      }
      return match;
    }

    if (attempt < UTXO_FETCH_MAX_RETRIES) {
      console.log(
        `[MARKETPLACE] UTxO ${txHash}#${outputIndex} not indexed yet — ` +
        `retrying in ${UTXO_FETCH_RETRY_DELAY_MS / 1000}s (attempt ${attempt}/${UTXO_FETCH_MAX_RETRIES})`
      );
      await new Promise((r) => setTimeout(r, UTXO_FETCH_RETRY_DELAY_MS));
    }
  }

  return null;
}

/**
 * Finds a suitable collateral UTxO from the wallet.
 * Cardano requires a pure-ADA UTxO (no tokens) of at least 5 ADA
 * as collateral when executing Plutus scripts.
 */
function findCollateralUtxo(utxos: any[]): any | null {
  return utxos.find(
    (u) =>
      u.output.amount.length === 1 &&
      u.output.amount[0].unit === "lovelace" &&
      parseInt(u.output.amount[0].quantity) >= COLLATERAL_MIN_LOVELACE
  ) ?? null;
}

// ─── POST /list ───────────────────────────────────────────────────────────────

/**
 * Lists an NFT for sale on the marketplace.
 *
 * Sends the NFT to the marketplace script address with an inline datum
 * encoding the sale terms. The Aiken validator will enforce these terms
 * when a buyer attempts to spend this UTxO.
 *
 * Body: { mnemonic, nft_unit, price_lovelace, royalty_policy_id }
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
      res.status(400).json({ error: "Wallet has no UTxOs — fund via the Cardano faucet" });
      return;
    }

    const sellerAddress = await wallet.getChangeAddress();

    // Extract policy ID (first 56 hex chars) and asset name (remainder)
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

    // Send NFT + min-ADA to marketplace script with inline datum.
    // The datum is stored on-chain so buyers can reconstruct sale terms
    // without any off-chain data (fully trustless).
    const unsignedTx = await txBuilder
      .txOut(marketplaceAddress, [
        { unit: nft_unit, quantity: "1" },
        { unit: "lovelace", quantity: LISTING_MIN_ADA_LOVELACE },
      ])
      .txOutInlineDatumValue(listingDatumCbor, "CBOR")
      .changeAddress(sellerAddress)
      .selectUtxosFrom(utxos)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    res.json({
      tx_hash: txHash,
      script_utxo: `${txHash}#0`,
      marketplace_address: marketplaceAddress,
      price_lovelace,
    });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[MARKETPLACE] List error:", message);
    res.status(500).json({ error: message });
  }
});

// ─── POST /buy ────────────────────────────────────────────────────────────────

/**
 * Purchases a listed NFT by spending the marketplace script UTxO.
 *
 * Uses the Buy redeemer (d87980) which triggers the Aiken validator to check:
 * - Buyer's signature is present
 * - Seller receives the correct ADA amount
 * - Royalty recipient receives the royalty amount (if any)
 *
 * Retries UTxO lookup to handle Blockfrost indexing delay.
 *
 * Body: { mnemonic, listing_utxo_hash, listing_utxo_index, seller_address,
 * price_lovelace, nft_unit, royalty_amount, royalty_address,
 * royalty_utxo_hash, royalty_utxo_index }
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

    // Collateral is required for all Plutus script executions.
    // We exclude this UTxO from fee selection to keep it permanently reserved.
    const collateralUtxo = findCollateralUtxo(utxos);
    if (!collateralUtxo) {
      res.status(400).json({
        error:
          "Your wallet needs a pure ADA UTxO of at least 5 ADA for collateral. " +
          "Fund your wallet at https://docs.cardano.org/cardano-testnet/tools/faucet",
      });
      return;
    }

    // Fetch listing UTxO with retry — handles Blockfrost indexing delay
    const listingUtxo = await fetchListingUtxoWithRetry(
      provider,
      listing_utxo_hash,
      parseInt(listing_utxo_index.toString())
    );

    if (!listingUtxo) {
      res.status(404).json({
        error:
          "Listing not found on-chain. It may have already been sold or cancelled. " +
          "Please refresh the page.",
      });
      return;
    }

    // Seller receives full price minus royalty amount
    const sellerAmount = price_lovelace - royalty_amount;
    const marketplaceScript = applyCborEncoding(marketplaceValidator.compiledCode);

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
      .txInRedeemerValue("d87980", "CBOR") // Buy redeemer (Constructor 0, empty fields)
      .txInScript(marketplaceScript)
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex,
        collateralUtxo.output.amount,
        collateralUtxo.output.address
      )
      .requiredSignerHash(resolvePaymentKeyHash(buyerAddress))
      .txOut(buyerAddress, [{ unit: nft_unit, quantity: "1" }])
      .txOut(seller_address, [{ unit: "lovelace", quantity: sellerAmount.toString() }])
      // Explicitly recreate a 5 ADA collateral UTxO for the buyer's next purchase
      .txOut(buyerAddress, [{ unit: "lovelace", quantity: "5000000" }]);

    if (royalty_amount > 0 && royalty_address) {
      txBuilder.txOut(royalty_address, [
        { unit: "lovelace", quantity: royalty_amount.toString() },
      ]);
    }

    if (royalty_utxo_hash) {
      txBuilder.readOnlyTxInReference(royalty_utxo_hash, royalty_utxo_index);
    }

    // Exclude collateral from fee selection so it is never accidentally consumed
    const unsignedTx = await txBuilder
      .changeAddress(buyerAddress)
      .selectUtxosFrom(
        utxos.filter(
          (u) =>
            u.input.txHash !== collateralUtxo.input.txHash ||
            u.input.outputIndex !== collateralUtxo.input.outputIndex
        )
      )
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    res.json({ tx_hash: txHash });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[MARKETPLACE] Buy error:", message);
    res.status(500).json({ error: message });
  }
});

// ─── POST /cancel ─────────────────────────────────────────────────────────────

/**
 * Cancels an active listing and returns the NFT to the seller.
 *
 * Uses the Cancel redeemer (d87a80) which triggers the Aiken validator to check:
 * - Original seller's signature is present (only seller can cancel)
 *
 * Retries UTxO lookup to handle Blockfrost indexing delay — critical because
 * users often try to cancel immediately after listing, before the listing tx
 * has been indexed by Blockfrost (~20s on Preprod).
 *
 * Body: { mnemonic, listing_utxo_hash, listing_utxo_index, nft_unit }
 */
router.post("/cancel", async (req: Request, res: Response) => {
  try {
    const {
      mnemonic,
      listing_utxo_hash,
      listing_utxo_index,
      nft_unit,
    } = req.body;

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

    const collateralUtxo = findCollateralUtxo(utxos);
    if (!collateralUtxo) {
      res.status(400).json({
        error: "Wallet needs a pure ADA UTxO of at least 5 ADA for collateral",
      });
      return;
    }

    // Fetch listing UTxO with retry — critical for users who cancel immediately
    // after listing (before Blockfrost has indexed the listing tx)
    const listingUtxo = await fetchListingUtxoWithRetry(
      provider,
      listing_utxo_hash,
      parseInt((listing_utxo_index ?? 0).toString())
    );

    if (!listingUtxo) {
      res.status(404).json({
        error:
          "Listing not found on-chain. It may have already been sold or cancelled. " +
          "Please refresh the page.",
      });
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
      .txInRedeemerValue("d87a80", "CBOR") // Cancel redeemer (Constructor 1, empty fields)
      .txInScript(marketplaceScript)
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex,
        collateralUtxo.output.amount,
        collateralUtxo.output.address
      )
      .txOut(sellerAddress, [{ unit: nft_unit, quantity: "1" }])
      .changeAddress(sellerAddress)
      .selectUtxosFrom(
        utxos.filter(
          (u) =>
            u.input.txHash !== collateralUtxo.input.txHash ||
            u.input.outputIndex !== collateralUtxo.input.outputIndex
        )
      )
      .requiredSignerHash(resolvePaymentKeyHash(sellerAddress))
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    res.json({ tx_hash: txHash });
  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[MARKETPLACE] Cancel error:", message);
    res.status(500).json({ error: message });
  }
});

export { marketplaceAddress };
export default router;