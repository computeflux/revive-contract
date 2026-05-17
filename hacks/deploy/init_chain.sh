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
BUILD="true"

usage() {
    cat <<EOF
Usage:
  $(basename "$0") --env <env> [options]

Options:
  --env <env>        Environment: local | test | main, loads configs/<env>.json
  --network <id>     SS58 network id, default: 42
  --build <bool>     Whether to run cargo wrevive build first, default: true

Config files (configs/<env>.json):
  url                Blockchain websocket url (required)
  suri               Signer secret uri (required)
  genesis            Genesis node configuration (required)

Examples:
  $(basename "$0") --env local
  $(basename "$0") --env test
  $(basename "$0") --env main
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="$2"; shift 2 ;;
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

if [[ "$BUILD" == "true" ]]; then
    cargo wrevive build -p pod
    cargo wrevive build -p subnet
    cargo wrevive build -p cloud
    cargo wrevive build -p proxy
fi

go run ./cmd/deploy-full \
    -env "$ENV" \
    -dir "$ROOT" \
    -network "$NETWORK"
