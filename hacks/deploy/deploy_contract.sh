#!/usr/bin/env bash
set -euo pipefail

SOURCE="$0"
while [ -h "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
ROOT="$(cd "$DIR/../../" && pwd)"

ENV="local"
CONTRACT_NAME=""
CONTRACT_DIR=""
CODE_PATH=""
NETWORK="42"
BUILD="true"

usage() {
    cat <<EOF
Usage:
  $(basename "$0") --env <env> --name <contract-name> [options]

Options:
  --env <env>        Environment: local | test | main, loads configs/<env>.json
  --name <name>      Contract name, e.g. cloud / subnet / proxy
  --dir <dir>        Contract crate directory (directory containing Cargo.toml)
  --code <path>      Compiled .polkavm file path, default: <workspace-root>/target/<name>.release.polkavm
  --network <id>     SS58 network id, default: 42
  --build <bool>     Whether to run cargo wrevive build first, default: true

Config files (configs/<env>.json):
  url                Blockchain websocket url (required)
  suri               Signer secret uri (required)

Examples:
  $(basename "$0") --env local --name cloud --dir "$ROOT/revives/Cloud"
  $(basename "$0") --env test --name proxy --dir "$ROOT/revives/proxy"
  $(basename "$0") --env main --name subnet --dir "$ROOT/revives/Subnet"
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="$2"; shift 2 ;;
        --name) CONTRACT_NAME="$2"; shift 2 ;;
        --dir) CONTRACT_DIR="$2"; shift 2 ;;
        --code) CODE_PATH="$2"; shift 2 ;;
        --network) NETWORK="$2"; shift 2 ;;
        --build) BUILD="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

# Validate env config exists
CONFIG_FILE="$DIR/configs/$ENV.json"
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: config file not found: $CONFIG_FILE" >&2
    exit 1
fi

# Resolve SURI: only from configs/<env>.json
SURI="$(jq -r '.suri // empty' "$CONFIG_FILE")"
if [[ -z "$SURI" ]]; then
    echo "Error: missing suri in $CONFIG_FILE" >&2
    exit 1
fi

# Resolve URL: only from configs/<env>.json
CHAIN_URL="$(jq -r '.url // empty' "$CONFIG_FILE")"
if [[ -z "$CHAIN_URL" ]]; then
    echo "Error: missing url in $CONFIG_FILE" >&2
    exit 1
fi

cd "$DIR"

if [[ -z "$CONTRACT_NAME" ]]; then
    usage
    exit 1
fi

if [[ -z "$CONTRACT_DIR" ]]; then
    if [[ -d "$ROOT/revives/$CONTRACT_NAME" ]]; then
        CONTRACT_DIR="$ROOT/revives/$CONTRACT_NAME"
    elif [[ -d "$ROOT/revives/${CONTRACT_NAME^}" ]]; then
        CONTRACT_DIR="$ROOT/revives/${CONTRACT_NAME^}"
    else
        echo "Unable to infer contract dir, please pass --dir" >&2
        exit 1
    fi
fi

if [[ ! -f "$CONTRACT_DIR/Cargo.toml" ]]; then
    echo "No Cargo.toml found in contract dir: $CONTRACT_DIR" >&2
    exit 1
fi

WORKSPACE_ROOT="$CONTRACT_DIR"
while [[ "$WORKSPACE_ROOT" != "/" ]]; do
    if [[ -f "$WORKSPACE_ROOT/Cargo.toml" ]] && grep -q "^\[workspace\]" "$WORKSPACE_ROOT/Cargo.toml"; then
        break
    fi
    WORKSPACE_ROOT="$(dirname "$WORKSPACE_ROOT")"
done

if [[ "$WORKSPACE_ROOT" == "/" ]]; then
    echo "Unable to locate workspace root from contract dir: $CONTRACT_DIR" >&2
    exit 1
fi

if [[ -z "$CODE_PATH" ]]; then
    CODE_PATH="$WORKSPACE_ROOT/target/$CONTRACT_NAME.release.polkavm"
fi

if [[ "$BUILD" == "true" ]]; then
    cargo wrevive build --manifest-path "$CONTRACT_DIR/Cargo.toml"
fi

go run ./cmd/deploy-contract \
    -env "$ENV" \
    -name "$CONTRACT_NAME" \
    -dir "$WORKSPACE_ROOT" \
    -code "$CODE_PATH" \
    -network "$NETWORK"
