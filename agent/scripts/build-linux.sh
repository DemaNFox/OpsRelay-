#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."
mkdir -p dist
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/opsrelay-agent-linux-amd64 ./cmd/opsrelay-agent
echo "Built dist/opsrelay-agent-linux-amd64"
