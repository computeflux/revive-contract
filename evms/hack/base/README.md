# Base 链部署与维护脚本（hack/base/）

Base（OP-Stack）链上 BLS 网关 + Token 的部署 / 维护脚本。所有 Base 相关的
Hardhat 网络配置在 `../../hardhat.config.base.js`（`hardhat.config.polkadot.js`
只保留 Polkadot Hub）。

## 合约

| 合约 | 说明 |
|------|------|
| `BaseBLS` | **Base 网络的 BLS 网关**（EIP-2537 `0x0f` pairing）。`submit(data, hmG1, negSig)` 验证通过后 `target.call(data)`。hmG1 = MAP_FP_TO_G1(keccak256(data))，由 tee-node（`ext/blscontract`，签名入口 `BLSSignEIP2537`）在链下算好随签名提交；链上验签把签名绑死在 hmG1，伪造 data 无法复用已发布签名。 |
| `Token` | 积分充值/提现（UUPS 可升级），与 `revives/Token`（PolkaVM 版）功能对齐。`subnet` = BaseBLS 地址，`withdraw` / `withdrawErc20` / `setRechargeFeeRate` / `setWithdrawFeeRate` 仅 subnet 可调。 |

## 环境变量

在 `evms/.env` 中配置（`common.js` 自动加载）：

```bash
# Base 私钥不在这里（见下方「Base 私钥」）；以下主要是 Polkadot / 通用配置。
PRIVATE_KEY=0x...                             # Polkadot Hub 测试网部署者私钥
BASE_RPC_URL=https://mainnet.base.org         # Base 主网（可选，有默认值）
BASE_SEPOLIA_RPC_URL=https://sepolia.base.org # Base Sepolia（默认测试环境）
```

## 链配置（hack/base/configs/）

与 `hack/polkadot/configs/` 对应，Base 链的配置文件提交入库（无密钥）：

| 文件 | 说明 |
|------|------|
| `baseSepolia.json` | chainId 84532，rpc `https://sepolia.base.org`，explorer sepolia.basescan |
| `base.json` | chainId 8453，rpc `https://mainnet.base.org`，explorer basescan |

字段：`rpc_url` / `chain_id` / `explorer` + `token`（Token 创世参数：`owner`（null = 部署者）、
`rate`、`token_unit`、`native_active`、`recharge_fee_rate`、`withdraw_fee_rate`、`erc20_tokens`）。
`--deploy-token` 部署后按配置应用（幂等，跳过与链上一致的项）；环境变量
`BASE_RPC_URL` / `BASE_SEPOLIA_RPC_URL` / `--rpc` 覆盖配置文件中的 rpc。

### Base 私钥（迁移到 config，与 Polkadot 完全分离）

Base 脚本的部署私钥直接写在 **Base 自己的链配置**里（像 Polkadot
`hack/polkadot/configs/test.json` 内嵌 `bls_validator_key`），与 Polkadot 的 key
（`.env` 的 `PRIVATE_KEY` / `init_chain.sh --private-key` / `flux_config.json`）互不影响：

- 每个 `hack/base/configs/<name>.json` 内嵌 `private_key`（`0x` + 64 hex）；两个 Base
  链都填 `~/.secrets/deploy-mainnet.key` 里的那把主网钥（地址 `0x9D573489…65Ce`）。
- **这些配置不入库**：`.gitignore` 忽略 `hack/base/configs/*.json`，只提交模板
  `hack/base/configs/*.example.json`（`private_key` 为占位）。新环境从模板复制
  `<name>.json` 再填真实密钥。
- 也支持外部文件：`private_key_file`（路径，`~` / `$ENV` / 相对 config 目录展开）+
  `private_key_name`（keyed `NAME=value` 文件取哪个字段，默认 `PRIVATE_KEY`）。
- 解析优先级：**config 的 `private_key`（或 `private_key_file`）> 环境变量
  `PRIVATE_KEY`**（config 是 Base 私钥的 source of truth，`.env` 里的 Polkadot 私钥不会
  劫持 Base）。仅当 config 无 key 时 env 生效。
- `ecdsa_key` / `ecdsa_key_file` 同理（预留给 tee-provider ECDSA，Base 目前不用）。

## 前置：编译

```bash
cd evms
npx hardhat compile --config hardhat.config.base.js
```

## 部署

```bash
cd evms

# dry-run（不发送交易，打印将执行的操作）
node hack/base/deploy.js --chain baseSepolia --deploy-token --dry-run

# 部署 Token（impl + ERC1967Proxy + initialize(owner=deployer)）→ TOKEN
node hack/base/deploy.js --chain baseSepolia --deploy-token

# 部署 BaseBLS（aggPubkey / g2Generator 来自 tee-node ext/blscontract/vectors/vectors.json，
# target = --target <地址>，生产为 Token 代理；target=部署者时自动 setAggPubkey）→ BLS
node hack/base/deploy.js --chain baseSepolia --deploy-bls --target <TOKEN_PROXY>

# 链上集成：Token.setSubnet(BLS) + BaseBLS.setTarget(Token)
node hack/base/deploy.js --chain baseSepolia --integrate --token <TOKEN_PROXY> --bls <BLS_ADDR>

# UUPS 升级 Token 实现（owner 权限）
node hack/base/deploy.js --chain baseSepolia --upgrade-token --token <TOKEN_PROXY>
```

`--chain` 取 `base` | `baseSepolia`（默认测试环境 baseSepolia，或 `--rpc <url>` 自定义）。
部署产物追加写入 `hack/base/deployments.json`（按 chain 分组，含地址与 tx hash）。

## 查询 / 维护

```bash
node hack/base/state.js --chain baseSepolia --token <TOKEN_PROXY> --bls <BLS_ADDR>
# 输出: chainId / 0x0f,0x10 预编译可用性、Token owner/subnet/rate/tokenUnit/fees/
#       nativeActive/ERC20 注册表/原生币余额、BaseBLS target/aggPubkey/g2Generator、
#       可选 --withdrawal-nonce <n> 查待提现记录
```

## 测试

```bash
cd evms
# Base Sepolia fork（EIP-2537 已激活；默认区块 58171467，可用环境变量覆盖）
# 全链路测试（9 用例：部署 + 配对验证 + 负向用例）
npm run test:base

# Go 侧向量生成（DKG → EIP-2537 → precompile 预验证 → vectors.json）
cd ../apps/tee-node && go test ./ext/blscontract/golang/ -run TestGenerateVectors -v
```

## Base Sepolia 端到端流程（示例）

1. `go test ./ext/blscontract/golang/` 生成 `ext/blscontract/vectors/vectors.json`
2. `node hack/base/deploy.js --chain baseSepolia --deploy-token` → `TOKEN`
3. `node hack/base/deploy.js --chain baseSepolia --deploy-bls --target $TOKEN` → `BLS`
4. `node hack/base/deploy.js --chain baseSepolia --integrate --token $TOKEN --bls $BLS`
5. submit 验证：vectors.json 的签名绑定具体 calldata（MockTarget），要驱动
   Token 的 `withdraw(user, points, nonce)` 需以该 calldata 重新走 DKG 签名
   （`ext/blscontract` 里替换 targetCalldatas 生成新向量）。网关本身可先用
   `--target <部署者 EOA>` 部署验证。
6. `node hack/base/state.js ...` 抽查链上状态

## 侧链对接

与 TEE 侧链测试网（tee-node）对接的完整实施计划见
`../../docs/base-integration-plan.md`。
