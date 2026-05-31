// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

/**
 * @title Token
 * @notice 积分充值/提现合约（UUPS 可升级）
 * @dev Subnet 合约作为管理员。用户充值直接调用，提现必须由 Subnet 合约调用。
 */
contract Token is UUPSUpgradeable, OwnableUpgradeable {
    address public subnet;
    uint256 public rate;
    mapping(address => uint256) public balances;

    // 预留存储空间（可升级合约兼容）
    uint256[50] private __gap;

    // ========== 事件 ==========

    event Recharged(
        address indexed user,
        uint256 ethAmount,
        uint256 pointsAmount,
        uint256 timestamp
    );

    event Withdrawn(
        address indexed user,
        uint256 ethAmount,
        uint256 pointsAmount,
        uint256 timestamp
    );

    event RateUpdated(uint256 oldRate, uint256 newRate);
    event SubnetSet(address indexed oldSubnet, address indexed newSubnet);

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
        rate = 1000;
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

    // ========== 充值/提现 ==========

    function recharge() external payable returns (uint256) {
        require(msg.value > 0, "amount must be > 0");
        uint256 pointsAmount = msg.value * rate;
        balances[msg.sender] += msg.value;
        emit Recharged(msg.sender, msg.value, pointsAmount, block.timestamp);
        return pointsAmount;
    }

    function withdraw(address user, uint256 ethAmount) external onlySubnet {
        require(ethAmount > 0, "amount must be > 0");
        require(balances[user] >= ethAmount, "insufficient balance");
        balances[user] -= ethAmount;
        uint256 pointsAmount = ethAmount * rate;
        (bool ok, ) = payable(user).call{value: ethAmount}("");
        require(ok, "transfer failed");
        emit Withdrawn(user, ethAmount, pointsAmount, block.timestamp);
    }

    // ========== 管理 ==========

    function emergencyWithdraw() external onlyOwner {
        payable(owner()).transfer(address(this).balance);
    }

    // ========== 查询 ==========

    function getBalance(address user) external view returns (uint256) {
        return balances[user];
    }

    function toPoints(uint256 ethAmount) external view returns (uint256) {
        return ethAmount * rate;
    }

    // ========== 修饰器 ==========

    modifier onlySubnet() {
        require(msg.sender == subnet, "only subnet contract");
        _;
    }

    // ========== 默认充值 ==========

    receive() external payable {
        balances[msg.sender] += msg.value;
        emit Recharged(
            msg.sender,
            msg.value,
            msg.value * rate,
            block.timestamp
        );
    }
}
