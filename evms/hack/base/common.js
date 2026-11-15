// Shared helpers for the Base deployment/maintenance scripts (hack/base/).
// Standalone: plain ethers v6 + hardhat artifacts — no hardhat runtime needed,
// so the scripts run from any cwd (`node evms/hack/base/deploy.js ...`).
const fs = require("fs");
const path = require("path");
const { ethers } = require("ethers");

const ROOT = path.join(__dirname, "..", "..");

// Load evms/.env (PRIVATE_KEY, BASE_RPC_URL, ...).
require("dotenv").config({ path: path.join(ROOT, ".env") });

// Base private keys live in the chain config (hack/base/configs/<name>.json),
// like Polkadot's configs embed genesis keys. The config holds the key inline
// via `private_key` (raw 0x hex) — or `private_key_file` for a path (expanded
// for `~` / $ENV / relative to the config dir, keyed `NAME=value` files pick
// the `private_key_name` field, default PRIVATE_KEY). These config files are
// gitignored so secrets stay out of the repo; the committed *.example.json
// shows the shape. Keys are never printed by the scripts.
function expandPath(p, dir) {
  p = p.replace(/^~(?=$|\/)/, process.env.HOME);
  p = p.replace(/\$\{?([A-Z_][A-Z0-9_]*)\}?/gi, (_, v) => process.env[v] ?? "");
  return path.isAbsolute(p) ? p : path.join(dir, p);
}

function assertHexKey(name, val, where) {
  if (!/^0x[0-9a-fA-F]{64}$/.test(val)) {
    throw new Error(`${name} in ${where}: not a 0x hex private key`);
  }
  return val;
}

function resolveKey(name, cfg, dir) {
  if (!cfg) return null;
  // 1. Inline key directly in the config JSON.
  const inline = cfg[`${name}_key`];
  if (inline) return assertHexKey(`${name}_key`, inline.trim(), `${name}.json`);
  // 2. Pointer to an external key file (raw hex or keyed NAME=value).
  const file = cfg[`${name}_key_file`];
  if (!file) return null;
  const p = expandPath(file, dir);
  if (!fs.existsSync(p)) {
    throw new Error(`${name}_key_file not found: ${p} (from ${dir})`);
  }
  const content = fs.readFileSync(p, "utf8");
  const raw = content.trim();
  if (/^0x[0-9a-fA-F]{64}$/.test(raw)) return raw;

  // Keyed file: pick the requested field (default PRIVATE_KEY).
  const want = (cfg[`${name}_key_name`] || "PRIVATE_KEY").toUpperCase();
  for (const line of content.split(/\r?\n/)) {
    const t = line.trim();
    if (!t || t.startsWith("#")) continue;
    const eq = t.indexOf("=");
    if (eq < 0) continue;
    if (t.slice(0, eq).trim().toUpperCase() !== want) continue;
    return assertHexKey(`${name}_key_file ${want}`, t.slice(eq + 1).trim(), p);
  }
  throw new Error(`${name}_key_file ${p}: no ${want}= field (raw hex not found)`);
}

const CHAINS = {
  base: {
    chainId: 8453,
    rpc: () => process.env.BASE_RPC_URL || "https://mainnet.base.org",
    explorer: "https://basescan.org",
  },
  baseSepolia: {
    chainId: 84532,
    rpc: () => process.env.BASE_SEPOLIA_RPC_URL || "https://sepolia.base.org",
    explorer: "https://sepolia.basescan.org",
  },
};

// Base chain configs: hack/base/configs/<name>.json (committed, no secrets).
// Chain defaults (rpc / chainId / explorer) come from the config file; env
// (BASE_RPC_URL / BASE_SEPOLIA_RPC_URL / --rpc) overrides. `token` holds the
// Token genesis parameters applied after `--deploy-token` (owner null = deployer).
function chainConfig(name) {
  const p = path.join(__dirname, "configs", `${name}.json`);
  return fs.existsSync(p) ? JSON.parse(fs.readFileSync(p, "utf8")) : null;
}

function parseChain(argv) {
  const r = argv.findIndex((a) => a === "--rpc");
  if (r !== -1) {
    const url = argv[r + 1];
    return { name: "custom", chainId: null, rpc: () => url, explorer: null };
  }
  const i = argv.findIndex((a) => a === "--chain");
  const name = i !== -1 ? argv[i + 1] : "baseSepolia";
  const chain = CHAINS[name];
  if (!chain) {
    throw new Error(`unknown --chain ${name} (base | baseSepolia)`);
  }
  const cfg = chainConfig(name);
  if (cfg) {
    return {
      name,
      chainId: cfg.chain_id,
      rpc: () =>
        (name === "base"
          ? process.env.BASE_RPC_URL
          : process.env.BASE_SEPOLIA_RPC_URL) || cfg.rpc_url,
      explorer: cfg.explorer,
      keyDir: path.join(__dirname, "configs"),
      cfg,
    };
  }
  // CHAINS values have `rpc` as a function (env-resolved); bind the name too.
  return { name, ...chain };
}

// Connect a signer via RPC. Verifies chainId when known.
async function connect(chain) {
  const provider = new ethers.JsonRpcProvider(chain.rpc(), undefined, {
    staticNetwork: true,
  });
  const chainId = Number((await provider.getNetwork()).chainId);
  if (chain.chainId && chainId !== chain.chainId) {
    throw new Error(
      `RPC ${chain.rpc()} has chainId ${chainId}, expected ${chain.chainId}`
    );
  }
  // Config is the source of truth for Base keys: use the chain config's
  // inline `private_key` (or `private_key_file`) when present; env PRIVATE_KEY
  // only applies when the config has no key (so a stray evms/.env PRIVATE_KEY
  // can't hijack a chain that carries its own key).
  const cfgHasKey =
    chain.keyDir &&
    chain.cfg &&
    (chain.cfg.private_key || chain.cfg.private_key_file);
  const fileKey = cfgHasKey
    ? resolveKey("private", chain.cfg, chain.keyDir)
    : null;
  const pk = fileKey || process.env.PRIVATE_KEY;
  if (!pk) {
    throw new Error(
      `no private key — set private_key in hack/base/configs/${chain.name}.json (or env PRIVATE_KEY)`
    );
  }
  const signer = new ethers.Wallet(pk, provider);
  const balance = await provider.getBalance(signer.address);
  return {
    provider,
    signer,
    chainId,
    balance,
    address: signer.address,
  };
}

// Load a hardhat artifact (compiled with hardhat.config.base.js).
function artifact(name) {
  const p = path.join(
    ROOT,
    "artifacts",
    "contracts",
    `${name}.sol`,
    `${name}.json`
  );
  if (!fs.existsSync(p)) {
    throw new Error(
      `artifact not found: ${p} — run: npx hardhat compile --config hardhat.config.base.js`
    );
  }
  return require(p);
}

// ERC1967Proxy factory (OZ build artifact).
function proxyFactory(signer) {
  const a = require("@openzeppelin/contracts/build/contracts/ERC1967Proxy.json");
  return new ethers.ContractFactory(a.abi, a.bytecode, signer);
}

// vectors.json generated by tee-node ext/blscontract (DKG → EIP-2537).
// Owned by tee-node: ext/blscontract/vectors/vectors.json. Inject the path
// with BLS_VECTORS_DIR (dir or file); default is TEE_NODE_DIR (env, default
// $HOME/work/cathome/workflow/apps/tee-node).
function vectorsPath() {
  const teeNode =
    process.env.TEE_NODE_DIR || `${process.env.HOME}/work/cathome/workflow/apps/tee-node`;
  const base = path.join(teeNode, "ext", "blscontract", "vectors");
  const env = process.env.BLS_VECTORS_DIR;
  const p = env
    ? fs.existsSync(env) && fs.statSync(env).isDirectory()
      ? path.join(env, "vectors.json")
      : env
    : path.join(base, "vectors.json");
  if (!fs.existsSync(p)) {
    throw new Error(
      `vectors.json not found: ${p} — set BLS_VECTORS_DIR or run go test ./ext/blscontract/golang/ in tee-node`
    );
  }
  return p;
}

function vectors() {
  return JSON.parse(fs.readFileSync(vectorsPath(), "utf8"));
}

// g2Generator (256 B) = last 256 bytes of any pairing input.
function g2Generator(v) {
  return "0x" + v.vectors[0].pairingInput.slice(2 + 512 * 2);
}

const DEPLOYMENTS = path.join(__dirname, "deployments.json");

function loadDeployments() {
  if (fs.existsSync(DEPLOYMENTS))
    return JSON.parse(fs.readFileSync(DEPLOYMENTS, "utf8"));
  return {};
}

function saveDeployment(chainName, record) {
  const all = loadDeployments();
  (all[chainName] = all[chainName] || []).push({
    ...record,
    timestamp: new Date().toISOString(),
  });
  fs.writeFileSync(DEPLOYMENTS, JSON.stringify(all, null, 2) + "\n");
}

async function tx(txr, label) {
  const rc = await txr.wait();
  console.log(
    `  ${label}: tx ${rc.hash} gas=${rc.gasUsed.toString()} block=${rc.blockNumber}` +
      ` status=${rc.status}`
  );
  return rc;
}

function fmtEth(wei) {
  return ethers.formatEther(wei);
}

module.exports = {
  ROOT,
  ethers,
  parseChain,
  chainConfig,
  resolveKey,
  connect,
  artifact,
  proxyFactory,
  vectors,
  vectorsPath,
  g2Generator,
  loadDeployments,
  saveDeployment,
  tx,
  fmtEth,
};
