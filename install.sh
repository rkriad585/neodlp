#!/usr/bin/env bash
# install.sh - Alias/identical file to installer.sh for neodlp
# Auto detects OS and architecture, downloads the release binary, sets up PATH, and supports self-uninstallation.

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
PROJECT_NAME="neodlp"
PUBLISHER_NAME="rkriad585"
GITHUB_REPO="rkriad585/neodlp"

# Paths
CONFIG_DIR="${HOME}/.config/neostore/${PROJECT_NAME}"
BIN_DIR="${CONFIG_DIR}/bin"
BINARY_PATH="${BIN_DIR}/${PROJECT_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Helper: Clean PATH from a profile file
clean_profile() {
    local profile="$1"
    if [[ -f "$profile" ]]; then
        # Platform-independent filtering without depending on sed -i differences
        grep -v "neostore/${PROJECT_NAME}/bin" "$profile" > "${profile}.tmp" || true
        mv "${profile}.tmp" "$profile"
    fi
}

# Helper: Add PATH to a profile file if not already present
add_to_profile() {
    local profile="$1"
    if [[ -f "$profile" ]]; then
        if ! grep -q "neostore/${PROJECT_NAME}/bin" "$profile"; then
            echo "" >> "$profile"
            echo "# neodlp PATH configuration" >> "$profile"
            echo "export PATH=\"\$HOME/.config/neostore/${PROJECT_NAME}/bin:\$PATH\"" >> "$profile"
        fi
    fi
}

# ── Check for Uninstallation Flag ───────────────────────────────────────────
IS_UNINSTALL=false
for arg in "$@"; do
    if [[ "$arg" == "--selfuninstall" || "$arg" == "-selfuninstall" || "$arg" == "--uninstall" || "$arg" == "-u" ]]; then
        IS_UNINSTALL=true
    fi
done

if [ "$IS_UNINSTALL" = true ]; then
    echo -e ""
    echo -e "${YELLOW}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║              neodlp Uninstaller                  ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════════════╝${NC}"
    echo -e ""

    # 1. Remove binary and directories
    if [[ -f "$BINARY_PATH" ]]; then
        printf "  Removing binary: %s ... " "$BINARY_PATH"
        rm -f "$BINARY_PATH"
        echo -e "${GREEN}OK${NC}"
    fi

    if [[ -d "$CONFIG_DIR" ]]; then
        printf "  Removing config directory: %s ... " "$CONFIG_DIR"
        rm -rf "$CONFIG_DIR"
        echo -e "${GREEN}OK${NC}"
    fi

    # 2. Clean shell profiles
    echo -e "  Cleaning shell profile PATH configurations ... "
    PROFILES=(
        "${HOME}/.bashrc"
        "${HOME}/.zshrc"
        "${HOME}/.profile"
        "${HOME}/.bash_profile"
    )
    for p in "${PROFILES[@]}"; do
        if [[ -f "$p" ]]; then
            printf "    Updating %s ... " "$(basename "$p")"
            clean_profile "$p"
            echo -e "${GREEN}OK${NC}"
        fi
    done

    echo -e ""
    echo -e "${GREEN}  neodlp has been successfully uninstalled from your system.${NC}"
    echo -e "${CYAN}  Please restart your shell or run: hash -r${NC}"
    echo -e ""
    exit 0
fi

# ── Installation / Update Flow ───────────────────────────────────────────────
echo -e ""
echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║               neodlp Installer                   ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
echo -e ""

# 1. Resolve Version from GitHub
printf "  Checking latest version from GitHub ... "
VERSION_URL="https://raw.githubusercontent.com/${GITHUB_REPO}/main/.version"
if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "$VERSION_URL" | tr -d '[:space:]')
elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "$VERSION_URL" | tr -d '[:space:]')
else
    echo -e "${RED}FAILED${NC}"
    echo "Error: curl or wget is required to run this installer." >&2
    exit 1
fi
echo -e "${GREEN}${VERSION}${NC}"

# 2. Detect System OS and Architecture
printf "  Detecting platform ... "
OS_NAME=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS_NAME" in
    linux*)  OS="linux" ;;
    darwin*) OS="darwin" ;;
    *)
        echo -e "${RED}FAILED${NC}"
        echo "Error: Unsupported operating system: ${OS_NAME}" >&2
        exit 1
        ;;
esac

ARCH_NAME=$(uname -m)
case "$ARCH_NAME" in
    x86_64)  ARCH="amd64" ;;
    amd64)   ARCH="amd64" ;;
    arm64)   ARCH="arm64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo -e "${RED}FAILED${NC}"
        echo "Error: Unsupported CPU architecture: ${ARCH_NAME}" >&2
        exit 1
        ;;
esac
echo -e "${GREEN}${OS}-${ARCH}${NC}"

# 3. Download the Release Binary
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${PROJECT_NAME}-${OS}-${ARCH}"
echo -e "  Downloading binary from ${DOWNLOAD_URL} ...\n"

# Ensure directories exist
mkdir -p "$BIN_DIR"

if command -v curl >/dev/null 2>&1; then
    curl -L --progress-bar -o "$BINARY_PATH" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget --show-progress -O "$BINARY_PATH" "$DOWNLOAD_URL"
else
    echo -e "  ${RED}✗ Failed to download binary. No curl or wget found.${NC}" >&2
    exit 1
fi

if [[ ! -f "$BINARY_PATH" ]]; then
    echo -e "  ${RED}✗ Downloaded file not found. Download failed.${NC}" >&2
    exit 1
fi

chmod +x "$BINARY_PATH"
SIZE=$(du -h "$BINARY_PATH" | cut -f1)
echo -e "  ${GREEN}✓ Successfully downloaded ${PROJECT_NAME} (${SIZE})${NC}"

# 3.1 Download yt-dlp dependency
case "${OS}" in
    linux)
        if [ "${ARCH}" = "amd64" ]; then
            YTDLP_ASSET="yt-dlp"
        else
            YTDLP_ASSET="yt-dlp_linux_aarch64"
        fi
        ;;
    darwin)
        YTDLP_ASSET="yt-dlp_macos"
        ;;
esac

YTDLP_DOWNLOAD_URL="https://github.com/yt-dlp/yt-dlp/releases/download/2026.03.17/${YTDLP_ASSET}"
YTDLP_PATH="${BIN_DIR}/yt-dlp"

echo -e "  Downloading yt-dlp dependency from ${YTDLP_DOWNLOAD_URL} ...\n"

if command -v curl >/dev/null 2>&1; then
    curl -L --progress-bar -o "$YTDLP_PATH" "$YTDLP_DOWNLOAD_URL" || true
elif command -v wget >/dev/null 2>&1; then
    wget --show-progress -O "$YTDLP_PATH" "$YTDLP_DOWNLOAD_URL" || true
fi

if [[ -f "$YTDLP_PATH" ]]; then
    chmod +x "$YTDLP_PATH"
    YTDLP_SIZE=$(du -h "$YTDLP_PATH" | cut -f1)
    echo -e "  ${GREEN}✓ Successfully downloaded yt-dlp (${YTDLP_SIZE})${NC}"
else
    echo -e "  ${YELLOW}⚠ Warning: Failed to pre-download yt-dlp. neodlp will attempt to install it on first run.${NC}"
fi

# 4. Configure PATH Environment Variable
printf "  Configuring shell profile PATH configurations ... "
PROFILES=(
    "${HOME}/.bashrc"
    "${HOME}/.zshrc"
    "${HOME}/.profile"
    "${HOME}/.bash_profile"
)
UPDATED_PROFILES=()

# Always check active shell profiles
ACTIVE_SHELL=$(basename "$SHELL")
if [ "$ACTIVE_SHELL" = "bash" ]; then
    # Ensure .bashrc exists or create it
    touch "${HOME}/.bashrc"
    add_to_profile "${HOME}/.bashrc"
    UPDATED_PROFILES+=(".bashrc")
elif [ "$ACTIVE_SHELL" = "zsh" ]; then
    # Ensure .zshrc exists or create it
    touch "${HOME}/.zshrc"
    add_to_profile "${HOME}/.zshrc"
    UPDATED_PROFILES+=(".zshrc")
else
    # Fallback to general profiles
    for p in "${PROFILES[@]}"; do
        if [[ -f "$p" ]]; then
            add_to_profile "$p"
            UPDATED_PROFILES+=("$(basename "$p")")
        fi
    done
fi

if [ ${#UPDATED_PROFILES[@]} -eq 0 ]; then
    # If no standard shell profiles exist, create .profile
    touch "${HOME}/.profile"
    add_to_profile "${HOME}/.profile"
    UPDATED_PROFILES+=(".profile")
fi

echo -e "${GREEN}OK${NC} (Updated: ${UPDATED_PROFILES[*]})"

# ── Success Banner ────────────────────────────────────────────────────────────
echo -e ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║         neodlp successfully installed!           ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo -e ""
echo -e "  Installation Path: ${BINARY_PATH}"
echo -e "  Version          : ${VERSION}"
echo -e ""
echo -e "${YELLOW}  Please RESTART your terminal window or run:${NC}"
echo -e "${CYAN}  source ~/${UPDATED_PROFILES[0]}${NC}"
echo -e "${YELLOW}  to start using neodlp immediately!${NC}"
echo -e ""
