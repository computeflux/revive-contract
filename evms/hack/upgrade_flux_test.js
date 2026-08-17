/**
 * 升级 FLUX_TEST（UUPS 模式，仅替换实现合约，代理地址不变）
 *
 * 用法:
 *   npm run upgrade:flux                    # 完整升级
 *   DRY_RUN=true npm run upgrade:flux       # 只重新编译 + 预检，不发升级交易
 *
 * 流程:
 *   1. 自动重新编译（保证部署的是最新源码）
 *   2. 预检: 签名账户必须是代理 owner；打印当前实现地址（ERC-1967 槽位）
 *   3. 部署新实现 → proxy.upgradeToAndCall(新实现, "0x")
 *   4. 链上确认: ERC-1967 槽位已指向新实现
 *   5. 更新 config 的 implementation_address
 *   6. Blockscout 验证新实现（失败不影响已完成的升级）
 *
 * 注意: 升级后不得改变已有 storage 变量布局（只能追加变量，不能删除/重排）。
 *       代理地址（token_address）不变，所有持有者余额不受影响。
 */

const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

const configPath = path.resolve(__dirname, "../../../tee-node/ext/actives/flux_config.json");

// ERC-1967 implementation 槽位
const IMPLEMENTATION_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

async function readImplSlot(provider, proxyAddr) {
  const word = await provider.getStorage(proxyAddr, IMPLEMENTATION_SLOT);
  return "0x" + word.slice(-40);
}

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
  const dryRun = process.env.DRY_RUN === "true";

  const Wallet = require("ethers").Wallet;
  const wallet = new Wallet(cfg.private_key, hre.ethers.provider);
  const provider = hre.ethers.provider;

  console.log("Network:", hre.network.name);
  console.log("Proxy:", proxyAddr);
  console.log("Signer:", wallet.address);

  // ── 1. 重新编译，确保用最新源码 ───────────────────────────
  console.log("\n[1/5] 重新编译 ...");
  await hre.run("compile");

  // ── 2. 预检 ───────────────────────────────────────────────
  console.log("[2/5] 预检 ...");
  const FLUX_TEST = await hre.ethers.getContractFactory("FLUX_TEST");
  const token = FLUX_TEST.attach(proxyAddr);

  const owner = await token.owner();
  console.log("Owner:", owner);
  if (owner.toLowerCase() !== wallet.address.toLowerCase()) {
    console.error("❌ 签名账户不是代理 owner，无法升级");
    process.exit(1);
  }
  console.log("✅ Signer 是 owner");

  const curImpl = await readImplSlot(provider, proxyAddr);
  console.log("当前实现 (ERC-1967):", curImpl);
  if (cfg.implementation_address) {
    const cfgImpl = hre.ethers.getAddress(cfg.implementation_address);
    const matched = cfgImpl.toLowerCase() === curImpl.toLowerCase();
    console.log(`config implementation_address: ${cfgImpl} ${matched ? "✅ 一致" : "⚠️ 与链上不一致"}`);
  }

  if (dryRun) {
    console.log("\n[dry-run] 预检完成，未部署/未升级。确认后去掉 DRY_RUN=true 再运行。");
    return;
  }

  // ── 3. 部署新实现并升级 ───────────────────────────────────
  console.log("\n[3/5] 部署新实现 ...");
  const newImpl = await FLUX_TEST.deploy();
  await newImpl.waitForDeployment();
  const newImplAddr = await newImpl.getAddress();
  console.log("New implementation:", newImplAddr);

  console.log("升级 (upgradeToAndCall) ...");
  const tx = await token.connect(wallet).upgradeToAndCall(newImplAddr, "0x");
  const receipt = await tx.wait();
  console.log(`✅ 升级交易已确认: ${receipt.hash}`);
  console.log(`   Blockscout: https://blockscout-testnet.polkadot.io/tx/${receipt.hash}`);

  // ── 4. 链上确认 ───────────────────────────────────────────
  console.log("\n[4/5] 链上确认 ...");
  const afterImpl = await readImplSlot(provider, proxyAddr);
  if (afterImpl.toLowerCase() !== newImplAddr.toLowerCase()) {
    console.error(`❌ 升级后实现槽位 (${afterImpl}) 与预期 (${newImplAddr}) 不一致！`);
    process.exit(1);
  }
  console.log("✅ ERC-1967 槽位已指向新实现");
  console.log(`   合约地址不变: ${proxyAddr}`);

  // ── 5. 更新 config ────────────────────────────────────────
  console.log("\n[5/5] 更新 config ...");
  cfg.implementation_address = newImplAddr;
  fs.writeFileSync(configPath, JSON.stringify(cfg, null, 2));
  console.log(`✅ 已更新 ${configPath} implementation_address = ${newImplAddr}`);

  // ── 6. 验证新实现（可选，失败不影响升级）────────────────
  console.log("\n验证新实现源码 ...");
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

  console.log("\n✅ 升级完成！");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});