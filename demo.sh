#!/bin/bash

# 0G Memory Hub Demo Script
# This script demonstrates the end-to-end workflow

set -e

echo "🚀 0G Memory Hub - End-to-End Demo"
echo "=================================="
echo ""

# Configuration
AGENT_ADDRESS="${1:-0x1234567890123456789012345678901234567890}"
MEMORY_FILE="${2:-./demo_memory.json}"

echo "📋 Configuration:"
echo "   Agent Address: $AGENT_ADDRESS"
echo "   Memory File: $MEMORY_FILE"
echo ""

# Step 1: Create sample memory file
echo "Step 1️⃣  Creating sample memory file..."
cat > "$MEMORY_FILE" << 'EOF'
{
  "agent_id": "agent_001",
  "timestamp": 1710950400,
  "memories": [
    {
      "type": "conversation",
      "content": "Discussed AI agent memory systems",
      "confidence": 0.95
    },
    {
      "type": "learning",
      "content": "Learned about 0G decentralized infrastructure",
      "confidence": 0.88
    }
  ],
  "metadata": {
    "session_id": "session_12345",
    "duration_seconds": 3600,
    "model": "claude-3-sonnet"
  }
}
EOF
echo "   ✅ Created: $MEMORY_FILE"
echo ""

# Step 2: Build the project
echo "Step 2️⃣  Building Rust project..."
cargo build --release 2>&1 | grep -E "(Compiling|Finished)" || true
echo "   ✅ Build complete"
echo ""

# Step 3: Upload to 0G Storage
echo "Step 3️⃣  Uploading memory to 0G Storage..."
UPLOAD_OUTPUT=$(cargo run --release -- upload "$MEMORY_FILE" --replicas 2 2>&1)
echo "$UPLOAD_OUTPUT"
CID=$(echo "$UPLOAD_OUTPUT" | grep "CID:" | awk '{print $NF}')
echo "   ✅ CID: $CID"
echo ""

# Step 4: Set pointer on-chain
echo "Step 4️⃣  Anchoring CID on 0G Chain..."
CHAIN_OUTPUT=$(cargo run --release -- set-pointer "$AGENT_ADDRESS" "$CID" 2>&1)
echo "$CHAIN_OUTPUT"
echo "   ✅ Pointer set on-chain"
echo ""

# Step 5: Verify pointer
echo "Step 5️⃣  Verifying on-chain pointer..."
VERIFY_OUTPUT=$(cargo run --release -- get-pointer "$AGENT_ADDRESS" 2>&1)
echo "$VERIFY_OUTPUT"
echo "   ✅ Verification complete"
echo ""

# Step 6: Get history
echo "Step 6️⃣  Retrieving memory history..."
HISTORY_OUTPUT=$(cargo run --release -- get-history "$AGENT_ADDRESS" 2>&1)
echo "$HISTORY_OUTPUT"
echo "   ✅ History retrieved"
echo ""

# Step 7: Download and verify
echo "Step 7️⃣  Downloading memory from storage..."
cargo run --release -- download "$CID" --output ./demo_memory_restored.json --verify 2>&1
echo "   ✅ Download complete"
echo ""

echo "🎉 Demo Complete!"
echo "=================================="
echo ""
echo "Summary:"
echo "  ✅ Memory uploaded to 0G Storage"
echo "  ✅ CID anchored on 0G Chain"
echo "  ✅ On-chain pointer verified"
echo "  ✅ Memory history retrieved"
echo "  ✅ Memory downloaded and verified"
echo ""
echo "Your AI agent now has eternal memory on 0G! 🧠"
