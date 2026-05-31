#!/usr/bin/env bash
set -euo pipefail

# ── Resolve script directory ───────────────────────────────
SOURCE="$0"
while [ -h "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
ROOT="$(cd "$DIR/.." && pwd)"

# ── Source .env if present ─────────────────────────────────
if [[ -f "$ROOT/.env" ]]; then
    set -a
    source "$ROOT/.env"
    set +a
fi

# ── Defaults ───────────────────────────────────────────────
ENV="local"
BUILD="true"
SKIP_DEPLOY="false"
PRIVATE_KEY="${PRIVATE_KEY:-}"
ECDSA_KEY="${ECDSA_KEY:-}"

# ── Usage ──────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage:
  $(basename "$0") --env <env> [options]

Options:
  --env <env>           Environment: local | test | main, loads configs/<env>.json
  --build <bool>        Whether to run "npx hardhat compile" first, default: true
  --skip-deploy         Skip contract deployment, only init subnet data
                        (uses SUBNET_CONTRACT_ADDRESS from env / .env)
  --private-key <hex>   Deployer private key (0x...), overrides PRIVATE_KEY env var
  --ecdsa-key <hex>     tee-provider ECDSA private key (0x...), overrides ECDSA_KEY env var

Config files (hack/configs/<env>.json):
  rpc_url               Blockchain RPC URL
  chain_id              Network chain ID
  hardhat_network       Hardhat network name (must match hardhat.config.polkadot.js)
  genesis               Subnet genesis data (regions, workers, secrets, etc.)

Environment variables (also read from .env):
  PRIVATE_KEY              Deployer private key (0x...), required for deploy
  ECDSA_KEY                tee-provider ECDSA private key (0x...), optional
  SUBNET_CONTRACT_ADDRESS  Required when using --skip-deploy
  TOKEN_CONTRACT_ADDRESS   Optional, used with --skip-deploy

Examples:
  $(basename "$0") --env local                              # Full deploy + init
  $(basename "$0") --env test --private-key 0x...            # Deploy + init on testnet
  $(basename "$0") --env test --skip-deploy                  # Only init existing contracts
EOF
}

# ── Parse args ─────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="$2"; shift 2 ;;
        --build) BUILD="$2"; shift 2 ;;
        --skip-deploy) SKIP_DEPLOY="true"; shift ;;
        --private-key) PRIVATE_KEY="$2"; shift 2 ;;
        --ecdsa-key) ECDSA_KEY="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

# ── Validate config file ───────────────────────────────────
CONFIG_FILE="$DIR/configs/$ENV.json"
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: config file not found: $CONFIG_FILE" >&2
    echo "Available configs:" >&2
    ls "$DIR/configs/" >&2
    exit 1
fi

# ── Load config ────────────────────────────────────────────
RPC_URL="$(jq -r '.rpc_url // empty' "$CONFIG_FILE")"
CHAIN_ID="$(jq -r '.chain_id // empty' "$CONFIG_FILE")"
HARDHAT_NETWORK="$(jq -r '.hardhat_network // empty' "$CONFIG_FILE")"

if [[ -z "$HARDHAT_NETWORK" ]]; then
    echo "Error: missing hardhat_network in $CONFIG_FILE" >&2
    exit 1
fi

# ── Validate private key (only needed for deploy) ──────────
if [[ "$SKIP_DEPLOY" != "true" ]]; then
    if [[ -z "$PRIVATE_KEY" ]]; then
        PRIVATE_KEY="$(jq -r '.private_key // empty' "$CONFIG_FILE")"
    fi
    if [[ -z "$PRIVATE_KEY" ]]; then
        echo "Error: PRIVATE_KEY not set. Provide via --private-key, PRIVATE_KEY env var, .env, or in $CONFIG_FILE" >&2
        exit 1
    fi
fi

# ── Resolve ECDSA key ──────────────────────────────────────
if [[ -z "$ECDSA_KEY" ]]; then
    ECDSA_KEY="$(jq -r '.ecdsa_key // empty' "$CONFIG_FILE")"
fi

# ── Print info ─────────────────────────────────────────────
echo "========================================"
echo " Subnet Contract Init"
echo "========================================"
echo "Environment:    $ENV"
echo "Config:         $CONFIG_FILE"
echo "RPC URL:        ${RPC_URL:-"(from hardhat config)"}"
echo "Chain ID:       ${CHAIN_ID:-"(from hardhat config)"}"
echo "Hardhat network: $HARDHAT_NETWORK"
echo "Build:          $BUILD"
echo "Skip deploy:    $SKIP_DEPLOY"
echo "ECDSA key set:  $(if [[ -n "$ECDSA_KEY" ]]; then echo "yes"; else echo "no"; fi)"
echo ""

# ── Export for hardhat / deploy script ─────────────────────
export PRIVATE_KEY
export ECDSA_KEY

cd "$ROOT"

# ── Build ──────────────────────────────────────────────────
if [[ "$BUILD" == "true" ]]; then
    echo "--- Compiling contracts ---"
    npx hardhat compile --config hardhat.config.polkadot.js
    echo ""
fi

# ── Deploy (or skip) ──────────────────────────────────────
if [[ "$SKIP_DEPLOY" == "true" ]]; then
    SUBNET_ADDR="${SUBNET_CONTRACT_ADDRESS:-}"
    CLOUD_ADDR="${TOKEN_CONTRACT_ADDRESS:-}"
    if [[ -z "$SUBNET_ADDR" ]]; then
        echo "Error: --skip-deploy requires SUBNET_CONTRACT_ADDRESS (set in .env or export it)" >&2
        exit 1
    fi
    echo "--- Skipping deploy, using existing contracts ---"
    echo "Token  (cloud) : ${CLOUD_ADDR:-unknown}"
    echo "Subnet          : $SUBNET_ADDR"
    echo ""
else
    echo "--- Deploying Subnet + Token ---"
    DEPLOY_OUTPUT="$(mktemp)"
    npx hardhat run hack/deploy_subnet_token.js \
        --config hardhat.config.polkadot.js \
        --network "$HARDHAT_NETWORK" | tee "$DEPLOY_OUTPUT"

    # ── Parse deployed addresses ─────────────────────────────
    SUBNET_ADDR="$(grep -o 'SUBNET_CONTRACT_ADDRESS=0x[0-9a-fA-F]*' "$DEPLOY_OUTPUT" | cut -d= -f2 || true)"
    CLOUD_ADDR="$(grep -o 'TOKEN_CONTRACT_ADDRESS=0x[0-9a-fA-F]*' "$DEPLOY_OUTPUT" | cut -d= -f2 || true)"
    rm -f "$DEPLOY_OUTPUT"

    if [[ -z "$SUBNET_ADDR" ]]; then
        echo "Error: could not parse SUBNET_CONTRACT_ADDRESS from deploy output" >&2
        exit 1
    fi
    echo "Token  (cloud) : ${CLOUD_ADDR:-unknown}"
    echo "Subnet          : $SUBNET_ADDR"
    echo ""
fi

# ── Init subnet chain data ────────────────────────────────
echo "--- Initializing Subnet chain data ---"
export SUBNET_CONTRACT_ADDRESS="$SUBNET_ADDR"
export CONFIG_FILE="$CONFIG_FILE"
npx hardhat run hack/init_subnet_data.js \
    --config hardhat.config.polkadot.js \
    --network "$HARDHAT_NETWORK"

echo ""
echo "Done."
