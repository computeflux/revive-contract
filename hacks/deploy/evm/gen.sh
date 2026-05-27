#!/bin/bash
# Generate EVM Go bindings from Solidity ABI JSON
# 从 Solidity ABI JSON 生成 EVM Go 绑定
#
# Usage: ./gen.sh
# Prerequisites: abigen in PATH (go install github.com/ethereum/go-ethereum/cmd/abigen@latest)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ABI_FILE="$ROOT_DIR/target/token.sol.abi.json"
OUT_FILE="$SCRIPT_DIR/binds/token.go"
PKG="tokensol"
TYPE="Token"

# Check abigen
if ! command -v abigen &>/dev/null; then
    echo "ERROR: abigen not found. Install with: go install github.com/ethereum/go-ethereum/cmd/abigen@latest"
    exit 1
fi

# Check ABI file
if [ ! -f "$ABI_FILE" ]; then
    echo "ERROR: ABI file not found at $ABI_FILE"
    echo "Build the token contract first: cargo build -p token"
    exit 1
fi

echo "Generating Go bindings..."
echo "  ABI: $ABI_FILE"
echo "  Out: $OUT_FILE"
echo "  Pkg: $PKG"
echo "  Type: $TYPE"

abigen --abi "$ABI_FILE" --pkg "$PKG" --type "$TYPE" --out "$OUT_FILE"

echo "Done: $OUT_FILE"
