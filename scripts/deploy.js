const hre = require("hardhat");

const EXPLORER_BASE_URLS = {
  "0g-testnet": process.env.OG_TESTNET_EXPLORER_URL || "https://chainscan-galileo.0g.ai",
  "0g-mainnet": process.env.OG_MAINNET_EXPLORER_URL || "",
};

async function main() {
  const contractName = process.env.CONTRACT_NAME || "MemoryAnchor";
  const [deployer] = await hre.ethers.getSigners();
  if (!deployer) {
    throw new Error("No deployer configured. Set PRIVATE_KEY before deploying to a remote network.");
  }

  console.log(`🚀 Deploying ${contractName}...\n`);
  console.log(`Network: ${hre.network.name}`);
  console.log(`Deployer: ${deployer.address}\n`);

  const contractFactory = await hre.ethers.getContractFactory(contractName);
  const contract = await contractFactory.deploy();
  await contract.waitForDeployment();

  const contractAddress = await contract.getAddress();
  const deploymentBlock = await hre.ethers.provider.getBlockNumber();

  console.log("✅ Contract deployed successfully!");
  console.log(`Contract: ${contractName}`);
  console.log(`Address: ${contractAddress}`);
  console.log(`Block: ${deploymentBlock}\n`);

  const explorerBaseURL = EXPLORER_BASE_URLS[hre.network.name];
  if (explorerBaseURL) {
    console.log(`Explorer: ${explorerBaseURL.replace(/\/$/, "")}/address/${contractAddress}`);
  }

  console.log("\n📋 Deployment Info:");
  console.log(
    JSON.stringify(
      {
        network: hre.network.name,
        contractName,
        contractAddress,
        deployer: deployer.address,
        deploymentBlock,
        timestamp: new Date().toISOString(),
      },
      null,
      2,
    ),
  );

  if (!["hardhat", "localhost"].includes(hre.network.name)) {
    console.log("\n⏳ Waiting for 5 confirmations before verification...");
    const deploymentTx = contract.deploymentTransaction();
    if (deploymentTx) {
      await deploymentTx.wait(5);
    }

    try {
      console.log("🔍 Verifying contract on explorer...");
      await hre.run("verify:verify", {
        address: contractAddress,
        constructorArguments: [],
      });
      console.log("✅ Contract verified!");
    } catch (error) {
      console.log("⚠️ Verification skipped or failed.");
      console.log(`Reason: ${error.message}`);
    }
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
