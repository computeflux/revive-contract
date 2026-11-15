// Finalize: wire Token.subnet -> EOA-target BaseBLS, verify, record.
const {ethers, parseChain, connect, saveDeployment} = require("/home/wetee/work/Query-Web3/revive-contract/evms/hack/base/common");

async function main() {
  const chain = parseChain(["--chain", "baseSepolia"]);
  const ctx = await connect(chain);
  const TOKEN = "0xb7f812414D87C5A4c1F24271500590F93c032971";
  const BLS = "0x160F27168c71413e9Eed99a83cB090EEE5Da7235"; // target = deployer EOA, aggPubkey set

  const token = new ethers.Contract(TOKEN, ["function subnet() view returns (address)", "function setSubnet(address)"], ctx.signer);
  const current = await token.subnet();
  console.log("Token.subnet before:", current);
  if (current.toLowerCase() !== BLS.toLowerCase()) {
    const rc = await (await token.setSubnet(BLS)).wait();
    console.log("Token.setSubnet:", rc.hash, "status", rc.status);
  }
  console.log("Token.subnet after:", await token.subnet());

  saveDeployment("baseSepolia", {
    contract: "integrate",
    token: TOKEN,
    bls: BLS,
    note: "Token.subnet -> BaseBLS-eoa (target=deployer EOA, aggPubkey set)",
    chainId: 84532,
  });
  console.log("recorded.");
}

main().catch(e => { console.error("FATAL:", e.message || e); process.exit(1); });
