#!/bin/bash

# 0G Memory Hub - Smart Contract Deployment Script
# This script deploys the MemoryChain contract to 0G Chain

set -e

echo "🚀 0G Memory Hub - Smart Contract Deployment"
echo "=============================================="
echo ""

# Configuration
RPC_URL="${1:-https://testnet-rpc.0g.ai}"
PRIVATE_KEY="${2:-0x}"
VERIFY="${3:-false}"

if [ -z "$PRIVATE_KEY" ] || [ "$PRIVATE_KEY" = "0x" ]; then
    echo "❌ Error: Private key not provided"
    echo "Usage: ./deploy.sh <RPC_URL> <PRIVATE_KEY> [verify]"
    echo ""
    echo "Example:"
    echo "  ./deploy.sh https://testnet-rpc.0g.ai 0x... true"
    exit 1
fi

echo "📋 Configuration:"
echo "   RPC URL: $RPC_URL"
echo "   Verify: $VERIFY"
echo ""

# Check if Foundry is installed
if ! command -v forge &> /dev/null; then
    echo "❌ Foundry not found. Installing..."
    curl -L https://foundry.paradigm.xyz | bash
    foundryup
fi

echo "Step 1️⃣  Compiling contract..."
forge build

echo ""
echo "Step 2️⃣  Deploying MemoryChain contract..."

if [ "$VERIFY" = "true" ]; then
    echo "   (with verification)"
    DEPLOY_CMD="forge create contracts/MemoryChain.sol:MemoryChain \
      --rpc-url $RPC_URL \
      --private-key $PRIVATE_KEY \
      --verify"
else
    DEPLOY_CMD="forge create contracts/MemoryChain.sol:MemoryChain \
      --rpc-url $RPC_URL \
      --private-key $PRIVATE_KEY"
fi

DEPLOY_OUTPUT=$($DEPLOY_CMD)

echo "$DEPLOY_OUTPUT"

# Extract contract address
CONTRACT_ADDRESS=$(echo "$DEPLOY_OUTPUT" | grep "Deployed to:" | awk '{print $NF}')

echo ""
echo "✅ Deployment successful!"
echo "   Contract Address: $CONTRACT_ADDRESS"
echo ""

# Save to .env
echo "Step 3️⃣  Updating .env file..."
if [ -f ".env" ]; then
    sed -i "s/CONTRACT_ADDRESS=.*/CONTRACT_ADDRESS=$CONTRACT_ADDRESS/" .env
else
    echo "CONTRACT_ADDRESS=$CONTRACT_ADDRESS" >> .env
fi

echo "   ✅ .env updated"
echo ""

echo "🎉 Deployment complete!"
echo "=============================================="
echo ""
echo "Next steps:"
echo "1. Verify contract on 0G Explorer:"
echo "   https://testnet-explorer.0g.ai/address/$CONTRACT_ADDRESS"
echo ""
echo "2. Update your configuration:"
echo "   export CONTRACT_ADDRESS=$CONTRACT_ADDRESS"
echo ""
echo "3. Test the contract:"
echo "   cargo run --release -- get-pointer 0x1234567890123456789012345678901234567890"
echo ""
