// Gas report for the BLS gateway flows (Base fork, Isthmus).
// Measures exact gasUsed per flow, separating success / revert paths.
const hre = require("hardhat");
const fs = require("fs");
const path = require("path");
const { vectorsPath } = require("./common");

async function main() {
  const v = JSON.parse(fs.readFileSync(vectorsPath(), "utf8"));
  const [dep] = await hre.ethers.getSigners();
  await hre.network.provider.request({
    method: "hardhat_setBalance",
    params: [dep.address, "0x21e19e0c9bab2400000"],
  });

  const MockTarget = await hre.ethers.getContractFactory("MockTarget", dep);
  const BaseBLS = await hre.ethers.getContractFactory("BaseBLS", dep);
  const SolTest = await hre.ethers.getContractFactory("SolTest", dep);

  const target = await MockTarget.deploy();
  await target.waitForDeployment();
  const targetAddr = await target.getAddress();
  const g2gen = "0x" + v.vectors[0].pairingInput.slice(2 + 512 * 2);

  const rows = [];
  async function measure(name, fn) {
    try {
      const tx = await fn();
      try {
        const rc = await tx.wait();
        rows.push([name, rc.status, rc.gasUsed.toString()]);
      } catch (e) {
        const rc = e && e.receipt;
        rows.push([name, rc ? rc.status : "THREW", rc ? rc.gasUsed.toString() : "-"]);
      }
    } catch (e) {
      const rc = e && e.receipt;
      rows.push([name, rc ? rc.status : "SEND-THREW", rc ? rc.gasUsed.toString() : "-"]);
    }
  }

  // deploy gas
  const bls = await BaseBLS.deploy(targetAddr, g2gen);
  await bls.waitForDeployment();
  const deployReceipt = await hre.ethers.provider.getTransactionReceipt(bls.deploymentTransaction().hash);
  rows.push(["deploy BaseBLS", deployReceipt.status, deployReceipt.gasUsed.toString()]);

  const sol = await SolTest.deploy();
  await sol.waitForDeployment();
  const solReceipt = await hre.ethers.provider.getTransactionReceipt(sol.deploymentTransaction().hash);
  rows.push(["deploy SolTest", solReceipt.status, solReceipt.gasUsed.toString()]);

  await hre.network.provider.request({ method: "hardhat_impersonateAccount", params: [targetAddr] });
  await hre.network.provider.request({ method: "hardhat_setBalance", params: [targetAddr, "0x1000000000000000000"] });
  const signer = await hre.ethers.getImpersonatedSigner(targetAddr);
  await measure("setAggPubkey", () => bls.connect(signer).setAggPubkey(v.manifest.aggPubkey));

  const single = v.vectors.find((x) => x.name === "submit-single");
  const batch = v.vectors.find((x) => x.name === "batchSubmit-batch");
  const callfail = v.vectors.find((x) => x.name === "submit-callfail");

  // success paths
  await measure("submit (good, 1 call)", () =>
    bls.connect(dep).submit(single.targetCalldatas[0], single.hmG1, single.negSig)
  );
  await measure("batchSubmit (good, 3 calls)", () =>
    bls.connect(dep).batchSubmit(batch.targetCalldatas, batch.hmG1, batch.negSig)
  );

  // pairing precompile gas in EVM (SolTest): good vs tampered (precompile error path)
  await measure("assertPair good", () =>
    sol.connect(dep).assertPair(single.pairingInput, true)
  );
  const pi = single.pairingInput.slice(2);
  const cut = (a, b) => "0x" + pi.slice(a * 2, b * 2);
  const tamperLast = (hex) =>
    "0x" + ((BigInt(hex) ^ 1n).toString(16)).padStart(hex.length - 2, "0");
  const tampered =
    cut(0, 128) + cut(128, 384).slice(2) + tamperLast(cut(384, 512)).slice(2) + cut(512, 768).slice(2);
  await measure("assertPair tampered", () =>
    sol.connect(dep).assertPair(tampered, false)
  );

  // revert paths (explicit gas cap so gasUsed reflects actual usage)
  await measure("submit (tampered negSig) -> revert", () =>
    bls.connect(dep).submit(single.targetCalldatas[0], single.hmG1,
      single.negSig.slice(0, -2) + ((parseInt(single.negSig.slice(-2), 16) ^ 1).toString(16).padStart(2, "0")),
      { gasLimit: 60000000 })
  );
  await measure("submit (target reverts) -> CallFailed", () =>
    bls.connect(dep).submit(callfail.targetCalldatas[0], callfail.hmG1, callfail.negSig,
      { gasLimit: 60000000 })
  );

  // print table
  const w = [34, 6, 12];
  console.log("flow".padEnd(w[0]), "st".padEnd(w[1]), "gasUsed".padEnd(w[2]));
  for (const r of rows) console.log(r[0].padEnd(w[0]), String(r[1]).padEnd(w[1]), r[2].padStart(10));
}

main().catch((e) => {
  console.error("GAS FAILED:", e.shortMessage || e.message); if (e.stack) console.error(e.stack.split("\n").slice(0,6).join("\n"));
  process.exit(1);
});
