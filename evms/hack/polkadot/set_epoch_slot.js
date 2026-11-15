/**
 * 设置 Subnet 合约的 epochSlot（epoch 间隔，单位：区块数）
 *
 * 用法:
 *   SUBNET_CONTRACT_ADDRESS=<proxy_address> EPOCH_SLOT=<value> \
 *     npx hardhat run hack/set_epoch_slot.js --config hardhat.config.polkadot.js --network <network>
 *
 * 环境变量:
 *   SUBNET_CONTRACT_ADDRESS   Subnet 代理地址 (必填)
 *   EPOCH_SLOT                 epoch 间隔，单位区块数 (必填)
 *
 * 示例:
 *   # 本地 hardhat 网络
 *   SUBNET_CONTRACT_ADDRESS=0x... EPOCH_SLOT=10 \
 *     npx hardhat run hack/set_epoch_slot.js --config hardhat.config.polkadot.js --network localhost
 *
 *   # Polkadot 测试网
 *   SUBNET_CONTRACT_ADDRESS=0x... EPOCH_SLOT=72000 \
 *     npx hardhat run hack/set_epoch_slot.js --config hardhat.config.polkadot.js --network polkadotTestnet
 */

const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  console.log("Caller:", deployer.address);
  console.log("Network:", hre.network.name);

  // ── 参数校验 ──────────────────────────────────────────────
  const subnetAddr = process.env.SUBNET_CONTRACT_ADDRESS;
  if (!subnetAddr) {
    console.error("Error: SUBNET_CONTRACT_ADDRESS not set");
    process.exit(1);
  }

  const epochSlotStr = process.env.EPOCH_SLOT;
  if (!epochSlotStr) {
    console.error("Error: EPOCH_SLOT not set");
    process.exit(1);
  }
  const epochSlot = parseInt(epochSlotStr, 10);
  if (isNaN(epochSlot) || epochSlot <= 0) {
    console.error("Error: EPOCH_SLOT must be a positive integer");
    process.exit(1);
  }

  // ── 连接 Subnet 合约 ──────────────────────────────────────
  const Subnet = await hre.ethers.getContractFactory("Subnet");
  const subnet = Subnet.attach(subnetAddr);

  // 查询当前值
  const currentSlot = await subnet.epochSlot();
  console.log("");
  console.log("Subnet proxy:", subnetAddr);
  console.log("Current epochSlot:", currentSlot.toString());
  console.log("New epochSlot:    ", epochSlot);
  console.log("");

  // ── 调用 setEpochSlot ─────────────────────────────────────
  console.log("Sending setEpochSlot transaction...");
  const tx = await subnet.setEpochSlot(epochSlot);
  console.log("  tx hash:", tx.hash);
  console.log("  Waiting for confirmation...");

  const receipt = await tx.wait();
  console.log("  Confirmed in block:", receipt.blockNumber);
  console.log("  Gas used:", receipt.gasUsed.toString());
  console.log("");

  // ── 验证结果 ──────────────────────────────────────────────
  const newSlot = await subnet.epochSlot();
  console.log("Verification:");
  console.log("  epochSlot after set:", newSlot.toString());
  if (newSlot.toString() === epochSlot.toString()) {
    console.log("  ✓ SUCCESS: epochSlot updated correctly");
  } else {
    console.log("  ✗ FAIL: epochSlot mismatch!");
    process.exit(1);
  }
}

main()
  .then(() => process.exit(0))
  .catch((e) => {
    console.error("Error:", e.message || e);
    process.exit(1);
  });
