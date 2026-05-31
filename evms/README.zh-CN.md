# Dtoken 合约（Hardhat）

<p align="right">
  <a href="./README.md">English</a> | <strong>简体中文</strong>
</p>

Dtoken 市场相关的智能合约（Hardhat 工程）。包含编译、测试、部署、（可选）验证脚本，支持 Polkadot Hub 测试网以及部分 EVM 网络。

## 前置条件

- Node.js + npm
- 用于部署的私钥（不要提交到仓库）

## 安装与配置

```bash
npm install
cp .env.example .env
```

## 常用命令

```bash
npm run compile
npm run test
```

### 部署

```bash
npm run deploy:polkadot
```

### 验证（Polkadot）

```bash
npm run verify:polkadot
```

## 生成 Go 绑定（供 `token` 后端使用）

仓库提供了脚本：

```bash
./generate-go-binding.sh
```

该脚本依赖 `abigen`，并将生成的绑定写入 `../service/ModelMarketplaceV3.go`（位于 `token` 工作区）。

## 相关项目

- Token 服务（DTOKEN）：`../token`
- DApp：`../dapp`
- 管理后台（Dashboard）：`../dashboard`

