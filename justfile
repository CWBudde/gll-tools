# gll-tools justfile
# Development automation for GLL file format tools

set shell := ["bash", "-uc"]

export GOPRIVATE := "github.com/cwbudde"

# Default recipe - show available commands
default:
    @just --list

# Note: Install dependencies manually or use the GitHub Actions workflow
# treefmt: Download from https://github.com/numtide/treefmt/releases
# Go tools: go install mvdan.cc/gofumpt@latest && go install github.com/daixiang0/gci@latest && go install mvdan.cc/sh/v3/cmd/shfmt@latest
# golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# prettier: npm install -g prettier

# Format all code using treefmt
fmt:
    treefmt --allow-missing-formatter

# Check if code is formatted correctly
check-formatted:
    treefmt --allow-missing-formatter --fail-on-change

# Run linters
lint:
    golangci-lint run --timeout=2m

# Run linters with auto-fix
lint-fix:
    golangci-lint run --fix --timeout=2m

# Ensure go.mod is tidy
check-tidy:
    go mod tidy
    git diff --exit-code go.mod go.sum

# Run all tests
test:
    go test -v -timeout 120s ./...

# Run tests with coverage
test-coverage:
    go test -v -timeout 120s -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run all checks (formatting, linting, tests, tidiness)
check: check-formatted lint test check-tidy

# Build gllinfo CLI tool
build-gllinfo:
    go build -o bin/gllinfo ./cmd/gllinfo

# Build WebAssembly module for web demo
build-wasm:
    GOOS=js GOARCH=wasm go build -buildvcs=false -o web/gll.wasm ./cmd/gllwasm
    @echo "Copying wasm_exec.js..."
    @GOROOT=$(go env GOROOT) && \
    if [ -f "$$GOROOT/misc/wasm/wasm_exec.js" ]; then \
        cp "$$GOROOT/misc/wasm/wasm_exec.js" web/; \
    elif [ -f "$$GOROOT/lib/wasm/wasm_exec.js" ]; then \
        cp "$$GOROOT/lib/wasm/wasm_exec.js" web/; \
    else \
        echo "Downloading wasm_exec.js..."; \
        curl -sL "https://raw.githubusercontent.com/golang/go/go1.22.0/misc/wasm/wasm_exec.js" -o web/wasm_exec.js; \
    fi
    @echo "WASM build complete: web/gll.wasm"

# Build all CLI tools
build: build-gllinfo build-wasm

# Install gllinfo to $GOPATH/bin
install-gllinfo:
    go install ./cmd/gllinfo

# Install all CLI tools
install: install-gllinfo

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f coverage.out coverage.html

# Run gllinfo on sample files
test-sample FILE="testdata/gll/CoRay4-V1_5.gll":
    go run ./cmd/gllinfo info "{{ FILE }}"

# Extract resources from a GLL file
extract-sample FILE="testdata/gll/CoRay4-V1_5.gll" OUTPUT="extracted":
    go run ./cmd/gllinfo extract "{{ FILE }}" --output "{{ OUTPUT }}"

# Show version information
version:
    go run ./cmd/gllinfo version

# Parse XGLL example
parse-xgll FILE="testdata/xgll/example-ls.xgll":
    go run ./cmd/xgllc parse "{{ FILE }}"

# Convert XGLL to binary container
convert-xgll FILE="testdata/xgll/example-ls.xgll" OUTPUT="example.xgllbin":
    go run ./cmd/xgllc convert "{{ FILE }}" --output "{{ OUTPUT }}" --format xgllbin

# Convert XGLL to pretty binary container
convert-xgll-pretty FILE="testdata/xgll/example-ls.xgll" OUTPUT="example-pretty.xgllbin":
    go run ./cmd/xgllc convert "{{ FILE }}" --output "{{ OUTPUT }}" --format xgllbin --pretty

# Validate XGLL example
validate-xgll FILE="testdata/xgll/example-la.xgll":
    go run ./cmd/xgllc validate "{{ FILE }}"

# Compile all XGLL testdata to GLL files
compile-xgll:
    #!/usr/bin/env bash
    for f in testdata/xgll/*.xgll; do
        basename="${f##*/}"
        out="testdata/gll/${basename%.xgll}.gll"
        echo "Compiling $f -> $out"
        go run ./cmd/xgllc convert "$f" -f gll -o "$out"
    done

# === Python Bindings ===

# Build Python shared library
build-python:
    CGO_ENABLED=1 go build -buildmode=c-shared -o python/gll/_libgll.so ./cmd/gllpy
    @echo "Python shared library built: python/gll/_libgll.so"

# Install Python package in development mode
install-python: build-python
    pip install -e ./python

# Run Python tests
test-python: build-python
    cd python && python -m pytest tests/ -v

# Build Python wheel (for current platform)
build-python-wheel: build-python
    cd python && pip wheel . -w dist/ --no-deps

# Clean Python build artifacts
clean-python:
    rm -f python/gll/_libgll.so python/gll/_libgll.h
    rm -rf python/dist/ python/build/ python/*.egg-info python/gll/*.egg-info
    find python -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true

# Type check Python code
typecheck-python:
    cd python && python -m mypy gll/

# Lint Python code
lint-python:
    cd python && python -m ruff check gll/ tests/

# Format Python code
fmt-python:
    cd python && python -m ruff format gll/ tests/

# Run all Python checks
check-python: build-python lint-python typecheck-python test-python
