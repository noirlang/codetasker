#!/usr/bin/env bash
# ==============================================================================
# CodeTasker CLI — Linux / macOS Installation Script
# ==============================================================================

set -e

WHITE='\033[1;37m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
GREEN='\033[1;32m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${WHITE}${BOLD}"
cat << "EOF"
         ⣶  ⣠⠖⠉⠙⢶⢰⠏⢹⡏⢳
        ⢠⡏ ⣼⠃   ⠘  ⢸⡇⠈
   ⣠⣾⠃  ⣼⠃⢸⡇       ⢸⡇ ⠘⢷⣄
 ⣠⡾⠋   ⢀⡟ ⢸⠃       ⢸⡇   ⠙⢷⣄
⣾⡋     ⢸⠇ ⢸⡆       ⢸⡇     ⢙⣷
⠈⠻⣦⡀   ⣿  ⠘⣧       ⢸⡇   ⢀⣴⠟⠉
  ⠈⠻⣦⡀⢰⡇   ⠹⣆   ⢀⡇ ⢸⡇ ⢀⣴⠟⠁
    ⠈⠁⢼⠁    ⠈⠓⠒⠒⠉⠁ ⠛⠃ ⠈⠁
EOF
echo -e "${NC}"
echo -e "${BOLD}Building and installing CodeTasker CLI...${NC}\n"

# Verify Go runtime
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH.${NC}"
    echo "Please install Go 1.22+ from https://go.dev/dl/ to build CodeTasker CLI."
    exit 1
fi

TEMP_WORK_DIR=""
TEMP_BIN_DIR=""
cleanup() {
    [ -n "$TEMP_WORK_DIR" ] && [ -d "$TEMP_WORK_DIR" ] && rm -rf "$TEMP_WORK_DIR"
    [ -n "$TEMP_BIN_DIR" ] && [ -d "$TEMP_BIN_DIR" ] && rm -rf "$TEMP_BIN_DIR"
}
trap cleanup EXIT INT TERM

# Determine repository root
REPO_ROOT=""

# 1. Check if running as a script file inside a cloned repo
if [ -n "${BASH_SOURCE[0]}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
    CANDIDATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." 2>/dev/null && pwd)"
    if [ -f "$CANDIDATE/backend/cmd/codetasker/main.go" ]; then
        REPO_ROOT="$CANDIDATE"
    fi
fi

# 2. Check if current directory has the repository
if [ -z "$REPO_ROOT" ]; then
    if [ -f "./backend/cmd/codetasker/main.go" ]; then
        REPO_ROOT="$(pwd)"
    fi
fi

# 3. Remote execution (e.g. curl/wget | bash) — clone or download source
if [ -z "$REPO_ROOT" ]; then
    echo -e "${BLUE}==>${NC} Fetching CodeTasker repository..."
    TEMP_WORK_DIR="$(mktemp -d -t codetasker-src-XXXXXX 2>/dev/null || mktemp -d /tmp/codetasker-src-XXXXXX)"
    REPO_URL="${CODETASKER_REPO_URL:-https://github.com/noirlang/codetasker.git}"
    TAR_URL="${CODETASKER_TAR_URL:-https://github.com/noirlang/codetasker/archive/refs/heads/main.tar.gz}"

    if command -v git &> /dev/null; then
        git clone --depth 1 "$REPO_URL" "$TEMP_WORK_DIR" --quiet
    elif command -v curl &> /dev/null; then
        curl -fsSL "$TAR_URL" | tar -xz -C "$TEMP_WORK_DIR" --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -qO- "$TAR_URL" | tar -xz -C "$TEMP_WORK_DIR" --strip-components=1
    else
        echo -e "${RED}Error: git, curl, or wget is required to download CodeTasker.${NC}"
        exit 1
    fi
    REPO_ROOT="$TEMP_WORK_DIR"
fi

BACKEND_DIR="$REPO_ROOT/backend"
if [ ! -f "$BACKEND_DIR/cmd/codetasker/main.go" ]; then
    echo -e "${RED}Error: CodeTasker source not found at $BACKEND_DIR${NC}"
    exit 1
fi

echo -e "${BLUE}==>${NC} Compiling binary from source..."
TEMP_BIN_DIR="$(mktemp -d -t codetasker-bin-XXXXXX 2>/dev/null || mktemp -d /tmp/codetasker-bin-XXXXXX)"
TEMP_BIN="$TEMP_BIN_DIR/codetasker"

(
    cd "$BACKEND_DIR"
    go build -ldflags="-s -w" -o "$TEMP_BIN" ./cmd/codetasker
)

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
fi
mkdir -p "$INSTALL_DIR"

TARGET_BIN="$INSTALL_DIR/codetasker"
echo -e "${BLUE}==>${NC} Installing to ${BOLD}$TARGET_BIN${NC}..."

cp "$TEMP_BIN" "$TARGET_BIN"
chmod +x "$TARGET_BIN"

# Check if PATH contains install directory
IN_PATH=false
if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
    IN_PATH=true
fi

if [ "$IN_PATH" = false ]; then
    ADDED_TO_RC=false
    for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$RC" ]; then
            if ! grep -q "$INSTALL_DIR" "$RC" 2>/dev/null; then
                echo -e "\n# Added by CodeTasker CLI installer\nexport PATH=\"\$PATH:$INSTALL_DIR\"" >> "$RC"
                ADDED_TO_RC=true
            fi
        fi
    done

    echo -e "\n${YELLOW}Note: $INSTALL_DIR is not currently in your \$PATH.${NC}"
    if [ "$ADDED_TO_RC" = true ]; then
        echo -e "We have added it to your shell configuration profile."
        echo -e "To use '${BOLD}codetasker${NC}' immediately in this terminal, run:"
        echo -e "  ${BOLD}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
        echo -e "or restart your terminal."
    else
        echo -e "Add the following line to your ~/.bashrc or ~/.zshrc:"
        echo -e "  ${BOLD}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    fi
fi

echo -e "\n${GREEN}${BOLD}✓ CodeTasker CLI successfully installed!${NC}\n"

echo -e "To verify and get started, run:"
echo -e "  ${BOLD}codetasker --help${NC}   (View all available commands)"
echo -e "  ${BOLD}codetasker scan .${NC}   (Scan local directory for TODO/FIXME)"
echo -e "  ${BOLD}codetasker repo list${NC} (List connected repositories)"
