#!/usr/bin/env node
// Base 链状态查询：Token（UUPS 代理）与 BaseBLS 的链上配置/余额。
//
// 用法:
//   node hack/base/state.js --chain baseSepolia [--token <proxy>] [--bls <addr>]
const {
  ethers,
  parseChain,
  connect,
  artifact,
  vectors,
  g2Generator,
  fmtEth,
} = require("./common");

function arg(name) {
  const i = process.argv.indexOf(`--${name}`);
  return i !== -1 ? process.argv[i + 1] : undefined;
}

async function precompiles(provider) {
  // eth_call the EIP-2537 precompiles: 0x0f pairing, 0x10 map_fp_to_g1.
  const out = {};
  const calls = {
    "0x000000000000000000000000000000000000000f": "0x" + "00".repeat(768),
    "0x0000000000000000000000000000000000000010": "0x" + "00".repeat(64),
  };
  for (const [name, data] of Object.entries(calls)) {
    try {
      const r = await provider.call({ to: name, data });
      out[name.slice(26)] = `ok (${(r.length - 2) / 2}B)`;
    } catch (e) {
      out[name.slice(26)] = `ERR ${String(e.shortMessage || e.message).slice(0, 60)}`;
    }
  }
  return out;
}

async function main() {
  const chain = parseChain(process.argv);
  const ctx = await connect(chain);
  const { provider } = ctx;

  const block = await provider.getBlockNumber();
  console.log(`Network ${chain.name} chainId=${ctx.chainId} block=${block}`);
  console.log(`Precompiles:`, await precompiles(provider));

  const tokenAddr = arg("token");
  if (tokenAddr) {
    const tokenCode = await provider.getCode(tokenAddr);
    if (tokenCode === "0x") {
      console.log(`\nToken ${tokenAddr}: no code at this address`);
    } else {
    const Token = artifact("Token");
    const token = new ethers.Contract(tokenAddr, Token.abi, provider);
    const [
      owner,
      subnet,
      rate,
      tokenUnit,
      nativeActive,
      rfr,
      wfr,
      erc20Count,
      bal,
    ] = await Promise.all([
      token.owner(),
      token.subnet(),
      token.rate(),
      token.tokenUnit(),
      token.nativeActive(),
      token.rechargeFeeRate(),
      token.withdrawFeeRate(),
      token.getERC20Count(),
      provider.getBalance(tokenAddr),
    ]);
    console.log(`\nToken ${tokenAddr}`);
    console.log(`  owner=${owner} subnet=${subnet}`);
    console.log(
      `  rate=${rate} tokenUnit=${tokenUnit} nativeActive=${nativeActive}` +
        ` rechargeFeeRate=${rfr} withdrawFeeRate=${wfr}`
    );
    console.log(`  native balance=${fmtEth(bal)} ETH erc20Count=${erc20Count}`);
    const n = Number(erc20Count);
    if (n > 0) {
      const { tokens, actives, rates, units } = await token.getERC20List();
      for (let i = 0; i < n; i++) {
        console.log(
          `  erc20[${i}] ${tokens[i]} active=${actives[i]} rate=${rates[i]} unit=${units[i]}`
        );
      }
    }
    const w = arg("withdrawal-nonce");
    if (w !== undefined) {
      const p = await token.getPendingWithdrawal(BigInt(w));
      console.log(
        `  pendingWithdrawal[${w}] user=${p.user} token=${p.token}` +
          ` amount=${p.amount} status=${p.status}`
      );
    }
    }
  }

  const blsAddr = arg("bls");
  if (blsAddr) {
    const blsCode = await provider.getCode(blsAddr);
    if (blsCode === "0x") {
      console.log(`\nBaseBLS ${blsAddr}: no code at this address`);
    } else {
    const BaseBLS = artifact("BaseBLS");
    const bls = new ethers.Contract(blsAddr, BaseBLS.abi, provider);
    const [target, aggPubkey, g2gen, bal] = await Promise.all([
      bls.getFunction("target").staticCall(),
      bls.aggPubkey(),
      bls.g2Generator(),
      provider.getBalance(blsAddr),
    ]);
    console.log(`\nBaseBLS ${blsAddr}`);
    console.log(`  target=${target}`);
    console.log(`  aggPubkey=${aggPubkey}`);
    console.log(`  g2Generator=${g2gen.slice(0, 34)}... (${g2gen.length / 2 - 1}B)`);
    console.log(`  native balance=${fmtEth(bal)} ETH`);

    // Compare with the local vectors.json DKG public key (test setup).
    try {
      const v = vectors();
      console.log(
        `  vectors.json aggPubkey match: ${
          aggPubkey.toLowerCase() === v.manifest.aggPubkey.toLowerCase()
        } (g2gen match: ${g2gen === g2Generator(v)})`
      );
    } catch (_) {}
    }
  }

  if (!tokenAddr && !blsAddr) console.log("\n(no --token / --bls given)");
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
