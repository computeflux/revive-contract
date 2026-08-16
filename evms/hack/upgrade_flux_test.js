/**
 * 升级 FLUX_TEST（UUPS 模式，仅替换实现合约，代理地址不变）
 *
 * 用法:
 *   npx hardhat run hack/upgrade_flux_test.js --config hardhat.config.polkadot.js --network polkadotTestnet
 *
 * 说明:
 *   从 config 读取 token_address（代理）与 private_key。
 *   部署新实现 → proxy.upgradeToAndCall(新实现, "0x")
 *
 * 注意: 升级后不得改变已有 storage 变量布局（只能追加变量，不能删除/重排）。
 *       修改合约后先重新编译（npm run compile）再运行本脚本。
 */

const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

const configPath = path.resolve(__dirname, "../../../tee-node/ext/actives/flux_config.json");

async function main() {
  if (!fs.existsSync(configPath)) {
    console.error(`❌ 未找到 ${configPath}`);
    process.exit(1);
  }
  const cfg = JSON.parse(fs.readFileSync(configPath, "utf8"));
  const proxyAddr = process.env.TOKEN_ADDRESS || cfg.token_address;
  if (!proxyAddr) {
    console.error("❌ 未找到 token_address，请检查 config");
    process.exit(1);
  }
  if (!cfg.private_key) {
    console.error(`❌ ${configPath} 中 private_key 为空`);
    process.exit(1);
  }

  const Wallet = require("ethers").Wallet;
  const wallet = new Wallet(cfg.private_key, hre.ethers.provider);
  console.log("Network:", hre.network.name);
  console.log("Proxy:", proxyAddr);
  console.log("Signer:", wallet.address);

  // ── 1. 部署新实现合约 ───────────────────────────────────
  const FLUX_TEST = await hre.ethers.getContractFactory("FLUX_TEST");
  const newImpl = await FLUX_TEST.deploy();
  await newImpl.waitForDeployment();
  const newImplAddr = await newImpl.getAddress();
  console.log("New implementation:", newImplAddr);

  // ── 2. 升级（无额外调用，data 传 "0x"）──────────────────
  const token = FLUX_TEST.attach(proxyAddr).connect(wallet);
  const tx = await token.upgradeToAndCall(newImplAddr, "0x");
  await tx.wait();
  console.log("✅ 升级成功, tx:", tx.hash);

  // ── 3. 更新 config ──────────────────────────────────────
  cfg.implementation_address = newImplAddr;
  fs.writeFileSync(configPath, JSON.stringify(cfg, null, 2));
  console.log(`已更新 ${configPath} implementation_address`);

  // ── 4. 验证新实现（可选，失败不影响升级）────────────────
  try {
    await hre.run("verify", {
      address: newImplAddr,
      contract: "contracts/FLUX_TEST.sol:FLUX_TEST",
      constructorArgsParams: [],
    });
    console.log("✅ 新实现验证成功！");
  } catch (err) {
    console.log(`⚠️  验证结果: ${err.message}`);
    console.log(`  npx hardhat verify --config hardhat.config.polkadot.js --network ${hre.network.name} ${newImplAddr} --contract contracts/FLUX_TEST.sol:FLUX_TEST`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
