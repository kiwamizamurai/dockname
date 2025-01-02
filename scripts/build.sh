#!/bin/bash

# Get absolute path of the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Project root directory (parent directory of the script)
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"
rm -rf build/*

go mod tidy

CGO_ENABLED=0 go build -o build/dockname ./cmd/dockname

echo "Build completed: build/dockname"