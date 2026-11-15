# Base Sepolia 对接计划 — TEE 侧链 × BaseBLS/Token

> 状态：待实施 ｜ 更新：2026-11-13
> 相关代码：`revive-contract/evms`（合约 + hack/base 脚本）、`tee-node`（侧链）

## 0. 目标与架构

侧链测试网（tee-node）与 Base Sepolia 打通积分充值/提现业务：

```
充值: 用户 ─ETH─▶ Token(Base Sepolia) ─事件 Recharged─▶ tee-node relay ─▶ gateway.Recharge ─▶ 积分入账
提现: 侧链 ApplyWithdraw ─▶ batch ─▶ DKG BLS 签名 ─▶ BaseBLS.submit(data,hmG1,negSig)
        ─0x0f pairing─▶ Token.withdraw(user, points, nonce) ─▶ ETH 到账 / PENDING→claim
```

关键约定：

- 网关 = `BaseBLS.sol`（hmG1 = MAP_FP_TO_G1(keccak256(data))，由 Go 侧
  `BLSSignEIP2537` 链下算好随签名提交；链上只做 0x0f 配对验证 + 目标执行）。
  `EthBLS.sol`（合约内 0x10 现算 hmG1）用于 ETH/Polkadot Hub 网络。
- `aggPubkey` = **侧链测试网当前纪元 DKG BLS 群公钥**（EIP-2537 G2，256B），
  不是 vectors.json 里的测试钥。
- chainID `84532`；换算 `rate=4000, tokenUnit=1e18`（1 ETH = 4000 pt，$4/ETH）。

## Phase 1 — tee-node 代码（Go）

### 1.1 BaseBLS Go binding

- `hack/evm/contract.sh` build 段新增：
  `artifacts/contracts/BaseBLS.sol/BaseBLS.json` → `abigen --type BaseBLS --out $BINDINGS_DIR/bls_base.go`
- ABI：`submit(bytes data, bytes hmG1, bytes negSig)` /
  `batchSubmit(bytes[] datas, bytes hmG1, bytes negSig)` / `setAggPubkey(bytes)`

### 1.2 `pkg/chains/evm` EvmChain 支持双网关 ABI

- `client.go`：按配置选择绑定 `bls`（EthBLS）或 `blsBase`（BaseBLS）——
  加 `ChainConfigType.BLSVariant`（默认 eth），或按 chainId 8453/84532→base、其余→eth。
- `extrinsic.go SignAndSubmit`：Base 变体走 `BLSSignEIP2537` 的 **hmG1EIP**（第 3 返回值），
  pack `submit(data, hmG1, negSig)` / `batchSubmit(datas, hmG1, negSig)`。
  msgHash 规则必须与合约一致：
  - submit: `keccak256(data)`
  - batchSubmit: `keccak256(abi.encode(bytes[]))`
  ⚠️ M1 第一件事：用 vectors.json 的 `batchSubmit-batch` 三元组对 Go 签名路径做
  逐字节断言（标准答案：`ext/blscontract/golang` 的 `packDynamicBytesArray`），不一致要修。
- `GetRelayEvents`（EVM relay）：从 Token 代理拉
  `Recharged(indexed user, amount, points, timestamp)`，事件 nonce = 链上单调序号
  （`(chainID, nonce)` 防重放）；EVM 地址 → `UniAddr{t:2}`。

### 1.3 runtime 白名单（需运行时升级）

- `gateway/credit.go`：`UserTypeToChainID[2] += 84532`
- 标准迁移流程：`NativeVersion` +1、`go vet/test`、`update_runtime.sh` 发测试网
  （本改动无存储结构变化，无数据迁移）。

### 1.4 链配置

- `config/baseTest.go`：

  ```go
  ChainID: 84532, ChainType: "evm", Cipher: SuiteBLS12381,
  Urls: [<私有 Base Sepolia 节点>], TokenAddress: <TOKEN>, BLSAddress: <BLS>
  ```

- 部署钥 = tee-provider ECDSA 账户（gas 付款 + submit tx from），纳入节点 keystore。
- ⚠️ 公共 `sepolia.base.org` 有速率限制（探测期间已被 403），relay 轮询必须私有 RPC。

## Phase 2 — Base Sepolia 合约部署（evms）

```bash
cd evms && npm run compile:base
node hack/base/deploy.js --chain baseSepolia --deploy-token                    # → TOKEN
node hack/base/deploy.js --chain baseSepolia --deploy-bls --target $TOKEN     # → BLS (aggPubkey=侧链纪元公钥)
node hack/base/deploy.js --chain baseSepolia --integrate --token $TOKEN --bls $BLS
node hack/base/state.js --chain baseSepolia --token $TOKEN --bls $BLS         # 抽查
```

- `deploy-bls` 前把 aggPubkey 换成侧链真实公钥（`gov.DkgPubKeys(BLS12381)` → EIP-2537 G2）。
  `setAggPubkey` 是 onlyTarget：集成后由 Subnet 角色驱动；联调期可先把 target 设部署者
  EOA 直接 set（`deploy.js` 已支持）。
- Token 初始化：`rate=4000 tokenUnit=1e18`（initialize 默认）、ERC20 按需 `setERC20Token`。
- **流动性注入**：给 Token 代理转提现垫付 ETH（测试额度，如 0.5 ETH）；
  部署钥 `0x9D57…65Ce` 现有 ~0.001 ETH，仅够部署 gas。

## Phase 3 — 联调验收

| # | 场景 | 验收 |
|---|------|------|
| T1 | 充值：EOA `recharge()` 0.001 ETH | 侧链 relay `Recharge(chain=84532)` 入账、积分 = `value*4000/1e18`、`GetRechargeHistory` 可查 |
| T2 | 充值防重放 | relay 重启/回扫不重复入账（`(84532,nonce)` 去重） |
| T3 | 提现（余额充足）：侧链 `ApplyWithdraw` | BLS submit tx status=1、Token `Withdrawn(...,DONE)`+`WithdrawalDone`、用户收款 |
| T4 | 提现（合约余额不足） | PENDING 记录 → 注入流动性 → 用户 `claimWithdrawal(nonce)` 到账 |
| T5 | nonce 幂等 | 同 nonce 重复 submit 不双花 |
| T6 | 纪元轮换 | DKG 新纪元后旧 aggPubkey 更新流程：新公钥重新 `setAggPubkey`，跨纪元提现不断 |
| T7 | 负向 | 篡改 negSig/hmG1/data → BaseBLS revert（fork 测试已覆盖，链上抽查一次） |

## Phase 4 — 运维与回退

- **监控**：`state.js` 定时任务（Token/BLS 余额、pending withdrawals）+
  tee-node `balance_change`/relay 日志告警。
- **升级**：Token UUPS `--upgrade-token`；BLS aggPubkey 轮换脚本化进 `deploy.js`。
- **回退**：`Token.setSubnet(0x0)` 即停所有 subnet 函数（withdraw/费率）；
  `emergencyWithdraw`/`emergencyWithdrawErc20` 兜底取回资金。
- **文档**：落地后更新 `hack/base/README.md`「侧链对接」章节。

## 里程碑与顺序

1. **M1**（Go，1~2d）：1.1 binding + 1.2 SignAndSubmit/relay + 单测（本地对 mock RPC）
2. **M2**（runtime，0.5d + 发版）：1.3 白名单升级
3. **M3**（链上，0.5d）：Phase 2 部署接线
4. **M4**（联调，1~2d）：T1–T7
5. **M5**：运维接入 + README

## 风险与决策点

| 风险 | 处置 |
|------|------|
| batchSubmit 签名消息与合约 `keccak(abi.encode(datas))` 不一致 | M1 第一件事：vectors.json `batchSubmit-batch` 三元组逐字节断言 |
| 公共 RPC 限流 | M1 并行申请私有 Base Sepolia RPC（Ankr/QuickNode/自建 op-node+op-besu） |
| 纪元轮换窗口丢提现 | T6 演练；必要时提现提交带重试队列 |
| gas 付款账户管理 | tee-provider ECDSA 私钥进 keystore，与部署钥分离 |

## 现状基线（已完成，作为计划起点）

- `BaseBLS.sol` / `EthBLS.sol`（改名自 BLS.sol）编译通过；`test/BLS.js` 9 passing
  （Base Sepolia fork，`npm run test:base`）。
- vectors.json = baseSepolia/84532 manifest，DKG→EIP-2537→预编译离线验证；
  文件归 tee-node `ext/blscontract/vectors/vectors.json`（由 Go 测试生成），evms
  侧通过 `BLS_VECTORS_DIR`（目录或文件）注入读取，默认取本机 tee-node checkout。
- `hack/base/`：deploy.js（deploy-token/deploy-bls/integrate/upgrade-token）、
  state.js、common.js、start-opgeth.sh（已删，本地环境已清理）、gas_report.js。
- tee-node：`SignAndSubmit`（EthBLS ABI）、relay `(chainID,nonce)` 防重放、
  `UserTypeToChainID = {2: {420420417}}`（缺 84532）、`BLSSignEIP2537` 返回
  (sig, negSig, hmG1EIP)。
- 伪造测试结论：BaseBLS 下伪造 data 无法复用已发布签名（hmG1' ≠ 签名点，配对失败），
  hmG1↔data 绑定由 Go 签名入口强制。
