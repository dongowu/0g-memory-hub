// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title MemoryChain
 * @dev Stores the latest memory CID pointer for each AI Agent on 0G Chain
 * Implements a simple on-chain anchor for agent memory heads
 */
contract MemoryChain {
    // Agent address => Latest memory CID (bytes32)
    mapping(address => bytes32) public memoryHeads;

    // Agent address => Memory history (array of CIDs)
    mapping(address => bytes32[]) public memoryHistory;

    // Events
    event MemoryUpdated(address indexed agent, bytes32 indexed cid, uint256 timestamp);
    event MemoryHistoryRecorded(address indexed agent, uint256 historyLength);

    /**
     * @dev Update the memory head pointer for the caller
     * @param cid The content identifier (CID) of the latest memory
     */
    function setMemoryHead(bytes32 cid) external {
        require(cid != bytes32(0), "CID cannot be zero");

        memoryHeads[msg.sender] = cid;
        memoryHistory[msg.sender].push(cid);

        emit MemoryUpdated(msg.sender, cid, block.timestamp);
        emit MemoryHistoryRecorded(msg.sender, memoryHistory[msg.sender].length);
    }

    /**
     * @dev Get the current memory head for an agent
     * @param agent The agent address
     * @return The latest memory CID
     */
    function getMemoryHead(address agent) external view returns (bytes32) {
        return memoryHeads[agent];
    }

    /**
     * @dev Get the full memory history for an agent
     * @param agent The agent address
     * @return Array of all memory CIDs in chronological order
     */
    function getMemoryHistory(address agent) external view returns (bytes32[] memory) {
        return memoryHistory[agent];
    }

    /**
     * @dev Get the memory history length for an agent
     * @param agent The agent address
     * @return The number of memory updates
     */
    function getMemoryHistoryLength(address agent) external view returns (uint256) {
        return memoryHistory[agent].length;
    }

    /**
     * @dev Get a specific memory CID from history by index
     * @param agent The agent address
     * @param index The index in the history array
     * @return The memory CID at that index
     */
    function getMemoryAt(address agent, uint256 index) external view returns (bytes32) {
        require(index < memoryHistory[agent].length, "Index out of bounds");
        return memoryHistory[agent][index];
    }
}
