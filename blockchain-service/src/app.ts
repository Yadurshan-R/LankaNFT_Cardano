import express from "express";
import dotenv from "dotenv";
import config from "./config";
import { validateSecret } from "./middleware";
import { generateWallet, signTransaction, getWalletAddress } from "./wallet";
import mintRouter from "./routes/mint";

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

// Mint routes
app.use("/api/mint", mintRouter);

app.listen(config.port, () => {
  console.log(`Blockchain service running on port ${config.port}`);
});

export default app;