# Token 合约 ERC20 支付方案

## 概述

在现有 Token 合约基础上增加 ERC20 代币支付能力。用户可通过已注册的 ERC20 代币（如 USDC、USDT、DAI）进行积分充值，
Subnet 可将积分以 ERC20 代币形式提现给用户。

基于 `wrevive_api::Env` 的 `call_evm` 接口调用 Solidity ERC20 合约。

---

## 1. 核心能力

| 操作 | ERC20 方法 | 4-byte selector | 说明 |
|---|---|---|---|
| 用户代付 | `transferFrom(from, to, amount)` | `0x23b872dd` | Token 合约从用户钱包扣 ERC20 |
| 合约转出 | `transfer(to, amount)` | `0xa9059cbb` | Token 合约向用户转 ERC20 |
| 查询余额 | `balanceOf(account)` | `0x70a08231` | 查询任意地址的 ERC20 余额 |

ABI 编码示例（**关键：EVM 使用大端序**）：

```
// U256: 必须使用 to_be_bytes() —— EVM ABI 是大端序
// 而 SCALE 编码 (Encode trait) 使用 to_le_bytes()
// wrevive U256 提供: to_be_bytes() / to_le_bytes() / as_bytes(CallMode)

// transferFrom(address,address,uint256) → 0x23b872dd
input = [0x23, 0xb8, 0x72, 0xdd]           // 4-byte selector
      + [0u8; 12] + from.as_ref()             // from (12零 + 20 bytes)
      + [0u8; 12] + to.as_ref()               // to (12零 + 20 bytes)
      + amount.to_be_bytes()                  // amount (32 bytes, 大端)

// transfer(address,uint256) → 0xa9059cbb
input = [0xa9, 0x05, 0x9c, 0xbb]           // 4-byte selector
      + [0u8; 12] + to.as_ref()               // to (12零 + 20 bytes)
      + amount.to_be_bytes()                  // amount (32 bytes, 大端)
```

---

## 2. 新增存储

```rust
/// 单个 ERC20 代币配置
#[derive(Clone, Encode, Decode, Debug)]
struct ERC20Config {
    active: bool,       // 是否启用
    rate: U256,         // 兑换率: 1 代币 = rate 积分
    decimals: u8,       // 代币小数位
}

/// 已注册的 ERC20 代币: token_address → config
const ERC20_TOKENS: Mapping<Address, ERC20Config> = mapping!(b"erc20_tokens");
```

---

## 3. 新增函数

### 3.1 管理（Owner only）

| 函数 | 参数 | 说明 |
|---|---|---|
| `set_erc20_token` | `token: Address, active: bool, rate: U256, decimals: u8` | 注册/更新/停用 ERC20 代币 |
| `emergency_withdraw_erc20` | `token: Address` | 将合约持有的该 ERC20 全部转给 Owner |

### 3.2 充值

| 函数 | 调用者 | 说明 |
|---|---|---|
| `recharge_erc20` | 用户 | SCALE 编码。执行 `transferFrom` 并换算积分 |
| `recharge_erc20_sol` | 用户 | Sol ABI 编码（适配 MetaMask 等钱包） |

### 3.3 提现

| 函数 | 调用者 | 说明 |
|---|---|---|
| `withdraw_erc20` | Subnet | SCALE 编码。扣积分，调用 `transfer` 转 ERC20 |
| `withdraw_erc20_sol` | Subnet | Sol ABI 编码 |

### 3.4 查询

| 函数 | 说明 |
|---|---|
| `get_erc20_config(token)` → `Option<ERC20Config>` | 查询代币配置 |
| `get_erc20_config_sol(token)` → `(bool, U256, u8)` | 同上（Sol ABI） |

---

## 4. 流程

### 4.1 充值（用户 → ERC20 → 积分）

```
1. 用户调用 ERC20.approve(Token合约地址, amount)   // 链下/钱包操作
2. 用户调用 Token.recharge_erc20(token地址, amount)
3. Token 合约:
   a. 校验 token 已注册且 active
   b. 计算 rate = config.rate, dec = 10^config.decimals
   c. 调用 call_evm: ERC20.transferFrom(caller, self, amount)
   d. 如果失败 → Err(ERC20TransferFailed)
   e. points = amount * rate / dec
   f. 校验 points > 0
   g. CREDITS[caller] += points
   h. 发出 RechargeERC20 事件
   i. 返回 points
```

### 4.2 提现（积分 → ERC20 → 用户）

```
1. Subnet 调用 Token.withdraw_erc20(token地址, user, points)
2. Token 合约:
   a. ensure_subnet()
   b. 校验 token 已注册且 active
   c. 计算 rate = config.rate, dec = 10^config.decimals
   d. amount = points * dec / rate
   e. 校验 amount > 0
   f. 校验 CREDITS[user] >= points
   g. CREDITS[user] -= points
   h. 调用 call_evm: ERC20.transfer(user, amount)
   i. 如果失败 → Err(ERC20TransferFailed)
   j. 发出 WithdrawERC20 事件
```

### 4.3 积分换算公式

每个 ERC20 代币有独立的 `decimals`（小数位），换算时必须除以 `10^decimals`：

```
充值: points = erc20_amount * rate / 10^decimals
提现: erc20_amount = points * 10^decimals / rate
```

与原生币对比：

```
原生币: points = value * RATE / TOKEN_UNIT     (TOKEN_UNIT = 10^15, DOT Planck)
ERC20:   points = amount * rate / 10^decimals   (如 USDC decimals=6 → 10^6)
```

示例（不同 decimals 需要不同的 rate 才能达到相同定价）：

| 代币 | decimals | 目标: 1币=N积分 | rate 配置 | 充值 100 币 | 得到积分 |
|---|---|---|---|---|---|
| USDC | 6 | 1000 | 1000 | 100,000,000 | 100,000 |
| USDT | 6 | 1000 | 1000 | 100,000,000 | 100,000 |
| DAI | 18 | 1000 | 1000 | 100×10^18 | 100,000 |
| WBTC | 8 | 1000 | 1000 | 100×10^8 | 100,000 |

---

## 5. 端序处理（关键）

### 端序差异

| 环境 | 整数编码 | U256 方法 |
|---|---|---|
| SCALE (PolkaVM 内部) | 小端 (LE) | `to_le_bytes()` |
| EVM ABI (Solidity) | 大端 (BE) | `to_be_bytes()` |

### 地址处理

- `Address` / `[u8; 20]` 无端序问题
- ABI 编码时左侧填充 12 个零字节 → 32 字节

### calldata 构造要点

```rust
fn build_transfer_from_calldata(token: &Address, from: &Address, to: &Address, amount: U256) -> Vec<u8> {
    let mut calldata = Vec::with_capacity(4 + 32 * 3);
    // 4-byte selector: transferFrom
    calldata.extend_from_slice(&[0x23, 0xb8, 0x72, 0xdd]);
    // from: left-pad 20 bytes to 32
    calldata.extend_from_slice(&[0u8; 12]);
    calldata.extend_from_slice(from.as_ref());
    // to: left-pad 20 bytes to 32
    calldata.extend_from_slice(&[0u8; 12]);
    calldata.extend_from_slice(to.as_ref());
    // amount: 大端 32 bytes
    calldata.extend_from_slice(&amount.to_be_bytes());
    calldata
}
```

### 返回值解析

`call_evm` 的 `output` 返回 EVM 返回值（大端序）。对于 `transferFrom`/`transfer`，
ERC20 返回 `bool`（1 byte），EVM ABI 编码为 32 字节大端 `uint256`：

```rust
fn decode_bool(output: &[u8]) -> bool {
    // output 至少 32 字节，最后一个字节为 true(1)/false(0)
    output.len() >= 32 && output[31] != 0
}
```

---

## 6. 关键设计决策

| 决策 | 理由 |
|---|---|
| 每代币独立 rate | 不同 ERC20 价值不同（USDC ≠ DAI），不能用统一定价 |
| 与原生币共用 CREDITS | 积分统一管理，用户不关心充值来源，简化账户模型 |
| `call_evm` 而非 `call` | ERC20 是 Solidity 合约，需要 EVM ABI 编码 |
| 手动 ABI encode | revive 环境不依赖 ethers，需要手动拼接 calldata |
| 合约持有 ERC20 余额 | 用户充值的 ERC20 留在合约地址，提现时从中转出，确保 "积分总量 ≤ ERC20 余额" |
| 充值需先 approve | 用户需要在 ERC20 合约中 approve Token 合约，符合标准 ERC20 授权模式 |

---

## 7. 新增错误类型

```rust
enum Error {
    // ... 已有 ...
    ERC20NotSupported,       // 未注册的 ERC20 代币
    ERC20Inactive,           // ERC20 代币已停用
    ERC20TransferFailed,     // EVM call_evm 失败
}
```

---

## 8. 事件

```
RechargeERC20:  user(UniAddr), token(Address), erc20_amount(U256), points(U256)
WithdrawERC20:  user(UniAddr), token(Address), erc20_amount(U256), points(U256)
```

---

## 9. 兼容性

- **原生币充值 `recharge()` / 提现 `withdraw()` 完全不变**
- ERC20 作为**额外支付方式**，与原生币并存
- `get_balance()` 返回**混合积分余额**（不区分来源）
- 后端 Gateway 查询积分余额同样不需要修改
- 如有需要，可后续增加 `get_balance_by_token(token, user)` 按来源区分

---

## 10. 待实现项

- [ ] `contract.rs`: 新增 ERC20Config、storage、函数
- [ ] `errors.rs`: 新增 3 个错误类型
- [ ] `tests.rs`: 单元测试（mock ERC20 或使用 off_chain `call_evm`）
- [ ] `hacks/`: 部署/配置脚本
- [ ] 前端: 充值支持选择 ERC20 代币
