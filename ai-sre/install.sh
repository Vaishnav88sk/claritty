#!/usr/bin/env bash
set -e

echo "========================================"
echo "Installing Claritty AI-SRE Engine..."
echo "========================================"

# Define directories
CLARITTY_DIR="$HOME/.claritty"
APP_DIR="$CLARITTY_DIR/app"
BIN_DIR="$HOME/.local/bin"

# Create necessary directories
mkdir -p "$CLARITTY_DIR"
mkdir -p "$BIN_DIR"

# 1. Clone or update the repository
if [ -d "$APP_DIR" ]; then
    echo "Updating existing installation..."
    cd "$APP_DIR"
    git fetch origin
    git checkout vaishnav-claritty --quiet
    git pull origin vaishnav-claritty --quiet
else
    echo "Downloading source code..."
    git clone -b vaishnav-claritty https://github.com/Vaishnav88sk/claritty.git "$APP_DIR" --quiet
fi

# 2. Setup Virtual Environment
echo "Setting up isolated Python environment..."
cd "$APP_DIR/ai-sre"
python3 -m venv venv
source venv/bin/activate

# 3. Install dependencies
echo "Installing dependencies (this may take a moment)..."
pip install --upgrade pip --quiet
pip install -e . --quiet

# 4. Create the global executable symlink
echo "Configuring clarctl CLI..."
ln -sf "$APP_DIR/ai-sre/venv/bin/clarctl" "$BIN_DIR/clarctl"

# Ensure ~/.local/bin is in the PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "WARNING: $BIN_DIR is not in your PATH."
    echo "Please add the following line to your ~/.bashrc or ~/.zshrc:"
    echo "export PATH=\"$BIN_DIR:\$PATH\""
fi

# 5. Create default .env if it doesn't exist
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
echo "Note: Make sure to add your API keys to ~/.claritty/.env"
