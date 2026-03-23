const fs = require("fs");
const path = require("path");

function calendarDate(timestamp) {
  return String(timestamp || "").slice(0, 10) || "0000-00-00";
}

function renderEvidenceMarkdown(deployment) {
  const date = calendarDate(deployment.timestamp);
  const proof = deployment.proof;
  const lines = [
    `# ${date} ${deployment.network} ${deployment.contractName} Deployment Proof`,
    "",
    "## Summary",
    "",
    `On **${deployment.timestamp || "unknown time"}**, \`${deployment.contractName}\` was deployed to **${deployment.network}** and recorded as a judge-facing proof artifact.`,
    "",
    "## Deployment",
    "",
    `- Contract: \`${deployment.contractName}\``,
    `- Address: \`${deployment.contractAddress || ""}\``,
    `- Deployer: \`${deployment.deployer || ""}\``,
    `- Deployment block: \`${deployment.deploymentBlock ?? ""}\``,
    `- Deployment tx: \`${deployment.deploymentTxHash || ""}\``,
  ];

  if (deployment.contractExplorerURL) {
    lines.push(`- Contract explorer: \`${deployment.contractExplorerURL}\``);
  }
  if (deployment.deploymentTxExplorerURL) {
    lines.push(`- Deployment tx explorer: \`${deployment.deploymentTxExplorerURL}\``);
  }

  if (proof) {
    lines.push(
      "",
      "## Anchor Proof",
      "",
      `- Workflow label: \`${proof.workflowLabel}\``,
      `- Workflow ID: \`${proof.workflowId}\``,
      `- Step index: \`${proof.stepIndex}\``,
      `- Root hash: \`${proof.rootHash}\``,
      `- CID hash: \`${proof.cidHash}\``,
      `- Anchor tx: \`${proof.txHash}\``,
      `- Anchor block: \`${proof.blockNumber}\``,
    );
    if (proof.txExplorerURL) {
      lines.push(`- Anchor tx explorer: \`${proof.txExplorerURL}\``);
    }
  } else {
    lines.push(
      "",
      "## Anchor Proof",
      "",
      "No on-chain proof was attached to this deployment artifact.",
    );
  }

  lines.push(
    "",
    "## Judge Notes",
    "",
    "- This file is generated from the deployment artifact JSON.",
    "- Use the contract address and tx explorer links directly in HackQuest submission fields.",
    "",
  );

  return lines.join("\n");
}

function defaultOutputPath(deployment) {
  const date = calendarDate(deployment.timestamp);
  return path.join(process.cwd(), "docs", "evidence", `${date}-${deployment.network}-memory-anchor-deployment-proof.md`);
}

function main() {
  const inputPath = process.argv[2];
  if (!inputPath) {
    console.error("Usage: node scripts/render_deployment_evidence.js <deployment-json-path> [output-path]");
    process.exit(1);
  }

  const outputPath = process.argv[3];
  const deployment = JSON.parse(fs.readFileSync(inputPath, "utf8"));
  const markdown = renderEvidenceMarkdown(deployment);
  const finalOutputPath = outputPath || defaultOutputPath(deployment);

  fs.mkdirSync(path.dirname(finalOutputPath), { recursive: true });
  fs.writeFileSync(finalOutputPath, `${markdown}\n`, "utf8");
  console.log(finalOutputPath);
}

if (require.main === module) {
  main();
}

module.exports = {
  renderEvidenceMarkdown,
  defaultOutputPath,
};

