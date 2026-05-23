#!/usr/bin/env bash
# build.sh - Cross-platform Go build script (Linux / macOS)
# Builds neodlp binaries for all supported OS/arch combinations.

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
BINARY_NAME="neodlp"
PUBLISHER_NAME="rkriad585"
PUBLISHER_EMAIL="rkriad585@gmail.com"
LDFLAGS_PREFIX="neodlp/internal/version"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Resolve version ─────────────────────────────────────────────────────────
VERSION_FILE="${SCRIPT_DIR}/.version"
if [[ -f "$VERSION_FILE" ]]; then
    VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
else
    VERSION="0.0.0"
    echo "⚠  .version file not found, defaulting to ${VERSION}"
fi

# ── Resolve Git commit ──────────────────────────────────────────────────────
COMMIT="$(git -C "$SCRIPT_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")"

# ── Detect host architecture ────────────────────────────────────────────────
HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
    x86_64)  HOST_ARCH="amd64" ;;
    aarch64) HOST_ARCH="arm64" ;;
    arm64)   HOST_ARCH="arm64" ;;
    *)       HOST_ARCH="amd64" ;;
esac

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║          neodlp Cross-Platform Builder           ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""
echo "  Version  : ${VERSION}"
echo "  Commit   : ${COMMIT}"
echo "  Publisher: ${PUBLISHER_NAME} <${PUBLISHER_EMAIL}>"
echo "  Host Arch: ${HOST_ARCH}"
echo ""

# ── Target matrix ────────────────────────────────────────────────────────────
TARGETS=(
    "windows amd64 .exe"
    "windows arm64 .exe"
    "linux   amd64"
    "linux   arm64"
    "darwin  amd64"
    "darwin  arm64"
)

# ── Prepare output directory ─────────────────────────────────────────────────
OUT_DIR="${SCRIPT_DIR}/bin"
mkdir -p "$OUT_DIR"

# ── Build ldflags ────────────────────────────────────────────────────────────
LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X ${LDFLAGS_PREFIX}.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X ${LDFLAGS_PREFIX}.Commit=${COMMIT}"
LDFLAGS="${LDFLAGS} -X ${LDFLAGS_PREFIX}.PublisherName=${PUBLISHER_NAME}"
LDFLAGS="${LDFLAGS} -X ${LDFLAGS_PREFIX}.PublisherEmail=${PUBLISHER_EMAIL}"

# ── Build loop ───────────────────────────────────────────────────────────────
BUILT=0
FAILED=0
TOTAL=${#TARGETS[@]}
START_TIME=$(date +%s)

for entry in "${TARGETS[@]}"; do
    # shellcheck disable=SC2086
    set -- $entry
    GOOS="$1"
    GOARCH="$2"
    EXT="${3:-}"

    export GOOS GOARCH
    export CGO_ENABLED=0

    OUT_NAME="${BINARY_NAME}-${GOOS}-${GOARCH}${EXT}"
    OUT_PATH="${OUT_DIR}/${OUT_NAME}"

    IDX=$(( BUILT + FAILED + 1 ))
    printf "  [%d/%d] Building %s ... " "$IDX" "$TOTAL" "$OUT_NAME"

    if go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_PATH" .; then
        SIZE=$(du -h "$OUT_PATH" | cut -f1)
        echo "OK (${SIZE})"
        BUILT=$(( BUILT + 1 ))
    else
        echo "FAILED"
        FAILED=$(( FAILED + 1 ))
    fi
done

# ── Summary ──────────────────────────────────────────────────────────────────
END_TIME=$(date +%s)
DURATION=$(( END_TIME - START_TIME ))

echo ""
echo "══════════════════════════════════════════════════"
echo "  Build complete in ${DURATION}s"
echo "  Success: ${BUILT} / ${TOTAL}"
if [[ $FAILED -gt 0 ]]; then
    echo "  Failed : ${FAILED} / ${TOTAL}"
fi
echo "  Output : ${OUT_DIR}"
echo "══════════════════════════════════════════════════"
echo ""

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
