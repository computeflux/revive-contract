# WeTEE 合约部署与调试 UI

基于 [cathome/workflow/board](https://github.com/)（React 19 + Vite + Tailwind 4）整套代码改造的
合约部署、调试与函数调用界面。前端产物由 `hacks/deploy/cmd/api-server` **内嵌托管**，
单服务单端口运行，同源请求无 CORS 问题。

## 快速开始（推荐：单服务）

```bash
# 一键构建前端 + 启动服务（默认端口 8000）
./api-ui.sh

# 指定端口
PORT=9000 ./api-ui.sh
```

浏览器打开 http://127.0.0.1:8000 即可（UI 与 API 同源）。

脚本流程：`npm run build` → 产物同步到 `cmd/api-server/web/`（go:embed）→ `go run ./cmd/api-server`。

## 开发模式（可选：vite dev + 热更新）

```bash
# 终端 1：API 服务（默认 8000，或 PORT=xxxx 指定）
go run ./cmd/api-server --port 8000

# 终端 2：vite 开发服务器（热更新，代理 /api 到后端）
VITE_API_PROXY_TARGET=http://127.0.0.1:8000 npm run dev -- --port 5175
```

> vite dev 模式下 `api.ts` 默认同源请求 `/api`，由 vite proxy 转发到后端；
> 生产构建（dist 内嵌）时同源请求直接命中 api-server，无需任何代理配置。

## 页面

| 页面 | 功能 |
|------|------|
| 总览 `/` | 环境列表（local/test/main）、RPC、签名账户、余额、合约地址 |
| 合约部署 `/deploy` | 单合约部署、全量部署（subnet+token+proxy+创世初始化）、热升级 |
| 合约调试 `/debug` | 选择环境/合约 → 动态生成函数表单 → 查询（DryRun）/提交交易/批量交易 → 结果展示 |
| ERC20 管理 `/erc20` | 已注册代币列表、注册新代币、调整汇率 rate / 计价单位 unit / 启停状态 |
| 账户管理 `/account` | 账户信息（SS58/H160/余额）、Revive Map Account |

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `PORT` | 8000 | api-ui.sh 服务端口 |
| `API_PORT` | 8000 | api-server 监听端口（单独启动时） |
| `VITE_API_URL` | 同源 `/api` | 前端 API 地址覆盖（一般无需设置） |
| `VITE_API_PROXY_TARGET` | http://127.0.0.1:8000 | vite dev 代理目标 |

## API 接口一览

```
GET  /api/health
GET  /api/envs
GET  /api/envs/{env}
POST /api/account/{env}/map
GET  /api/contracts/{env}/{name}/methods
POST /api/contracts/{env}/{name}/call
POST /api/contracts/{env}/{name}/batch
POST /api/deploy
POST /api/deploy-full
POST /api/upgrade
```

完整说明见 `hacks/deploy/Readme.md` 的「API 服务」一节。
