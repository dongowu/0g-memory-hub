const fs = require("fs");
const path = require("path");
const { Wallet } = require("ethers");

const TESTNET_CHAIN_ID = 16602;
const TESTNET_RPC_URL = "https://evmrpc-testnet.0g.ai";
const TESTNET_EXPLORER_URL = "https://chainscan-galileo.0g.ai";
const FAUCETS = [
  "https://faucet.0g.ai",
  "https://cloud.google.com/application/web3/faucet/0g/galileo",
];

function createWalletBundle() {
  const wallet = Wallet.createRandom();

  return {
    address: wallet.address,
    privateKey: wallet.privateKey,
    mnemonic: wallet.mnemonic?.phrase || "",
    chainId: TESTNET_CHAIN_ID,
    rpcURL: TESTNET_RPC_URL,
    explorerURL: TESTNET_EXPLORER_URL,
    faucets: FAUCETS,
    exportPrivateKeyLine: `export PRIVATE_KEY=${wallet.privateKey}`,
    exportAddressLine: `export TESTNET_WALLET_ADDRESS=${wallet.address}`,
  };
}

function defaultWalletOutputPath(address) {
  return path.join(process.cwd(), ".wallets", `0g-galileo-${address}.json`);
}

function formatWalletBundle(bundle) {
  return [
    "0G Galileo testnet wallet",
    "=========================",
    "",
    `Address: ${bundle.address}`,
    `Private Key: ${bundle.privateKey}`,
    `Mnemonic: ${bundle.mnemonic}`,
    "",
    `Chain ID: ${bundle.chainId}`,
    `RPC: ${bundle.rpcURL}`,
    `Explorer: ${bundle.explorerURL}`,
    "",
    "Next steps:",
    `1. Fund this address from a faucet: ${bundle.faucets[0]}`,
    ...(bundle.faucets[1] ? [`   Backup faucet: ${bundle.faucets[1]}`] : []),
    `2. Export the key: ${bundle.exportPrivateKeyLine}`,
    "3. Run: npm run preflight:testnet",
    "4. Run: npm run deploy:proof",
  ].join("\n");
}

function main() {
  const bundle = createWalletBundle();
  const args = new Set(process.argv.slice(2));
  const save = args.has("--save");
  const json = args.has("--json");

  if (save) {
    const outputPath = defaultWalletOutputPath(bundle.address);
    fs.mkdirSync(path.dirname(outputPath), { recursive: true });
    fs.writeFileSync(outputPath, `${JSON.stringify(bundle, null, 2)}\n`, "utf8");
  }

  if (json) {
    process.stdout.write(`${JSON.stringify(bundle, null, 2)}\n`);
    return;
  }

  process.stdout.write(`${formatWalletBundle(bundle)}\n`);
  if (save) {
    process.stdout.write(`Saved wallet JSON: ${defaultWalletOutputPath(bundle.address)}\n`);
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  TESTNET_CHAIN_ID,
  TESTNET_RPC_URL,
  TESTNET_EXPLORER_URL,
  FAUCETS,
  createWalletBundle,
  formatWalletBundle,
  defaultWalletOutputPath,
};

