// Diagnostic: why does setAggPubkey not persist on Base Sepolia?
const {ethers, parseChain, connect} = require("/home/wetee/work/Query-Web3/revive-contract/evms/hack/base/common");

async function main() {
  const chain = parseChain(["--chain", "baseSepolia"]);
  const ctx = await connect(chain);
  const v = require("/home/wetee/work/cathome/workflow/apps/tee-node/ext/blscontract/vectors/vectors.json");
  const g2gen = "0x" + v.vectors[0].pairingInput.slice(2 + 512 * 2);

  // Deploy a fresh BaseBLS with target = deployer EOA (no shim)
  const art = require("/home/wetee/work/Query-Web3/revive-contract/evms/artifacts/contracts/BaseBLS.sol/BaseBLS.json");
  const f = new ethers.ContractFactory(art.abi, art.bytecode, ctx.signer);
  const bls = await f.deploy(ctx.address, g2gen);
  await bls.waitForDeployment();
  const addr = await bls.getAddress();
  console.log("new BaseBLS (target=EOA):", addr);

  // setAggPubkey directly from EOA (= target)
  const rc = await (await bls.setAggPubkey(v.manifest.aggPubkey)).wait();
  console.log("setAggPubkey tx:", rc.hash, "status", rc.status, "gasUsed", rc.gasUsed.toString());

  // read back
  const got = await bls.aggPubkey();
  console.log("aggPubkey readback len:", (got.length - 2) / 2, "match:", got.toLowerCase() === v.manifest.aggPubkey.toLowerCase());

  // raw storage probe: slot1 = length word of aggPubkey (dynamic bytes at slot 1)
  const s1 = await ctx.provider.getStorage(addr, ethers.zeroPadValue("0x01", 32));
  console.log("slot1 (aggPubkey length):", s1);
}

main().catch(e => { console.error("FATAL:", e.message || e); process.exit(1); });
