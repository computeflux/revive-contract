// Hardhat 2.x provider patch: fork OP-Stack chains (e.g. Base, chainId 8453) at
// blocks whose hardfork (Fusaka / Osaka with EIP-2537 BLS12-381 precompiles) is
// not in EDR's bundled chain DB. We create the provider with EDR's OP chain
// spec and inject the hardfork activation history via the EDR field
// `network.chainOverrides[].hardforkActivationOverrides` (the top-level
// `hardforkActivations` key is silently ignored by EDR).
//
// Usage: require("./hack/base/l1-fork-provider") from the hardhat config, and set
//   HH_L1_FORK_CHAIN_ID  (default 8453)
// The OP chain spec in this EDR build tops out at `Isthmus`, so use OP hardfork
// names (Bedrock / Regolith / Canyon / Ecotone / Fjord / Granite / Holocene /
// Isthmus) in the env var HH_OP_FORK_ACTIVATIONS, e.g.:
//   HH_OP_FORK_ACTIVATIONS="Bedrock:0,Isthmus:35116176"
// (the config's chains[8453].hardforkHistory uses L1 hardfork names to satisfy
// Hardhat's schema; it is ignored by this patch.)
//
// Note: with this patch the EIP-2537 precompiles work for EVM execution
// (0x0f pairing in particular); the OP-spec 0x10 map_fp_to_g1 is NOT
// implemented in the tx/EVM path of this EDR build, only in RPC eth_call
// (fresh provider). Production EthBLS.sol._mapFpToG1 relies on Base mainnet
// Fusaka's real 0x10.
const {
  EdrContext,
  OP_CHAIN_TYPE,
  opProviderFactory,
  opHardforkFromString,
  opHardforkToString,
  ContractDecoder,
} = require("@nomicfoundation/edr");

// construction.js resolves the module namespace and destructures
// `createHardhatNetworkProvider` at call time, so patching the exports
// object intercepts provider creation for the `hardhat` network.
const providerMod = require("hardhat/internal/hardhat-network/provider/provider");

const origCreate = providerMod.createHardhatNetworkProvider;

providerMod.createHardhatNetworkProvider = async function (config, loggerConfig, artifacts) {
  const forkUrl = config.forkConfig?.jsonRpcUrl;
  const l1ChainId = Number(process.env.HH_L1_FORK_CHAIN_ID || 8453);
  // The local `hardhat` network keeps its own chainId (31337); the forked
  // chain is identified by HH_L1_FORK_CHAIN_ID. Treat any fork as the OP fork.
  const isL1Fork = forkUrl !== undefined;

  if (!isL1Fork) {
    return origCreate.call(this, config, loggerConfig, artifacts);
  }

  // Hardfork activations for the forked OP chain come from
  // `HH_OP_FORK_ACTIVATIONS` ("Name:block,Name:block").
  const rawActivations = process.env.HH_OP_FORK_ACTIVATIONS;
  if (!rawActivations) {
    throw new Error(
      "l1-fork-provider: HH_OP_FORK_ACTIVATIONS not set (e.g. Bedrock:0,Isthmus:35116176)"
    );
  }
  const hardforkActivations = rawActivations.split(",").map((entry) => {
    const [name, block] = entry.split(":");
    return {
      condition: { blockNumber: BigInt(block) },
      hardfork: opHardforkToString(opHardforkFromString(name)),
    };
  });

  const context = new EdrContext();
  await context.registerProviderFactory(OP_CHAIN_TYPE, opProviderFactory());

  const providerConfig = {
    allowBlocksWithSameTimestamp: config.allowBlocksWithSameTimestamp ?? false,
    allowUnlimitedContractSize: config.allowUnlimitedContractSize,
    bailOnCallFailure: false,
    bailOnTransactionFailure: false,
    blockGasLimit: BigInt(config.blockGasLimit),
    defaultTransactionGasLimit: BigInt(config.blockGasLimit),
    chainId: BigInt(config.chainId),
    coinbase: Buffer.from((config.coinbase ?? providerMod.DEFAULT_COINBASE).slice(2), "hex"),
    network: {
      blockNumber:
        config.forkConfig.blockNumber !== undefined
          ? BigInt(config.forkConfig.blockNumber)
          : undefined,
      cacheDir: (config.forkCachePath || "./cache") + "/edr-op-fork",
      httpHeaders: Object.entries(config.forkConfig.httpHeaders ?? {}).map(([name, value]) => ({ name, value })),
      url: forkUrl,
      chainOverrides: [
        {
          chainId: BigInt(l1ChainId),
          name: "base",
          hardforkActivationOverrides: hardforkActivations,
        },
      ],
    },
    // For forked blocks, storage overrides are not supported; system contracts
    // come from the forked chain state itself. Funded genesis accounts
    // (Hardhat's default dev accounts) are injected on top of the fork state.
    genesisState: (config.genesisAccounts ?? []).map((account) => ({
      address: Buffer.from(
        require("@ethereumjs/util").privateToAddress(
          Buffer.from(account.privateKey.slice(2), "hex")
        ),
      ),
      balance: BigInt(account.balance),
      code: new Uint8Array(),
    })),
    hardfork: opHardforkToString(opHardforkFromString("Isthmus")),
    initialBaseFeePerGas:
      config.initialBaseFeePerGas !== undefined ? BigInt(config.initialBaseFeePerGas) : undefined,
    minGasPrice: BigInt(config.minGasPrice),
    mining: {
      autoMine: config.automine,
      interval: undefined,
      memPool: { order: "Priority" },
    },
    networkId: BigInt(config.networkId),
    observability: {},
    ownedAccounts: (config.genesisAccounts ?? []).map((account) => account.privateKey),
    precompileOverrides: [],
    // Osaka EIP-7825 tx gas cap is below many local test gas limits.
    transactionGasCap: false,
  };

  const loggerCfg = {
    enable: loggerConfig.enabled,
    decodeConsoleLogInputsCallback: () => [],
    printLineCallback: () => {},
  };
  const subscriptionCfg = { subscriptionCallback: () => {} };
  const contractDecoder = ContractDecoder.withContracts({});

  const provider = await context.createProvider(
    OP_CHAIN_TYPE,
    providerConfig,
    loggerCfg,
    subscriptionCfg,
    contractDecoder
  );

  // Minimal EthereumJS VM shim so Hardhat internals (tracing events, etc.)
  // keep working.
  const { getMinimalEthereumJsVm } = require("hardhat/internal/hardhat-network/provider/vm/minimal-vm");
  const minimalNode = { _vm: getMinimalEthereumJsVm(provider) };

  const EdrProviderWrapper = providerMod.EdrProviderWrapper;
  return new EdrProviderWrapper(
    provider,
    providerConfig,
    loggerCfg,
    minimalNode,
    subscriptionCfg,
    config.genesisAccounts,
    config.forkCachePath,
    []
  );
};
