# revive-contract — Building a Subnet System on Polkadot PVM

[English](#english) · [中文](#中文)

---

<a name="english"></a>

## English

`revive-contract` is a set of smart contracts that build a **decentralized cloud / subnet system** on top of the **Polkadot PVM** (PolkaVM, via `pallet-revive`). The contracts under [`revives/`](revives/) are written in Rust and compiled to PolkaVM bytecode with [`cargo-wrevive`](https://github.com/paritytech/cargo-pvm-contract), using the [`wrevive`](../../wetee/wrevive) API + macro toolkit.

The subnet lets anyone contribute **compute nodes (Workers)** and **TEE consensus nodes (Secret Nodes)** to a shared cloud, backed by on-chain staking, slashing, epoch-based validator rotation, and an upgradeable point-payment token.

### Why PVM / `pallet-revive`?

`pallet-revive` runs RISC-V (PolkaVM) contracts on any Substrate chain. Compared with ink!/Wasm, PVM contracts are `no_std` Rust binaries that expose EVM-compatible calling conventions and can `delegate_call` / `call_evm`, which makes:

- **Transparent proxy upgrades** possible (see [`Proxy`](revives/Proxy)),
- **ERC20 interop** possible (the `Token` contract calls Solidity ERC20 via `call_evm`),
- storage and gas cheap enough to keep a full subnet registry on-chain.

### Architecture

```
                ┌─────────────────────────────────────────────┐
   Governance   │                 Proxy (per contract)         │  ← upgradeable entrypoint
   / DAO  ─────▶│  admin: upgrade() / transfer_admin()         │
                │  fallback: delegate_call → implementation    │
                └───────────────┬──────────────┬───────────────┘
                                │              │
                       ┌────────▼──────┐  ┌────▼──────────────┐
                       │   Subnet      │  │      Token        │
                       │ (registry &   │  │ (points/recharge, │
                       │  consensus)   │◀─│  withdrawals)     │
                       └───┬───────┬───┘  └───────────────────┘
                           │       │
            ┌──────────────▼─┐  ┌──▼────────────────┐
            │  Workers       │  │  Secret Nodes     │
            │ (K8s compute,  │  │ (TEE validators,  │
            │  staked)       │  │  epoch rotation)  │
            └────────────────┘  └───────────────────┘
                    ▲                    ▲
            Node owners stake     Side-chain (TEE) multisig
            resources & run       starts workers & advances epochs
```

The three contracts under `revives/`:

| Contract | Path | Role |
|---|---|---|
| **Subnet** | [`revives/Subnet`](revives/Subnet) | The core registry & consensus contract: regions, workers, staking/slashing, secret nodes, validator set & epoch rotation, TEE code versions. |
| **Token** | [`revives/Token`](revives/Token) | Point recharge/withdrawal contract. Users recharge native DOT or ERC20 into points; Subnet authorizes withdrawals. UUPS-upgradeable via Proxy. |
| **Proxy** | [`revives/Proxy`](revives/Proxy) | Transparent proxy for upgradeability: holds `implementation` + `admin`, forwards unmatched calls via `delegate_call`. |

Shared data types live in [`primitives`](primitives) (`K8sCluster`, `Ip`, `RunPrice`, `AssetInfo`, `NodeID`, …), so Subnet, Token, and off-chain clients agree on SCALE encodings.

### How the Subnet works

#### 1. Governance bootstrap
`init()` records the caller as the **governance contract** (`gov`). Governance-only setters configure the network: `set_region`, `set_level_price`, `set_asset`, `set_min_mortgage`, `set_level_min_mortgage`, `set_epoch_slot`, `set_boot_nodes`, and TEE `add_code_version` / `delete_code_version`.

#### 2. Worker (compute) lifecycle
Compute providers register and stake real funds:

1. `worker_register(name, p2p_id, ip, port, level, region_id)` — one Worker per address, bound to a region.
2. `worker_mortgage(id, cpu, mem, cvm_cpu, cvm_mem, disk, gpu, deposit)` — stake native tokens and declare resources. The **actual** transferred value is recorded (not the claimed amount), preventing false deposits.
3. `worker_start(id)` — called by the **side-chain (TEE) multisig**; verifies total stake ≥ the level minimum and that at least one resource is staked, then sets status to *running (1)*.
4. `worker_stop`, `worker_unmortgage` — owner stops the node and reclaims stake (only when not running).
5. `slash_worker_mortgage(worker_id, amount, to)` — governance slashes misbehaving nodes, deducting stake record-by-record and transferring to a recipient.

#### 3. Secret Node (TEE validator) lifecycle & epochs
Secret Nodes form the TEE consensus / validator set:

1. `secret_register(name, validator_id, p2p_id, ip, port, bls)` — register a validator (BLS key required). Node `0` bootstraps the running set.
2. `secret_deposit` / `secret_delete` — stake and de-register (must not be active/pending, stake must be zero).
3. `validator_join(id)` / `validator_delete(id)` — governance queues weight changes into a **pending** set.
4. `set_next_epoch(node_id)` — the side-chain multisig advances the epoch once `epoch_slot` blocks have passed; `calc_new_validators()` merges pending → running, dropping zero-weight nodes.
5. `validators()`, `next_epoch_validators()`, `boot_nodes()` — read the active/next validator sets for off-chain consensus clients.

#### 4. TEE code versions
`add_code_version(signer, signature)` registers signed TEE program versions so nodes can verify they run governance-approved enclave code.

#### 5. Token / payment layer
The `Token` contract mints **points** from native DOT (`points = value * RATE / TOKEN_UNIT`) or registered ERC20 tokens, and processes Subnet-authorized withdrawals with a pending-withdrawal/nonce system. See [`docs/token-erc20-design.md`](docs/token-erc20-design.md).

### Build

```bash
# Install the PVM contract toolchain
cargo install cargo-wrevive

# Build all contracts to PolkaVM bytecode (release profile: lto, opt-level=z, panic=abort)
cargo wrevive build --release

# Run the contract unit tests (off_chain feature)
cargo test
```

Each contract is both a `[[bin]]` (deployable PVM blob) and a `[lib]` (importable with the `api` feature, which disables the crate's `global_allocator` to avoid conflicts).

### Deploy

Deployment, genesis initialization and upgrades are driven by the Go + shell tooling in [`hacks/deploy`](hacks/deploy):

```bash
cd hacks/deploy
./init_chain.sh        # full deploy + genesis (secrets, boot_nodes, validators, regions…)
./deploy_contract.sh   # deploy a single contract
./upgrade_contract.sh  # upgrade an implementation behind its Proxy
```

Environments are configured under `hacks/deploy/configs/<env>.json` (`url`, `suri`, deployed `contracts.*` addresses, and `genesis` data). See [`hacks/deploy/Readme.md`](hacks/deploy/Readme.md) for full field docs.

### Repository layout

```
revive-contract/
├── revives/          # PolkaVM subnet contracts (Subnet, Token, Proxy)
├── primitives/       # shared SCALE data types
├── evms/             # EVM-side contracts / interop
├── hacks/deploy/     # Go + shell deploy / upgrade / genesis tooling
├── docs/             # design docs (e.g. token ERC20 design)
└── audits/           # audit material
```

---

<a name="中文"></a>

## 中文

`revive-contract` 是一组在 **Polkadot PVM**（PolkaVM，基于 `pallet-revive`）之上构建**去中心化云 / 子网系统**的智能合约。[`revives/`](revives/) 目录下的合约用 Rust 编写，通过 [`cargo-wrevive`](https://github.com/paritytech/cargo-pvm-contract) 编译为 PolkaVM 字节码，并使用 [`wrevive`](../../wetee/wrevive) 的 API + 宏工具集。

该子网允许任何人向共享云贡献**算力节点（Worker）**和 **TEE 共识节点（Secret Node）**，并由链上抵押、罚没、按 epoch 轮换的验证者集合，以及可升级的积分支付代币来支撑。

### 为什么用 PVM / `pallet-revive`？

`pallet-revive` 让任意 Substrate 链都能运行 RISC-V（PolkaVM）合约。与 ink!/Wasm 相比，PVM 合约是 `no_std` 的 Rust 二进制，暴露兼容 EVM 的调用约定，可以 `delegate_call` / `call_evm`，因而支持：

- **透明代理升级**（见 [`Proxy`](revives/Proxy)）；
- **ERC20 互操作**（`Token` 合约通过 `call_evm` 调用 Solidity ERC20）；
- 足够便宜的存储与 gas，可将完整子网注册表保存在链上。

### 架构

```
                ┌─────────────────────────────────────────────┐
     治理/DAO   │              Proxy（每个合约一个）           │  ← 可升级入口
        ───────▶│  管理员: upgrade() / transfer_admin()        │
                │  fallback: delegate_call → 实现合约          │
                └───────────────┬──────────────┬───────────────┘
                                │              │
                       ┌────────▼──────┐  ┌────▼──────────────┐
                       │   Subnet      │  │      Token        │
                       │ （注册表与    │  │ （积分充值 /      │
                       │   共识）      │◀─│   提现）          │
                       └───┬───────┬───┘  └───────────────────┘
                           │       │
            ┌──────────────▼─┐  ┌──▼────────────────┐
            │  Worker        │  │  Secret Node      │
            │ （K8s 算力,    │  │ （TEE 验证者,     │
            │   已抵押）     │  │   epoch 轮换）    │
            └────────────────┘  └───────────────────┘
                    ▲                    ▲
            节点主抵押资源并运行   侧链（TEE）多签启动 Worker 并推进 epoch
```

`revives/` 下的三个合约：

| 合约 | 路径 | 职责 |
|---|---|---|
| **Subnet** | [`revives/Subnet`](revives/Subnet) | 核心注册表与共识合约：区域、Worker、抵押/罚没、Secret 节点、验证者集合与 epoch 轮换、TEE 代码版本。 |
| **Token** | [`revives/Token`](revives/Token) | 积分充值/提现合约。用户用原生 DOT 或 ERC20 充值积分，由 Subnet 授权提现，通过 Proxy 实现 UUPS 可升级。 |
| **Proxy** | [`revives/Proxy`](revives/Proxy) | 透明代理，实现可升级：保存 `implementation` + `admin`，将未匹配的调用通过 `delegate_call` 转发。 |

共享数据类型放在 [`primitives`](primitives)（`K8sCluster`、`Ip`、`RunPrice`、`AssetInfo`、`NodeID` 等），使 Subnet、Token 与链下客户端在 SCALE 编码上保持一致。

### 子网如何运作

#### 1. 治理引导
`init()` 将调用者记录为**治理合约**（`gov`）。仅治理可调用的设置项配置整个网络：`set_region`、`set_level_price`、`set_asset`、`set_min_mortgage`、`set_level_min_mortgage`、`set_epoch_slot`、`set_boot_nodes`，以及 TEE 的 `add_code_version` / `delete_code_version`。

#### 2. Worker（算力）生命周期
算力提供者注册并抵押真实资金：

1. `worker_register(name, p2p_id, ip, port, level, region_id)` —— 每个地址仅限一个 Worker，绑定到某个区域。
2. `worker_mortgage(id, cpu, mem, cvm_cpu, cvm_mem, disk, gpu, deposit)` —— 抵押原生代币并声明资源。合约记录**实际**转入金额（而非声明金额），防止虚报抵押。
3. `worker_start(id)` —— 由**侧链（TEE）多签**调用；校验抵押总额 ≥ 该等级最低要求，且至少抵押了一种资源，然后将状态置为*运行中（1）*。
4. `worker_stop`、`worker_unmortgage` —— 拥有者停止节点并取回抵押（仅在非运行状态下）。
5. `slash_worker_mortgage(worker_id, amount, to)` —— 治理对作恶节点进行罚没，逐条抵押记录扣除并转账给接收方。

#### 3. Secret Node（TEE 验证者）生命周期与 epoch
Secret 节点组成 TEE 共识 / 验证者集合：

1. `secret_register(name, validator_id, p2p_id, ip, port, bls)` —— 注册验证者（需 BLS 公钥）。节点 `0` 会引导初始运行集合。
2. `secret_deposit` / `secret_delete` —— 抵押与注销（不得处于运行/待处理状态，且抵押须为 0）。
3. `validator_join(id)` / `validator_delete(id)` —— 治理将权重变更加入**待处理（pending）**集合。
4. `set_next_epoch(node_id)` —— 侧链多签在经过 `epoch_slot` 个区块后推进 epoch；`calc_new_validators()` 将 pending 合并进 running，剔除权重为 0 的节点。
5. `validators()`、`next_epoch_validators()`、`boot_nodes()` —— 供链下共识客户端读取当前 / 下一个验证者集合。

#### 4. TEE 代码版本
`add_code_version(signer, signature)` 注册经签名的 TEE 程序版本，使节点可验证自己运行的是治理批准的 enclave 代码。

#### 5. Token / 支付层
`Token` 合约将原生 DOT（`积分 = value * RATE / TOKEN_UNIT`）或已注册的 ERC20 代币铸造为**积分**，并通过待提现/nonce 机制处理由 Subnet 授权的提现。详见 [`docs/token-erc20-design.md`](docs/token-erc20-design.md)。

### 构建

```bash
# 安装 PVM 合约工具链
cargo install cargo-wrevive

# 将所有合约编译为 PolkaVM 字节码（release：lto、opt-level=z、panic=abort）
cargo wrevive build --release

# 运行合约单元测试（off_chain feature）
cargo test
```

每个合约同时是 `[[bin]]`（可部署的 PVM blob）和 `[lib]`（用 `api` feature 导入，该 feature 会禁用本 crate 的 `global_allocator` 以避免冲突）。

### 部署

部署、创世初始化与升级由 [`hacks/deploy`](hacks/deploy) 下的 Go + shell 工具驱动：

```bash
cd hacks/deploy
./init_chain.sh        # 全量部署 + 创世（secrets、boot_nodes、validators、regions…）
./deploy_contract.sh   # 部署单个合约
./upgrade_contract.sh  # 升级 Proxy 背后的实现合约
```

环境在 `hacks/deploy/configs/<env>.json` 中配置（`url`、`suri`、已部署的 `contracts.*` 地址与 `genesis` 数据）。完整字段说明见 [`hacks/deploy/Readme.md`](hacks/deploy/Readme.md)。

### 仓库结构

```
revive-contract/
├── revives/          # PolkaVM 子网合约（Subnet、Token、Proxy）
├── primitives/       # 共享 SCALE 数据类型
├── evms/             # EVM 侧合约 / 互操作
├── hacks/deploy/     # Go + shell 部署 / 升级 / 创世工具
├── docs/             # 设计文档（如 token ERC20 设计）
└── audits/           # 审计材料
```
