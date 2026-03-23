// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title MemoryAnchor
 * @dev Workflow-centric checkpoint anchoring contract for AI runtime traces.
 *
 * Each workflow can publish checkpoints identified by:
 * - workflowId
 * - stepIndex
 * - rootHash
 * - cidHash (hash of storage CID / content identifier)
 */
contract MemoryAnchor {
    struct Checkpoint {
        uint64 stepIndex;
        bytes32 rootHash;
        bytes32 cidHash;
        uint64 timestamp;
        address submitter;
    }

    // workflowId => latest checkpoint
    mapping(bytes32 => Checkpoint) private latestCheckpoint;

    // workflowId => full checkpoint history
    mapping(bytes32 => Checkpoint[]) private checkpointHistory;

    event CheckpointAnchored(
        bytes32 indexed workflowId,
        uint64 indexed stepIndex,
        bytes32 indexed rootHash,
        bytes32 cidHash,
        address submitter,
        uint64 timestamp
    );

    /**
     * @dev Anchor a workflow checkpoint.
     */
    function anchorCheckpoint(
        bytes32 workflowId,
        uint64 stepIndex,
        bytes32 rootHash,
        bytes32 cidHash
    ) external {
        require(workflowId != bytes32(0), "workflowId cannot be zero");
        require(rootHash != bytes32(0), "rootHash cannot be zero");
        require(cidHash != bytes32(0), "cidHash cannot be zero");

        Checkpoint memory current = latestCheckpoint[workflowId];
        if (current.timestamp != 0) {
            require(stepIndex >= current.stepIndex, "stepIndex must be monotonic");
        }

        uint64 ts = uint64(block.timestamp);
        Checkpoint memory next = Checkpoint({
            stepIndex: stepIndex,
            rootHash: rootHash,
            cidHash: cidHash,
            timestamp: ts,
            submitter: msg.sender
        });

        latestCheckpoint[workflowId] = next;
        checkpointHistory[workflowId].push(next);

        emit CheckpointAnchored(
            workflowId,
            stepIndex,
            rootHash,
            cidHash,
            msg.sender,
            ts
        );
    }

    /**
     * @dev Read the latest checkpoint for a workflow.
     */
    function getLatestCheckpoint(
        bytes32 workflowId
    )
        external
        view
        returns (
            uint64 stepIndex,
            bytes32 rootHash,
            bytes32 cidHash,
            uint64 timestamp,
            address submitter
        )
    {
        Checkpoint memory cp = latestCheckpoint[workflowId];
        return (cp.stepIndex, cp.rootHash, cp.cidHash, cp.timestamp, cp.submitter);
    }

    /**
     * @dev Number of checkpoints for a workflow.
     */
    function getCheckpointCount(bytes32 workflowId) external view returns (uint256) {
        return checkpointHistory[workflowId].length;
    }

    /**
     * @dev Read checkpoint at index from workflow history.
     */
    function getCheckpointAt(
        bytes32 workflowId,
        uint256 index
    )
        external
        view
        returns (
            uint64 stepIndex,
            bytes32 rootHash,
            bytes32 cidHash,
            uint64 timestamp,
            address submitter
        )
    {
        require(index < checkpointHistory[workflowId].length, "index out of bounds");
        Checkpoint memory cp = checkpointHistory[workflowId][index];
        return (cp.stepIndex, cp.rootHash, cp.cidHash, cp.timestamp, cp.submitter);
    }
}
