#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# ──────────────────────────────────────────────
# 生成 FLUX_TEST 部署配置
#  1. 生成新的以太坊私钥（如果未通过 PRIVATE_KEY 指定）
#  2. 部署 FLUX_TEST 合约，部署者即为合约 Owner
#  3. 自动等待确认并验证合约源码
#  4. 写入 flux_config.json，actives 服务可直接使用
# ──────────────────────────────────────────────

NETWORK="${1:-polkadotTestnet}"

echo "========================================="
echo "FLUX_TEST 部署 + 验证"
echo "========================================="
echo ""

# ── 1. 私钥 ──────────────────────────────────
if [ -z "${PRIVATE_KEY:-}" ]; then
    echo "步骤 1: 生成新以太坊私钥..."
    PRIVATE_KEY="0x$(openssl rand -hex 32)"
fi
export PRIVATE_KEY

# 推导地址
DEPLOYER_ADDR=$(node -e "
const {Wallet} = require('ethers');
const w = new Wallet('$PRIVATE_KEY');
console.log(w.address);
")
echo "  Deployer: $DEPLOYER_ADDR"

# ── 2. 部署 + 验证 + 写入配置 ──────────────────
echo ""
echo "步骤 2: 部署 FLUX_TEST 到 $NETWORK ..."
echo "  部署完成后自动等待确认，然后验证源码并写入配置"
echo ""

npx hardhat run hack/deploy_flux_test.js \
    --config hardhat.config.polkadot.js \
    --network "$NETWORK" 2>&1

echo ""
echo "✅ 完成！"
