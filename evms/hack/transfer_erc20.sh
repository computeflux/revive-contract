#!/bin/bash
set -euo pipefail

# ============================================================
# ERC20 转账脚本 - Polkadot Hub 测试网
#
# 用法:
#   chmod +x transfer_erc20.sh
#   ./transfer_erc20.sh <合约地址> <接收地址> <数量>
#
# 示例:
#   ./transfer_erc20.sh 0x1234... 0xabcd... 100
#
# 环境变量:
#   PRIVATE_KEY=0x...  (必填，也可放在 .env 文件中)
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# ── 参数检查 ──
if [ $# -lt 3 ]; then
    echo -e "${RED}用法: $0 <合约地址> <接收地址> <数量>${NC}"
    echo "示例: $0 0x1234... 0xabcd... 100"
    exit 1
fi

TOKEN_ADDRESS=$1
TO_ADDRESS=$2
AMOUNT=$3

# ── 加载 .env ──
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [ -f "$PROJECT_DIR/.env" ]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

if [ -z "${PRIVATE_KEY:-}" ]; then
    echo -e "${RED}错误: 请设置 PRIVATE_KEY 环境变量或在 $PROJECT_DIR/.env 中配置${NC}"
    exit 1
fi

# ── 显示转账信息 ──
echo "============================================"
echo " ERC20 转账 - Polkadot Hub 测试网"
echo "============================================"
echo "代币合约: $TOKEN_ADDRESS"
echo "接收地址: $TO_ADDRESS"
echo "转账数量: $AMOUNT"
echo "============================================"
echo ""

# ── 执行转账 ──
cd "$PROJECT_DIR"

export ERC20_ADDRESS="$TOKEN_ADDRESS"
export ERC20_TO="$TO_ADDRESS"
export ERC20_AMOUNT="$AMOUNT"

npx hardhat run hack/transfer_erc20.js \
    --config hardhat.config.polkadot.js \
    --network polkadotTestnet

echo ""
echo -e "${GREEN}转账完成!${NC}"
echo "查看交易: https://blockscout-testnet.polkadot.io/address/$TOKEN_ADDRESS"
