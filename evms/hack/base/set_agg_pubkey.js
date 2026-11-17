#!/usr/bin/env node
// Set BaseBLS.aggPubkey.
//
// Accepts either the final 256-byte EIP-2537 G2 key, or the 96-byte compressed
// DKG group pubkey printed by tee-node as `DKG PUBKEY bls12381 ...` (converted
// here with @noble/curves; noble prints x1||x0||y1||y0, EIP-2537 needs
// x0||x1||y0||y1).
//
// Usage:
//   node hack/base/set_agg_pubkey.js --chain baseSepolia --agg-pubkey 0x<512 hex>
//   node hack/base/set_agg_pubkey.js --chain baseSepolia --dkg-pubkey 0x<192 hex>
//   ... [--bls 0x...] [--dry]
//
// setAggPubkey is onlyTarget: the signer must be the contract's current target.
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

function dkgToEip2537(hex) {
  const h = String(hex).replace(/^0x/i, "");
  if (!/^[0-9a-fA-F]+$/.test(h) || h.length !== 192) {
    throw new Error(`--dkg-pubkey must be 96 compressed bytes (192 hex), got ${h.length / 2} bytes`);
  }
  const { bls12_381 } = require("@noble/curves/bls12-381");
  const point = bls12_381.G2.ProjectivePoint.fromHex(h);
  const raw = Buffer.from(point.toRawBytes(false)); // 192 B: x1||x0||y1||y0
  if (raw.length !== 192) throw new Error(`unexpected uncompressed G2 length ${raw.length}`);
  const w = [0, 1, 2, 3].map((i) => raw.subarray(i * 48, (i + 1) * 48).toString("hex"));
  const pad = (s) => "00".repeat(16) + s;
  // EIP-2537: x0 || x1 || y0 || y1
  return "0x" + pad(w[1]) + pad(w[0]) + pad(w[3]) + pad(w[2]);
}

(async () => {
  const argv = process.argv.slice(2);
  const chain = parseChain(argv);
  const dry = argv.includes("--dry");

  const blsAddr = arg("--bls", latestBls(chain.name));
  if (!blsAddr) throw new Error("no BaseBLS address: pass --bls or deploy first");
  if (!ethers.isAddress(blsAddr)) throw new Error(`invalid --bls ${blsAddr}`);

  let agg = arg("--agg-pubkey");
  if (!agg) {
    const dkg = arg("--dkg-pubkey");
    if (!dkg) throw new Error("pass --agg-pubkey 0x<256B> or --dkg-pubkey 0x<96B>");
    agg = dkgToEip2537(dkg);
  }
  if (!/^0x[0-9a-fA-F]{512}$/.test(agg)) {
    throw new Error(`aggPubkey must be 256 bytes (0x + 512 hex), got ${(agg.length - 2) / 2} bytes`);
  }

  const abi = artifact("BaseBLS").abi;
  if (!abi.some((f) => f.type === "function" && f.name === "setAggPubkey")) {
    throw new Error("BaseBLS artifact has no setAggPubkey — compile contracts first");
  }

  const ctx = await connect(chain);
  const bls = new ethers.Contract(blsAddr, abi, ctx.signer);
  const [current, target] = await Promise.all([bls.aggPubkey(), bls.getFunction("target").staticCall()]);

  console.log(`chain:   ${chain.name} (${ctx.chainId})`);
  console.log(`BLS:     ${blsAddr}`);
  console.log(`signer:  ${ctx.address}`);
  console.log(`target:  ${target}`);
  console.log(`current: ${current}`);
  console.log(`new:     ${agg}`);

  if (current.toLowerCase() === agg.toLowerCase()) {
    console.log("aggPubkey already set; nothing to do");
    return;
  }
  if (target.toLowerCase() !== ctx.address.toLowerCase()) {
    throw new Error("signer is not the current target — setAggPubkey is onlyTarget and would revert");
  }
  if (dry) {
    console.log("[dry-run] not sending");
    return;
  }

  const txr = await bls.setAggPubkey(agg);
  console.log(`tx:      ${txr.hash}`);
  const rc = await txr.wait();
  console.log(`status:  ${rc.status}  gas: ${rc.gasUsed}  block: ${rc.blockNumber}`);

  const after = await bls.aggPubkey();
  console.log(`after:   ${after}`);
  if (after.toLowerCase() !== agg.toLowerCase()) throw new Error("readback mismatch");
})().catch((e) => {
  console.error("error:", e.message || e);
  process.exit(1);
});
