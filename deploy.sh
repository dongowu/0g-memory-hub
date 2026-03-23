#!/usr/bin/env bash

# 0G Memory Hub - current deploy wrapper
# Canonical path: Hardhat + MemoryAnchor + Galileo / mainnet explorer settings

set -euo pipefail

NETWORK_NAME="${4:-${NETWORK_NAME:-0g-testnet}}"
RUN_PROOF="${3:-${RUN_PROOF:-true}}"

if [[ "$NETWORK_NAME" == "0g-mainnet" ]]; then
  DEFAULT_RPC="https://evmrpc.0g.ai"
  DEFAULT_CHAIN_ID="16661"
  DEFAULT_EXPLORER="https://chainscan.0g.ai"
  DEPLOY_CMD_NO_PROOF="npm run deploy:mainnet"
  DEPLOY_CMD_WITH_PROOF="npm run deploy:proof:mainnet"
else
  NETWORK_NAME="0g-testnet"
  DEFAULT_RPC="https://evmrpc-testnet.0g.ai"
  DEFAULT_CHAIN_ID="16602"
  DEFAULT_EXPLORER="https://chainscan-galileo.0g.ai"
  DEPLOY_CMD_NO_PROOF="npm run deploy"
  DEPLOY_CMD_WITH_PROOF="npm run deploy:proof"
fi

RPC_URL="${1:-${OG_CHAIN_RPC:-$DEFAULT_RPC}}"
PRIVATE_KEY_INPUT="${2:-${PRIVATE_KEY:-}}"

if [[ -z "$PRIVATE_KEY_INPUT" || "$PRIVATE_KEY_INPUT" == "0x" ]]; then
  echo "❌ Error: Private key not provided"
  echo "Usage: ./deploy.sh [rpc_url] [private_key] [run_proof=true|false] [network=0g-testnet|0g-mainnet]"
  echo ""
  echo "Examples:"
  echo "  ./deploy.sh https://evmrpc-testnet.0g.ai 0x... true"
  echo "  ./deploy.sh https://evmrpc.0g.ai 0x... false 0g-mainnet"
  exit 1
fi

export OG_CHAIN_RPC="$RPC_URL"
export PRIVATE_KEY="$PRIVATE_KEY_INPUT"

if [[ "$NETWORK_NAME" == "0g-mainnet" ]]; then
  export OG_MAINNET_RPC="${OG_MAINNET_RPC:-$RPC_URL}"
  export OG_MAINNET_CHAIN_ID="${OG_MAINNET_CHAIN_ID:-$DEFAULT_CHAIN_ID}"
  export OG_MAINNET_EXPLORER_URL="${OG_MAINNET_EXPLORER_URL:-$DEFAULT_EXPLORER}"
else
  export OG_TESTNET_CHAIN_ID="${OG_TESTNET_CHAIN_ID:-$DEFAULT_CHAIN_ID}"
  export OG_TESTNET_EXPLORER_URL="${OG_TESTNET_EXPLORER_URL:-$DEFAULT_EXPLORER}"
fi

echo "🚀 0G Memory Hub - MemoryAnchor deploy wrapper"
echo "=============================================="
echo "Network: $NETWORK_NAME"
echo "RPC URL: $RPC_URL"
echo "Run proof: $RUN_PROOF"
echo ""

echo "Step 1️⃣  Preflight"
if [[ "$NETWORK_NAME" == "0g-testnet" ]]; then
  npm run preflight:testnet
else
  echo "Mainnet mode: skipping testnet preflight helper"
fi

echo ""
echo "Step 2️⃣  Compile"
npx hardhat compile

echo ""
echo "Step 3️⃣  Contract tests"
npx hardhat test test/MemoryAnchor.js

echo ""
echo "Step 4️⃣  Deploy MemoryAnchor"
if [[ "$RUN_PROOF" == "1" || "$RUN_PROOF" == "true" || "$RUN_PROOF" == "yes" ]]; then
  eval "$DEPLOY_CMD_WITH_PROOF"
else
  eval "$DEPLOY_CMD_NO_PROOF"
fi

ARTIFACT_PATH="deployments/${NETWORK_NAME}/MemoryAnchor.latest.json"

if [[ -f "$ARTIFACT_PATH" ]]; then
  echo ""
  echo "Step 5️⃣  Deployment artifact"
  ARTIFACT_PATH="$ARTIFACT_PATH" node - <<'EOF'
const fs = require("fs");
const path = require("path");
const artifactPath = path.join(process.cwd(), process.env.ARTIFACT_PATH);
const deployment = JSON.parse(fs.readFileSync(artifactPath, "utf8"));
console.log(`Artifact: ${process.env.ARTIFACT_PATH}`);
console.log(`Contract: ${deployment.contractAddress}`);
if (deployment.contractExplorerURL) console.log(`Explorer: ${deployment.contractExplorerURL}`);
if (deployment.deploymentTxExplorerURL) console.log(`Deployment tx: ${deployment.deploymentTxExplorerURL}`);
if (deployment.proof?.txExplorerURL) console.log(`Proof tx: ${deployment.proof.txExplorerURL}`);
console.log("");
console.log("Suggested orchestrator exports:");
console.log(`export ORCH_CHAIN_RPC_URL=${process.env.OG_CHAIN_RPC}`);
console.log(`export ORCH_CHAIN_ID=${process.env.OG_TESTNET_CHAIN_ID || process.env.OG_MAINNET_CHAIN_ID || ""}`);
console.log(`export ORCH_CHAIN_CONTRACT_ADDRESS=${deployment.contractAddress}`);
console.log("export ORCH_CHAIN_PRIVATE_KEY=$PRIVATE_KEY");
EOF
else
  echo "⚠️ Deployment artifact not found at $ARTIFACT_PATH"
fi
