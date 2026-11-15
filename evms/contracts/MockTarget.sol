// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title MockTarget — test target for BLS gateway
/// Accepts calls only from the BLS contract; records last call data.
contract MockTarget {
    bytes public lastData;
    uint256 public lastValue;
    uint256 public callCount;

    event Executed(bytes data, uint256 value);

    function execute(bytes calldata data, uint256 value) external {
        lastData = data;
        lastValue = value;
        callCount++;
        emit Executed(data, value);
    }

    function alwaysFails() external pure {
        revert("boom");
    }
}
