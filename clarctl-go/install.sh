#!/usr/bin/env bash
# install.sh — One-line installer for clarctl AI-SRE CLI
# Usage: curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/clarctl-go/clarctl-go/install.sh | bash
set -e

REPO="Vaishnav88sk/claritty"
BINARY="clarctl"
CLARITTY_DIR="$HOME/.claritty"
BIN_DIR="$HOME/.local/bin"
INSTALL_PATH="$BIN_DIR/$BINARY"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Installing clarctl — Claritty AI-SRE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ── Detect OS and architecture ──────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)   ARCH="amd64" ;;
  aarch64)  ARCH="arm64" ;;
  arm64)    ARCH="arm64" ;;
  *)
    echo "ERROR: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux | darwin) ;;
  msys* | cygwin* | mingw*) OS="windows" ;;
  *)
    echo "ERROR: Unsupported OS: $OS"
    exit 1
    ;;
esac

ASSET_NAME="${BINARY}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  ASSET_NAME="${ASSET_NAME}.exe"
fi

# ── Fetch the latest release URL ─────────────────────────────────────────────
echo "Fetching latest release info..."
LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"browser_download_url"' \
  | grep "${ASSET_NAME}" \
  | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
  echo "ERROR: Could not find release asset '${ASSET_NAME}' in the latest release."
  echo "Please check: https://github.com/${REPO}/releases"
  exit 1
fi

# ── Download binary ──────────────────────────────────────────────────────────
echo "Downloading ${ASSET_NAME}..."
mkdir -p "$BIN_DIR"
rm -f "$INSTALL_PATH"
curl -sL "$LATEST" -o "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

# ── Ensure PATH includes ~/.local/bin ────────────────────────────────────────
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo ""
  echo "⚠  $BIN_DIR is not in your PATH."
  echo "   Add this line to your ~/.bashrc or ~/.zshrc:"
  echo ""
  echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# ── Create default .env if it doesn't exist ──────────────────────────────────
mkdir -p "$CLARITTY_DIR"
if [ ! -f "$CLARITTY_DIR/.env" ]; then
  cat <<EOF > "$CLARITTY_DIR/.env"
# Claritty AI-SRE Configuration
# Get a free key at https://console.groq.com
LLM_PROVIDER=groq
LLM_MODEL=groq/llama-3.3-70b-versatile
GROQ_API_KEY=your_groq_api_key_here
# OPENAI_API_KEY=
# MISTRAL_API_KEY=
EOF
  echo "Created configuration file: $CLARITTY_DIR/.env"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Installation Complete! ✓"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Binary installed: $INSTALL_PATH"
echo "Binary size:      $(du -sh "$INSTALL_PATH" | cut -f1)"
echo ""
echo "Next steps:"
echo "  1. Add your API key to ~/.claritty/.env"
echo "  2. Run: clarctl status"
echo "  3. Run: clarctl scan --apply"
echo ""
echo "For all commands: clarctl --help"
