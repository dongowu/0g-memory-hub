const fs = require("fs");
const path = require("path");

const REQUIRED_FOR_DEPLOY = ["PRIVATE_KEY"];
const RECOMMENDED = [
  ["OG_CHAIN_RPC", "https://evmrpc-testnet.0g.ai"],
  ["OG_TESTNET_CHAIN_ID", "16602"],
  ["OG_TESTNET_EXPLORER_URL", "https://chainscan-galileo.0g.ai"],
];

function loadLatestDeployment() {
  const deploymentPath = path.join(process.cwd(), "deployments", "0g-testnet", "MemoryAnchor.latest.json");
  if (!fs.existsSync(deploymentPath)) {
    return null;
  }
  return JSON.parse(fs.readFileSync(deploymentPath, "utf8"));
}

function main() {
  const missing = REQUIRED_FOR_DEPLOY.filter((key) => !process.env[key]);
  const deployment = loadLatestDeployment();

  console.log("0G testnet preflight");
  console.log("====================");
  console.log("");

  for (const [key, fallback] of RECOMMENDED) {
    console.log(`${key}=${process.env[key] || fallback} ${process.env[key] ? "(from env)" : "(recommended default)"}`);
  }
  console.log(`PRIVATE_KEY=${process.env.PRIVATE_KEY ? "SET" : "MISSING"}`);
  console.log("");

  if (deployment) {
    console.log("Latest deployment artifact detected:");
    console.log(`- Address: ${deployment.contractAddress}`);
    console.log(`- Deployment tx: ${deployment.deploymentTxHash || "(none)"}`);
    if (deployment.proof?.txHash) {
      console.log(`- Proof tx: ${deployment.proof.txHash}`);
    }
    console.log("");
    console.log("Suggested orchestrator exports:");
    console.log(`export ORCH_CHAIN_RPC_URL=${process.env.OG_CHAIN_RPC || "https://evmrpc-testnet.0g.ai"}`);
    console.log(`export ORCH_CHAIN_ID=${process.env.OG_TESTNET_CHAIN_ID || "16602"}`);
    console.log(`export ORCH_CHAIN_CONTRACT_ADDRESS=${deployment.contractAddress}`);
    if (process.env.PRIVATE_KEY) {
      console.log("export ORCH_CHAIN_PRIVATE_KEY=$PRIVATE_KEY");
    } else {
      console.log("# export ORCH_CHAIN_PRIVATE_KEY=<same PRIVATE_KEY used for deployment>");
    }
    console.log("");
  }

  if (missing.length > 0) {
    console.error(`Missing required deployment secret(s): ${missing.join(", ")}`);
    process.exit(1);
  }

  console.log("Preflight looks ready for testnet deployment.");
}

if (require.main === module) {
  main();
}

