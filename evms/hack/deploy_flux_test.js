/**
 * 部署 FLUX_TEST 代币到 Polkadot Hub 测试网
 *
 * 用法:
 *   npx hardhat run hack/deploy_flux_test.js --config hardhat.config.polkadot.js --network polkadotTestnet
 *
 * 说明:
 *   会自动读取 ext/actives/flux_config.json 中的 private_key 进行部署。
 *   如果 flux_config.json 不存在或 private_key 为空，会生成新密钥并保存。
 *   部署后自动等待交易确认，并自动验证合约源码。
 */

const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

// ── 自动从 actives 配置加载私钥 ────────────────────────────
const configPath = path.resolve(__dirname, "../../../tee-node/ext/actives/flux_config.json");

// 必须存在 flux_config.json 且包含 private_key
if (!fs.existsSync(configPath)) {
  console.error(`❌ 未找到 ${configPath}`);
  process.exit(1);
}

let activesCfg;
try {
  activesCfg = JSON.parse(fs.readFileSync(configPath, "utf8"));
} catch (e) {
  console.error(`❌ 无法解析 ${configPath}: ${e.message}`);
  process.exit(1);
}

if (!activesCfg.private_key || activesCfg.private_key === "") {
  console.error(`❌ ${configPath} 中 private_key 为空`);
  console.error(`   请先启动 actives 服务自动生成私钥，或手动填入`);
  process.exit(1);
}

const pk = activesCfg.private_key;
process.env.PRIVATE_KEY = pk;
const Wallet = require("ethers").Wallet;
const deployerAddr = new Wallet(pk).address;
console.log(`  Deployer: ${deployerAddr}`);

// ── 部署 ────────────────────────────────────────────────────
async function main() {
  console.log("Network:", hre.network.name);
  console.log("Network:", hre.network.name);

  console.log("");
  console.log("Deploying FLUX_TEST (FLUXT) ...");

  const FLUX_TEST = await hre.ethers.getContractFactory("FLUX_TEST");
  const token = await FLUX_TEST.deploy(deployerAddr);
  await token.waitForDeployment();

  const tokenAddress = await token.getAddress();
  console.log(`\n✅ FLUX_TEST deployed to:`, tokenAddress);

  console.log("Owner:", await token.owner());
  console.log("Native per user:", hre.ethers.formatEther(await token.REWARD_AMOUNT()));

  const ownerAddr = await token.owner();
  if (ownerAddr.toLowerCase() === deployerAddr.toLowerCase()) {
    console.log("✅ Owner 与 Deployer 一致");
  }

  // ── 等待交易确认并验证 ───────────────────────────────
  console.log("\n等待交易确认...");

  // 获取部署交易的 receipt
  const deployTx = token.deploymentTransaction();
  if (deployTx) {
    const receipt = await deployTx.wait(2); // 等待 2 个区块确认
    console.log(`确认区块: ${receipt.blockNumber}`);
  }

  console.log("验证合约源码 (Verify & Publish) ...");
  try {
    await hre.run("verify", {
      address: tokenAddress,
      constructorArgsParams: [deployerAddr],
      // Hardhat will figure out the contract automatically
    });
    console.log("✅ 合约验证成功！");
  } catch (err) {
    console.log(`⚠️  验证结果: ${err.message}`);
    console.log("可稍后手动运行:");
    console.log(`  npx hardhat verify --config hardhat.config.polkadot.js --network ${hre.network.name} ${tokenAddress} "${deployerAddr}"`);
  }

  // ── 写入 token_address 到 config ──────────────────────
  if (fs.existsSync(configPath)) {
    try {
      const cfg = JSON.parse(fs.readFileSync(configPath, "utf8"));
      cfg.rpc_url = hre.network.config.url;
      cfg.private_key = pk;
      cfg.token_address = tokenAddress;
      fs.writeFileSync(configPath, JSON.stringify(cfg, null, 2));
      console.log(`\n✅ 已自动更新 ${configPath}`);
      console.log(JSON.stringify(cfg, null, 2));
    } catch (e) {
      console.log(`\n⚠️  无法写入 config: ${e.message}`);
    }
  }

  console.log(`\n请向合约转入 native token 作为分发资金:`);
  console.log(`  cast send ${tokenAddress} --value 10ether`);

  console.log(`\nBlockscout: https://blockscout-testnet.polkadot.io/address/${tokenAddress}`);
  console.log("\n✅ 完成！");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
