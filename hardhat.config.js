require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();

const TESTNET_RPC_URL = process.env.OG_CHAIN_RPC || "https://evmrpc-testnet.0g.ai";
const MAINNET_RPC_URL = process.env.OG_MAINNET_RPC || "https://rpc.0g.ai";
const PRIVATE_KEY = process.env.PRIVATE_KEY || "";

function configuredAccounts() {
  return PRIVATE_KEY ? [PRIVATE_KEY] : [];
}

function configuredChainID(value) {
  if (!value) {
    return undefined;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

const testnetChainID = configuredChainID(process.env.OG_TESTNET_CHAIN_ID || "16601");
const mainnetChainID = configuredChainID(process.env.OG_MAINNET_CHAIN_ID);

module.exports = {
  solidity: {
    version: "0.8.20",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
      },
    },
  },
  networks: {
    "0g-testnet": {
      url: TESTNET_RPC_URL,
      accounts: configuredAccounts(),
      chainId: testnetChainID,
    },
    "0g-mainnet": {
      url: MAINNET_RPC_URL,
      accounts: configuredAccounts(),
      ...(mainnetChainID ? { chainId: mainnetChainID } : {}),
    },
    hardhat: {
      chainId: 1337,
    },
    localhost: {
      url: "http://127.0.0.1:8545",
    },
  },
  etherscan: {
    apiKey: {
      "0g-testnet": process.env.OG_TESTNET_EXPLORER_API_KEY || "0g",
      "0g-mainnet": process.env.OG_MAINNET_EXPLORER_API_KEY || "0g",
    },
    customChains: [
      {
        network: "0g-testnet",
        chainId: testnetChainID || 16601,
        urls: {
          apiURL: process.env.OG_TESTNET_EXPLORER_API_URL || "https://chainscan-galileo.0g.ai/api",
          browserURL: process.env.OG_TESTNET_EXPLORER_URL || "https://chainscan-galileo.0g.ai",
        },
      },
      ...(mainnetChainID
        ? [
            {
              network: "0g-mainnet",
              chainId: mainnetChainID,
              urls: {
                apiURL: process.env.OG_MAINNET_EXPLORER_API_URL || "",
                browserURL: process.env.OG_MAINNET_EXPLORER_URL || "",
              },
            },
          ]
        : []),
    ],
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS === "true",
    currency: "USD",
    coinmarketcap: process.env.COINMARKETCAP_API_KEY,
  },
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts",
  },
};
