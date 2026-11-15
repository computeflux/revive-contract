/**
 * 部署 QueryToken (ERC20) 到 Polkadot Hub 测试网
 *
 * 用法:
 *   npx hardhat run hack/deploy_erc20.js --config hardhat.config.polkadot.js --network polkadotTestnet
 *
 * 环境变量 (在 .env 文件中):
 *   PRIVATE_KEY=0x...   部署者私钥
 *   ERC20_NAME=QueryToken       代币名称 (可选，默认 QueryToken)
 *   ERC20_SYMBOL=QTK            代币符号 (可选，默认 QTK)
 *   ERC20_INITIAL_SUPPLY=1000000 初始供应量 (可选，默认 1000000)
 */

const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  console.log("Deployer:", deployer.address);
  console.log("Network:", hre.network.name);

  const name = process.env.ERC20_NAME || "QueryToken";
  const symbol = process.env.ERC20_SYMBOL || "QTK";
  const initialSupply = process.env.ERC20_INITIAL_SUPPLY || "1000000";

  // 初始供应量: 默认 100 万 * 10^18
  const initialSupplyWei = hre.ethers.parseUnits(initialSupply, 18);

  console.log("");
  console.log(`Deploying ${name} (${symbol})`);
  console.log(`Initial supply: ${initialSupply} ${symbol}`);

  // 部署合约
  const QueryToken = await hre.ethers.getContractFactory("QueryToken");
  const token = await QueryToken.deploy(name, symbol, initialSupplyWei);
  await token.waitForDeployment();

  const tokenAddress = await token.getAddress();
  console.log(`\n✅ ${name} deployed to:`, tokenAddress);

  // 验证余额
  const balance = await token.balanceOf(deployer.address);
  console.log("Deployer balance:", hre.ethers.formatUnits(balance, 18), symbol);

  // 输出验证命令
  console.log("\n--- Verify on Blockscout ---");
  console.log(
    `npx hardhat verify --config hardhat.config.polkadot.js --network polkadotTestnet ${tokenAddress} "${name}" "${symbol}" "${initialSupplyWei}"`
  );
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
