/**
 * 部署 FLUX_TEST（UUPS 可升级模式）到 Polkadot Hub 测试网
 *
 * 用法:
 *   npx hardhat run hack/deploy_flux_test.js --config hardhat.config.polkadot.js --network polkadotTestnet
 *
 * 说明:
 *   会自动读取 ext/actives/flux_config.json 中的 private_key 进行部署。
 *   部署流程: 实现合约 → ERC1967Proxy → initialize
 *   如果配置中已有旧的 token_address，会先保存为 legacy_token_address
 *   （供 hack/migrate_flux_test.js 做余额迁移）。
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

/// 获取 ERC1967Proxy 工厂：优先 hardhat 编译产物，缺失时回退到
/// @openzeppelin/contracts 自带的编译产物（hardhat clean 后 artifacts 会被清掉）
async function getERC1967ProxyFactory() {
  try {
    return await hre.ethers.getContractFactory("ERC1967Proxy");
  } catch (e) {
    if (String(e.message).includes("HH700")) {
      const ozArtifact = require("@openzeppelin/contracts/build/contracts/ERC1967Proxy.json");
      const [signer] = await hre.ethers.getSigners();
      return new hre.ethers.ContractFactory(ozArtifact.abi, ozArtifact.bytecode, signer);
    }
    throw e;
  }
}

// ── 部署 ────────────────────────────────────────────────────
async function main() {
  console.log("Network:", hre.network.name);
  console.log("");
  console.log("Deploying FLUX_TEST (FLUXT) [UUPS upgradeable] ...");

  // ── 1. 部署实现合约 ─────────────────────────────────────
  const FLUX_TEST = await hre.ethers.getContractFactory("FLUX_TEST");
  const impl = await FLUX_TEST.deploy();
  await impl.waitForDeployment();
  const implAddr = await impl.getAddress();
  console.log("Implementation:", implAddr);

  // ── 2. 编码 initialize 调用 ─────────────────────────────
  const initData = FLUX_TEST.interface.encodeFunctionData("initialize", [
    deployerAddr,
  ]);

  // ── 3. 部署 ERC1967 代理（构造时即执行 initialize）──────
  const ERC1967Proxy = await getERC1967ProxyFactory();
  const proxy = await ERC1967Proxy.deploy(implAddr, initData);
  await proxy.waitForDeployment();
  const tokenAddress = await proxy.getAddress();
  console.log("Proxy (token address):", tokenAddress);

  // ── 4. 附加接口并检查 ───────────────────────────────────
  const token = FLUX_TEST.attach(tokenAddress);
  console.log("Owner:", await token.owner());
  console.log("rewardAmount:", hre.ethers.formatEther(await token.rewardAmount()));
  console.log("transferTarget:", await token.transferTarget());

  const ownerAddr = await token.owner();
  if (ownerAddr.toLowerCase() === deployerAddr.toLowerCase()) {
    console.log("✅ Owner 与 Deployer 一致");
  }

  // ── 等待交易确认并验证 ───────────────────────────────
  console.log("\n等待交易确认...");

  // 获取代理部署交易的 receipt
  const deployTx = proxy.deploymentTransaction();
  if (deployTx) {
    const receipt = await deployTx.wait(2); // 等待 2 个区块确认
    console.log(`确认区块: ${receipt.blockNumber}`);
  }

  console.log("验证实现合约源码 (Verify & Publish) ...");
  try {
    await hre.run("verify", {
      address: implAddr,
      contract: "contracts/FLUX_TEST.sol:FLUX_TEST",
      constructorArgsParams: [],
    });
    console.log("✅ 合约验证成功！");
  } catch (err) {
    console.log(`⚠️  验证结果: ${err.message}`);
    console.log("可稍后手动运行:");
    console.log(`  npx hardhat verify --config hardhat.config.polkadot.js --network ${hre.network.name} ${implAddr} --contract contracts/FLUX_TEST.sol:FLUX_TEST`);
  }

  // ── 写入 token_address 到 config ──────────────────────
  if (fs.existsSync(configPath)) {
    try {
      const cfg = JSON.parse(fs.readFileSync(configPath, "utf8"));
      // 保留旧地址，供余额迁移脚本使用
      if (cfg.token_address && cfg.token_address.toLowerCase() !== tokenAddress.toLowerCase()) {
        cfg.legacy_token_address = cfg.token_address;
      }
      cfg.rpc_url = hre.network.config.url;
      cfg.private_key = pk;
      cfg.token_address = tokenAddress;
      cfg.implementation_address = implAddr;
      fs.writeFileSync(configPath, JSON.stringify(cfg, null, 2));
      console.log(`\n✅ 已自动更新 ${configPath}`);
      console.log(JSON.stringify(cfg, null, 2));
    } catch (e) {
      console.log(`\n⚠️  无法写入 config: ${e.message}`);
    }
  }

  if (activesCfg.token_address) {
    console.log("\n如需迁移旧合约余额，先 dry-run 再执行:");
    console.log("  DRY_RUN=true npx hardhat run hack/migrate_flux_test.js --config hardhat.config.polkadot.js --network polkadotTestnet");
    console.log("  npx hardhat run hack/migrate_flux_test.js --config hardhat.config.polkadot.js --network polkadotTestnet");
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
