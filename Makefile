.PHONY: build build-cfg build-jtag build-all run run-debug clean test test-coverage test-integration lint fmt tidy install install-cfg install-jtag install-all help build-prod build-cfg-prod build-jtag-prod build-prod-all check-gdb check-certs build-cfg-with-gdb docs docs-serve docs-clean release release-dry-run tag

# Variables
BINARY_NAME=smartap-server
CFG_BINARY_NAME=smartap-cfg
JTAG_BINARY_NAME=smartap-jtag
BUILD_DIR=bin
MAIN_PATH=./cmd/smartap-server
CFG_MAIN_PATH=./cmd/smartap-cfg
JTAG_MAIN_PATH=./cmd/smartap-jtag
GO=go
GOFLAGS=-v

# Version information (from git or environment)
# Can be overridden: make build VERSION=v1.2.3 COMMIT=abc123
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "")

# Build ldflags for version injection
VERSION_PKG=github.com/muurk/smartap/internal/version
LDFLAGS=-X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT)

# Default target - build all binaries
all: build-all

# Build all binaries
build-all: build build-cfg build-jtag

# Build server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build configuration utility
build-cfg:
	@echo "Building $(CFG_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(CFG_BINARY_NAME) $(CFG_MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(CFG_BINARY_NAME)"

# Build JTAG utility
build-jtag:
	@echo "Building $(JTAG_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(JTAG_BINARY_NAME) $(JTAG_MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(JTAG_BINARY_NAME)"

# Build all for production (optimized, smaller binaries)
build-prod-all: build-prod build-cfg-prod build-jtag-prod

# Build server for production
build-prod:
	@echo "Building $(BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Production build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build configuration utility for production
build-cfg-prod:
	@echo "Building $(CFG_BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(CFG_BINARY_NAME) $(CFG_MAIN_PATH)
	@echo "Production build complete: $(BUILD_DIR)/$(CFG_BINARY_NAME)"

# Build JTAG utility for production
build-jtag-prod:
	@echo "Building $(JTAG_BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(JTAG_BINARY_NAME) $(JTAG_MAIN_PATH)
	@echo "Production build complete: $(BUILD_DIR)/$(JTAG_BINARY_NAME)"

# Run the server (requires CERT_PATH and KEY_PATH environment variables)
run: build
	@if [ -z "$(CERT_PATH)" ] || [ -z "$(KEY_PATH)" ]; then \
		echo "Error: CERT_PATH and KEY_PATH environment variables must be set"; \
		echo "Example: make run CERT_PATH=../certs/fullchain.pem KEY_PATH=../certs/privkey.pem"; \
		exit 1; \
	fi
	$(BUILD_DIR)/$(BINARY_NAME) server --cert $(CERT_PATH) --key $(KEY_PATH)

# Run with debug logging
run-debug: build
	@if [ -z "$(CERT_PATH)" ] || [ -z "$(KEY_PATH)" ]; then \
		echo "Error: CERT_PATH and KEY_PATH environment variables must be set"; \
		exit 1; \
	fi
	$(BUILD_DIR)/$(BINARY_NAME) server --cert $(CERT_PATH) --key $(KEY_PATH) --log-level debug

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f *.log
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run integration tests (requires hardware and OpenOCD)
test-integration:
	@echo "Running integration tests..."
	@echo "Note: Integration tests require actual hardware with OpenOCD running"
	$(GO) test -v -tags=integration ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Install all binaries to $GOPATH/bin
install-all: install install-cfg install-jtag

# Install server binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(MAIN_PATH)
	@echo "Installed to: $$(which $(BINARY_NAME))"

# Install configuration utility to $GOPATH/bin
install-cfg:
	@echo "Installing $(CFG_BINARY_NAME)..."
	$(GO) install $(CFG_MAIN_PATH)
	@echo "Installed to: $$(which $(CFG_BINARY_NAME))"

# Install JTAG utility to $GOPATH/bin
install-jtag:
	@echo "Installing $(JTAG_BINARY_NAME)..."
	$(GO) install $(JTAG_MAIN_PATH)
	@echo "Installed to: $$(which $(JTAG_BINARY_NAME))"

# GDB Operations
# Check for GDB prerequisites (arm-none-eabi-gdb, OpenOCD)
check-gdb:
	@echo "Checking GDB prerequisites..."
	@if command -v arm-none-eabi-gdb >/dev/null 2>&1; then \
		echo "✓ arm-none-eabi-gdb found: $$(command -v arm-none-eabi-gdb)"; \
		arm-none-eabi-gdb --version | head -n 1; \
	else \
		echo "✗ arm-none-eabi-gdb not found"; \
		echo "  Install on macOS: brew install --cask gcc-arm-embedded"; \
		echo "  Install on Linux: sudo apt-get install gdb-multiarch"; \
		exit 1; \
	fi
	@echo ""
	@if command -v openocd >/dev/null 2>&1; then \
		echo "✓ openocd found: $$(command -v openocd)"; \
		openocd --version 2>&1 | head -n 1; \
	else \
		echo "⚠ openocd not found (optional, but required for JTAG operations)"; \
		echo "  Install on macOS: brew install openocd"; \
		echo "  Install on Linux: sudo apt-get install openocd"; \
	fi
	@echo ""
	@echo "GDB prerequisites check complete."

# Check for certificate files
check-certs:
	@echo "Checking certificate files..."
	@if [ -f "../custom-certs/ca-root-cert.pem" ]; then \
		echo "✓ ca-root-cert.pem found"; \
	else \
		echo "✗ ca-root-cert.pem not found at ../custom-certs/"; \
		exit 1; \
	fi
	@if [ -f "../custom-certs/ca-root-cert.der" ]; then \
		echo "✓ ca-root-cert.der found"; \
	else \
		echo "✗ ca-root-cert.der not found at ../custom-certs/"; \
		exit 1; \
	fi
	@if [ -f "../custom-certs/ca-root-key.pem" ]; then \
		echo "✓ ca-root-key.pem found"; \
	else \
		echo "✗ ca-root-key.pem not found at ../custom-certs/"; \
		exit 1; \
	fi
	@echo "Certificate files check complete."

# Build configuration utility with GDB support (same as build-cfg, but with verbose output)
build-cfg-with-gdb: check-certs
	@echo "Building $(CFG_BINARY_NAME) with embedded GDB support..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(CFG_BINARY_NAME) $(CFG_MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(CFG_BINARY_NAME)"
	@echo ""
	@echo "Embedded assets:"
	@ls -lh ../custom-certs/ca-root-cert.* 2>/dev/null || echo "  (certificates will be embedded)"
	@echo ""
	@echo "Binary size:"
	@ls -lh $(BUILD_DIR)/$(CFG_BINARY_NAME)

# =============================================================================
# Documentation
# =============================================================================

# Build documentation
docs:
	@echo "Building documentation..."
	@if ! command -v mkdocs >/dev/null 2>&1; then \
		echo "Error: mkdocs is not installed"; \
		echo "Install with: pip install -r requirements.txt"; \
		exit 1; \
	fi
	mkdocs build --clean

# Start local documentation server
docs-serve:
	@echo "Starting documentation server..."
	@echo "Documentation will be available at http://127.0.0.1:8000"
	@if ! command -v mkdocs >/dev/null 2>&1; then \
		echo "Error: mkdocs is not installed"; \
		echo "Install with: pip install -r requirements.txt"; \
		exit 1; \
	fi
	mkdocs serve

# Clean documentation build artifacts
docs-clean:
	@echo "Cleaning documentation build artifacts..."
	rm -rf site/

# =============================================================================
# Release Management
# =============================================================================

# Supported platforms for cross-compilation
PLATFORMS = linux/amd64 linux/arm64 linux/arm darwin/amd64 darwin/arm64 windows/amd64

# Create a new release (interactive)
# Usage: make release VERSION=v1.0.0
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$$'; then \
		echo "Error: VERSION must be in format vX.Y.Z or vX.Y.Z-suffix"; \
		echo "Examples: v1.0.0, v1.2.3-beta.1, v2.0.0-rc1"; \
		exit 1; \
	fi
	@if ! git diff --quiet HEAD 2>/dev/null; then \
		echo "Error: Working directory has uncommitted changes"; \
		echo "Please commit or stash your changes before releasing"; \
		exit 1; \
	fi
	@echo ""
	@echo "Creating release $(VERSION)..."
	@echo ""
	@read -p "Create and push tag $(VERSION)? [y/N] " confirm && \
		if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
			git tag -a $(VERSION) -m "Release $(VERSION)" && \
			echo "Tag $(VERSION) created locally" && \
			git push origin $(VERSION) && \
			echo "" && \
			echo "Release $(VERSION) tag pushed!" && \
			echo "GitHub Actions will now build and publish the release." && \
			echo "Check: https://github.com/muurk/smartap/actions"; \
		else \
			echo "Aborted."; \
		fi

# Dry-run release build (build all platforms locally without tagging)
release-dry-run:
	@echo "Building release binaries for all platforms (dry run)..."
	@echo "Version: $(VERSION)"
	@echo ""
	@mkdir -p $(BUILD_DIR)/release
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d/ -f1); \
		GOARCH=$$(echo $$platform | cut -d/ -f2); \
		if [ "$$GOARCH" = "arm" ]; then \
			GOARM=7; \
			export GOARM; \
			suffix="-$$GOOS-armv7"; \
		else \
			suffix="-$$GOOS-$$GOARCH"; \
		fi; \
		if [ "$$GOOS" = "windows" ]; then \
			ext=".exe"; \
		else \
			ext=""; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -ldflags="-s -w $(LDFLAGS)" \
			-o $(BUILD_DIR)/release/$(BINARY_NAME)$$suffix$$ext $(MAIN_PATH) || exit 1; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -ldflags="-s -w $(LDFLAGS)" \
			-o $(BUILD_DIR)/release/$(CFG_BINARY_NAME)$$suffix$$ext $(CFG_MAIN_PATH) || exit 1; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -ldflags="-s -w $(LDFLAGS)" \
			-o $(BUILD_DIR)/release/$(JTAG_BINARY_NAME)$$suffix$$ext $(JTAG_MAIN_PATH) || exit 1; \
	done
	@echo ""
	@echo "Release binaries built:"
	@ls -lh $(BUILD_DIR)/release/

# Create and push a version tag (non-interactive)
# Usage: make tag VERSION=v1.0.0
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$$'; then \
		echo "Error: VERSION must be in format vX.Y.Z or vX.Y.Z-suffix"; \
		exit 1; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "Tag $(VERSION) pushed. GitHub Actions will create the release."

# =============================================================================
# Help
# =============================================================================

# Show help
help:
	@echo "Smartap Project - Makefile targets:"
	@echo ""
	@echo "Building:"
	@echo "  make build             - Build server binary only"
	@echo "  make build-cfg         - Build configuration utility only"
	@echo "  make build-jtag        - Build JTAG utility only"
	@echo "  make build-all         - Build all three binaries (default)"
	@echo "  make build-prod        - Build server for production"
	@echo "  make build-cfg-prod    - Build config utility for production"
	@echo "  make build-jtag-prod   - Build JTAG utility for production"
	@echo "  make build-prod-all    - Build all for production"
	@echo "  make build-cfg-with-gdb- Build config utility with GDB support (checks certs)"
	@echo ""
	@echo "Running:"
	@echo "  make run               - Build and run server (requires CERT_PATH and KEY_PATH)"
	@echo "  make run-debug         - Build and run server with debug logging"
	@echo ""
	@echo "Installation:"
	@echo "  make install           - Install server to GOPATH/bin"
	@echo "  make install-cfg       - Install config utility to GOPATH/bin"
	@echo "  make install-jtag      - Install JTAG utility to GOPATH/bin"
	@echo "  make install-all       - Install all binaries"
	@echo ""
	@echo "Development:"
	@echo "  make test              - Run unit tests"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-integration  - Run integration tests (requires hardware)"
	@echo "  make lint              - Run golangci-lint"
	@echo "  make fmt               - Format code with go fmt"
	@echo "  make tidy              - Tidy go.mod dependencies"
	@echo "  make clean             - Remove build artifacts"
	@echo ""
	@echo "JTAG Operations:"
	@echo "  make check-gdb         - Verify GDB prerequisites (arm-none-eabi-gdb, openocd)"
	@echo "  make check-certs       - Verify certificate files exist"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs              - Build documentation with mkdocs"
	@echo "  make docs-serve        - Start local documentation server (http://127.0.0.1:8000)"
	@echo "  make docs-clean        - Clean generated documentation"
	@echo ""
	@echo "Release Management:"
	@echo "  make release VERSION=v1.0.0   - Create and push a release tag (interactive)"
	@echo "  make release-dry-run          - Build all platform binaries locally (test)"
	@echo "  make tag VERSION=v1.0.0       - Create and push tag (non-interactive)"
	@echo ""
	@echo "Example usage:"
	@echo "  make build-all"
	@echo "  make run CERT_PATH=../certs/fullchain.pem KEY_PATH=../certs/privkey.pem"
	@echo "  ./bin/smartap-cfg wizard"
	@echo "  ./bin/smartap-jtag inject-certs"
	@echo "  make docs-serve              # Preview docs locally"
	@echo "  make release VERSION=v1.0.0  # Create a release"
