// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

struct Ip {
    uint8 a;
    uint8 b;
    uint8 c;
    uint8 d;
}

struct K8sCluster {
    uint64 id;
    bytes name;
    address owner;
    uint8 level;
    uint32 regionId;
    uint256 startBlock;
    uint256 stopBlock;
    uint256 terminalBlock;
    Ip ip;
    uint32 port;
    uint8 status;
    bytes32 p2pId;
}

struct SecretNode {
    uint64 id;
    bytes name;
    address owner;
    bytes32 validatorId;
    uint8 level;
    uint32 regionId;
    uint256 startBlock;
    uint256 stopBlock;
    uint256 terminalBlock;
    Ip ip;
    uint32 port;
    uint8 status;
    bytes32 p2pId;
    bytes bls;
}

struct EpochInfo {
    uint32 epoch;
    uint32 epochSlot;
    uint256 lastEpochBlock;
    uint32 now;
    address sideChainPub;
}

struct RunPrice {
    uint256 cpu;
    uint256 mem;
    uint256 storageAmount;
    uint256 network;
}

struct AssetInfo {
    string name;
    uint8 assetType;
    uint256 totalSupply;
}

struct AssetDeposit {
    uint256 amount;
    uint32 cpu;
    uint32 mem;
    uint32 cvmCpu;
    uint32 cvmMem;
    uint32 disk;
    uint32 gpu;
    uint8 deleted;
}
