#!/bin/bash

# kubectl-broker Installation Script
# This script installs kubectl-broker as a kubectl plugin

set -e

INSTALL_DIR="$HOME/.kubectl-broker"
BINARY_NAME="kubectl-broker"
SHELL_RC=""

echo "🚀 Installing kubectl-broker as kubectl plugin..."

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Build the binary with optimization
echo "📦 Building optimized kubectl-broker..."
if command -v make >/dev/null 2>&1; then
    echo "🔧 Using Make for optimized build (35MB vs 53MB)..."
    make build-small
    cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "⚠️  Make not found, using fallback build with basic optimization..."
    CGO_ENABLED=0 go build -ldflags="-w -s" -trimpath -o "$INSTALL_DIR/$BINARY_NAME" ./cmd/kubectl-broker
fi

# Make it executable
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Detect shell and RC file
if [ -n "$ZSH_VERSION" ]; then
    SHELL_RC="$HOME/.zshrc"
elif [ -n "$BASH_VERSION" ]; then
    SHELL_RC="$HOME/.bashrc"
    # macOS typically uses .bash_profile instead of .bashrc
    if [[ "$OSTYPE" == "darwin"* ]] && [ -f "$HOME/.bash_profile" ]; then
        SHELL_RC="$HOME/.bash_profile"
    fi
fi

# Check if PATH already contains our directory
if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
    echo "✅ $INSTALL_DIR already in PATH"
else
    echo "🔧 Adding $INSTALL_DIR to PATH..."
    
    if [ -n "$SHELL_RC" ]; then
        # Add PATH export to shell RC file
        echo "" >> "$SHELL_RC"
        echo "# kubectl-broker plugin" >> "$SHELL_RC"
        echo "export PATH=\"\$HOME/.kubectl-broker:\$PATH\"" >> "$SHELL_RC"
        echo "📝 Added PATH export to $SHELL_RC"
    else
        echo "⚠️  Could not detect shell type. Please manually add the following to your shell RC file:"
        echo "export PATH=\"\$HOME/.kubectl-broker:\$PATH\""
    fi
fi

# Test installation
echo "🧪 Testing installation..."
if "$INSTALL_DIR/$BINARY_NAME" --help > /dev/null 2>&1; then
    echo "✅ kubectl-broker binary is working"
else
    echo "❌ kubectl-broker binary test failed"
    exit 1
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "📋 Next steps:"
echo "1. Restart your terminal or run: source $SHELL_RC"
echo "2. Verify installation: kubectl plugin list | grep broker"
echo "3. Test the plugin: kubectl broker --help"
echo "4. Discover HiveMQ brokers: kubectl broker --discover"
echo ""
echo "💡 Usage examples:"
echo "   kubectl broker                                              # Uses intelligent defaults"
echo "   kubectl broker --discover                                   # Find all HiveMQ brokers"
echo "   kubectl broker --pod broker-0 --namespace my-namespace"
echo "   kubectl broker --statefulset broker --namespace my-namespace"