const test = require("node:test");
const assert = require("node:assert/strict");

const { createWalletBundle } = require("../scripts/generate_testnet_wallet");

test("createWalletBundle returns wallet fields and faucet guidance", () => {
  const bundle = createWalletBundle();

  assert.match(bundle.address, /^0x[a-fA-F0-9]{40}$/);
  assert.match(bundle.privateKey, /^0x[a-fA-F0-9]{64}$/);
  assert.ok(bundle.mnemonic);
  assert.equal(bundle.chainId, 16602);
  assert.equal(bundle.rpcURL, "https://evmrpc-testnet.0g.ai");
  assert.equal(bundle.explorerURL, "https://chainscan-galileo.0g.ai");
  assert.ok(bundle.faucets.length >= 1);
  assert.match(bundle.exportPrivateKeyLine, /^export PRIVATE_KEY=0x[a-fA-F0-9]{64}$/);
});

