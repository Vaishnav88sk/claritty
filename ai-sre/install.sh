#!/usr/bin/env bash
set -e

echo "========================================"
echo "Installing Claritty AI-SRE Engine..."
echo "========================================"

# Define directories
CLARITTY_DIR="$HOME/.claritty"
BIN_DIR="$HOME/.local/bin"

# Ensure docker is installed
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed or not in PATH."
    echo "Please install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi

# Create necessary directories
mkdir -p "$CLARITTY_DIR"
mkdir -p "$BIN_DIR"

# 1. Build the Docker image from remote repository
echo "Building the clarctl Docker image directly from GitHub (this may take a few minutes)..."
docker build -t claritty-sre https://github.com/Vaishnav88sk/claritty.git#vaishnav-claritty:ai-sre

# 2. Create the global executable bash wrapper
echo "Configuring clarctl CLI wrapper..."
WRAPPER_PATH="$BIN_DIR/clarctl"

cat <<'EOF' > "$WRAPPER_PATH"
#!/usr/bin/env bash
# Claritty AI-SRE Docker Wrapper

# Ensure necessary directories exist to mount
mkdir -p "$HOME/.claritty"
mkdir -p "$HOME/.kube"

docker run -it --rm \
  --network host \
  -v "$HOME/.kube:/root/.kube:ro" \
  -v "$HOME/.claritty:/root/.claritty" \
  claritty-sre "$@"
EOF

chmod +x "$WRAPPER_PATH"

# Ensure ~/.local/bin is in the PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "WARNING: $BIN_DIR is not in your PATH."
    echo "Please add the following line to your ~/.bashrc or ~/.zshrc:"
    echo "export PATH=\"$BIN_DIR:\$PATH\""
fi

# 3. Create default .env if it doesn't exist
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
echo "Your host machine remains completely clean!"
echo ""
echo "You can now run the CLI using:"
echo "  clarctl status"
echo ""
echo "For help and more commands, use:"
echo "  clarctl --help"
echo ""
echo "Note: Make sure to add your API keys to ~/.claritty/.env"
