.PHONY: help test lint lint-fix build clean install audit

# Default target
.DEFAULT_GOAL := help

# Add Go bin to PATH for all targets
GOPATH ?= $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies, dev tools, and git hooks
	@echo "[INFO] Installing development environment..."
	@echo ""
	@echo "[1/3] Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies installed"
	@echo ""
	@echo "[2/3] Installing development tools..."
	@command -v golangci-lint >/dev/null 2>&1 || \
		(echo "  Installing golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@command -v staticcheck >/dev/null 2>&1 || \
		(echo "  Installing staticcheck..." && \
		go install honnef.co/go/tools/cmd/staticcheck@latest)
	@command -v ineffassign >/dev/null 2>&1 || \
		(echo "  Installing ineffassign..." && \
		go install github.com/gordonklaus/ineffassign@latest)
	@command -v misspell >/dev/null 2>&1 || \
		(echo "  Installing misspell..." && \
		go install github.com/client9/misspell/cmd/misspell@latest)
	@command -v errcheck >/dev/null 2>&1 || \
		(echo "  Installing errcheck..." && \
		go install github.com/kisielk/errcheck@latest)
	@command -v gocyclo >/dev/null 2>&1 || \
		(echo "  Installing gocyclo..." && \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
	@echo "✓ Development tools installed"
	@echo ""
	@echo "[3/3] Installing git hooks..."
	@bash .githooks/install.sh
	@echo ""
	@echo "✅ Installation complete! Ready to develop."
	@echo ""
	@echo "Next steps:"
	@echo "  • Run 'make test' to verify your setup"
	@echo "  • Run 'make audit' to check code quality"
	@echo "  • See 'make help' for all available commands"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

lint: ## Run linter
	@echo "Running golangci-lint..."
	@$$(go env GOPATH)/bin/golangci-lint run ./...

lint-fix: ## Run linter with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@$$(go env GOPATH)/bin/golangci-lint run --fix ./...

audit: ## Run all quality checks
	@echo "========================================"
	@echo "  CI Quality Checks (Local)"
	@echo "========================================"
	@echo ""
	@echo "[1/8] Checking formatting (gofmt -s)..."
	@output=$$(gofmt -s -l $$(find . -type f -name '*.go' ! -path "./vendor/*" ! -path "./generated/*")); \
	if [ -n "$$output" ]; then \
		echo "❌ The following files are not formatted with gofmt -s:"; \
		echo "$$output"; \
		echo "   Run 'make lint-fix' to fix"; \
		exit 1; \
	fi
	@echo "✓ gofmt passed"
	@echo ""
	@echo "[2/8] Running go vet..."
	@go vet $$(go list ./... | grep -v '/vendor/' | grep -v '/generated/')
	@echo "✓ go vet passed"
	@echo ""
	@echo "[3/8] Running staticcheck..."
	@staticcheck $$(go list ./... | grep -v '/vendor/' | grep -v '/generated/')
	@echo "✓ staticcheck passed"
	@echo ""
	@echo "[4/8] Running ineffassign..."
	@ineffassign ./...
	@echo "✓ ineffassign passed"
	@echo ""
	@echo "[5/8] Running misspell..."
	@misspell -error $$(find . -type f -name '*.go' ! -path "./vendor/*" ! -path "./generated/*")
	@echo "✓ misspell passed"
	@echo ""
	@echo "[6/8] Running errcheck..."
	@errcheck -ignoretests ./...
	@echo "✓ errcheck passed"
	@echo ""
	@echo "[7/8] Running gocyclo (threshold: 15)..."
	@gocyclo_output=$$(gocyclo -over 15 . | grep -v 'vendor/' | grep -v 'generated/' || true); \
	if [ -n "$$gocyclo_output" ]; then \
		echo "❌ Functions with cyclomatic complexity > 15:"; \
		echo "$$gocyclo_output"; \
		exit 1; \
	fi
	@echo "✓ gocyclo passed"
	@echo ""
	@echo "[8/8] Running golangci-lint..."
	@golangci-lint run
	@echo "✓ golangci-lint passed"
	@echo ""
	@echo "========================================"
	@echo "✅ All CI checks passed!"
	@echo "========================================"
	@echo ""
	@echo "Quality Summary:"
	@echo "  ✓ gofmt -s (formatting)"
	@echo "  ✓ go vet (correctness)"
	@echo "  ✓ staticcheck (static analysis)"
	@echo "  ✓ ineffassign (ineffectual assignments)"
	@echo "  ✓ misspell (spelling in .go files)"
	@echo "  ✓ errcheck (error handling)"
	@echo "  ✓ gocyclo (complexity ≤ 15)"
	@echo "  ✓ golangci-lint (comprehensive linting)"
	@echo ""

build: ## Build verification
	@echo "Building plugin..."
	@go build -v ./...
	@echo "✓ Build successful"

clean: ## Clean build artifacts and caches
	@echo "Cleaning..."
	@go clean -cache -testcache -modcache
	@rm -f coverage.out
	@echo "✓ Cleaned"
