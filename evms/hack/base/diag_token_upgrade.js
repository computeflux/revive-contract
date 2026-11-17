// 临时诊断：为什么 Token 代理 upgradeToAndCall 执行 revert "amount must be > 0"
const { ethers, parseChain, connect } = require("./common");

const PROXY = "0xC7701fC65125C0a5cdBc760806F8fCf33D2aD2E0";
const NEW_IMPL_RUN = "0xe084c4ef3306c0620b0fe95478a316b096c16b8b";
const IMPL_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";
const ADMIN_SLOT = "0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103";

async function main() {
  const chain = parseChain(["--chain", "baseSepolia"]);
  const ctx = await connect(chain);
  const p = ctx.provider;

  const implRaw = await p.getStorage(PROXY, IMPL_SLOT);
  const adminRaw = await p.getStorage(PROXY, ADMIN_SLOT);
  const impl = "0x" + implRaw.slice(26);
  const admin = "0x" + adminRaw.slice(26);
  console.log("proxy          :", PROXY);
  console.log("impl slot      :", implRaw);
  console.log("impl           :", impl);
  console.log("admin slot     :", adminRaw);
  console.log("admin          :", admin);
  console.log("proxy codesize :", (await p.getCode(PROXY)).length / 2 - 1);
  console.log("impl  codesize :", (await p.getCode(impl)).length / 2 - 1);
  console.log("newImpl codesize:", (await p.getCode(NEW_IMPL_RUN)).length / 2 - 1);

  // 代理接管 reads
  const iface = new ethers.Interface([
    "function proxiableUUID() view returns (bytes32)",
    "function owner() view returns (address)",
    "function upgradeTo(address)",
    "function upgradeToAndCall(address,bytes)",
    "function getLatestNonce() view returns (uint64)",
  ]);

  for (const fn of ["proxiableUUID", "owner"]) {
    try {
      const out = await p.call({ to: PROXY, data: iface.encodeFunctionData(fn) });
      console.log(`${fn}:`, out);
    } catch (e) {
      console.log(`${fn}: ERR`, e.shortMessage || e.message);
    }
  }

  // 新实现自身
  try {
    const out = await p.call({ to: NEW_IMPL_RUN, data: iface.encodeFunctionData("proxiableUUID") });
    console.log("newImpl.proxiableUUID:", out);
  } catch (e) {
    console.log("newImpl.proxiableUUID: ERR", e.shortMessage || e.message);
  }

  // 差别对比：upgradeTo vs upgradeToAndCall（只 eth_call，不发交易）
  for (const [name, data] of [
    ["upgradeTo(impl)", iface.encodeFunctionData("upgradeTo", [NEW_IMPL_RUN])],
    ["upgradeToAndCall(impl,0x)", iface.encodeFunctionData("upgradeToAndCall", [NEW_IMPL_RUN, "0x"])],
  ]) {
    try {
      const out = await p.call({ from: ctx.address, to: PROXY, data });
      console.log(`${name}: OK`, out);
    } catch (e) {
      console.log(`${name}: REVERT`, e.shortMessage || e.message);
      const d = e.data || e.info?.error?.data;
      if (d) console.log("   revert data:", d);
    }
  }

  // 代理上是否已有 relay 存储（升级后应能读到 0）
  try {
    const out = await p.call({ to: PROXY, data: iface.encodeFunctionData("getLatestNonce") });
    console.log("getLatestNonce (proxy):", out);
  } catch (e) {
    console.log("getLatestNonce (proxy): 尚不存在 ->", e.shortMessage || e.message);
  }
}

main().catch((e) => {
  console.error("FATAL:", e.message || e);
  process.exit(1);
});
