import { MeshWallet, BlockfrostProvider } from "@meshsdk/core";

const getProvider = (): BlockfrostProvider => {
  const projectId = process.env.BLOCKFROST_PROJECT_ID;
  if (!projectId) throw new Error("BLOCKFROST_PROJECT_ID is not set");
  return new BlockfrostProvider(projectId);
};

export const generateWallet = async (): Promise<{
  mnemonic: string[];
  address: string;
}> => {
  const provider = getProvider();
  const mnemonic = MeshWallet.brew() as string[];
  const wallet = new MeshWallet({
    networkId: 0,
    fetcher: provider,
    submitter: provider,
    key: { type: "mnemonic", words: mnemonic },
  });
  const address = await wallet.getChangeAddress();
  return { mnemonic, address };
};

export const signTransaction = async (
  mnemonic: string[],
  unsignedTx: string
): Promise<string> => {
  const provider = getProvider();
  const wallet = new MeshWallet({
    networkId: 0,
    fetcher: provider,
    submitter: provider,
    key: { type: "mnemonic", words: mnemonic },
  });
  return await wallet.signTx(unsignedTx, true);
};

export const getWalletAddress = async (
  mnemonic: string[]
): Promise<string> => {
  const provider = getProvider();
  const wallet = new MeshWallet({
    networkId: 0,
    fetcher: provider,
    submitter: provider,
    key: { type: "mnemonic", words: mnemonic },
  });
  return await wallet.getChangeAddress();
};
