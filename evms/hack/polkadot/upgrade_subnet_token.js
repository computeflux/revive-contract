/**
 * 升级 Subnet / Token 合约（UUPS 模式）
 *
 * 用法:
 *   npx hardhat run hack/upgrade_subnet_token.js --network polkadotTestnet
 *
 * 环境变量:
 *   TOKEN_PROXY_ADDRESS=0x...    Token 代理地址
 *   SUBNET_PROXY_ADDRESS=0x...   Subnet 代理地址
 *   UPGRADE_TOKEN=true           升级 Token（默认两个都升级）
 *   UPGRADE_SUBNET=true          升级 Subnet
 */

const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  console.log("Deployer:", deployer.address);
  console.log("Network:", hre.network.name);
  console.log("");

  const upgradeToken = process.env.UPGRADE_TOKEN !== "false";
  const upgradeSubnet = process.env.UPGRADE_SUBNET !== "false";

  // ── 升级 Token ──────────────────────────────────────────
  if (upgradeToken) {
    const tokenProxyAddr =
      process.env.TOKEN_PROXY_ADDRESS || process.env.TOKEN_CONTRACT_ADDRESS;
    if (!tokenProxyAddr) {
      console.log("SKIP Token: 未设置 TOKEN_PROXY_ADDRESS");
    } else {
      console.log("--- Upgrade Token ---");
      console.log("Proxy:", tokenProxyAddr);

      const TokenV2 = await hre.ethers.getContractFactory("Token");
      const newImpl = await TokenV2.deploy();
      await newImpl.waitForDeployment();
      console.log("New implementation:", await newImpl.getAddress());

      const token = TokenV2.attach(tokenProxyAddr);
      const tx = await token.upgradeToAndCall(
        await newImpl.getAddress(),
        "0x" // 不调用额外初始化
      );
      await tx.wait();
      console.log("Token upgraded successfully");
    }
  }

  // ── 升级 Subnet ─────────────────────────────────────────
  if (upgradeSubnet) {
    const subnetProxyAddr =
      process.env.SUBNET_PROXY_ADDRESS || process.env.SUBNET_CONTRACT_ADDRESS;
    if (!subnetProxyAddr) {
      console.log("SKIP Subnet: 未设置 SUBNET_PROXY_ADDRESS");
    } else {
      console.log("\n--- Upgrade Subnet ---");
      console.log("Proxy:", subnetProxyAddr);

      const SubnetV2 = await hre.ethers.getContractFactory("Subnet");
      const newImpl = await SubnetV2.deploy();
      await newImpl.waitForDeployment();
      console.log("New implementation:", await newImpl.getAddress());

      const subnet = SubnetV2.attach(subnetProxyAddr);
      const tx = await subnet.upgradeToAndCall(
        await newImpl.getAddress(),
        "0x"
      );
      await tx.wait();
      console.log("Subnet upgraded successfully");
    }
  }

  console.log("\nDone.");
}

main()
  .then(() => process.exit(0))
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
