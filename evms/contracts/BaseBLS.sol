// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/// @title BaseBLS — Base 网络（OP-Stack）的 BLS 网关
/// @notice BLS12-381 阈值签名网关：submit(data, hmG1, negSig) 验证配对后执行 target.call(data)
/// @dev hmG1 = MAP_FP_TO_G1(keccak256(data))，由 tee-node（ext/blscontract，生产 DKG
///      签名路径 BLSSignEIP2537）在链下用真实 MapFpToG1 算好并随签名一同提交；
///      合约只做 EIP-2537 0x0f 配对验证 + 目标执行。
/// @dev 与 EthBLS.sol（ETH 网络版，合约内调 0x10 现算 hmG1）的验证语义等价：
///      伪造 data 无法复用已发布签名（hmG1' = map(keccak(data')) ≠ 签名点，配对失败），
///      hmG1↔data 绑定由 Go 侧签名入口强制，链上验证 negSig↔hmG1。
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

    /// @notice 使用调用方提供的 hmG1（= MAP_FP_TO_G1(keccak256(data))，EIP-2537 格式）提交
    function submit(bytes calldata data, bytes calldata hmG1, bytes calldata negSig) external {
        _verifyBLS(negSig, hmG1);
        (bool callSuccess, ) = target.call(data);
        if (!callSuccess) revert CallFailed();
    }

    /// @notice 批量提交：hmG1 = MAP_FP_TO_G1(keccak256(abi.encode(datas)))
    function batchSubmit(bytes[] calldata datas, bytes calldata hmG1, bytes calldata negSig) external {
        _verifyBLS(negSig, hmG1);
        for (uint256 i = 0; i < datas.length; i++) {
            (bool callSuccess, ) = target.call(datas[i]);
            if (!callSuccess) revert CallFailed();
        }
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
