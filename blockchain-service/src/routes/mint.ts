// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/mint.ts
//
// CIP-68 + CIP-102 Compliant Single NFT Minting Route
//
// Mints THREE tokens in one transaction:
//   (100) Reference NFT  → locked at immutable_lock script with CIP-68 datum
//   (222) User NFT       → sent to the creator's wallet
//   (500) Royalty NFT    → locked at royalty_lock script with CIP-102 datum
//
// The minting policy is one-shot: parameterized by a UTxO reference.
// Once that UTxO is consumed, the policy ID can never mint again.
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

const router = Router();

// ─── Load Blueprint ───────────────────────────────────────────────────────────
const blueprintPath = path.join(__dirname, "../../plutus.json");
const blueprintJson = JSON.parse(fs.readFileSync(blueprintPath, "utf-8"));

// Helper to find a validator by title
function getValidator(title: string) {
  const v = blueprintJson.validators.find((v: any) => v.title === title);
  if (!v) throw new Error(`Validator "${title}" not found in plutus.json`);
  return v;
}

// Load validators from blueprint
const mintSingleValidator = getValidator(
  "minting_policy.cip68_mint_single.mint"
);
const immutableLockValidator = getValidator(
  "reference_lock.immutable_lock.spend"
);
const royaltyLockValidator = getValidator("royalty_lock.royalty_lock.spend");

// ─── Compute Script Addresses ─────────────────────────────────────────────────
// serializePlutusScript returns { address: any }
// networkId 0 = preprod/testnet, 1 = mainnet

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

console.log("✅ Immutable lock address:", immutableLockAddress);
console.log("✅ Royalty lock address:", royaltyLockAddress);

// ─── Route ────────────────────────────────────────────────────────────────────

/**
 * POST /api/mint/single
 *
 * Request body:
 *   mnemonic      string[]  — 24 word mnemonic of the custodial wallet
 *   asset_name    string    — base name of the NFT (e.g. "MidnightApe001")
 *   metadata_ipfs string    — ipfs:// URI of the CIP-68 metadata JSON
 *   image_ipfs    string    — ipfs:// URI of the NFT image
 *   royalties     number    — royalty percentage (e.g. 5 = 5%)
 *
 * Response:
 *   tx_hash       string    — Cardano transaction hash
 *   policy_id     string    — unique policy ID of this NFT
 *   ref_token     string    — full unit of the (100) reference NFT
 *   user_token    string    — full unit of the (222) user NFT
 *   royalty_token string    — full unit of the (500) royalty NFT
 */
router.post("/single", async (req: Request, res: Response) => {
  try {
    const {
      mnemonic,
      asset_name,
      metadata_ipfs,
      image_ipfs,
      royalties = 0,
    } = req.body;

    // ── Input validation ──────────────────────────────────────────────────
    if (!mnemonic || !asset_name || !metadata_ipfs) {
      res.status(400).json({
        error: "mnemonic, asset_name and metadata_ipfs are required",
      });
      return;
    }

    // ── Setup provider and wallet ─────────────────────────────────────────
    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    // Load custodial wallet from decrypted mnemonic
    const wallet = new MeshWallet({
      networkId: 0,
      fetcher: provider,
      submitter: provider,
      key: { type: "mnemonic", words: mnemonic },
    });

    // ── Check wallet UTxOs ────────────────────────────────────────────────
    const utxos = await wallet.getUtxos();
    if (utxos.length === 0) {
      res.status(400).json({
        error: "Wallet has no UTxOs — fund the preprod wallet via the faucet",
      });
      return;
    }

    // One-shot UTxO: consumed to make this policy unique forever
    const oneShotUtxo = utxos[0];
    const txHash = oneShotUtxo.input.txHash;
    const txIndex = oneShotUtxo.input.outputIndex;

    // Collateral UTxO: must be pure ADA only — no native tokens allowed
    const collateralUtxo = utxos.find(
      (u) =>
        !(u.input.txHash === txHash && u.input.outputIndex === txIndex) &&
        u.output.amount.every((a) => a.unit === "lovelace")
    );

    if (!collateralUtxo) {
      res.status(400).json({
        error:
          "No pure-ADA UTxO available for collateral. " +
          "Send at least 5 tADA to the wallet address and try again.",
      });
      return;
    }

    // ── Parameterize the minting policy ───────────────────────────────────
    // OutputReference = ConStr0 [ txHash, txIndex ]
    const outputRef = mConStr0([txHash, txIndex]);
    const mintBlueprint = new MintingBlueprint("V3");
    mintBlueprint.paramScript(
      mintSingleValidator.compiledCode,
      [outputRef],
      "Mesh"
    );
    const policyId = mintBlueprint.hash;
    const mintScript = mintBlueprint.cbor;

    // ── CIP-68 token names ────────────────────────────────────────────────
    // Full token name = 4-byte label prefix + hex encoded base name
    const assetNameHex = stringToHex(asset_name);
    const refTokenName = "000643b0" + assetNameHex;     // (100) Reference NFT
    const userTokenName = "001bc280" + assetNameHex;    // (222) User NFT
    const royaltyTokenName = "001f4d70" + assetNameHex; // (500) Royalty NFT

    // Creator address — receives the (222) user NFT
    const creatorAddress = await wallet.getChangeAddress();

    // ── CIP-68 metadata datum ─────────────────────────────────────────────
    // Attached as inline datum to the (100) Reference NFT
    // Marketplaces read this to display NFT name, image, description
    // Format: ConStr0 [ metadata_map, version ]
    const cip68Datum = mConStr0([
      [
        [stringToHex("name"),        stringToHex(asset_name)],
        [stringToHex("image"),       stringToHex(image_ipfs)],
        [stringToHex("mediaType"),   stringToHex("image/png")],
        [stringToHex("description"), stringToHex("")],
        [stringToHex("files"),       []],
      ],
      1, // CIP-68 version number
    ]);
    const cip68DatumCbor = serializeData(cip68Datum);

    // ── CIP-102 royalty datum ─────────────────────────────────────────────
    // Attached as inline datum to the (500) Royalty NFT
    // Marketplaces read this as a reference input to enforce royalty splits
    // Rate = (royalty% / 100) * 1,000,000
    // e.g. 5% royalty → rate = 50,000
    const royaltyRate = Math.floor((royalties / 100) * 1_000_000);

    // RoyaltyLockDatum { owner, royalty: RoyaltyDatum { recipients, min_ada }, collection_policy }
    const royaltyDatum = mConStr0([
      stringToHex(creatorAddress), // owner — can update royalty terms
      mConStr0([                   // RoyaltyDatum
        [
          mConStr0([               // RoyaltyRecipient
            stringToHex(creatorAddress), // address — creator gets royalties
            royaltyRate,                 // rate e.g. 50000 for 5%
          ]),
        ],
        2_000_000,                 // min_ada — minimum 2 ADA per payment
      ]),
      policyId,                    // collection_policy — links to this NFT
    ]);
    const royaltyDatumCbor = serializeData(royaltyDatum);

    // ── Mint redeemer ─────────────────────────────────────────────────────
    // MintSingle { asset_name: ByteArray, include_royalty: Bool }
    // Constructor 0 = MintSingle
    // mConStr1([]) = Bool True (constructor 1, no fields)
    const redeemer = mConStr0([
      assetNameHex, // asset_name as hex bytes
      mConStr1([]), // include_royalty = True
    ]);
    const redeemerCbor = serializeData(redeemer);

    // ── Build transaction ─────────────────────────────────────────────────
    const txBuilder = new MeshTxBuilder({
      fetcher: provider,
      submitter: provider,
    });

    const unsignedTx = await txBuilder
      // Consume the one-shot UTxO — locks this policy permanently
      .txIn(txHash, txIndex)

      // Mint (100) Reference NFT
      .mintPlutusScriptV3()
      .mint("1", policyId, refTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")

      // Mint (222) User NFT
      .mintPlutusScriptV3()
      .mint("1", policyId, userTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")

      // Mint (500) Royalty NFT
      .mintPlutusScriptV3()
      .mint("1", policyId, royaltyTokenName)
      .mintingScript(mintScript)
      .mintRedeemerValue(redeemerCbor, "CBOR")

      // Send (222) User NFT to creator wallet — proof of ownership
      .txOut(creatorAddress, [
        { unit: policyId + userTokenName, quantity: "1" },
      ])

      // Send (100) Reference NFT to immutable_lock with CIP-68 datum
      .txOut(immutableLockAddress, [
        { unit: policyId + refTokenName, quantity: "1" },
      ])
      .txOutInlineDatumValue(cip68DatumCbor, "CBOR")

      // Send (500) Royalty NFT to royalty_lock with CIP-102 datum
      .txOut(royaltyLockAddress, [
        { unit: policyId + royaltyTokenName, quantity: "1" },
      ])
      .txOutInlineDatumValue(royaltyDatumCbor, "CBOR")

      // Remaining ADA goes back to creator
      .changeAddress(creatorAddress)

      // Select UTxOs for fees
      .selectUtxosFrom(utxos)

      // Plutus V3 collateral — must be pure ADA
      .txInCollateral(
        collateralUtxo.input.txHash,
        collateralUtxo.input.outputIndex
      )
      .complete();

    // Sign and submit to Blockfrost
    const signedTx = await wallet.signTx(unsignedTx);
    const submittedTxHash = await wallet.submitTx(signedTx);

    console.log(`✅ NFT minted — tx: ${submittedTxHash}`);
    console.log(`   Policy ID:     ${policyId}`);
    console.log(`   User NFT:      ${policyId + userTokenName}`);
    console.log(`   Ref locked at: ${immutableLockAddress}`);
    console.log(`   Roy locked at: ${royaltyLockAddress}`);

    res.json({
      tx_hash: submittedTxHash,
      policy_id: policyId,
      asset_name,
      ref_token: policyId + refTokenName,
      user_token: policyId + userTokenName,
      royalty_token: policyId + royaltyTokenName,
    });
  } catch (error: any) {
    const message =
      typeof error === "string"
        ? error
        : error?.message || JSON.stringify(error);
    console.error("Mint error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;