#!/usr/bin/env bash
set -euo pipefail

SOURCE="$0"
while [ -h "$SOURCE" ]; do
    DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
ROOT="$(cd "$DIR/../../" && pwd)"

cd "$ROOT"

echo "=== Building contracts to generate ABI JSON ==="
for pkg in cloud subnet pod proxy dao; do
    echo "  -> $pkg"
    cargo wrevive build -p "$pkg" --quiet
done

cd "$DIR/contracts"

echo ""
echo "=== Generating Go bindings ==="
for contract in cloud subnet pod proxy dao; do
    json="../../../target/$contract.json"
    if [[ -f "$json" ]]; then
        echo "  -> $contract"
        go-ink-gen -json "$json"
    else
        echo "  !! $json not found, skipping $contract"
    fi
done


# Remove generated dao/pod packages (not yet maintained, needs manual type fixes)Restoring hand-maintained types
rm -rf contracts/dao contracts/pod 2>/dev/null || true
echo "  -> removed generated dao/pod (needs manual type definitions)"

cd "$DIR"
echo ""
echo "=== Verifying Go build ==="
go build ./... && echo "OK" || echo "FAILED (see errors above)"
