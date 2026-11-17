#!/usr/bin/env node
// Set BaseBLS.target.
//
// NOTE: setTarget(address) exists in EthBLS.sol and in BaseBLS.sol only after
// this change — the currently deployed BaseBLS 0x2921... does NOT implement it
// and cannot be repointed. Redeploy BaseBLS to use this script.
//
// Usage:
//   node hack/base/set_target.js --chain baseSepolia --target 0x...
//   ... [--bls 0x...] [--dry]
//
// setTarget is onlyTarget: the signer must be the contract's current target.
const {
  parseChain,
  connect,
  artifact,
  loadDeployments,
  ethers,
} = require("./common");

function arg(name, def) {
  const i = process.argv.indexOf(name);
  return i !== -1 && process.argv[i + 1] ? process.argv[i + 1] : def;
}

function latestBls(chainName) {
  const all = loadDeployments();
  const list = all[chainName] || [];
  for (let i = list.length - 1; i >= 0; i--) {
    if (list[i].contract === "BaseBLS" && list[i].address) return list[i].address;
  }
  return null;
}

(async () => {
  const argv = process.argv.slice(2);
  const chain = parseChain(argv);
  const dry = argv.includes("--dry");

  const blsAddr = arg("--bls", latestBls(chain.name));
  if (!blsAddr) throw new Error("no BaseBLS address: pass --bls or deploy first");
  if (!ethers.isAddress(blsAddr)) throw new Error(`invalid --bls ${blsAddr}`);

  const target = arg("--target");
  if (!target || !ethers.isAddress(target)) {
    throw new Error("--target 0x... required");
  }

  const abi = artifact("BaseBLS").abi;
  if (!abi.some((f) => f.type === "function" && f.name === "setTarget")) {
    throw new Error(
      "BaseBLS artifact has no setTarget — the deployed contract cannot be repointed; " +
        "add setTarget to contracts/BaseBLS.sol, recompile and redeploy"
    );
  }

  const ctx = await connect(chain);
  const bls = new ethers.Contract(blsAddr, abi, ctx.signer);
  const current = await bls.getFunction("target").staticCall();
  const code = await ctx.provider.getCode(target);

  console.log(`chain:   ${chain.name} (${ctx.chainId})`);
  console.log(`BLS:     ${blsAddr}`);
  console.log(`signer:  ${ctx.address}`);
  console.log(`current: ${current}`);
  console.log(`new:     ${target} (code ${(code.length - 2) / 2} bytes)`);

  if (current.toLowerCase() === target.toLowerCase()) {
    console.log("target already set; nothing to do");
    return;
  }
  if (current.toLowerCase() !== ctx.address.toLowerCase()) {
    throw new Error("signer is not the current target — setTarget is onlyTarget and would revert");
  }
  if (code === "0x") {
    console.warn("warning: new target has no contract code — submit/batchSubmit will be a no-op");
  }
  if (dry) {
    console.log("[dry-run] not sending");
    return;
  }

  const txr = await bls.setTarget(target);
  console.log(`tx:      ${txr.hash}`);
  const rc = await txr.wait();
  console.log(`status:  ${rc.status}  gas: ${rc.gasUsed}  block: ${rc.blockNumber}`);

  const after = await bls.getFunction("target").staticCall();
  console.log(`after:   ${after}`);
  if (after.toLowerCase() !== target.toLowerCase()) throw new Error("readback mismatch");
})().catch((e) => {
  console.error("error:", e.message || e);
  process.exit(1);
});
