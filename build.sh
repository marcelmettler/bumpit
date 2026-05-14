#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

echo "==> Running go mod tidy..."
go mod tidy

echo "==> Building..."
go build -o chorekit .

echo "==> Build successful! Binary: ./chorekit"
echo ""
echo "Run: ./chorekit [directory]"
