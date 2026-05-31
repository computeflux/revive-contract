/**
 * 初始化 Subnet 链上数据（Region、LevelPrice、Asset、Worker、Secret、BootNode、Validator 等）
 *
 * 用法:
 *   npx hardhat run hack/init_subnet_data.js --config hardhat.config.polkadot.js --network <network>
 *
 * 环境变量:
 *   SUBNET_CONTRACT_ADDRESS   Subnet 代理地址 (必填)
 *   CONFIG_FILE               指向 config JSON 的路径 (必填)
 */

const hre = require("hardhat");
const fs = require("fs");
const { decodeAddress } = require("@polkadot/util-crypto");

// ── helpers ────────────────────────────────────────────────

/** "192.168.110.205" → { a: 192, b: 168, c: 110, d: 205 } */
function parseIp(ipStr) {
  const parts = (ipStr || "0.0.0.0").split(".");
  return {
    a: parseInt(parts[0]) || 0,
    b: parseInt(parts[1]) || 0,
    c: parseInt(parts[2]) || 0,
    d: parseInt(parts[3]) || 0,
  };
}

/**
 * Convert SS58 / hex / string → bytes32 (raw public key).
 *
 * SS58 addresses (e.g. "5EC7yhh...") are decoded to their raw 32-byte
 * public key.  Hex strings (0x...) are zero-padded to 32 bytes.
 * Other strings are keccak256-hashed (fallback).
 */
function toBytes32(val) {
  if (!val) return hre.ethers.ZeroHash;
  // Already a 0x-prefixed hex (≤ 66 chars) → zero-pad to bytes32
  if (val.startsWith("0x") && val.length <= 66) {
    return hre.ethers.zeroPadValue(val, 32);
  }
  // Try SS58 decode → raw 32-byte public key
  try {
    const raw = decodeAddress(val);
    // raw is a Uint8Array (32 bytes for ed25519/sr25519)
    if (raw.length === 32) {
      return "0x" + Buffer.from(raw).toString("hex");
    }
  } catch (_) {
    // Not a valid SS58, fall through
  }
  // Fallback: keccak256 hash
  return hre.ethers.keccak256(hre.ethers.toUtf8Bytes(String(val)));
}

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  console.log("Caller:", deployer.address);
  console.log("Network:", hre.network.name);
  console.log("");

  // ── 参数 ──────────────────────────────────────────────
  const subnetAddr =
    process.env.SUBNET_CONTRACT_ADDRESS || process.env.SUBNET_PROXY_ADDRESS;
  if (!subnetAddr) {
    console.error("Error: SUBNET_CONTRACT_ADDRESS not set");
    process.exit(1);
  }

  const configPath = process.env.CONFIG_FILE || "hack/configs/local.json";
  if (!fs.existsSync(configPath)) {
    console.error("Error: config file not found:", configPath);
    process.exit(1);
  }
  const cfg = JSON.parse(fs.readFileSync(configPath, "utf-8"));
  const genesis = cfg.genesis;
  if (!genesis) {
    console.error("Error: missing genesis in config");
    process.exit(1);
  }

  const Subnet = await hre.ethers.getContractFactory("Subnet");
  const subnet = Subnet.attach(subnetAddr);

  console.log("Subnet proxy:", subnetAddr);
  console.log("");

  // ── 0. 设置 epochSlot 为较小值，让 tee-provider 能立即触发 epoch ──
  const epochSlot = genesis.epoch_slot || 5;
  console.log("--- 0. Set Epoch Slot ---");
  const tx0 = await subnet.setEpochSlot(epochSlot);
  const r0 = await tx0.wait();
  console.log(`  epochSlot = ${epochSlot}, tx=${r0.hash}`);
  console.log("");

  // ── 1. 设置 Region ────────────────────────────────────
  console.log("--- 1. Set Regions ---");
  const regionIds = [];
  for (const name of genesis.regions) {
    const nameBytes = hre.ethers.toUtf8Bytes(name);
    const tx = await subnet.setRegion(nameBytes);
    const receipt = await tx.wait();
    const id = regionIds.length;
    regionIds.push(id);
    console.log(`  Region "${name}" -> id=${id}, tx=${receipt.hash}`);
  }
  console.log("");

  // ── 2. 设置 Level Prices ──────────────────────────────
  console.log("--- 2. Set Level Prices ---");
  for (const lp of genesis.level_prices) {
    const tx = await subnet.setLevelPrice(lp.level, {
      cpu: lp.cpu,
      mem: lp.mem,
      storageAmount: lp.storage_amount,
      network: lp.network,
    });
    const receipt = await tx.wait();
    console.log(
      `  Level ${lp.level}: cpu=${lp.cpu}, mem=${lp.mem}, storage=${lp.storage_amount}, net=${lp.network}, tx=${receipt.hash}`
    );
  }
  console.log("");

  // ── 3. 设置 Min Mortgage ──────────────────────────────
  console.log("--- 3. Set Min Mortgage ---");
  if (genesis.min_mortgage > 0) {
    const tx = await subnet.setMinMortgage(genesis.min_mortgage);
    const receipt = await tx.wait();
    console.log(`  Min mortgage: ${genesis.min_mortgage}, tx=${receipt.hash}`);
  }
  for (const lm of genesis.level_min_mortgages || []) {
    const tx = await subnet.setLevelMinMortgage(lm.level, lm.amount);
    const receipt = await tx.wait();
    console.log(
      `  Level ${lm.level} min mortgage: ${lm.amount}, tx=${receipt.hash}`
    );
  }
  console.log("");

  // ── 4. 设置 Assets ────────────────────────────────────
  console.log("--- 4. Set Assets ---");
  for (const asset of genesis.assets || []) {
    const tx = await subnet.setAsset(
      {
        name: asset.name,
        assetType: asset.asset_type,
        totalSupply: asset.total_supply,
      },
      asset.price
    );
    const receipt = await tx.wait();
    console.log(
      `  Asset "${asset.name}": price=${asset.price}, tx=${receipt.hash}`
    );
  }
  console.log("");

  // ── 5. 注册 Secrets (validator nodes) ──────────────────
  console.log("--- 5. Register Secrets ---");
  const secretIdByIndex = [];
  for (const s of genesis.secrets || []) {
    const validatorId = toBytes32(s.validator_id || s.ss58);
    const p2pId = toBytes32(s.p2p_id || s.p_ss58);
    const ip = parseIp(s.ip);
    const validatorBls = s.bls_validator_key
      ? hre.ethers.getBytes("0x" + s.bls_validator_key)
      : new Uint8Array(0);
    const tx = await subnet.secretRegister(
      hre.ethers.toUtf8Bytes(s.name),
      validatorId,
      p2pId,
      ip,
      s.port || 0,
      validatorBls
    );
    const receipt = await tx.wait();
    // secretRegister returns the new id via event; we track by insertion order
    const idx = secretIdByIndex.length;
    secretIdByIndex.push(idx);
    console.log(
      `  Secret "${s.name}" -> index=${idx}, validatorId=${validatorId}, blsLen=${validatorBls.length}, tx=${receipt.hash}`
    );
  }
  console.log("");

  // ── 6. 设置 Boot Nodes (按 secret index) ───────────────
  console.log("--- 6. Set Boot Nodes ---");
  if ((genesis.boot_nodes || []).length > 0) {
    const tx = await subnet.setBootNodes(genesis.boot_nodes);
    const receipt = await tx.wait();
    console.log(`  Boot nodes: [${genesis.boot_nodes}], tx=${receipt.hash}`);
  } else {
    console.log("  (none)");
  }
  console.log("");

  // ── 7. Validator Join ──────────────────────────────────
  console.log("--- 7. Validator Join ---");
  for (const vid of genesis.validators || []) {
    const tx = await subnet.validatorJoin(vid);
    const receipt = await tx.wait();
    console.log(`  Validator join id=${vid}, tx=${receipt.hash}`);
  }
  if (!(genesis.validators || []).length) {
    console.log("  (none)");
  }
  console.log("");

  // ── 8. 注册 Workers ───────────────────────────────────
  console.log("--- 8. Register Workers ---");
  for (const w of genesis.workers || []) {
    const nameBytes = hre.ethers.toUtf8Bytes(w.name);
    const p2pId = toBytes32(w.p2p_id || w.p_ss58);
    const ip = parseIp(w.ip);
    const tx = await subnet.workerRegister(
      nameBytes,
      p2pId,
      ip,
      w.port || 0,
      w.level || 0,
      w.region_id || 0
    );
    const receipt = await tx.wait();
    console.log(`  Worker "${w.name}" p2p=${p2pId}, tx=${receipt.hash}`);
  }
  if (!(genesis.workers || []).length) {
    console.log("  (none)");
  }
  console.log("");

  // ── 9. Worker Mortgages ───────────────────────────────
  console.log("--- 9. Worker Mortgages ---");
  for (const w of genesis.workers || []) {
    if (!w.mortgage || w.mortgage <= 0) continue;
    console.log(
      `  (skip mortgage for "${w.name}" — need worker id, use separate script)`
    );
  }
  if (!(genesis.workers || []).filter((w) => w.mortgage > 0).length) {
    console.log("  (none)");
  }
  console.log("");

  // ── 输出 ──────────────────────────────────────────────
  console.log("========================================");
  console.log("Subnet data initialization complete");
  console.log("========================================");
  console.log("Subnet: ", subnetAddr);
  console.log("");
}

main()
  .then(() => process.exit(0))
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
