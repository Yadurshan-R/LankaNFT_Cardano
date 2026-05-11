import dotenv from "dotenv";
dotenv.config();

const config = {
  port: process.env.PORT || 3001,
  sharedSecret: process.env.SHARED_SECRET || "",
  blockfrost: {
    projectId: process.env.BLOCKFROST_PROJECT_ID || "",
    network: "preprod",
  },
};

if (!config.blockfrost.projectId) {
  throw new Error("BLOCKFROST_PROJECT_ID is not set in .env");
}

if (!config.sharedSecret) {
  throw new Error("SHARED_SECRET is not set in .env");
}

export default config;
