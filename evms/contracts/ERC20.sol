// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title QueryToken
 * @notice Standard ERC20 token with minting capability, deployable to Polkadot Hub testnet.
 *         The deployer becomes the owner and can mint new tokens for testing transfers.
 */
contract QueryToken is ERC20, Ownable {
    /**
     * @param name_ Token name
     * @param symbol_ Token symbol
     * @param initialSupply_ Initial supply minted to the deployer (in wei, e.g. 1000000 * 10^18)
     */
    constructor(
        string memory name_,
        string memory symbol_,
        uint256 initialSupply_
    ) ERC20(name_, symbol_) {
        _mint(msg.sender, initialSupply_);
    }

    /**
     * @notice Mint new tokens to a specified address. Only callable by the owner.
     * @param to Recipient address
     * @param amount Amount to mint (in wei)
     */
    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    /**
     * @notice Returns the number of decimals (18 by default).
     */
    function decimals() public pure override returns (uint8) {
        return 18;
    }
}
