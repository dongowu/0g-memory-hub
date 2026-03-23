const { expect } = require("chai");
const { anyValue } = require("@nomicfoundation/hardhat-chai-matchers/withArgs");
const { ethers } = require("hardhat");

describe("MemoryAnchor", function () {
  async function deployMemoryAnchor() {
    const factory = await ethers.getContractFactory("MemoryAnchor");
    const contract = await factory.deploy();
    await contract.waitForDeployment();
    return contract;
  }

  it("anchors a checkpoint and exposes it as the latest checkpoint", async function () {
    const [submitter] = await ethers.getSigners();
    const contract = await deployMemoryAnchor();

    const workflowId = ethers.id("wf-judge-01");
    const rootHash = ethers.id("root-1");
    const cidHash = ethers.id("cid-1");

    await expect(contract.anchorCheckpoint(workflowId, 1, rootHash, cidHash))
      .to.emit(contract, "CheckpointAnchored")
      .withArgs(workflowId, 1, rootHash, cidHash, submitter.address, anyValue);

    const latest = await contract.getLatestCheckpoint(workflowId);
    expect(latest.stepIndex).to.equal(1n);
    expect(latest.rootHash).to.equal(rootHash);
    expect(latest.cidHash).to.equal(cidHash);
    expect(latest.submitter).to.equal(submitter.address);
  });

  it("rejects non-monotonic checkpoint step indexes", async function () {
    const contract = await deployMemoryAnchor();

    const workflowId = ethers.id("wf-judge-02");
    const rootHash1 = ethers.id("root-1");
    const cidHash1 = ethers.id("cid-1");
    const rootHash2 = ethers.id("root-2");
    const cidHash2 = ethers.id("cid-2");

    await contract.anchorCheckpoint(workflowId, 5, rootHash1, cidHash1);

    await expect(contract.anchorCheckpoint(workflowId, 4, rootHash2, cidHash2)).to.be.revertedWith(
      "stepIndex must be monotonic",
    );
  });
});
