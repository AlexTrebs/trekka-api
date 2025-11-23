.PHONY: help build run test clean dev install-deps tidy

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building server..."
	@go build -o bin/server cmd/server/main.go
	@echo "Building update-metadata..."
	@go build -o bin/update-metadata cmd/update-metadata/main.go

run: ## Run the application
	@echo "Running server..."
	@go run cmd/server/main.go

sync-update-metadata: ## Update metadata for ALL files in Firestore (extract GPS from Storage/Drive)
	@echo "Updating metadata for ALL files in Firestore..."
	@go run cmd/update-metadata/main.go

sync-update-metadata-empty: ## Update metadata ONLY for files missing GPS/location data
	@echo "Updating metadata for files with empty GPS/location fields..."
	@go run cmd/update-metadata/main.go -only-empty

sync-update-metadata-backfill: ## Force download from Drive for all files (slower but more reliable)
	@echo "Backfilling metadata from Google Drive..."
	@go run cmd/update-metadata/main.go -backfill

sync-update-metadata-backfill-empty: ## Force download from Drive for files missing GPS/location only
	@echo "Backfilling metadata from Google Drive for empty fields..."
	@go run cmd/update-metadata/main.go -backfill -only-empty

sync-update-metadata-dry-run: ## Preview metadata updates without making changes
	@echo "Previewing metadata updates (dry run)..."
	@go run cmd/update-metadata/main.go -dry-run

sync-update-metadata-empty-dry-run: ## Preview updates for empty fields only (dry run)
	@echo "Previewing metadata updates for empty fields (dry run)..."
	@go run cmd/update-metadata/main.go -only-empty -dry-run

sync-update-metadata-backfill-dry-run: ## Preview backfill updates (dry run)
	@echo "Previewing backfill updates (dry run)..."
	@go run cmd/update-metadata/main.go -backfill -dry-run

dev: ## Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
	@air

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

install-deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@golangci-lint run
