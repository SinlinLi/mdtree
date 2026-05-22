BINARY  := bin/mdtree
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: help frontend build backend dev test cover lint fmt run clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

frontend: ## Build the frontend into web/dist
	cd web && npm install && npm run build

backend: ## Build the binary using whatever is already in web/dist
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mdtree

build: frontend backend ## Build the frontend, then the self-contained binary

dev: ## Run the backend and the Vite dev server together (hot reload)
	./scripts/dev.sh

test: ## Run the Go test suite
	go test ./...

cover: ## Run tests and report combined coverage
	go test -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | tail -1

lint: ## Run go vet and golangci-lint (if installed)
	go vet ./...
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipped"

fmt: ## Format Go code
	gofmt -w cmd internal

run: build ## Build everything and run mdtree
	$(BINARY)

clean: ## Remove build artifacts and dependencies
	rm -rf bin web/node_modules coverage.out
	find web/dist -mindepth 1 ! -name .gitkeep -delete
