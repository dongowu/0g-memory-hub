const fs = require("fs");
const path = require("path");

const hre = require("hardhat");

const EXPLORER_BASE_URLS = {
  "0g-testnet": process.env.OG_TESTNET_EXPLORER_URL || "https://chainscan-galileo.0g.ai",
  "0g-mainnet": process.env.OG_MAINNET_EXPLORER_URL || "https://chainscan.0g.ai",
};

function isTruthy(value) {
  return ["1", "true", "yes", "on"].includes(String(value || "").toLowerCase());
}

function explorerAddressURL(networkName, contractAddress) {
  const baseURL = EXPLORER_BASE_URLS[networkName];
  if (!baseURL) {
    return "";
  }
  return `${baseURL.replace(/\/$/, "")}/address/${contractAddress}`;
}

function explorerTxURL(networkName, txHash) {
  const baseURL = EXPLORER_BASE_URLS[networkName];
  if (!baseURL || !txHash) {
    return "";
  }
  return `${baseURL.replace(/\/$/, "")}/tx/${txHash}`;
}

function deploymentOutputPath(networkName, contractName) {
  return path.join(process.cwd(), "deployments", networkName, `${contractName}.latest.json`);
}

function persistDeploymentInfo(networkName, contractName, deploymentInfo) {
  const outputPath = deploymentOutputPath(networkName, contractName);
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, `${JSON.stringify(deploymentInfo, null, 2)}\n`, "utf8");
  return outputPath;
}

async function runMemoryAnchorProof(contract, networkName) {
  const workflowLabel = process.env.PROOF_WORKFLOW_LABEL || `judge-${Date.now()}`;
  const stepIndex = Number(process.env.PROOF_STEP_INDEX || "1");
  const workflowId = hre.ethers.id(`workflow:${workflowLabel}`);
  const rootHash = hre.ethers.id(`root:${workflowLabel}:${stepIndex}`);
  const cidHash = hre.ethers.id(`cid:${workflowLabel}:${stepIndex}`);

  console.log("\n🧪 Running MemoryAnchor proof transaction...");
  console.log(`Workflow Label: ${workflowLabel}`);
  console.log(`Workflow ID: ${workflowId}`);
  console.log(`Root Hash: ${rootHash}`);
  console.log(`CID Hash: ${cidHash}`);

  const tx = await contract.anchorCheckpoint(workflowId, stepIndex, rootHash, cidHash);
  const receipt = await tx.wait();
  const latest = await contract.getLatestCheckpoint(workflowId);

  if (
    latest.stepIndex !== BigInt(stepIndex) ||
    latest.rootHash !== rootHash ||
    latest.cidHash !== cidHash
  ) {
    throw new Error("anchor proof validation failed: on-chain checkpoint did not match the submitted proof");
  }

  const proof = {
    workflowLabel,
    workflowId,
    stepIndex,
    rootHash,
    cidHash,
    txHash: receipt.hash,
    txExplorerURL: explorerTxURL(networkName, receipt.hash),
    blockNumber: receipt.blockNumber,
  };

  console.log("✅ MemoryAnchor proof succeeded!");
  console.log(`Proof Tx: ${proof.txHash}`);
  if (proof.txExplorerURL) {
    console.log(`Proof Explorer: ${proof.txExplorerURL}`);
  }

  return proof;
}

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
  const deploymentTx = contract.deploymentTransaction();
  const deploymentTxHash = deploymentTx ? deploymentTx.hash : "";

  console.log("✅ Contract deployed successfully!");
  console.log(`Contract: ${contractName}`);
  console.log(`Address: ${contractAddress}`);
  console.log(`Block: ${deploymentBlock}\n`);

  const addressExplorerURL = explorerAddressURL(hre.network.name, contractAddress);
  if (addressExplorerURL) {
    console.log(`Explorer: ${addressExplorerURL}`);
  }

  const deploymentInfo = {
    network: hre.network.name,
    contractName,
    contractAddress,
    contractExplorerURL: addressExplorerURL,
    deploymentTxHash,
    deploymentTxExplorerURL: explorerTxURL(hre.network.name, deploymentTxHash),
    deployer: deployer.address,
    deploymentBlock,
    timestamp: new Date().toISOString(),
  };

  if (isTruthy(process.env.RUN_ANCHOR_PROOF) && contractName === "MemoryAnchor") {
    deploymentInfo.proof = await runMemoryAnchorProof(contract, hre.network.name);
  }

  const outputPath = persistDeploymentInfo(hre.network.name, contractName, deploymentInfo);

  console.log("\n📋 Deployment Info:");
  console.log(JSON.stringify(deploymentInfo, null, 2));
  console.log(`Saved: ${outputPath}`);

  if (!["hardhat", "localhost"].includes(hre.network.name)) {
    console.log("\n⏳ Waiting for 5 confirmations before verification...");
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
