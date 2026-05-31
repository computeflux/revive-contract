#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "========================================="
echo "生成合约 Go 绑定"
echo "========================================="
echo ""

# 1. 检查 abigen 是否安装
echo "步骤 1: 检查 abigen..."
if ! command -v abigen &> /dev/null; then
    echo "❌ abigen 未安装"
    echo "请运行: go install github.com/ethereum/go-ethereum/cmd/abigen@latest"
    exit 1
fi
echo "✅ abigen 已安装"
echo ""

# 2. 编译合约
echo "步骤 2: 编译合约..."
npx hardhat compile --config hardhat.config.polkadot.js
if [ $? -ne 0 ]; then
    echo "❌ 合约编译失败"
    exit 1
fi
echo "✅ 合约编译成功"
echo ""

# 3. 生成 Go 绑定
OUT_DIR="$SCRIPT_DIR/../../token/service"

generate_binding() {
    local contract_name="$1"
    local artifact_path="artifacts/contracts/${contract_name}.sol/${contract_name}.json"
    local abi_file="/tmp/${contract_name}.abi"

    if [ ! -f "$artifact_path" ]; then
        echo "⚠️  跳过 ${contract_name}: artifact 不存在"
        return
    fi

    echo "生成 ${contract_name} 绑定..."
    jq '.abi' "$artifact_path" > "$abi_file"
    abigen \
        --abi="$abi_file" \
        --pkg=service \
        --type="${contract_name}" \
        --out="${OUT_DIR}/${contract_name}.go"

    if [ $? -eq 0 ]; then
        echo "  ✅ ${OUT_DIR}/${contract_name}.go"
    else
        echo "  ❌ ${contract_name} 绑定生成失败"
    fi
}

echo "步骤 3: 生成 Go 绑定..."
generate_binding "Subnet"
generate_binding "Token"

echo ""
echo "完成！"
