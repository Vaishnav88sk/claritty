#!/usr/bin/env bash
# install.sh — One-line installer for clarctl AI-SRE CLI
# Usage: curl -fsSL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/clarctl-go/install.sh | bash
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
RELEASE=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
LATEST=$(printf '%s\n' "$RELEASE" | grep '"browser_download_url"' | grep -F "/${ASSET_NAME}\"" | cut -d '"' -f 4)
STACK_URL=$(printf '%s\n' "$RELEASE" | grep '"browser_download_url"' | grep -F '/clarctl-stack.tar.gz"' | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
  echo "ERROR: Could not find release asset '${ASSET_NAME}' in the latest release."
  echo "Please check: https://github.com/${REPO}/releases"
  exit 1
fi

# ── Download and stage the release ──────────────────────────────────────────
echo "Downloading ${ASSET_NAME}..."
mkdir -p "$BIN_DIR"
BIN_TMP=$(mktemp "$BIN_DIR/.clarctl.XXXXXX")
STACK_TMP=""
cleanup() {
  rm -f "$BIN_TMP"
  if [ -n "$STACK_TMP" ]; then
    rm -rf "$STACK_TMP"
  fi
}
trap cleanup EXIT
curl -fsSL "$LATEST" -o "$BIN_TMP"
chmod +x "$BIN_TMP"

if [ -n "$STACK_URL" ]; then
  echo "Installing the local stack..."
  STACK_ROOT="$CLARITTY_DIR/stack"
  STACK_PATH="$STACK_ROOT/claritty"
  mkdir -p "$STACK_ROOT"
  STACK_TMP=$(mktemp -d "$STACK_ROOT/.install.XXXXXX")
  mkdir "$STACK_TMP/claritty"
  curl -fsSL "$STACK_URL" -o "$STACK_TMP/stack.tar.gz"
  tar -xzf "$STACK_TMP/stack.tar.gz" -C "$STACK_TMP/claritty"
  test -f "$STACK_TMP/claritty/docker-compose.yml"
  if [ -f "$STACK_PATH/.env" ]; then
    cp -p "$STACK_PATH/.env" "$STACK_TMP/claritty/.env"
  fi
  if [ -e "$STACK_PATH" ]; then
    mv "$STACK_PATH" "$STACK_TMP/previous"
  fi
  if ! mv "$STACK_TMP/claritty" "$STACK_PATH"; then
    if [ -e "$STACK_TMP/previous" ]; then
      mv "$STACK_TMP/previous" "$STACK_PATH" || STACK_TMP=""
    fi
    exit 1
  fi
fi
mv -f "$BIN_TMP" "$INSTALL_PATH"

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
# LLM_MODEL can be left empty for provider defaults or customized to any model
LLM_PROVIDER=groq
LLM_MODEL=groq/openai/gpt-oss-120b
GROQ_API_KEY=your_groq_api_key_here
# OPENAI_API_KEY=
# ANTHROPIC_API_KEY=
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
if [ -n "$STACK_URL" ]; then
  echo "  Start the local dashboard with clarctl up, then stop it with clarctl down."
fi
echo "  1. Add your API key to ~/.claritty/.env"
echo "  2. Run: clarctl status"
echo "  3. Run: clarctl -h"
echo "  4. Run: clarctl scan [optional: --apply]"
echo ""
echo "For all commands: clarctl --help"
