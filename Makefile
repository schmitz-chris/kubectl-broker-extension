# kubectl-broker Makefile
# Build and install kubectl plugin for HiveMQ broker health diagnostics

BINARY_NAME=kubectl-broker
INSTALL_DIR=$(HOME)/.kubectl-broker
BUILD_DIR=.
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build: $(BINARY_NAME)

$(BINARY_NAME): $(GO_FILES)
	@echo "Building kubectl-broker..."
	go build -o $(BINARY_NAME) ./cmd/kubectl-broker
	@echo "Build complete: $(BINARY_NAME)"

# Install as kubectl plugin (standard build)
.PHONY: install
install: build
	@echo "Installing kubectl-broker as kubectl plugin..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "To complete installation, add to your PATH:"
	@echo "   export PATH=\"\$$HOME/.kubectl-broker:\$$PATH\""
	@echo ""
	@echo "Test installation:"
	@echo "   kubectl plugin list | grep broker"
	@echo "   kubectl broker --help"

# Install dual-plugin with symlink (standard build)
.PHONY: install-dual
install-dual: build
	@echo "Installing kubectl-broker and kubectl-pulse as dual kubectl plugins..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	ln -sf $(BINARY_NAME) $(INSTALL_DIR)/kubectl-pulse
	@echo "Installed kubectl-broker to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Created symlink kubectl-pulse -> $(BINARY_NAME)"
	@echo ""
	@echo "To complete installation, add to your PATH:"
	@echo "   export PATH=\"\$$HOME/.kubectl-broker:\$$PATH\""
	@echo ""
	@echo "Test installation:"
	@echo "   kubectl plugin list | grep -E '(broker|pulse)'"
	@echo "   kubectl broker --help"
	@echo "   kubectl pulse --help"

# Install with optimized build (35MB vs 53MB)
.PHONY: install-small
install-small: build-small
	@echo "Installing optimized kubectl-broker as kubectl plugin..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed optimized binary to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "To complete installation, add to your PATH:"
	@echo "   export PATH=\"\$$HOME/.kubectl-broker:\$$PATH\""
	@echo ""
	@echo "Test installation:"
	@echo "   kubectl plugin list | grep broker"
	@echo "   kubectl broker --help"

# Install dual-plugin with optimized build
.PHONY: install-dual-small
install-dual-small: build-small
	@echo "Installing optimized kubectl-broker and kubectl-pulse as dual kubectl plugins..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	ln -sf $(BINARY_NAME) $(INSTALL_DIR)/kubectl-pulse
	@echo "Installed optimized kubectl-broker to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Created symlink kubectl-pulse -> $(BINARY_NAME)"
	@echo ""
	@echo "To complete installation, add to your PATH:"
	@echo "   export PATH=\"\$$HOME/.kubectl-broker:\$$PATH\""
	@echo ""
	@echo "Test installation:"
	@echo "   kubectl plugin list | grep -E '(broker|pulse)'"
	@echo "   kubectl broker --help"
	@echo "   kubectl pulse --help"

# Automated installation with PATH setup (uses install.sh)
.PHONY: install-auto
install-auto:
	@echo "Running automated installation with optimized binary..."
	@./install.sh

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Uninstall the plugin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling kubectl-broker and kubectl-pulse..."
	rm -rf $(INSTALL_DIR)
	@echo "Uninstalled kubectl-broker and kubectl-pulse"
	@echo "Don't forget to remove from your PATH if added manually"

# Test the plugin functionality
.PHONY: test
test: build
	@echo "Testing kubectl-broker functionality..."
	./$(BINARY_NAME) --help
	@echo "Basic functionality test passed"

# Development build with race detector
.PHONY: dev
dev:
	@echo "Building development version with race detector..."
	go build -race -o $(BINARY_NAME) ./cmd/kubectl-broker
	@echo "Development build complete"

# Release build with optimizations
.PHONY: release
release:
	@echo "Building release version..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_NAME) ./cmd/kubectl-broker
	@echo "Release build complete"

# Small build with maximum optimization
.PHONY: build-small
build-small:
	@echo "Building with maximum size optimization..."
	CGO_ENABLED=0 go build -ldflags="-w -s -X 'main.version=$(shell git describe --tags --always)'" -trimpath -o $(BINARY_NAME) ./cmd/kubectl-broker
	@echo "Small build complete"

# Cross-compile for multiple platforms
.PHONY: cross-compile
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	mkdir -p dist
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o dist/kubectl-broker-linux-amd64 ./cmd/kubectl-broker
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o dist/kubectl-broker-linux-arm64 ./cmd/kubectl-broker
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o dist/kubectl-broker-darwin-amd64 ./cmd/kubectl-broker
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o dist/kubectl-broker-darwin-arm64 ./cmd/kubectl-broker
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o dist/kubectl-broker-windows-amd64.exe ./cmd/kubectl-broker
	@echo "Cross-compilation complete. Binaries in dist/"



# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	gofmt -s -w $(shell find . -name "*.go" -not -path "./vendor/*")
	@echo "Code formatted"

# Run Go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet checks passed"

# Run all checks
.PHONY: check
check: fmt vet
	@echo "All checks passed"

# Show help
.PHONY: help
help:
	@echo "kubectl-broker Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build              Build the binary"
	@echo "  install            Install as kubectl plugin (standard build)"
	@echo "  install-dual       Install as dual kubectl plugins with symlink (standard build)"
	@echo "  install-small      Install as kubectl plugin (optimized 35MB build)"
	@echo "  install-dual-small Install as dual kubectl plugins with symlink (optimized build)"
	@echo "  install-auto       Install with automatic PATH setup (uses install.sh)"
	@echo "  clean              Remove build artifacts"
	@echo "  uninstall          Remove installed plugins"
	@echo "  test               Test basic functionality"
	@echo "  dev                Build with race detector"
	@echo "  release            Build optimized release version"
	@echo "  build-small        Build with maximum size optimization"
	@echo ""
	@echo "Quality:"
	@echo "  cross-compile Build for multiple platforms"
	@echo "  fmt           Format Go code"
	@echo "  vet           Run go vet"
	@echo "  check         Run code quality checks (fmt, vet)"
	@echo "  help          Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  make install-dual-small # Build and install both kubectl-broker and kubectl-pulse"
	@echo "  kubectl broker --help   # Test broker installation"
	@echo "  kubectl pulse --help    # Test pulse installation"