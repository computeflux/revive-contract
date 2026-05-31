// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title BLS Gateway
/// @notice 通过 BLS12-381 阈值签名验证提交交易到目标合约
/// @dev 使用 EIP-2537 预编译进行 BLS 配对验证
/// @dev 权限由目标合约（Subnet）代理，BLS 本身不存储 govContract
contract BLS {
    address public target;

    // 聚合公钥（G2 point，256 bytes EIP-2537 格式）
    bytes public aggPubkey;
    // G2 generator（256 bytes EIP-2537 格式）
    bytes public g2Generator;

    error MustCallByTarget();
    error PubkeyNotSet();
    error InvalidSignatureLength();
    error InvalidHmG1Length();
    error InvalidPubkeyLength();
    error InvalidGeneratorLength();
    error PairingCallFailed();
    error BLSSignatureInvalid();
    error MapFpToG1Failed();
    error CallFailed();

    modifier onlyTarget() {
        if (msg.sender != target) revert MustCallByTarget();
        _;
    }

    constructor(address _target, bytes memory _g2Generator) {
        target = _target;
        if (_g2Generator.length != 256) revert InvalidGeneratorLength();
        g2Generator = _g2Generator;
    }

    function setAggPubkey(bytes calldata _aggPubkey) external onlyTarget {
        if (_aggPubkey.length != 256) revert InvalidPubkeyLength();
        aggPubkey = _aggPubkey;
    }

    function setTarget(address _target) external onlyTarget {
        target = _target;
    }

    /// @notice 通过 BLS 签名提交单个交易到目标合约
    /// @param data 目标合约调用数据
    /// @param negSig 取反后的 BLS G1 签名（128 bytes EIP-2537 格式）
    function submit(bytes calldata data, bytes calldata negSig) external {
        bytes32 msgHash = keccak256(data);
        bytes memory hmG1 = _mapFpToG1(msgHash);
        _verifyAndExecute(data, negSig, hmG1);
    }

    /// @notice 通过 BLS 签名批量提交多个交易到目标合约
    /// @param datas 目标合约调用数据数组
    /// @param negSig 取反后的 BLS G1 签名（128 bytes EIP-2537 格式）
    function batchSubmit(bytes[] calldata datas, bytes calldata negSig) external {
        bytes32 msgHash = keccak256(abi.encode(datas));
        bytes memory hmG1 = _mapFpToG1(msgHash);
        // 验证一次签名
        _verifyBLS(negSig, hmG1);

        // 逐个执行目标调用
        for (uint256 i = 0; i < datas.length; i++) {
            (bool callSuccess, ) = target.call(datas[i]);
            if (!callSuccess) revert CallFailed();
        }
    }

    function _verifyAndExecute(bytes calldata data, bytes calldata negSig, bytes memory hmG1) internal {
        _verifyBLS(negSig, hmG1);

        // 验证通过，执行目标调用
        (bool callSuccess, ) = target.call(data);
        if (!callSuccess) revert CallFailed();
    }

    function _verifyBLS(bytes calldata negSig, bytes memory hmG1) internal view {
        if (aggPubkey.length == 0) revert PubkeyNotSet();
        if (negSig.length != 128) revert InvalidSignatureLength();
        if (hmG1.length != 128) revert InvalidHmG1Length();

        // 构造 EIP-2537 配对输入: hmG1(128) || aggPubkey(256) || negSig(128) || g2Generator(256) = 768 bytes
        bytes memory input = new bytes(768);
        _copy(input, 0, hmG1);
        _copy(input, 128, aggPubkey);
        _copy(input, 384, negSig);
        _copy(input, 512, g2Generator);

        // 调用 BLS12_PAIRING 预编译 (0x0f)
        (bool success, bytes memory result) = address(0x0f).staticcall(input);
        if (!success) revert PairingCallFailed();
        if (result.length != 32 || uint256(bytes32(result)) != 1) revert BLSSignatureInvalid();
    }

    /// @dev 使用 EIP-2537 BLS12_MAP_FP_TO_G1 (0x10) 预编译将 keccak256 哈希映射到 G1
    function _mapFpToG1(bytes32 input) internal view returns (bytes memory) {
        bytes memory fpInput = new bytes(64);
        for (uint i = 0; i < 32; i++) {
            fpInput[16 + i] = input[i];
        }
        (bool success, bytes memory result) = address(0x10).staticcall(fpInput);
        if (!success || result.length != 128) revert MapFpToG1Failed();
        return result;
    }

    function _copy(bytes memory dst, uint256 offset, bytes memory src) internal pure {
        for (uint256 i = 0; i < src.length; i++) {
            dst[offset + i] = src[i];
        }
    }
}
