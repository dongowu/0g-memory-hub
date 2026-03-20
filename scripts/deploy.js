// Hardhat deployment script for MemoryChain contract
// Usage: npx hardhat run scripts/deploy.js --network 0g-testnet

async function main() {
  console.log("🚀 Deploying MemoryChain contract...\n");

  // Get the contract factory
  const MemoryChain = await ethers.getContractFactory("MemoryChain");

  // Deploy the contract
  console.log("📝 Deploying contract...");
  const memoryChain = await MemoryChain.deploy();
  await memoryChain.deployed();

  console.log("✅ Contract deployed successfully!");
  console.log(`   Contract Address: ${memoryChain.address}\n`);

  // Save deployment info
  const deploymentInfo = {
    network: hre.network.name,
    contractAddress: memoryChain.address,
    deploymentBlock: await ethers.provider.getBlockNumber(),
    timestamp: new Date().toISOString(),
  };

  console.log("📋 Deployment Info:");
  console.log(JSON.stringify(deploymentInfo, null, 2));

  // Verify on block explorer (if applicable)
  if (hre.network.name !== "hardhat" && hre.network.name !== "localhost") {
    console.log("\n⏳ Waiting for block confirmations before verification...");
    await memoryChain.deployTransaction.wait(5);

    console.log("🔍 Verifying contract on block explorer...");
    try {
      await hre.run("verify:verify", {
        address: memoryChain.address,
        constructorArguments: [],
      });
      console.log("✅ Contract verified!");
    } catch (error) {
      console.log("⚠️  Verification failed (this is normal for some networks)");
      console.log(`   Error: ${error.message}`);
    }
  }

  // Test the contract
  console.log("\n🧪 Testing contract functions...");

  const testAgent = "0x1234567890123456789012345678901234567890";
  const testCID = ethers.utils.formatBytes32String("test_cid_001");

  try {
    // Note: This will fail if called from a different address
    // In production, you'd need to use the agent's private key
    console.log("   (Skipping write test - requires agent's private key)");
  } catch (error) {
    console.log(`   Error: ${error.message}`);
  }

  console.log("\n🎉 Deployment complete!");
  console.log("\nNext steps:");
  console.log(`1. Update your .env file with CONTRACT_ADDRESS=${memoryChain.address}`);
  console.log(`2. View on explorer: https://testnet-explorer.0g.ai/address/${memoryChain.address}`);
  console.log("3. Run the CLI: cargo run --release -- get-pointer <agent_address>");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
