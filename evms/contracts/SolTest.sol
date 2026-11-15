// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title SolTest — 在 EVM 内断言 EIP-2537 预编译行为（tx-mode）。
/// 覆盖 BLS 网关的全部验证语义：配对结果、无效点。
contract SolTest {
    // 0x0f pairing：good 输入 => (true, 32, 1)；任何篡改 => (false, 0, 0)
    function assertPair(bytes calldata input, bool expectValid) external {
        (bool ok, bytes memory result) = address(0x0f).call(input);
        if (expectValid) {
            require(ok && result.length == 32 && uint256(bytes32(result)) == 1, "pair should be valid");
        } else {
            require(!ok || result.length != 32 || uint256(bytes32(result)) != 1, "pair should be invalid");
        }
    }
}
