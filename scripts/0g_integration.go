// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title IZGS
 * @dev Interface for 0G Storage Flow contract
 * This interface defines the functions needed to interact with the 0G Storage system
 */
interface IZGS {
    /**
     * @dev Log a data entry on-chain
     * @param prevRoot Previous merkle root
     * @param data Data to be logged
     */
    function log(bytes32 prevRoot, bytes calldata data) external;

    /**
     * @dev Get the current merkle root
     * @return Current merkle root
     */
    function getRoot() external view returns (bytes32);
}

/**
 * @title Flow
 * @dev Main contract for 0G Storage integration
 * Stores the flow configuration and provides access to the ZGS interface
 */
contract Flow {
    // ZGS contract address
    address public zgsAddress;

    // Merkle tree configuration
    uint256 public constant LEAF_SIZE = 32;
    uint256 public constant TREE_DEPTH = 26;

    // Events
    event DataLogged(bytes32 indexed root, bytes32 indexed prevRoot, uint256 timestamp);
    event ZGSUpdated(address indexed newAddress);

    constructor(address _zgsAddress) {
        zgsAddress = _zgsAddress;
    }

    /**
     * @dev Update the ZGS contract address
     * @param _newAddress New ZGS contract address
     */
    function updateZGS(address _newAddress) external {
        require(_newAddress != address(0), "Invalid address");
        zgsAddress = _newAddress;
        emit ZGSUpdated(_newAddress);
    }

    /**
     * @dev Log data to the ZGS contract
     * @param prevRoot Previous merkle root
     * @param data Data to be logged
     */
    function logToZGS(bytes32 prevRoot, bytes calldata data) external {
        require(zgsAddress != address(0), "ZGS not set");
        IZGS(zgsAddress).log(prevRoot, data);
        emit DataLogged(IZGS(zgsAddress).getRoot(), prevRoot, block.timestamp);
    }
}
