// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/app.ts
//
// Blockchain Service Entry Point
//
// This Node.js service handles all Cardano blockchain operations:
//   - Wallet generation and signing
//   - Single NFT minting (CIP-68 + CIP-102, one-shot Plutus policy)
//   - Batch NFT minting (CIP-68, native script collection policy)
// ─────────────────────────────────────────────────────────────────────────────

import express from "express";
import dotenv from "dotenv";
import config from "./config";
import { validateSecret } from "./middleware";
import { generateWallet, signTransaction, getWalletAddress } from "./wallet";
import mintRouter from "./routes/mint";
import batchRouter from "./routes/batch";
import marketplaceRouter from "./routes/marketplace";
import transferRouter from "./routes/transfer";
import { BlockfrostProvider } from "@meshsdk/core";

dotenv.config();

const app = express();
app.use(express.json({ limit: "50mb" }));

// Health check — no secret needed
app.get("/health", (_req, res) => {
  res.json({ status: "ok", message: "Blockchain service running on preprod" });
});

// All routes below require shared secret header
app.use(validateSecret);

// Wallet routes
app.post("/api/wallet/generate", async (_req, res) => {
  try {
    const { mnemonic, address } = await generateWallet();
    res.json({ mnemonic, address });
  } catch (error) {
    console.error("Wallet generation error:", error);
    res.status(500).json({ error: "Failed to generate wallet" });
  }
});

app.post("/api/wallet/sign", async (req, res) => {
  try {
    const { mnemonic, unsignedTx } = req.body;
    if (!mnemonic || !unsignedTx) {
      res.status(400).json({ error: "mnemonic and unsignedTx are required" });
      return;
    }
    const signedTx = await signTransaction(mnemonic, unsignedTx);
    res.json({ signedTx });
  } catch (error) {
    console.error("Signing error:", error);
    res.status(500).json({ error: "Failed to sign transaction" });
  }
});

app.post("/api/wallet/address", async (req, res) => {
  try {
    const { mnemonic } = req.body;
    if (!mnemonic) {
      res.status(400).json({ error: "mnemonic is required" });
      return;
    }
    const address = await getWalletAddress(mnemonic);
    res.json({ address });
  } catch (error) {
    console.error("Address error:", error);
    res.status(500).json({ error: "Failed to get address" });
  }
});

// Single NFT minting — Plutus one-shot policy (unique policy per NFT)
app.use("/api/mint", mintRouter);

// Batch NFT minting — Native script collection policy (all NFTs in one tx)
app.use("/api/mint", batchRouter);

// Marketplace routes
app.use("/api/marketplace", marketplaceRouter);

// NFT transfer — send any NFT to another address
app.use("/api/transfer", transferRouter);

// POST /api/submit
//
// Submits a signed transaction CBOR to Cardano via Blockfrost.
// Used by external wallet users — Lace signs but cannot reliably submit.
// Blockfrost submission is more reliable than wallet.submitTx().
//
// Body: { unsigned_cbor: string, witness_cbor: string }
// Returns: { tx_hash: string }
app.post("/api/submit", async (req, res) => {
  try {
    const { unsigned_cbor, witness_cbor } = req.body;
    if (!unsigned_cbor || !witness_cbor) {
      res.status(400).json({ error: "unsigned_cbor and witness_cbor are required" });
      return;
    }

    // Cardano Serialization Library — assembles tx correctly
    // Available as @emurgo/cardano-serialization-lib-nodejs (MeshSDK dependency)
    const CSL = require('@emurgo/cardano-serialization-lib-nodejs');

    // Parse the unsigned transaction built by MeshSDK
    const unsignedTx = CSL.Transaction.from_hex(unsigned_cbor);

    // Parse the witness set returned by Lace's signTx()
    const witnessSet = CSL.TransactionWitnessSet.from_hex(witness_cbor);

    // Assemble the full signed transaction:
    //   transaction = [body, witness_set, is_valid, auxiliary_data]
    // The body hash stays identical so Lace's signature remains valid.
    const signedTx = CSL.Transaction.new(
      unsignedTx.body(),
      witnessSet,
      unsignedTx.auxiliary_data() ?? undefined
    );

    // Convert to hex and submit via Blockfrost
    const signedCbor = Buffer.from(signedTx.to_bytes()).toString('hex');
    const provider = new BlockfrostProvider(config.blockfrost.projectId);
    const txHash = await provider.submitTx(signedCbor);

    console.log("[SUBMIT] Transaction submitted:", txHash);
    res.json({ tx_hash: txHash });

  } catch (error: any) {
    const message = typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("[SUBMIT] Error:", message);
    res.status(500).json({ error: message });
  }
});

app.listen(config.port, () => {
  console.log(`Blockchain service running on port ${config.port}`);
});

export default app;