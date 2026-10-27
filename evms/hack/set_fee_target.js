/**
 * 设置 FLUX_TEST 的 feeTarget 为 SUBNET 地址
 *
 * 用法:
 *   CONFIG_PATH=configs/flux_config_test.json npx hardhat run hack/set_fee_target.js --config hardhat.config.polkadot.js --network polkadotTestnet
 */

const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

const configPath = process.env.CONFIG_PATH
  ? path.resolve(process.env.CONFIG_PATH)
  : path.resolve(__dirname, "../configs/flux_config_test.json");

async function main() {
  const cfg = JSON.parse(fs.readFileSync(configPath, "utf8"));
  const tokenAddr = cfg.token_address;
  const subnetAddr = "0x576afc0dab34389170845e9b996eca9017d2a505";

  if (!cfg.private_key) {
    console.error("❌ private_key 为空");
    process.exit(1);
  }

  const Wallet = require("ethers").Wallet;
  const wallet = new Wallet(cfg.private_key, hre.ethers.provider);

  console.log("Network:", hre.network.name);
  console.log("FLUX_TEST:", tokenAddr);
  console.log("SUBNET (feeTarget):", subnetAddr);
  console.log("Signer:", wallet.address);

  const FLUX_TEST = await hre.ethers.getContractFactory("FLUX_TEST");
  const token = FLUX_TEST.attach(tokenAddr).connect(wallet);

  // 检查当前 feeTarget
  const currentFeeTarget = await token.feeTarget();
  console.log("当前 feeTarget:", currentFeeTarget);

  if (currentFeeTarget.toLowerCase() === subnetAddr.toLowerCase()) {
    console.log("✅ feeTarget 已经是 SUBNET 地址，无需操作");
    return;
  }

  // 调用 setFeeTarget
  console.log("调用 setFeeTarget ...");
  const tx = await token.setFeeTarget(subnetAddr);
  const receipt = await tx.wait();
  console.log(`✅ 交易已确认: ${receipt.hash}`);
  console.log(`   Blockscout: https://blockscout-testnet.polkadot.io/tx/${receipt.hash}`);

  // 确认
  const after = await token.feeTarget();
  console.log("新 feeTarget:", after);
  if (after.toLowerCase() !== subnetAddr.toLowerCase()) {
    console.error("❌ feeTarget 设置失败！");
    process.exit(1);
  }
  console.log("✅ feeTarget 设置成功！");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
