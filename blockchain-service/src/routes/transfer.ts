// ─────────────────────────────────────────────────────────────────────────────
// blockchain-service/src/routes/transfer.ts
//
// NFT Transfer Route
//
// Sends a CIP-68 (222) user token from one custodial wallet to any address.
// This is a simple UTxO transaction — no script spending required.
// The recipient address can be any valid Cardano address (custodial or external).
// ─────────────────────────────────────────────────────────────────────────────
import { Router, Request, Response } from "express";
import {
  BlockfrostProvider,
  MeshTxBuilder,
  MeshWallet,
} from "@meshsdk/core";
import config from "../config";

const router = Router();

// ─── Transfer ─────────────────────────────────────────────────────────────────
// POST /api/transfer
//
// Sends an NFT from the sender's custodial wallet to the recipient address.
// Body:
//   mnemonic          string[]  — sender's decrypted mnemonic
//   nft_unit          string    — full on-chain unit (policyId + hex asset name)
//   recipient_address string    — bech32 destination address
router.post("/", async (req: Request, res: Response) => {
  try {
    const { mnemonic, nft_unit, recipient_address } = req.body;

    if (!mnemonic || !nft_unit || !recipient_address) {
      res.status(400).json({
        error: "mnemonic, nft_unit and recipient_address are required",
      });
      return;
    }

    const provider = new BlockfrostProvider(config.blockfrost.projectId);

    // Load sender's custodial wallet
    const wallet = new MeshWallet({
      networkId: 0,
      fetcher: provider,
      submitter: provider,
      key: { type: "mnemonic", words: mnemonic },
    });

    const utxos = await wallet.getUtxos();
    if (utxos.length === 0) {
      res.status(400).json({ error: "Sender wallet has no UTxOs" });
      return;
    }

    const senderAddress = await wallet.getChangeAddress();

    // Verify the sender actually holds the NFT
    // Find the UTxO containing the NFT unit
    const nftUtxo = utxos.find((u) =>
      u.output.amount.some((a) => a.unit === nft_unit)
    );

    if (!nftUtxo) {
      res.status(400).json({
        error: "NFT not found in sender wallet — may have already been transferred",
      });
      return;
    }

    // Build a simple transfer transaction
    // Send NFT to recipient + ADA back to sender as change
    const txBuilder = new MeshTxBuilder({ fetcher: provider, submitter: provider });

    const unsignedTx = await txBuilder
      // Send the NFT to the recipient with minimum ADA (2 ADA for min-UTxO)
      .txOut(recipient_address, [
        { unit: nft_unit, quantity: "1" },
        { unit: "lovelace", quantity: "2000000" },
      ])
      .changeAddress(senderAddress)
      .selectUtxosFrom(utxos)
      .complete();

    const signedTx = await wallet.signTx(unsignedTx);
    const txHash = await wallet.submitTx(signedTx);

    res.json({
      tx_hash: txHash,
      nft_unit,
      recipient_address,
    });
  } catch (error: any) {
    const message =
      typeof error === "string" ? error : error?.message || JSON.stringify(error);
    console.error("Transfer error:", message);
    res.status(500).json({ error: message });
  }
});

export default router;