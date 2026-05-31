/**
 * 部署 Subnet + Token 合约（UUPS 可升级模式）
 *
 * 用法:
 *   npm run deploy          # 部署到 polkadotTestnet
 *   npm run deploy:local    # 部署到本地网络
 *
 * 环境变量 (在 .env 文件中):
 *   PRIVATE_KEY=0x...   部署者私钥
 *   ECDSA_KEY=0x...     tee-provider 的 ECDSA 私钥 (用于 sideChainMultiKey)
 *
 * 部署顺序:
 *   1. 部署 Token 实现合约 → ERC1967Proxy → 初始化
 *   2. 部署 Subnet 实现合约 → ERC1967Proxy → 初始化
 *   3. Token.setSubnet(Subnet代理地址)
 *   4. Subnet.setTokenContract(Token代理地址)
 *   5. 如果配置了 ECDSA_KEY，Subnet.setCloudContract(tee-provider地址)
 */

const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  console.log("Deployer:", deployer.address);
  console.log("Network:", hre.network.name);
  console.log("");

  // ── 1. 部署 Token（UUPS 代理模式）──────────────────────
  console.log("--- 1. Deploy Token (UUPS) ---");

  // 1a. 部署实现合约
  const Token = await hre.ethers.getContractFactory("Token");
  const tokenImpl = await Token.deploy();
  await tokenImpl.waitForDeployment();
  console.log("Token implementation:", await tokenImpl.getAddress());

  // 1b. 编码初始化调用
  const tokenInitData = Token.interface.encodeFunctionData("initialize", [
    deployer.address,
  ]);

  // 1c. 部署代理合约
  const ERC1967Proxy = await hre.ethers.getContractFactory("ERC1967Proxy");
  const tokenProxy = await ERC1967Proxy.deploy(
    await tokenImpl.getAddress(),
    tokenInitData
  );
  await tokenProxy.waitForDeployment();
  const tokenAddr = await tokenProxy.getAddress();
  console.log("Token proxy:", tokenAddr);

  // 1d. 附加 Token 接口到代理
  const token = Token.attach(tokenAddr);

  // ── 2. 部署 Subnet 实现合约（先部署实现，代理稍后）──────
  console.log("\n--- 2. Deploy Subnet Implementation ---");
  const Subnet = await hre.ethers.getContractFactory("Subnet");
  const subnetImpl = await Subnet.deploy();
  await subnetImpl.waitForDeployment();
  console.log("Subnet implementation:", await subnetImpl.getAddress());

  // ── 3. 部署 BLS 合约 ────────────────────────────────────
  console.log("\n--- 3. Deploy BLS ---");
  const BLS = await hre.ethers.getContractFactory("BLS");
  // G2 generator in EIP-2537 format (256 bytes zero-padded)
  // x0=0x180... y0=0x... (placeholder zeros for now; real value set post-deploy)
  const g2Generator = new Uint8Array(256);
  const bls = await BLS.deploy(hre.ethers.ZeroAddress, g2Generator);
  await bls.waitForDeployment();
  const blsAddr = await bls.getAddress();
  console.log("BLS contract:", blsAddr);

  // ── 4. 部署 Subnet 代理合约并初始化（传入 BLS 地址）──────
  console.log("\n--- 4. Deploy Subnet Proxy (UUPS) ---");
  const subnetInitData = Subnet.interface.encodeFunctionData("initialize", [
    deployer.address,
    deployer.address, // sideChainMultiKey 初始设为 deployer
    blsAddr,
  ]);

  const subnetProxy = await ERC1967Proxy.deploy(
    await subnetImpl.getAddress(),
    subnetInitData
  );
  await subnetProxy.waitForDeployment();
  const subnetAddr = await subnetProxy.getAddress();
  console.log("Subnet proxy:", subnetAddr);

  // 附加 Subnet 接口到代理
  const subnet = Subnet.attach(subnetAddr);

  // 设置 BLS 的 target 为 Subnet 代理地址
  console.log("\n--- 4b. BLS.setTarget(Subnet) ---");
  let tx = await bls.setTarget(subnetAddr);
  await tx.wait();
  console.log("BLS.setTarget done");

  // ── 5. Token.setSubnet(Subnet) ──────────────────────────
  console.log("\n--- 5. Token.setSubnet(Subnet) ---");
  tx = await token.setSubnet(subnetAddr);
  await tx.wait();
  console.log("Token.setSubnet done");

  // ── 6. Subnet.setTokenContract(Token) ────────────────────
  console.log("\n--- 6. Subnet.setTokenContract(Token) ---");
  tx = await subnet.setTokenContract(tokenAddr);
  await tx.wait();
  console.log("Subnet.setTokenContract done");

  // ── 7. 设置 sideChainMultiKey (tee-provider 的 ECDSA 地址) ──
  const ecdsaKey = process.env.ECDSA_KEY;
  if (ecdsaKey) {
    const provider = hre.ethers.provider;
    const wallet = new hre.ethers.Wallet(ecdsaKey, provider);
    console.log("\n--- 7. Set sideChainMultiKey ---");
    console.log("tee-provider ECDSA address:", wallet.address);

    tx = await subnet.setCloudContract(wallet.address);
    await tx.wait();
    console.log("Subnet.setCloudContract done (cloudContract = tee-provider)");
  }

  // ── 输出配置 ────────────────────────────────────────────
  console.log("\n========================================");
  console.log("Deployment Complete (UUPS Upgradeable)");
  console.log("========================================");
  console.log("Token : ", tokenAddr);
  console.log("Subnet: ", subnetAddr);
  console.log("BLS   : ", blsAddr);
  console.log("");
  console.log("Add to .env or environment:");
  console.log(`  TOKEN_CONTRACT_ADDRESS=${tokenAddr}`);
  console.log(`  SUBNET_CONTRACT_ADDRESS=${subnetAddr}`);
  console.log(`  BLS_CONTRACT_ADDRESS=${blsAddr}`);
  console.log("");
  console.log("Upgrade commands:");
  console.log("  npx hardhat run hack/upgrade_subnet_token.js --network ...");
}

main()
  .then(() => process.exit(0))
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
