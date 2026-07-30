// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title FLUX_TEST
 * @notice ERC20 test token for the Actives campaign.
 *         batchMint mints ERC20 tokens AND sends 0.5 native token to each user.
 */
contract FLUX_TEST is ERC20, Ownable {
    /// @notice Fixed native amount sent to each user in batchMint
    uint256 public constant REWARD_AMOUNT = 0.5 ether;

    event Distributed(address indexed to, uint256 nativeAmount, uint256 tokenAmount);

    constructor(address initialOwner) ERC20("FLUX TEST", "FLUXT") {
        _transferOwnership(initialOwner);
    }

    /// @notice Mint ERC20 tokens and send 0.5 native token to each user.
    /// @param tos Array of recipient addresses
    /// @param amounts Array of ERC20 token amounts to mint per user
    function batchMint(address[] calldata tos, uint256[] calldata amounts)
        external
        onlyOwner
    {
        require(tos.length == amounts.length, "length mismatch");

        uint256 nativeTotal = tos.length * REWARD_AMOUNT;
        require(address(this).balance >= nativeTotal, "insufficient balance");

        for (uint256 i = 0; i < tos.length; i++) {
            _mint(tos[i], amounts[i]);
            (bool ok, ) = payable(tos[i]).call{value: REWARD_AMOUNT}("");
            require(ok, "transfer failed");
            emit Distributed(tos[i], REWARD_AMOUNT, amounts[i]);
        }
    }

    function decimals() public pure override returns (uint8) {
        return 18;
    }

    receive() external payable {}
}
