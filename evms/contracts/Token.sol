// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/math/Math.sol";

/**
 * @title Token
 * @notice 积分充值/提现合约（UUPS 可升级），与 revives/Token（PolkaVM/wrevive 版）功能对齐。
 * @dev 积分规则 | Point rules:
 *      points = value * RATE / TOKEN_UNIT   (原生币最小单位 → 积分)
 *      eth    = points * TOKEN_UNIT / RATE  (积分 → 原生币最小单位)
 *      例: ETH=$4, RATE=4000, TOKEN_UNIT=10^18 → 1 ETH = 4000 积分, 1000 积分 ≈ $1
 * @dev 积分余额由 TEE 子链管理，合约只发事件（与 rust 版一致）。
 *      Subnet 合约作为管理员（BLS 网关代理）。提现必须由 Subnet 通过 BaseBLS.submit(data)
 *      调用（withdraw / withdrawErc20）。
 */
contract Token is UUPSUpgradeable, OwnableUpgradeable {
    using Math for uint256;

    address public subnet;

    // 兑换率: 1 原生币 = RATE 积分 | points per 1 native token
    uint256 public rate;
    // 主链币最小单位 (ETH=10^18 Wei, DOT=10^15 Planck)
    uint256 public tokenUnit;

    // ERC20 代币注册表: token => (active, rate, unit)
    struct ERC20TokenConfig {
        bool active;
        uint256 rate;
        uint256 unit;
    }
    mapping(address => ERC20TokenConfig) public erc20Tokens;
    address[] public erc20TokenList;

    // Native token 充值开关 (默认开启)
    bool public nativeActive = true;

    // 协议费率 (基点, 10000 = 100%, 默认 500 = 5%)
    uint256 public rechargeFeeRate = 500;
    uint256 public withdrawFeeRate = 500;

    // 提现状态常量
    uint8 public constant WITHDRAWAL_STATUS_PENDING = 0;
    uint8 public constant WITHDRAWAL_STATUS_DONE = 1;
    uint8 public constant WITHDRAWAL_STATUS_CANCELLED = 2;

    // ERC20 充值最低积分（与 rust 版一致）
    uint256 public constant ERC20_MIN_POINTS = 1_999_000;

    // 待提现记录: nonce => PendingWithdrawal
    struct PendingWithdrawal {
        address user;
        address token; // address(0)=原生币, 其他=ERC20
        uint256 amount; // 净额（已扣提现费）
        uint8 status;
    }
    mapping(uint64 => PendingWithdrawal) public pendingWithdrawals;

    /// @notice Relay 事件记录（与 revives/Token 的 EventRecord 一致）
    /// @dev eventData 为 SCALE 编码，字节与 PolkaVM 版 emit_event 完全一致，relay 可直接复用同一解码器
    struct EventRecord {
        bytes targetContract; // 固定 "gateway"
        bytes eventType; // "Recharge" / "WithdrawalDone" / "WithdrawalCancelled"
        bytes[] eventData;
    }

    // ========== Relay 事件存储 ==========
    // 注意：新增状态变量必须追加在原有变量之后，保持 UUPS 存储布局兼容
    // 全局事件 nonce（从 1 起单调递增，relay 用 (chainID, nonce) 防重放）
    uint64 private _eventNonce;
    // nonce => EventRecord
    mapping(uint64 => EventRecord) private _relayEvents;

    // ========== 事件 ==========

    event Recharged(address indexed user, uint256 amount, uint256 points, uint256 timestamp);
    event Withdrawn(uint64 indexed nonce, address indexed user, address token, uint256 amount, uint8 status);
    event WithdrawalDone(uint64 indexed nonce);
    event WithdrawalCancelled(uint64 indexed nonce);

    event RateUpdated(uint256 oldRate, uint256 newRate);
    event TokenUnitUpdated(uint256 oldUnit, uint256 newUnit);
    event SubnetSet(address indexed oldSubnet, address indexed newSubnet);
    event NativeActiveSet(bool active);
    event RechargeFeeRateSet(uint256 rate);
    event WithdrawFeeRateSet(uint256 rate);
    event ERC20TokenSet(address indexed token, bool active, uint256 rate, uint256 unit);

    // ========== 初始化 ==========

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    /**
     * @notice 代理初始化（替代 constructor）
     * @param _owner 合约管理员地址（address(0) 则使用 msg.sender）
     */
    function initialize(address _owner) public initializer {
        __Ownable_init();
        __UUPSUpgradeable_init();
        rate = 4000; // $4/ETH → $1 = 1000pt
        tokenUnit = 1 ether;
        if (_owner != address(0)) {
            _transferOwnership(_owner);
        }
    }

    /// @dev UUPS 升级授权：仅 owner 可升级
    function _authorizeUpgrade(address) internal override onlyOwner {}

    // ========== 配置 ==========

    function setSubnet(address _subnet) external onlyOwner {
        require(_subnet != address(0), "zero address");
        address old = subnet;
        subnet = _subnet;
        emit SubnetSet(old, _subnet);
    }

    function setRate(uint256 newRate) external onlyOwner {
        require(newRate > 0, "rate must be > 0");
        uint256 oldRate = rate;
        rate = newRate;
        emit RateUpdated(oldRate, newRate);
    }

    function setTokenUnit(uint256 unit) external onlyOwner {
        require(unit > 0, "unit must be > 0");
        uint256 oldUnit = tokenUnit;
        tokenUnit = unit;
        emit TokenUnitUpdated(oldUnit, unit);
    }

    /// @notice 启用/禁用 Native token 充值
    function setNativeActive(bool active) external onlyOwner {
        nativeActive = active;
        emit NativeActiveSet(active);
    }

    /// @notice 设置充值协议费率 (基点, 10000 = 100%)，仅 SUBNET 可设置
    function setRechargeFeeRate(uint256 newRate) external onlySubnet {
        require(newRate < 10_000, "fee rate too high");
        rechargeFeeRate = newRate;
        emit RechargeFeeRateSet(newRate);
    }

    /// @notice 设置提现协议费率 (基点, 10000 = 100%)，仅 SUBNET 可设置
    function setWithdrawFeeRate(uint256 newRate) external onlySubnet {
        require(newRate < 10_000, "fee rate too high");
        withdrawFeeRate = newRate;
        emit WithdrawFeeRateSet(newRate);
    }

    // ========== ERC20 管理 ==========

    /// @notice 注册/更新 ERC20 代币
    function setERC20Token(address token, bool active, uint256 tokenRate, uint256 unit) external onlyOwner {
        require(token != address(0), "zero address");
        require(unit > 0, "unit must be > 0");
        // 未注册过的代币（unit == 0）加入列表
        if (erc20Tokens[token].unit == 0) {
            erc20TokenList.push(token);
        }
        erc20Tokens[token] = ERC20TokenConfig(active, tokenRate, unit);
        emit ERC20TokenSet(token, active, tokenRate, unit);
    }

    // ========== 充值 / 提现 ==========

    /// @notice 原生币充值（自动扣协议费转 SUBNET，积分按净额计算）
    function recharge() external payable returns (uint256) {
        return _rechargeImpl();
    }

    /// @notice 提现（由 Subnet 通过 BLS 网关调用）。
    ///         nonce 幂等：重复提交直接返回。余额充足直接转账，不足记录待提现由用户 claim。
    function withdraw(address user, uint256 points, uint64 nonce) external onlySubnet {
        _withdrawImpl(user, points, nonce);
    }

    /// @notice ERC20 充值（用户需先 approve 本合约）。自动扣协议费转 SUBNET。
    function rechargeErc20(address token, uint256 amount) external returns (uint256) {
        return _rechargeErc20Impl(token, amount);
    }

    /// @notice ERC20 提现（由 Subnet 通过 BLS 网关调用）
    function withdrawErc20(address token, address user, uint256 points, uint64 nonce) external onlySubnet {
        _withdrawErc20Impl(token, user, points, nonce);
    }

    // ========== 用户提现操作 ==========

    /// @notice 用户领取待提现资金（按 nonce）
    function claimWithdrawal(uint64 nonce) external {
        PendingWithdrawal storage pendingWd = pendingWithdrawals[nonce];
        require(pendingWd.user == msg.sender, "not your withdrawal");
        require(pendingWd.status == WITHDRAWAL_STATUS_PENDING, "not pending");

        if (pendingWd.token == address(0)) {
            require(address(this).balance >= pendingWd.amount, "insufficient balance");
            (bool ok, ) = payable(msg.sender).call{value: pendingWd.amount}("");
            require(ok, "transfer failed");
        } else {
            require(
                IERC20(pendingWd.token).transfer(msg.sender, pendingWd.amount),
                "erc20 transfer failed"
            );
        }

        pendingWd.status = WITHDRAWAL_STATUS_DONE;
        emit WithdrawalDone(nonce);
        _emitRelayEvent("WithdrawalDone", _singleNonceData(nonce));
    }

    /// @notice 用户取消待提现（按 nonce）
    function cancelWithdrawal(uint64 nonce) external {
        PendingWithdrawal storage pendingWd = pendingWithdrawals[nonce];
        require(pendingWd.user == msg.sender, "not your withdrawal");
        require(pendingWd.status == WITHDRAWAL_STATUS_PENDING, "not pending");

        pendingWd.status = WITHDRAWAL_STATUS_CANCELLED;
        emit WithdrawalCancelled(nonce);
        _emitRelayEvent("WithdrawalCancelled", _singleNonceData(nonce));
    }

    // ========== 管理 ==========

    function emergencyWithdraw() external onlyOwner {
        (bool ok, ) = payable(owner()).call{value: address(this).balance}("");
        require(ok, "transfer failed");
    }

    /// @notice 提取合约持有的 ERC20（仅 owner）
    function emergencyWithdrawErc20(address token) external onlyOwner {
        uint256 bal = IERC20(token).balanceOf(address(this));
        require(
            IERC20(token).transfer(owner(), bal),
            "erc20 transfer failed"
        );
    }

    // ========== 查询 ==========

    /// @notice 原生币最小单位 → 积分换算
    function toPoints(uint256 amount) external view returns (uint256) {
        return _valueToPoints(amount);
    }

    /// @notice 积分 → 原生币最小单位换算
    function toValue(uint256 points) external view returns (uint256) {
        return _pointsToValue(points);
    }

    function getERC20Config(address token) external view returns (bool, uint256, uint256) {
        ERC20TokenConfig memory c = erc20Tokens[token];
        return (c.active, c.rate, c.unit);
    }

    function getERC20Count() external view returns (uint256) {
        return erc20TokenList.length;
    }

    function getERC20List() external view returns (address[] memory tokens, bool[] memory actives, uint256[] memory rates, uint256[] memory units) {
        uint256 n = erc20TokenList.length;
        tokens = new address[](n);
        actives = new bool[](n);
        rates = new uint256[](n);
        units = new uint256[](n);
        for (uint256 i = 0; i < n; i++) {
            address token = erc20TokenList[i];
            ERC20TokenConfig memory c = erc20Tokens[token];
            tokens[i] = token;
            actives[i] = c.active;
            rates[i] = c.rate;
            units[i] = c.unit;
        }
    }

    function getPendingWithdrawal(uint64 nonce) external view returns (address user, address token, uint256 amount, uint8 status) {
        PendingWithdrawal memory p = pendingWithdrawals[nonce];
        return (p.user, p.token, p.amount, p.status);
    }

    // ========== Relay 查询 ==========

    /// @notice 当前最新事件 nonce（0 表示尚无事件）
    function getLatestNonce() external view returns (uint64) {
        return _eventNonce;
    }

    /// @notice 批量查询事件，to == 0 表示查询到最新
    /// @dev 跳过不存在的 nonce，语义与 PolkaVM 版 get_events 一致
    function getEvents(uint64 from, uint64 to) external view returns (EventRecord[] memory) {
        uint64 end = to == 0 ? _eventNonce : to;
        if (end > _eventNonce) {
            end = _eventNonce;
        }
        if (_eventNonce == 0 || from > end) {
            return new EventRecord[](0);
        }

        uint64 span = end - from + 1;
        uint64 count = 0;
        for (uint64 i = 0; i < span; i++) {
            if (_relayEvents[from + i].targetContract.length != 0) {
                count++;
            }
        }

        EventRecord[] memory out = new EventRecord[](count);
        uint64 j = 0;
        for (uint64 i = 0; i < span; i++) {
            uint64 n = from + i;
            if (_relayEvents[n].targetContract.length != 0) {
                out[j] = _relayEvents[n];
                j++;
            }
        }
        return out;
    }

    /// @notice 按 nonce 查询单个事件；不存在时返回空记录（targetContract.length == 0）
    function getEvent(uint64 nonce) external view returns (EventRecord memory) {
        return _relayEvents[nonce];
    }

    // ========== 默认充值（fallback） ==========

    receive() external payable {
        require(nativeActive, "native disabled");
        require(msg.value > 0, "amount must be > 0");
        _rechargeImpl();
    }

    // ========== 内部实现 ==========

    function _rechargeImpl() internal returns (uint256) {
        require(nativeActive, "native disabled");
        uint256 value = msg.value;
        require(value > 0, "amount must be > 0");

        // 收取协议费用，转到 SUBNET 地址
        uint256 fee = value * rechargeFeeRate / 10_000;
        uint256 netValue = value - fee;
        if (fee > 0) {
            require(subnet != address(0), "subnet not set");
            (bool ok, ) = payable(subnet).call{value: fee}("");
            require(ok, "fee transfer failed");
        }

        uint256 points = _valueToPoints(netValue);
        require(points > 0, "points must be > 0");

        // 积分由 TEE 子链管理，合约只发事件
        emit Recharged(msg.sender, netValue, points, block.timestamp);

        // relay 事件：Recharge(UniAddr{2, caller}, points)
        bytes[] memory relayData = new bytes[](2);
        relayData[0] = _scaleUniAddr(msg.sender);
        relayData[1] = _scaleU256(points);
        _emitRelayEvent("Recharge", relayData);
        return points;
    }

    function _withdrawImpl(address user, uint256 points, uint64 nonce) internal {
        require(points > 0, "points must be > 0");

        // nonce 幂等：侧链重复提交直接返回
        if (pendingWithdrawals[nonce].user != address(0)) {
            return;
        }

        uint256 nativeAmount = _pointsToValue(points);
        require(nativeAmount > 0, "amount must be > 0");

        // 收取协议费用，转到 SUBNET 地址，用户收到净额
        uint256 fee = nativeAmount * withdrawFeeRate / 10_000;
        uint256 netAmount = nativeAmount - fee;

        // 余额充足则直接转账，无需用户后续 claim
        if (address(this).balance >= nativeAmount) {
            if (fee > 0) {
                require(subnet != address(0), "subnet not set");
                (bool ok, ) = payable(subnet).call{value: fee}("");
                require(ok, "fee transfer failed");
            }
            (bool ok2, ) = payable(user).call{value: netAmount}("");
            require(ok2, "transfer failed");

            pendingWithdrawals[nonce] = PendingWithdrawal({
                user: user,
                token: address(0),
                amount: netAmount,
                status: WITHDRAWAL_STATUS_DONE
            });
            emit Withdrawn(nonce, user, address(0), netAmount, WITHDRAWAL_STATUS_DONE);
            emit WithdrawalDone(nonce);
            _emitRelayEvent("WithdrawalDone", _singleNonceData(nonce));
            return;
        }

        // 余额不足，记录待提现，用户后续自行 claim
        pendingWithdrawals[nonce] = PendingWithdrawal({
            user: user,
            token: address(0),
            amount: netAmount,
            status: WITHDRAWAL_STATUS_PENDING
        });
        emit Withdrawn(nonce, user, address(0), netAmount, WITHDRAWAL_STATUS_PENDING);
    }

    function _rechargeErc20Impl(address token, uint256 amount) internal returns (uint256) {
        ERC20TokenConfig memory config = erc20Tokens[token];
        require(config.unit > 0, "erc20 not supported");
        require(config.active, "erc20 inactive");
        require(amount > 0, "amount must be > 0");

        // transferFrom(caller, self, amount)
        require(IERC20(token).transferFrom(msg.sender, address(this), amount), "erc20 transfer failed");

        // 收取协议费用，转到 SUBNET 地址
        uint256 fee = amount * rechargeFeeRate / 10_000;
        uint256 netAmount = amount - fee;
        if (fee > 0) {
            require(subnet != address(0), "subnet not set");
            require(IERC20(token).transfer(subnet, fee), "erc20 fee transfer failed");
        }

        // 换算积分: points = net_amount * rate / unit
        uint256 points = netAmount * config.rate / config.unit;
        require(points >= ERC20_MIN_POINTS, "points below minimum");

        // 积分由 TEE 子链管理，合约只发事件
        emit Recharged(msg.sender, netAmount, points, block.timestamp);

        // relay 事件：Recharge(UniAddr{2, caller}, points)
        bytes[] memory relayData = new bytes[](2);
        relayData[0] = _scaleUniAddr(msg.sender);
        relayData[1] = _scaleU256(points);
        _emitRelayEvent("Recharge", relayData);
        return points;
    }

    function _withdrawErc20Impl(address token, address user, uint256 points, uint64 nonce) internal {
        ERC20TokenConfig memory config = erc20Tokens[token];
        require(config.unit > 0, "erc20 not supported");
        require(config.active, "erc20 inactive");
        require(points > 0, "points must be > 0");

        // nonce 幂等
        if (pendingWithdrawals[nonce].user != address(0)) {
            return;
        }

        // 换算 ERC20 数量: amount = points * unit / rate
        uint256 amount = points * config.unit / config.rate;
        require(amount > 0, "amount must be > 0");

        uint256 fee = amount * withdrawFeeRate / 10_000;
        uint256 netAmount = amount - fee;

        uint256 contractBalance = IERC20(token).balanceOf(address(this));

        // 余额充足则直接转账，无需用户后续 claim
        if (contractBalance >= amount) {
            if (fee > 0) {
                require(subnet != address(0), "subnet not set");
                require(IERC20(token).transfer(subnet, fee), "erc20 fee transfer failed");
            }
            require(IERC20(token).transfer(user, netAmount), "erc20 transfer failed");

            pendingWithdrawals[nonce] = PendingWithdrawal({
                user: user,
                token: token,
                amount: netAmount,
                status: WITHDRAWAL_STATUS_DONE
            });
            emit Withdrawn(nonce, user, token, netAmount, WITHDRAWAL_STATUS_DONE);
            emit WithdrawalDone(nonce);
            _emitRelayEvent("WithdrawalDone", _singleNonceData(nonce));
            return;
        }

        // 余额不足，记录待提现，用户后续自行 claim
        pendingWithdrawals[nonce] = PendingWithdrawal({
            user: user,
            token: token,
            amount: netAmount,
            status: WITHDRAWAL_STATUS_PENDING
        });
        emit Withdrawn(nonce, user, token, netAmount, WITHDRAWAL_STATUS_PENDING);
    }

    // ========== Relay 事件写入 ==========

    /// @dev 记录一条 relay 事件。nonce 从 1 起单调递增，与 PolkaVM 版 emit_event 一致。
    ///      写入点与 relay 关注的链上日志（Recharged / WithdrawalDone / WithdrawalCancelled）一一对应，
    ///      因此查询得到的 nonce 与按日志顺序统计的 nonce 相同。
    function _emitRelayEvent(bytes memory eventType, bytes[] memory eventData) internal {
        uint64 nonce = _eventNonce + 1;
        EventRecord storage rec = _relayEvents[nonce];
        rec.targetContract = bytes("gateway");
        rec.eventType = eventType;
        for (uint256 i = 0; i < eventData.length; i++) {
            rec.eventData.push(eventData[i]);
        }
        _eventNonce = nonce;
    }

    /// @dev SCALE 编码 UniAddr{t = 2, v = 20 字节地址}
    ///      u32 小端 (0x02000000) + compact(20) = 0x50 + 20 字节地址
    function _scaleUniAddr(address addr) internal pure returns (bytes memory) {
        return abi.encodePacked(bytes4(0x02000000), bytes1(0x50), addr);
    }

    /// @dev SCALE 编码 u256：32 字节小端
    function _scaleU256(uint256 v) internal pure returns (bytes memory out) {
        out = new bytes(32);
        for (uint256 i = 0; i < 32; i++) {
            out[i] = bytes1(uint8(v >> (8 * i)));
        }
    }

    /// @dev SCALE 编码 u64：8 字节小端
    function _scaleU64(uint64 v) internal pure returns (bytes memory out) {
        out = new bytes(8);
        for (uint256 i = 0; i < 8; i++) {
            out[i] = bytes1(uint8(v >> (8 * i)));
        }
    }

    /// @dev 仅含 nonce 的 eventData（WithdrawalDone / WithdrawalCancelled）
    function _singleNonceData(uint64 nonce) internal pure returns (bytes[] memory data) {
        data = new bytes[](1);
        data[0] = _scaleU64(nonce);
    }

    // ========== 积分换算 ==========

    /// 原生币最小单位 → 积分: value * RATE / TOKEN_UNIT
    function _valueToPoints(uint256 value) internal view returns (uint256) {
        return value * rate / tokenUnit;
    }

    /// 积分 → 原生币最小单位: points * TOKEN_UNIT / RATE
    function _pointsToValue(uint256 points) internal view returns (uint256) {
        return points * tokenUnit / rate;
    }

    // ========== 修饰器 ==========

    modifier onlySubnet() {
        require(msg.sender == subnet, "only subnet contract");
        _;
    }
}
