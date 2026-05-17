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
NETWORK="42"
NAME=""
BUILD="true"

usage() {
    cat <<EOF
Usage:
  $(basename "$0") --env <env> --name <name> [options]

Options:
  --env <env>        Environment: local | test | main, loads configs/<env>.json
  --name <name>      Contract to upgrade: token | subnet
  --network <id>     SS58 network id, default: 42
  --build <bool>     Whether to run cargo wrevive build first, default: true

Config files (configs/<env>.json):
  url                Blockchain websocket url (required)
  suri               Signer secret uri (required)
  contracts          Deployed contract addresses (required)
    - token          Token proxy address
    - subnet         Subnet proxy address

Valid names:
  token              Deploy new Token implementation and upgrade proxy
  subnet             Deploy new Subnet implementation and upgrade proxy

Examples:
  $(basename "$0") --env local --name token
  $(basename "$0") --env test --name subnet --build false
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="$2"; shift 2 ;;
        --name) NAME="$2"; shift 2 ;;
        --network) NETWORK="$2"; shift 2 ;;
        --build) BUILD="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

if [[ -z "$NAME" ]]; then
    echo "Error: missing required --name flag" >&2
    usage
    exit 1
fi

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

if [[ "$BUILD" == "true" ]]; then
    case "$NAME" in
        token)
            cargo wrevive build -p token
            ;;
        subnet)
            cargo wrevive build -p subnet
            ;;
    esac
fi

ARGS=(
    -env "$ENV"
    -name "$NAME"
    -dir "$ROOT"
    -network "$NETWORK"
)

go run ./cmd/upgrade-contract "${ARGS[@]}"
