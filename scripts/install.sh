#!/bin/bash
set -e

REPO="github.com/jupiterozeye/context"
INSTALL_DIR="/usr/local/bin"
SHELL_DIR="/usr/local/share/context/shell"

echo "Installing context..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        echo "Building from source instead..."
        go install $REPO/cmd/context@latest
        exit 0
        ;;
esac

# Try to download pre-built binary
echo "Attempting to download pre-built binary for $OS/$ARCH..."

BINARY_URL="https://github.com/jupiterozeye/context/releases/latest/download/context-${OS}-${ARCH}"

if command -v curl &> /dev/null; then
    curl -sL "$BINARY_URL" -o /tmp/context || true
elif command -v wget &> /dev/null; then
    wget -q "$BINARY_URL" -O /tmp/context || true
fi

if [[ -f /tmp/context ]] && [[ -s /tmp/context ]]; then
    chmod +x /tmp/context
    echo "Installing binary to $INSTALL_DIR..."
    sudo mv /tmp/context $INSTALL_DIR/context
else
    echo "Pre-built binary not available, building from source..."
    if command -v go &> /dev/null; then
        go install $REPO/cmd/context@latest
    else
        echo "Error: Go is not installed and no pre-built binary available"
        echo "Please install Go or build manually"
        exit 1
    fi
fi

# Install shell scripts
echo "Installing shell integration scripts..."
sudo mkdir -p $SHELL_DIR

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -d "$SCRIPT_DIR/../shell" ]]; then
    sudo cp "$SCRIPT_DIR/../shell"/* $SHELL_DIR/
else
    echo "Warning: Shell scripts not found. Please manually copy shell/ directory to $SHELL_DIR"
fi

echo ""
echo "Installation complete!"
echo ""
echo "To enable 'context last', add to your shell config:"
echo ""
echo "  Bash:  echo 'source $SHELL_DIR/context.bash' >> ~/.bashrc"
echo "  Zsh:   echo 'source $SHELL_DIR/context.zsh' >> ~/.zshrc"
echo "  Fish:  echo 'source $SHELL_DIR/context.fish' >> ~/.config/fish/config.fish"
echo ""
echo "Then restart your terminal or run: source ~/.bashrc  # or ~/.zshrc, etc."