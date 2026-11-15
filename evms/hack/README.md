# hack/ 脚本索引（按链维护）

Hardhat 配置一链一文件，脚本目录与之一一对应：

| 目录 | 链 | Hardhat 配置 |
|------|----|--------------|
| `hack/polkadot/` | Polkadot Hub（AssetHub, chainId 420420417） | `hardhat.config.polkadot.js` |
| `hack/base/` | Base（OP-Stack，主网 / Sepolia 测试环境） | `hardhat.config.base.js` |
| `hack/eth/` | Ethereum L1（mainnet / Sepolia） | `hardhat.config.eth.js` |

Polkadot genesis 配置：`hack/polkadot/configs/`（Subnet genesis 数据）、根目录 `.env`。
Base 链配置：`hack/base/configs/`（chainId / rpc / Token 创世参数，无密钥）。

## hack/polkadot/ — Polkadot Hub

| 脚本 | 用途 |
|------|------|
| `init_chain.sh --env <local\|test\|main>` | 一键：编译 + 部署 Subnet/Token + 初始化链上数据（读 `hack/polkadot/configs/<env>.json`） |
| `deploy_subnet_token.js` | 部署 Subnet + Token（UUPS）+ BLS（Polkadot 版）并接线 |
| `upgrade_subnet_token.js` | UUPS 升级 Subnet / Token 实现 |
| `init_subnet_data.js` | 初始化 Subnet 链上数据（Region/LevelPrice/Asset/Worker/Secret/Validator） |
| `set_epoch_slot.js` | 设置 Subnet epochSlot |
| `deploy_flux_test.js` / `upgrade_flux_test.js` / `gen_deploy_flux_test.sh` | FLUX_TEST（actives）部署 / 升级（私钥读 `tee-node/ext/actives/flux_config.json`） |
| `set_fee_target.js` | 设置 FLUX_TEST feeTarget |
| `deploy_erc20.js` / `transfer_erc20.js` / `transfer_erc20.sh` | 测试 ERC20 部署与转账 |
| `generate-go-binding.sh` | 由 artifacts 生成 Go binding（输出到 `../../token/service`） |

示例（均在 `evms/` 下）：

```bash
npm run deploy:polkadot       # = npx hardhat run hack/polkadot/deploy_subnet_token.js ...
bash hack/polkadot/init_chain.sh --env test
```

## hack/base/ — Base

详见 `hack/base/README.md`。

| 脚本 | 用途 |
|------|------|
| `deploy.js` | Token（UUPS）/ BaseBLS 部署、链上集成、Token 升级；`--chain base\|baseSepolia --dry-run`（默认 baseSepolia） |
| `state.js` | 链上状态查询（预编译可用性、Token/BaseBLS 配置、余额） |
| `start-opgeth.sh` | 本地 op-geth dev 节点（Isthmus/Osaka，EIP-2537） |
| `common.js` | 共享库（链定义 / artifact / vectors / deployments.json） |
| `l1-fork-provider.js` | Hardhat EDR OP-fork provider patch（被 `hardhat.config.base.js` require） |
| `gas_report.js` | BLS 网关各流程 gas 测量（Base fork） |

```bash
npm run deploy:base -- --chain baseSepolia --deploy-token --dry-run
npm run state:base -- --chain baseSepolia --token <proxy>
npm run test:base
```

## hack/eth/ — Ethereum L1

暂无脚本；部署时把脚本放这里，用 `--config hardhat.config.eth.js --network sepolia|mainnet` 运行。
