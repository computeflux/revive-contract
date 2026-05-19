#!/usr/bin/env bash
# Usage: ./token_update.sh [local|test|main]  (default: local)
SOURCE="$0"
while [ -h "$SOURCE"  ]; do
    DIR="$( cd -P "$( dirname "$SOURCE"  )" && pwd  )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /*  ]] && SOURCE="$DIR/$SOURCE"
done
DIR="$( cd -P "$( dirname "$SOURCE"  )" && pwd  )"

ENV="${1:-local}"
CONFIG="$DIR/configs/${ENV}.json"

if [ ! -f "$CONFIG" ]; then
    echo "Config not found: $CONFIG"
    exit 1
fi

cd "$DIR/../../"
cargo wrevive build -p token

cd $DIR
go test -run ^TestTokenUpdate$ -args -env="$ENV"
