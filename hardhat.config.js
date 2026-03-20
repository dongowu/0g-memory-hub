require("@nomicfoundation/hardhat-toolbox");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-etherscan");
require("hardhat-gas-reporter");
require("solidity-coverage");
require("dotenv").config();

const PRIVATE_KEY = process.env.PRIVATE_KEY || "0x0000000000000000000000000000000000000000000000000000000000000000";
const RPC_URL = process.env.OG_CHAIN_RPC || "https://testnet-rpc.0g.ai";

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
      url: RPC_URL,
      accounts: [PRIVATE_KEY],
      chainId: 1, // Update with actual 0G Chain ID
    },
    "0g-mainnet": {
      url: "https://rpc.0g.ai", // Update with mainnet RPC
      accounts: [PRIVATE_KEY],
      chainId: 1, // Update with actual mainnet Chain ID
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
      "0g-testnet": "0g", // Placeholder
      "0g-mainnet": "0g", // Placeholder
    },
    customChains: [
      {
        network: "0g-testnet",
        chainId: 1,
        urls: {
          apiURL: "https://testnet-explorer.0g.ai/api",
          browserURL: "https://testnet-explorer.0g.ai",
        },
      },
      {
        network: "0g-mainnet",
        chainId: 1,
        urls: {
          apiURL: "https://explorer.0g.ai/api",
          browserURL: "https://explorer.0g.ai",
        },
      },
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
