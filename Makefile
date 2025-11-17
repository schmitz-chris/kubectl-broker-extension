# kubectl-broker build system

BINARY_NAME ?= kubectl-broker
INSTALL_DIR ?= $(HOME)/.kubectl-broker
SRC_DIR     ?= ./cmd/kubectl-broker
GO_FILES    := $(shell find . -name "*.go" -not -path "./vendor/*")
GO_LDFLAGS  ?= -s -w
GO_BUILD    := CGO_ENABLED=0 go build -trimpath -ldflags "$(GO_LDFLAGS)"

.PHONY: all build install install-dual install-auto clean uninstall dev test release cross-compile fmt vet check help

all: build

build:
	@echo "Building $(BINARY_NAME) (optimized release)..."
	$(GO_BUILD) -o $(BINARY_NAME) $(SRC_DIR)
	@echo "Build complete: $(BINARY_NAME)"

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	install -d $(INSTALL_DIR)
	install $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed. Add to PATH with: export PATH=\"$(INSTALL_DIR):$$PATH\""

install-dual: build
	@echo "Installing broker/pulse dual plugins to $(INSTALL_DIR)..."
	install -d $(INSTALL_DIR)
	install $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	ln -sf $(BINARY_NAME) $(INSTALL_DIR)/kubectl-pulse
	@echo "Installed kubectl-broker and symlinked kubectl-pulse."

install-auto:
	@echo "Running installer script..."
	@./install.sh

clean:
	@echo "Removing build artifacts..."
	rm -f $(BINARY_NAME)

uninstall:
	@echo "Removing $(INSTALL_DIR)..."
	rm -rf $(INSTALL_DIR)

dev:
	@echo "Building development binary with race detector..."
	go build -race -o $(BINARY_NAME) $(SRC_DIR)

test: build
	@echo "Running go test..."
	go test ./...

release: build

cross-compile:
	@echo "Cross-compiling broker for common platforms..."
	mkdir -p dist
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o dist/$(BINARY_NAME)-linux-amd64 $(SRC_DIR)
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o dist/$(BINARY_NAME)-linux-arm64 $(SRC_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o dist/$(BINARY_NAME)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o dist/$(BINARY_NAME)-darwin-arm64 $(SRC_DIR)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/$(BINARY_NAME)-windows-amd64.exe $(SRC_DIR)
	@echo "Binaries available in dist/."

fmt:
	@echo "Running gofmt..."
	gofmt -s -w $(GO_FILES)

vet:
	@echo "Running go vet..."
	go vet ./...

check: fmt vet
	@echo "Formatting and vetting complete."

help:
	@echo "kubectl-broker targets:"
	@echo "  make build          Build optimized binary (default)."
	@echo "  make install        Install plugin under $(INSTALL_DIR)."
	@echo "  make install-dual   Install broker + pulse symlink."
	@echo "  make install-auto   Run install.sh helper."
	@echo "  make dev            Build with race detector."
	@echo "  make test           Run go test ./..."
	@echo "  make cross-compile  Produce dist/* binaries."
	@echo "  make fmt|vet|check  Format/Vet helpers."
	@echo "  make clean|uninstall Remove artifacts or install dir."
