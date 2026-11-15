// Hardhat config for Base (OP-Stack) — BLS gateway (EIP-2537) + Token.
// Keep all Base-related network/fork configuration here; hardhat.config.polkadot.js
// is for Polkadot Hub only.
//
// Common commands:
//   npx hardhat compile --config hardhat.config.base.js
//   HH_FORK_URL=https://sepolia.base.org HH_FORK_BLOCK=58171467 \
//     HH_L1_FORK_CHAIN_ID=84532 HH_OP_FORK_ACTIVATIONS="Bedrock:0,Isthmus:0" \
//     npx hardhat test --config hardhat.config.base.js test/BLS.js
//   （npm run test:base 已内置这组默认值）
require("@nomicfoundation/hardhat-toolbox");
require("@nomicfoundation/hardhat-verify");
require("dotenv").config();
// Base fork: build the provider with EDR's OP chain spec so that the OP
// hardfork history (Isthmus = EIP-2537 BLS precompiles) applies at the fork
// block. Reads HH_OP_FORK_ACTIVATIONS / HH_L1_FORK_CHAIN_ID.
require("./hack/base/l1-fork-provider");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    compilers: [
      {
        version: "0.8.19",
        settings: {
          optimizer: {
            enabled: true,
            runs: 200,
          },
        },
      },
      {
        version: "0.8.20",
        settings: {
          optimizer: {
            enabled: true,
            runs: 200,
          },
          viaIR: true, // 解决 Stack too deep 错误
        },
      },
    ],
  },
  networks: {
    // Base fork（本地网络 + EDR OP provider，hack/base/l1-fork-provider.js patch）：
    //   默认测试环境 Base Sepolia：HH_FORK_URL=https://sepolia.base.org
    //   HH_FORK_BLOCK=<Isthmus 之后的区块>，HH_L1_FORK_CHAIN_ID=84532
    //   HH_OP_FORK_ACTIVATIONS="Bedrock:0,Isthmus:0"（Sepolia 的 Isthmus 在
    //   区块 46516795，早于 fork 区块即全部激活）
    hardhat: {
      chainId: 31337,
      forking: process.env.HH_FORK_URL
        ? {
            url: process.env.HH_FORK_URL,
            ...(process.env.HH_FORK_BLOCK
              ? { blockNumber: parseInt(process.env.HH_FORK_BLOCK, 10) }
              : {}),
          }
        : undefined,
      // Fusaka (EIP-2537 BLS12-381 预编译) 在本地网络启用
      hardfork: "osaka",
      // EDR 的 OP chain spec 在 Isthmus 已包含 EIP-2537；激活区块由
      // hack/l1-fork-provider.js 从 HH_OP_FORK_ACTIVATIONS 注入，这里只满足 schema。
      chains: {
        8453: { hardforkHistory: {} },
        84532: { hardforkHistory: {} },
      },
    },

    // Base 主网（真实部署，Fusaka 后 0x0f/0x10 预编译可用）
    base: {
      url: process.env.BASE_RPC_URL || "https://mainnet.base.org",
      chainId: 8453,
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    },
    // Base Sepolia 测试网（Isthmus/Osaka 已激活，EIP-2537 可用）
    baseSepolia: {
      url: process.env.BASE_SEPOLIA_RPC_URL || "https://sepolia.base.org",
      chainId: 84532,
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    },

        // 本地 hardhat node
    localhost: {
      url: "http://127.0.0.1:8545",
      chainId: 31337,
    },
  },
  etherscan: {
    apiKey: {
      basesepolia: process.env.BASESCAN_API_KEY || "no-api-key-needed",
      base: process.env.BASESCAN_API_KEY || "no-api-key-needed",
    },
    customChains: [
      {
        network: "base",
        chainId: 8453,
        urls: {
          apiURL: "https://api.basescan.org/api",
          browserURL: "https://basescan.org",
        },
      },
      {
        network: "basesepolia",
        chainId: 84532,
        urls: {
          apiURL: "https://api-sepolia.basescan.org/api",
          browserURL: "https://sepolia.basescan.org",
        },
      },
    ],
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    currency: "USD",
  },
};
