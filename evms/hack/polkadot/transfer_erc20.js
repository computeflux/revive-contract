/**
 * ERC20 转账脚本
 *
 * 用法:
 *   npx hardhat run hack/transfer_erc20.js --config hardhat.config.polkadot.js --network polkadotTestnet
 *
 * 环境变量:
 *   PRIVATE_KEY=0x...         发送者私钥
 *   ERC20_ADDRESS=0x...       代币合约地址
 *   ERC20_TO=0x...            接收者地址
 *   ERC20_AMOUNT=100          转账数量（默认 100）
 */

const hre = require("hardhat");

async function main() {
  const tokenAddress = process.env.ERC20_ADDRESS;
  const to = process.env.ERC20_TO;
  const amount = process.env.ERC20_AMOUNT || "100";

  if (!tokenAddress) throw new Error("请设置 ERC20_ADDRESS");
  if (!to) throw new Error("请设置 ERC20_TO");

  const [sender] = await hre.ethers.getSigners();
  const token = await hre.ethers.getContractAt("QueryToken", tokenAddress);

  // 查看转账前余额
  const balanceBefore = await token.balanceOf(sender.address);
  console.log(`Sender: ${sender.address}`);
  console.log(`Balance before: ${hre.ethers.formatUnits(balanceBefore, 18)}`);

  // 执行转账
  const amountWei = hre.ethers.parseUnits(amount, 18);
  const tx = await token.transfer(to, amountWei);
  console.log(`Transferring ${amount} tokens to ${to}...`);
  await tx.wait();

  // 查看转账后余额
  const balanceAfter = await token.balanceOf(sender.address);
  const balanceTo = await token.balanceOf(to);
  console.log(`\n✅ Transfer complete!`);
  console.log(`Tx: ${tx.hash}`);
  console.log(`Sender balance after: ${hre.ethers.formatUnits(balanceAfter, 18)}`);
  console.log(`Recipient balance: ${hre.ethers.formatUnits(balanceTo, 18)}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
