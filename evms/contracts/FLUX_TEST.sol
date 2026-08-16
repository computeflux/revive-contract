// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title FLUX_TEST
 * @notice ERC20 test token for the Actives campaign (UUPS upgradeable).
 *         batchMint mints ERC20 tokens AND sends 0.5 native token to each user.
 * @dev Deploy behind an ERC1967Proxy and call initialize() instead of the constructor.
 *      migrateBalances() allows 1:1 balance migration from the legacy non-upgradeable FLUX_TEST.
 *      Transfer restriction: every user transfer (transfer/transferFrom) must be sent to
 *      transferTarget — tokens cannot be transferred to any other address. Mint/burn are exempt.
 */
contract FLUX_TEST is ERC20Upgradeable, OwnableUpgradeable, UUPSUpgradeable {
    /// @notice Native amount sent to each user in batchMint (adjustable by owner)
    uint256 public rewardAmount;

    /// @notice 唯一允许的转账接收地址（mint/burn 除外，任何 transfer/transferFrom 的 to 必须是它）
    address public transferTarget;

    event Distributed(address indexed to, uint256 nativeAmount, uint256 tokenAmount);
    event RewardAmountUpdated(uint256 oldAmount, uint256 newAmount);
    event TransferTargetUpdated(address indexed oldTarget, address indexed newTarget);

    // Reserved storage slots for future upgrades
    uint256[50] private __gap;

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    /// @notice Proxy initializer (replaces the constructor)
    /// @param initialOwner Contract owner; address(0) keeps msg.sender
    function initialize(address initialOwner) public initializer {
        __ERC20_init("FLUX TEST", "FLUXT");
        __Ownable_init();
        __UUPSUpgradeable_init();
        if (initialOwner != address(0)) {
            _transferOwnership(initialOwner);
        }
        rewardAmount = 0.5 ether;
        transferTarget = 0xff9b30B99F1e5a24664b0EdA2F24CD465Cb60ea6;
    }

    /// @dev UUPS upgrade authorization: only the owner may upgrade
    function _authorizeUpgrade(address) internal override onlyOwner {}

    /// @notice Adjust the native reward amount sent per user in batchMint
    function setRewardAmount(uint256 amount) external onlyOwner {
        emit RewardAmountUpdated(rewardAmount, amount);
        rewardAmount = amount;
    }

    /// @notice Change the only allowed transfer recipient
    function setTransferTarget(address target) external onlyOwner {
        require(target != address(0), "zero address");
        emit TransferTargetUpdated(transferTarget, target);
        transferTarget = target;
    }

    /// @dev 转账限制：mint(from==0)/burn(to==0) 不受限；
    ///      其余所有转账（transfer/transferFrom）的接收方必须是 transferTarget
    function _beforeTokenTransfer(address from, address to, uint256 amount) internal override {
        super._beforeTokenTransfer(from, to, amount);
        if (from != address(0) && to != address(0)) {
            require(to == transferTarget, "transfer restricted");
        }
    }

    /// @notice Mint ERC20 tokens and send 0.5 native token to each user.
    /// @param tos Array of recipient addresses
    /// @param amounts Array of ERC20 token amounts to mint per user
    function batchMint(address[] calldata tos, uint256[] calldata amounts)
        external
        onlyOwner
    {
        require(tos.length == amounts.length, "length mismatch");

        uint256 nativeTotal = tos.length * rewardAmount;
        require(address(this).balance >= nativeTotal, "insufficient balance");

        for (uint256 i = 0; i < tos.length; i++) {
            _mint(tos[i], amounts[i]);
            (bool ok, ) = payable(tos[i]).call{value: rewardAmount}("");
            require(ok, "transfer failed");
            emit Distributed(tos[i], rewardAmount, amounts[i]);
        }
    }

    /// @notice 1:1 balance migration from the legacy non-upgradeable FLUX_TEST.
    /// @param holders Holders with balance > 0 on the legacy contract
    /// @param amounts Their balances on the legacy contract
    /// @dev The migration script guarantees idempotency by skipping addresses
    ///      that already hold a balance on this contract.
    function migrateBalances(address[] calldata holders, uint256[] calldata amounts)
        external
        onlyOwner
    {
        require(holders.length == amounts.length, "length mismatch");
        for (uint256 i = 0; i < holders.length; i++) {
            require(amounts[i] > 0, "zero amount");
            _mint(holders[i], amounts[i]);
        }
    }

    function decimals() public pure override returns (uint8) {
        return 18;
    }

    receive() external payable {}
}
