#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

echo "==> Running go mod tidy..."
go mod tidy

echo "==> Building..."
go build -o bumpit .

echo "==> Build successful! Binary: ./bumpit"
echo ""
echo "Run: ./bumpit [directory]"
