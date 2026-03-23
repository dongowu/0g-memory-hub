const test = require("node:test");
const assert = require("node:assert/strict");

const { renderEvidenceMarkdown } = require("../scripts/render_deployment_evidence");

test("renderEvidenceMarkdown includes deployment and proof fields", () => {
  const markdown = renderEvidenceMarkdown({
    network: "0g-testnet",
    contractName: "MemoryAnchor",
    contractAddress: "0xabc",
    contractExplorerURL: "https://chainscan-galileo.0g.ai/address/0xabc",
    deploymentTxHash: "0xdeploy",
    deploymentTxExplorerURL: "https://chainscan-galileo.0g.ai/tx/0xdeploy",
    deployer: "0xsigner",
    deploymentBlock: 123,
    timestamp: "2026-03-23T15:00:00.000Z",
    proof: {
      workflowLabel: "judge-run",
      workflowId: "0xworkflow",
      stepIndex: 1,
      rootHash: "0xroot",
      cidHash: "0xcid",
      txHash: "0xproof",
      txExplorerURL: "https://chainscan-galileo.0g.ai/tx/0xproof",
      blockNumber: 124,
    },
  });

  assert.match(markdown, /MemoryAnchor/);
  assert.match(markdown, /0xabc/);
  assert.match(markdown, /0xdeploy/);
  assert.match(markdown, /judge-run/);
  assert.match(markdown, /0xproof/);
  assert.match(markdown, /chainscan-galileo/);
});

