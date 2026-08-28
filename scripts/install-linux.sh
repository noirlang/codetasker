#!/usr/bin/env bash
# ==============================================================================
# CodeTasker CLI — Linux / macOS Installation Script
# ==============================================================================

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${GREEN}${BOLD}"
cat << "EOF"
          ⣶  ⣠⠖⠋⠉⠻⡇  ⣠⠞⠉⠉⠲⡄  ⢻⡏⠑⢦⡀ ⢸⡏⠉⢹⢸⠋⢹⡏⢹ ⢻⡇ ⢀⡞⠉⢻ ⠘⣿ ⠘⠁⢻⡏⠉⢿⠈⣿⠉⢻⡄
         ⢰⡏ ⣼⠏    ⠁ ⣰⠃    ⢹⡄ ⢸⡇ ⠈⣷ ⢸⡇  ⠈ ⢸⡇  ⢸⡇ ⢸⠁ ⠈  ⣿   ⢸⡇ ⠈ ⣿  ⣿
    ⣠⡾   ⣼⠁⢰⡟      ⢠⡟      ⣿ ⢸⡇  ⢸⡇⢸⡇    ⢸⡇   ⣷ ⢸⣧    ⣿   ⢸⡇   ⣿  ⣿ ⢷⣄
 ⢀⣴⡾⠋   ⢠⡟ ⢸⡇      ⢸⡇  ⢠⡀  ⢹⡆⢸⡇  ⢸⡇⢸⡧⠦⡄  ⢸⡇   ⣿  ⢻⣷⡄  ⣿⢲⡄ ⢸⡷⢦  ⣿⢀⣰⠏  ⠙⢿⣦⡀
⣴⡿⠋     ⣸⠇ ⢸⡇      ⢸⡇  ⠹⠋  ⢸⠇⢸⡇  ⢸⡇⢸⡇    ⢸⡇   ⣿   ⠙⣿⡄ ⣿ ⣿ ⢸⡇   ⣿⠈⢹⡄    ⠙⢿⣦
⠻⣦⣀     ⣿  ⠸⣇      ⠘⣇      ⣿ ⢸⡇  ⢸⡇⢸⡇    ⢸⡇   ⢸⡆   ⠈⣷ ⣿ ⣿ ⢸⡇   ⣿ ⢸⡇    ⢀⣴⠟
 ⠈⠻⣷⣄  ⢸⡇   ⢻⡄    ⢀⡄⢻⡄    ⢰⠇ ⢸⡇  ⡾ ⢸⡇    ⢸⡇ ⢠⢤⣸⡇ ⡄  ⣿ ⣿ ⣿ ⢸⡇ ⢀ ⣿ ⠸⡇  ⣠⣾⠟⠁
   ⠈⠻⣷ ⣾⠁    ⠻⣄⡀ ⣀⣼⡁ ⠻⣄⡀⢀⡴⠋  ⣸⡃⣀⠼⠁ ⢸⣇ ⣰⡇ ⣸⡇ ⡄ ⢸⣇ ⣷⡀⢠⠏⢀⣿ ⣿ ⣸⡇ ⣼⢀⣿  ⡇ ⣾⠟⠁
       ⠋       ⠉⠉      ⠉⠁    ⠁⠈    ⠈  ⠁  ⠁  ⠁ ⠈⠈  ⠈⠁ ⠈⠈ ⢸⡄⠁ ⠈     ⢷
EOF
echo -e "${NC}"
echo -e "${BOLD}Building and installing CodeTasker CLI...${NC}\n"

# Determine script root & project path
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH.${NC}"
    echo "Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi

echo -e "${BLUE}==>${NC} Compiling binary from source..."
TEMP_BIN="$(mktemp -d)/codetasker"
cd "$BACKEND_DIR"
go build -ldflags="-s -w" -o "$TEMP_BIN" ./cmd/codetasker

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

TARGET_BIN="$INSTALL_DIR/codetasker"
echo -e "${BLUE}==>${NC} Installing to ${BOLD}$TARGET_BIN${NC}..."

if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_DIR" ]; then
    sudo cp "$TEMP_BIN" "$TARGET_BIN"
    sudo chmod +x "$TARGET_BIN"
else
    cp "$TEMP_BIN" "$TARGET_BIN"
    chmod +x "$TARGET_BIN"
fi

rm -f "$TEMP_BIN"

echo -e "\n${GREEN}✓ CodeTasker CLI successfully installed!${NC}\n"

# Verify PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not currently in your \$PATH.${NC}"
    echo -e "Add the following line to your ~/.bashrc or ~/.zshrc:"
    echo -e "  ${BOLD}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}\n"
fi

echo -e "To verify and get started, run:"
echo -e "  ${BOLD}codetasker --help${NC}   (View all available commands)"
echo -e "  ${BOLD}codetasker scan .${NC}   (Scan local directory for TODO/FIXME)"
echo -e "  ${BOLD}codetasker tui${NC}      (Launch interactive terminal dashboard)"
