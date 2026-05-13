#!/usr/bin/env bash
set -e

echo "========================================"
echo "Installing Claritty AI-SRE Engine..."
echo "========================================"

# Define directories
CLARITTY_DIR="$HOME/.claritty"
VENV_DIR="$CLARITTY_DIR/venv"
BIN_DIR="$HOME/.local/bin"

# Create necessary directories
mkdir -p "$CLARITTY_DIR"
mkdir -p "$BIN_DIR"

# 1. Setup Virtual Environment
echo "Setting up isolated Python environment..."
if [ ! -d "$VENV_DIR" ]; then
    python3 -m venv "$VENV_DIR"
fi
source "$VENV_DIR/bin/activate"

# 2. Install dependencies and the CLI from remote
echo "Downloading and installing clarctl CLI (this may take a moment)..."
pip install --upgrade pip --quiet
# Install directly from the github branch without cloning locally
pip install --upgrade "git+https://github.com/Vaishnav88sk/claritty.git@vaishnav-claritty#subdirectory=ai-sre" --quiet

# 3. Create the global executable symlink
echo "Configuring clarctl CLI..."
ln -sf "$VENV_DIR/bin/clarctl" "$BIN_DIR/clarctl"

# Ensure ~/.local/bin is in the PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "WARNING: $BIN_DIR is not in your PATH."
    echo "Please add the following line to your ~/.bashrc or ~/.zshrc:"
    echo "export PATH=\"$BIN_DIR:\$PATH\""
fi

# 4. Create default .env if it doesn't exist
if [ ! -f "$CLARITTY_DIR/.env" ]; then
    echo "Creating empty configuration file at $CLARITTY_DIR/.env"
    cat <<EOF > "$CLARITTY_DIR/.env"
# Claritty AI-SRE Configuration
LLM_PROVIDER=groq
LLM_MODEL=groq/llama-3.3-70b-versatile
GROQ_API_KEY=your_groq_api_key_here
# OPENAI_API_KEY=
# MISTRAL_API_KEY=
EOF
fi

echo "========================================"
echo "Installation Complete! ✓"
echo "========================================"
echo "You can now run the CLI using:"
echo "  clarctl status"
echo ""
echo "For help and more commands, use:"
echo "  clarctl --help"
echo ""
echo "Note: Make sure to add your API keys to ~/.claritty/.env"
