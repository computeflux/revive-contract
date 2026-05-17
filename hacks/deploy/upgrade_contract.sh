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
POD_ID=""
BUILD="true"

usage() {
    cat <<EOF
Usage:
  $(basename "$0") --env <env> --name <name> [options]

Options:
  --env <env>        Environment: local | test | main, loads configs/<env>.json
  --name <name>      Contract to upgrade: cloud | subnet | pod-code | pod-contract
  --pod-id <id>      Pod ID (required when name=pod-contract)
  --network <id>     SS58 network id, default: 42
  --build <bool>     Whether to run cargo wrevive build first, default: true

Config files (configs/<env>.json):
  url                Blockchain websocket url (required)
  suri               Signer secret uri (required)
  contracts          Deployed contract addresses (required)
    - cloud          Cloud proxy address
    - subnet         Subnet proxy address

Upgrade types:
  cloud              Deploy new Cloud implementation and upgrade proxy
  subnet             Deploy new Subnet implementation and upgrade proxy
  pod-code           Upload new Pod code and update Cloud's pod code hash
  pod-contract       Update an existing pod's contract via Cloud

Examples:
  $(basename "$0") --env local --name cloud
  $(basename "$0") --env test --name subnet --build false
  $(basename "$0") --env local --name pod-code
  $(basename "$0") --env local --name pod-contract --pod-id 1
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="$2"; shift 2 ;;
        --name) NAME="$2"; shift 2 ;;
        --pod-id) POD_ID="$2"; shift 2 ;;
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
        cloud)
            cargo wrevive build -p cloud
            ;;
        subnet)
            cargo wrevive build -p subnet
            ;;
        pod-code|pod-contract)
            cargo wrevive build -p pod
            ;;
    esac
fi

if [[ "$NAME" == "pod-contract" && -z "$POD_ID" ]]; then
    echo "Error: --pod-id is required when --name=pod-contract" >&2
    usage
    exit 1
fi

ARGS=(
    -env "$ENV"
    -name "$NAME"
    -dir "$ROOT"
    -network "$NETWORK"
)

if [[ "$NAME" == "pod-contract" ]]; then
    ARGS+=( -pod-id "$POD_ID" )
fi

go run ./cmd/upgrade-contract "${ARGS[@]}"
