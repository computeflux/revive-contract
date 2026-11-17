// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title BaseBLS — Base 网络（OP-Stack）的 BLS 网关
/// @notice BLS12-381 阈值签名网关：submit(data, hmG1, negSig) 先校验
///         hmG1 == MAP_FP_TO_G1(keccak256(data))，再验证配对并执行 target.call(data)
/// @dev 与 EthBLS.sol 语义一致：0x10 在链上现算 hash→G1，把 data 与 hmG1 绑定，
///      因此伪造/替换 data 无法复用已发布的 (hmG1, negSig)；0x0f 只验证 negSig↔hmG1。
contract BaseBLS {
    address public target;

    bytes public aggPubkey;
    bytes public g2Generator;

    error MustCallByTarget();
    error PubkeyNotSet();
    error InvalidSignatureLength();
    error InvalidHmG1Length();
    error InvalidPubkeyLength();
    error InvalidGeneratorLength();
    error PairingCallFailed();
    error BLSSignatureInvalid();
    error CallFailed();
    error MapFpToG1Failed();
    error HmG1Mismatch();

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

    /// @notice 使用调用方提供的 hmG1（= MAP_FP_TO_G1(keccak256(data))，EIP-2537 格式）提交
    function submit(bytes calldata data, bytes calldata hmG1, bytes calldata negSig) external {
        _checkHmG1(hmG1, _mapFpToG1(keccak256(data)));
        _verifyBLS(negSig, hmG1);
        (bool callSuccess, ) = target.call(data);
        if (!callSuccess) revert CallFailed();
    }

    /// @notice 批量提交：hmG1 = MAP_FP_TO_G1(keccak256(abi.encode(datas)))
    function batchSubmit(bytes[] calldata datas, bytes calldata hmG1, bytes calldata negSig) external {
        _checkHmG1(hmG1, _mapFpToG1(keccak256(abi.encode(datas))));
        _verifyBLS(negSig, hmG1);
        for (uint256 i = 0; i < datas.length; i++) {
            (bool callSuccess, ) = target.call(datas[i]);
            if (!callSuccess) revert CallFailed();
        }
    }

    /// @dev EIP-2537 0x10 BLS12_MAP_FP_TO_G1：input 为 64 字节 Fp
    ///      （16 字节零 + 48 字节大端值），keccak256 结果左补零到 64 字节。
    function _mapFpToG1(bytes32 msgHash) internal view returns (bytes memory) {
        bytes memory input = new bytes(64);
        for (uint256 i = 0; i < 32; i++) {
            input[32 + i] = msgHash[i];
        }
        (bool success, bytes memory out) = address(0x10).staticcall(input);
        if (!success || out.length != 128) revert MapFpToG1Failed();
        return out;
    }

    /// @dev 链上强制 hmG1 == MAP_FP_TO_G1(keccak256(data))，把签名绑定到 data。
    function _checkHmG1(bytes calldata hmG1, bytes memory expected) internal pure {
        if (hmG1.length != 128 || expected.length != 128) revert InvalidHmG1Length();
        if (keccak256(hmG1) != keccak256(expected)) revert HmG1Mismatch();
    }

    function _verifyBLS(bytes calldata negSig, bytes calldata hmG1) internal view {
        if (aggPubkey.length == 0) revert PubkeyNotSet();
        if (negSig.length != 128) revert InvalidSignatureLength();
        if (hmG1.length != 128) revert InvalidHmG1Length();

        // EIP-2537 配对输入: hmG1(128) || aggPubkey(256) || negSig(128) || g2Generator(256) = 768
        bytes memory input = new bytes(768);
        _copy(input, 0, hmG1);
        _copy(input, 128, aggPubkey);
        _copy(input, 384, negSig);
        _copy(input, 512, g2Generator);

        (bool success, bytes memory result) = address(0x0f).staticcall(input);
        // EIP-2537: 无效点（被篡改的签名/公钥/哈希）使预编译直接报错，
        // 与配对结果为 0 同义 —— 都是签名无效。
        if (!success) revert BLSSignatureInvalid();
        if (result.length != 32 || uint256(bytes32(result)) != 1) revert BLSSignatureInvalid();
    }

    function _copy(bytes memory dst, uint256 offset, bytes memory src) internal pure {
        for (uint256 i = 0; i < src.length; i++) {
            dst[offset + i] = src[i];
        }
    }
}
