#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

VERSION=$(cat .version | tr -d '[:space:]')
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DIR="$PROJECT_DIR/build"

echo "Building neodlp $VERSION (commit: $COMMIT)"

mkdir -p "$BUILD_DIR"

build() {
    local os=$1 arch=$2 suffix=$3
    local output="$BUILD_DIR/neodlp-${os}-${arch}${suffix}"
    echo "  → $os/$arch"
    GOOS=$os GOARCH=$arch go build \
        -ldflags="-X github.com/rkriad585/neodlp/internal/version.Commit=$COMMIT" \
        -o "$output" \
        .
}

build windows amd64 .exe
build linux amd64 ""
build darwin amd64 ""
build darwin arm64 ""

echo "Build complete. Artifacts in $BUILD_DIR/"
ls -lh "$BUILD_DIR/"
