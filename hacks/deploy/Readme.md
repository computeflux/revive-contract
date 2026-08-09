# WeTEE 合约部署工具

基于 PolkaVM / pallet-revive 的合约部署、初始化与升级工具集。使用 Go 客户端与链交互，Shell 脚本统一入口。

---

## 目录结构

```
deploy/
├── configs/                  # 环境配置（不要提交含私钥的配置）
│   ├── local.json            # 本地开发节点
│   ├── test.json             # 测试网
│   └── main.json             # 主网
├── cmd/
│   ├── deploy-contract/      # 单合约部署程序
│   ├── deploy-full/          # 全量部署（含创世初始化）程序
│   ├── upgrade-contract/     # 合约升级程序
│   └── gen-key/              # 密钥生成程序
├── contracts/                # Go ABI 绑定（自动生成）
├── init_chain.sh             # 全量部署入口（推荐首次部署使用）
├── deploy_contract.sh        # 单合约部署入口
├── upgrade_contract.sh       # 合约升级入口
├── gen-key.sh                # 密钥生成入口
├── cloud_update.sh           # 快速更新 Cloud 合约（旧版，建议用 upgrade_contract.sh）
├── pod_update.sh             # 快速更新 Pod 合约代码（旧版）
├── subnet_update.sh          # 快速更新 Subnet 合约（旧版）
├── set_pod_code.sh           # 单独设置 Pod 代码哈希（旧版）
└── go.mod
```

---

## 前置条件

| 工具 | 版本要求 | 说明 |
|------|----------|------|
| Go | ≥ 1.23 | 部署程序运行时 |
| `cargo wrevive` | 已安装 | PolkaVM 合约编译工具 |
| `jq` | 任意 | Shell 脚本解析 JSON 配置 |
| 链节点 | 已运行 | 支持 pallet-revive 的 Substrate 节点 |

安装 `cargo-wrevive`：

```bash
cargo install cargo-wrevive
```

---

## 配置文件

每个环境对应 `configs/<env>.json`，包含连接信息、签名账户和创世数据。

### 字段说明

| 字段 | 说明 | 示例 |
|------|------|------|
| `url` | 链节点 WebSocket 地址 | `"ws://127.0.0.1:9944"` |
| `suri` | 签名账户的 secret URI | `"//Alice"` 或助记词 |
| `contracts.subnet` | 已部署 Subnet 代理合约地址 | `"0x1234..."` |
| `contracts.cloud` | 已部署 Cloud 代理合约地址 | `"0x5678..."` |
| `genesis` | 仅 `init_chain.sh` 需要，创世初始化数据 | 见下 |

### `genesis` 字段详解

```jsonc
"genesis": {
  // Secret 节点（TEE 验证者）列表
  "secrets": [
    {
      "name": "node0",           // 节点名称
      "ss58": "5G9j...",         // 节点 Substrate 账户地址（SS58）
      "p_ss58": "5FKg...",       // 节点 P2P 公钥地址（SS58）
      "ip": "192.168.1.10",      // 节点 IPv4 地址
      "port": 20110              // 节点监听端口
    }
  ],
  "boot_nodes": [0, 1, 2],      // 引导节点 ID（对应 secrets 数组下标）
  "validators": [1, 2],          // 初始验证者 ID（对应 secrets 数组下标）
  "region": "default",           // 默认区域名称
  // Worker 节点列表
  "workers": [
    {
      "name": "worker0",
      "ss58": "5GSB...",          // Worker 账户地址
      "domain": "example.com",   // Worker 域名（优先于 IP）
      "port": 10000,
      "level": 1,                // Worker 硬件等级（1=基础, 2=高级...）
      "region": 0,               // 所属区域 ID
      "cpu": 10000,              // CPU 资源量（毫核）
      "memory": 10000,           // 内存资源量（MiB）
      "disk": 0,                 // 磁盘资源量（GiB）
      "gpu": 0,                  // GPU 数量
      "mortgage": 10000000       // 抵押金额（链上最小单位）
    }
  ]
}
```

### 配置示例（本地开发）

```json
{
  "url": "ws://127.0.0.1:9944",
  "suri": "//Alice",
  "contracts": {
    "subnet": "0x0000000000000000000000000000000000000000",
    "cloud":  "0x0000000000000000000000000000000000000000"
  },
  "genesis": {
    "secrets": [],
    "boot_nodes": [],
    "validators": [],
    "region": "local",
    "workers": []
  }
}
```

> ⚠️ **安全提示**：`suri` 包含私钥信息，生产环境请勿将 `configs/main.json` 提交到版本库。建议通过环境变量或 secret manager 注入。

---

## 部署工作流

### 一、首次全量部署（推荐）

全量部署会按顺序完成：
1. 编译所有合约（Pod、Subnet、Cloud、Proxy）
2. 上传 Pod 合约代码
3. 部署 Subnet（实现合约 + 代理合约）并初始化
4. 部署 Cloud（实现合约 + 代理合约）并初始化，关联 Subnet 和 Pod
5. 按 `genesis` 配置初始化 Secret 节点、引导节点、验证者、区域和 Worker

```bash
./init_chain.sh --env local
```

**参数说明**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--env` | 必填 | 环境名：`local` / `test` / `main` |
| `--network` | `42` | SS58 网络 ID（Substrate 默认 42） |
| `--build` | `true` | 是否先编译合约，`false` 跳过编译直接使用已有 `.polkavm` |

部署完成后会输出两个地址，**务必记录并填入 `configs/<env>.json` 的 `contracts` 字段**：

```
========================================
subnet address (proxy) => 0xabc...
cloud  address (proxy) => 0xdef...
========================================
```

---

### 二、单合约部署

用于独立部署单个合约（如仅测试某个合约）。

```bash
./deploy_contract.sh --env local --name proxy
./deploy_contract.sh --env test  --name cloud --dir /path/to/revives/Cloud
./deploy_contract.sh --env main  --name subnet --build false
```

**参数说明**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--env` | 必填 | 环境名 |
| `--name` | 必填 | 合约名：`cloud` / `subnet` / `proxy` / `pod` 等 |
| `--dir` | 自动推断 | 合约 crate 目录（含 `Cargo.toml`） |
| `--code` | 自动推断 | 已编译的 `.polkavm` 文件路径 |
| `--network` | `42` | SS58 网络 ID |
| `--build` | `true` | 是否先编译 |

---

### 三、合约升级

在不停服的情况下升级已部署合约的逻辑（热升级）。系统采用代理模式：代理合约地址不变，只更新实现合约。

```bash
./upgrade_contract.sh --env local --name <类型>
```

**升级类型**

| `--name` | 说明 |
|----------|------|
| `cloud` | 编译并上传新 Cloud 实现合约，更新代理指向 |
| `subnet` | 编译并上传新 Subnet 实现合约，更新代理指向 |
| `pod-code` | 编译并上传新 Pod 代码，更新 Cloud 合约中的 pod_code_hash（影响后续新建 Pod） |
| `pod-contract` | 对特定 Pod 执行合约升级（需配合 `--pod-id`） |

**示例**

```bash
# 升级 Cloud 合约逻辑
./upgrade_contract.sh --env local --name cloud

# 升级 Subnet 合约逻辑（跳过重新编译）
./upgrade_contract.sh --env test --name subnet --build false

# 更新 Pod 代码哈希（新建 Pod 将使用新代码）
./upgrade_contract.sh --env local --name pod-code

# 升级指定 Pod 实例的合约（ID=5 的 Pod）
./upgrade_contract.sh --env local --name pod-contract --pod-id 5
```

**参数说明**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--env` | 必填 | 环境名 |
| `--name` | 必填 | 升级类型 |
| `--pod-id` | — | `pod-contract` 时必填，目标 Pod ID |
| `--network` | `42` | SS58 网络 ID |
| `--build` | `true` | 是否先编译 |

> `configs/<env>.json` 的 `contracts.cloud` 和 `contracts.subnet` 必须已填写正确地址，升级程序依赖它们定位代理合约。

---

### 四、密钥生成

为 Secret 节点或 Worker 生成 Sr25519 密钥对。

```bash
./gen-key.sh [选项]
```

生成的密钥可填入 `configs/<env>.json` 的 `genesis.secrets` 或 `genesis.workers`。

---

## 合约架构说明

```
用户/DApp
   │
   ▼
Cloud Proxy (代理合约，地址固定)
   │  delegate_call
   ▼
Cloud Implementation (业务逻辑，可升级)
   │  跨合约调用
   ├──► Subnet Proxy → Subnet Implementation
   │
   └──► Pod Proxy (每个 Pod 一个实例)
           │  delegate_call
           ▼
         Pod Implementation (业务逻辑，可升级)
```

- **Proxy 合约**：地址对外不变，所有调用通过 `delegate_call` 转发到实现合约
- **实现合约**：包含业务逻辑，状态存储在 Proxy 中，升级时只换实现地址
- **Pod 合约**：每个 Pod 对应一个独立的 Proxy 实例，共享同一套 Pod 实现代码

---

## 常见问题

**Q: 部署失败提示 "Account not mapped"**  
A: 账户首次使用 pallet-revive 前需 map account，工具会自动执行此操作。确保账户有足够余额支付 gas。

**Q: 编译时提示找不到 `cargo wrevive`**  
A: 运行 `cargo install cargo-wrevive` 安装，或传入 `--build false` 跳过编译并手动指定 `--code` 路径。

**Q: 升级后旧数据是否保留**  
A: 是。Proxy 合约持有所有状态，升级只更换实现合约地址，状态完全保留。

**Q: 如何验证部署是否成功**  
A: 部署结束后程序会打印合约地址，可通过 Go 测试文件中的测试用例验证：
```bash
cd hacks/deploy
go test -run ^TestCloudUpdate$ -v
```

---

## API 服务与 Web UI

部署、升级与合约函数调用全部封装为 REST API，前端产物由 api-server **内嵌托管**
（单服务单端口，同源请求无 CORS），UI 基于 cathome/workflow/board 改造，位于 `ui/`。

```bash
# 一键构建前端 + 启动（默认端口 8000）
./api-ui.sh

# 指定端口
PORT=9000 ./api-ui.sh
```

打开 http://127.0.0.1:8000 即可在浏览器中完成：

- **总览**：环境（local/test/main）、签名账户、余额、合约地址
- **合约部署**：单合约部署 / 全量部署（subnet+token+proxy+创世初始化）/ 热升级
- **合约调试**：动态函数表单，查询（DryRun）与交易调用，结果展示
- **账户管理**：查看账户、执行 Revive Map Account

### API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/envs` | 环境列表（脱敏） |
| GET | `/api/envs/{env}` | 环境详情 + 账户信息 |
| POST | `/api/account/{env}/map` | Revive Map Account |
| GET | `/api/contracts/{env}/{name}/methods` | 合约方法清单（含参数定义，驱动前端表单） |
| POST | `/api/contracts/{env}/{name}/call` | 通用合约调用（exec / query） |
| POST | `/api/contracts/{env}/{name}/batch` | 批量交易（CallOf + batch_all，一次签名） |
| POST | `/api/deploy` | 单合约部署 |
| POST | `/api/deploy-full` | 全量部署 |
| POST | `/api/upgrade` | 合约热升级（token / subnet） |
| GET | `/`（及 SPA 路由） | 前端页面（内嵌 dist，构建产物同步自 `ui/dist`） |

**通用调用示例**（等价于 `go test` 里的 `QueryGetRate` / `ExecSetRate`）：

```bash
# 查询（DryRun）
curl -X POST http://127.0.0.1:8000/api/contracts/test/token/call \
  -H 'Content-Type: application/json' \
  -d '{"method":"get_rate","kind":"query","args":{}}'

# 交易（需要签名账户有 gas）
curl -X POST http://127.0.0.1:8000/api/contracts/test/token/call \
  -H 'Content-Type: application/json' \
  -d '{"method":"set_rate","kind":"exec","args":{"new_rate":"20000"}}'

# 批量交易（等价于 token_test.go 的 TestBatchSetUnitAndRate）
curl -X POST http://127.0.0.1:8000/api/contracts/test/token/batch \
  -H 'Content-Type: application/json' \
  -d '{"calls":[{"method":"set_rate","args":{"new_rate":"20000"}},{"method":"set_token_unit","args":{"unit":"1000000000000000000"}}]}'
```

支持的合约与函数清单见 `cmd/api-server/registry.go`，添加新函数只需在注册表中增加一行。
