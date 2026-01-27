# gll-tools justfile
# Development automation for GLL file format tools

set shell := ["bash", "-uc"]

export GOPRIVATE := "github.com/MeKo-Christian"

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
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run all checks (formatting, linting, tests, tidiness)
check: check-formatted lint test check-tidy

# Build gllinfo CLI tool
build-gllinfo:
    go build -o bin/gllinfo ./cmd/gllinfo

# Build all CLI tools
build: build-gllinfo

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
