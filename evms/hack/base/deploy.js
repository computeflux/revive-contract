#!/usr/bin/env node
// Base 链部署脚本：Token（UUPS 代理）/ BaseBLS / 链上集成 / Token 升级。
//
// 用法（在 evms/ 下，先 npx hardhat compile --config hardhat.config.base.js）:
//   node hack/base/deploy.js --chain baseSepolia --deploy-token [--dry-run]
//   node hack/base/deploy.js --chain baseSepolia --deploy-bls --target <addr> [--dry-run]
//   node hack/base/deploy.js --chain baseSepolia --integrate --token <proxy> --bls <addr> [--dry-run]
//   node hack/base/deploy.js --chain baseSepolia --upgrade-token --token <proxy> [--dry-run]
//
// 环境变量（evms/.env）: PRIVATE_KEY, BASE_RPC_URL, BASE_SEPOLIA_RPC_URL
const {
  ethers,
  parseChain,
  chainConfig,
  connect,
  artifact,
  proxyFactory,
  vectors,
  g2Generator,
  saveDeployment,
  tx,
  fmtEth,
} = require("./common");

function arg(name) {
  const i = process.argv.indexOf(`--${name}`);
  return i !== -1 ? process.argv[i + 1] : undefined;
}
const flag = (name) => process.argv.includes(`--${name}`);

// Apply Token genesis parameters from hack/base/configs/<chain>.json after
// deployment (owner null = deployer). Skips no-op calls; fails on chain mismatch.
async function applyTokenGenesis(ctx, chain, token, dry) {
  const cfg = chainConfig(chain.name);
  if (!cfg || !cfg.token) {
    console.log("  (no token genesis config for chain, skipping)");
    return;
  }
  const t = cfg.token;
  const owner = t.owner || ctx.address;
  const calls = [];
  if (BigInt(t.rate) !== (await token.rate())) {
    calls.push(["setRate", [t.rate]]);
  }
  if (BigInt(t.token_unit) !== (await token.tokenUnit())) {
    calls.push(["setTokenUnit", [t.token_unit]]);
  }
  if ((await token.nativeActive()) !== t.native_active) {
    calls.push(["setNativeActive", [t.native_active]]);
  }
  for (const e of t.erc20_tokens || []) {
    calls.push(["setERC20Token", [e.token, e.active, e.rate, e.unit]]);
  }

  if (dry) {
    console.log(`[dry-run] applyTokenGenesis(${chain.name}): owner=${owner}`);
    for (const [m, a] of calls) console.log(`  Token.${m}(${a.join(", ")})`);
    return;
  }

  await tx(await token.transferOwnership(owner), "Token.transferOwnership");
  for (const [m, a] of calls) {
    await tx(await token[m](...a), `Token.${m}`);
  }
}

async function deployToken(ctx, chain, dry) {
  const { signer } = ctx;
  const Token = artifact("Token");

  if (dry) {
    console.log("[dry-run] deployToken:");
    console.log(`  impl: Token (${(Token.bytecode.length - 2) / 2} bytes)`);
    console.log(`  proxy: ERC1967Proxy(impl, initialize(${signer.address}))`);
    return;
  }

  console.log("--- Deploy Token (UUPS) ---");
  const implF = new ethers.ContractFactory(
    Token.abi,
    Token.bytecode,
    signer
  );
  const impl = await implF.deploy();
  await impl.waitForDeployment();
  const implAddr = await impl.getAddress();
  console.log("Token implementation:", implAddr);

  const initData = implF.interface.encodeFunctionData("initialize", [
    signer.address,
  ]);
  const Proxy = proxyFactory(signer);
  const proxy = await Proxy.deploy(implAddr, initData);
  await proxy.waitForDeployment();
  const tokenAddr = await proxy.getAddress();
  console.log("Token proxy:", tokenAddr);

  // UUPS initialize() sets rate=4000 / tokenUnit=1e18 defaults; apply the
  // per-chain genesis config on top (owner defaults to the deployer).
  if (!dry) {
    const token = new ethers.Contract(tokenAddr, Token.abi, signer);
    await applyTokenGenesis(ctx, chain, token, dry);
  }

  saveDeployment(chain.name, {
    contract: "Token",
    implementation: implAddr,
    proxy: tokenAddr,
    deployer: signer.address,
    chainId: ctx.chainId,
  });
  return tokenAddr;
}

async function deployBLS(ctx, chain, dry) {
  const { signer } = ctx;
  const BaseBLS = artifact("BaseBLS");
  const v = vectors();
  // Production deploys must pass the live DKG group key explicitly; falling
  // back to vectors.json silently wires a test-only key and every submit will
  // revert with BLSSignatureInvalid.
  const aggFlag = arg("agg-pubkey");
  if (!aggFlag) {
    console.log(
      "  WARN: --agg-pubkey not set; using vectors.json manifest.aggPubkey (TEST KEY)"
    );
  }
  const aggPubkey = aggFlag || v.manifest.aggPubkey;
  if (!/^0x[0-9a-fA-F]{512}$/.test(aggPubkey)) {
    throw new Error(`--agg-pubkey must be 256 bytes (0x + 512 hex), got ${(aggPubkey.length - 2) / 2} bytes`);
  }
  const g2gen = g2Generator(v);
  const target = arg("target");
  if (!target) throw new Error("--target <addr> required for --deploy-bls");

  if (dry) {
    console.log("[dry-run] deployBLS:");
    console.log(`  BaseBLS(target=${target}, g2Generator=${g2gen.slice(0, 18)}... 256B)`);
    console.log(`  setAggPubkey(${aggPubkey.slice(0, 18)}... 256B)`);
    return;
  }

  console.log("--- Deploy BaseBLS ---");
  const f = new ethers.ContractFactory(
    BaseBLS.abi,
    BaseBLS.bytecode,
    signer
  );
  const bls = await f.deploy(target, g2gen);
  await bls.waitForDeployment();
  const blsAddr = await bls.getAddress();
  console.log("BaseBLS:", blsAddr);

  // setAggPubkey is onlyTarget: when target is the deployer EOA we can set it
  // directly; with a contract target the owner must drive it through the
  // target (e.g. `--integrate` wiring + the DKG pubkey update flow).
  if (target.toLowerCase() === signer.address.toLowerCase()) {
    await tx(await bls.setAggPubkey(aggPubkey), "setAggPubkey");
  } else {
    console.log(
      "  (target is not the deployer — aggPubkey must be set by the target)"
    );
  }

  saveDeployment(chain.name, {
    contract: "BaseBLS",
    address: blsAddr,
    target,
    aggPubkey,
    deployer: signer.address,
    chainId: ctx.chainId,
  });
  return blsAddr;
}

async function integrate(ctx, chain, dry) {
  const { signer, provider } = ctx;
  const tokenAddr = arg("token");
  const blsAddr = arg("bls");
  if (!tokenAddr || !blsAddr) throw new Error("--integrate needs --token and --bls");

  const Token = artifact("Token");
  const BaseBLS = artifact("BaseBLS");
  const token = new ethers.Contract(tokenAddr, Token.abi, signer);
  const bls = new ethers.Contract(blsAddr, BaseBLS.abi, signer);

  const code = await provider.getCode(tokenAddr);
  const subnet =
    code && code !== "0x" ? await token.subnet() : ethers.ZeroAddress;
  const target =
    code && code !== "0x" ? await bls.getFunction("target").staticCall() : ethers.ZeroAddress;
  if (!code || code === "0x") {
    console.log(
      `  warn: --token ${tokenAddr} has no code (not deployed on this network?)`
    );
  }
  console.log(`current: Token.subnet=${subnet} BaseBLS.target=${target}`);

  if (dry) {
    console.log("[dry-run] integrate:");
    console.log(`  Token.setSubnet(${blsAddr})`);
    console.log(`  BaseBLS.setTarget(${tokenAddr})`);
    return;
  }

  if (subnet.toLowerCase() !== blsAddr.toLowerCase()) {
    await tx(await token.setSubnet(blsAddr), "Token.setSubnet");
  } else {
    console.log("  Token.subnet already set");
  }
  if (target.toLowerCase() !== tokenAddr.toLowerCase()) {
    await tx(await bls.setTarget(tokenAddr), "BaseBLS.setTarget");
  } else {
    console.log("  BaseBLS.target already set");
  }

  saveDeployment(chain.name, {
    contract: "integrate",
    token: tokenAddr,
    bls: blsAddr,
    chainId: ctx.chainId,
  });
}

async function upgradeToken(ctx, chain, dry) {
  const { signer } = ctx;
  const tokenAddr = arg("token");
  if (!tokenAddr) throw new Error("--upgrade-token needs --token <proxy>");

  const Token = artifact("Token");
  if (dry) {
    console.log("[dry-run] upgradeToken:");
    console.log(`  new impl: Token (${(Token.bytecode.length - 2) / 2} bytes)`);
    console.log(`  Token.upgradeTo(${tokenAddr})`);
    return;
  }

  console.log("--- Upgrade Token implementation ---");
  const implF = new ethers.ContractFactory(Token.abi, Token.bytecode, signer);
  const impl = await implF.deploy();
  await impl.waitForDeployment();
  const implAddr = await impl.getAddress();
  console.log("new implementation:", implAddr);

  const token = new ethers.Contract(tokenAddr, Token.abi, signer);
  // 不用 upgradeToAndCall(impl, "0x")：OZ 的 UUPS upgradeToAndCall 以 forceCall=true
  // 强制 delegatecall 进新实现，空 data 会命中 Token.receive()，其
  // require(msg.value > 0) 使升级必然 revert "amount must be > 0"。
  await tx(await token["upgradeTo(address)"](implAddr), "UUPS upgradeTo");
  saveDeployment(chain.name, {
    contract: "Token-upgrade",
    proxy: tokenAddr,
    implementation: implAddr,
    chainId: ctx.chainId,
  });
}

async function main() {
  const chain = parseChain(process.argv);
  const dry = flag("dry-run");
  const ctx = await connect(chain);
  console.log(
    `Network: ${chain.name} chainId=${ctx.chainId} deployer=${ctx.address}` +
      ` balance=${fmtEth(ctx.balance)} ETH` +
      (dry ? " [DRY-RUN]" : "")
  );

  if (flag("deploy-token")) await deployToken(ctx, chain, dry);
  else if (flag("deploy-bls")) await deployBLS(ctx, chain, dry);
  else if (flag("integrate")) await integrate(ctx, chain, dry);
  else if (flag("upgrade-token")) await upgradeToken(ctx, chain, dry);
  else {
    console.error(
      "nothing to do: --deploy-token | --deploy-bls --target <addr> | --integrate --token <p> --bls <a> | --upgrade-token --token <p>"
    );
    process.exit(2);
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
