.PHONY: dev-backend dev-frontend dev build test lint clean

# Backend development (with hot reload if air is installed)
dev-backend:
	@echo "🚀 Starting Go backend..."
	go run ./cmd/server/

# Frontend development
dev-frontend:
	@echo "🎨 Starting Vite dev server..."
	cd web && npm run dev

# Run both concurrently
dev:
	@echo "🚀 Starting Trace-Point in development mode..."
	@make dev-backend &
	@make dev-frontend &
	@wait

# Build everything
build: build-frontend build-backend

build-backend:
	@echo "🔨 Building Go binary..."
	go build -o bin/trace-point ./cmd/server/

build-frontend:
	@echo "🔨 Building frontend..."
	cd web && npm run build

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./... -v

# Lint
lint:
	@echo "🔍 Running linter..."
	golangci-lint run ./...

# Clean
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf data/

# Install dependencies
deps:
	go mod tidy
	cd web && npm install

# Database reset
db-reset:
	rm -f data/trace-point.db
	@echo "✅ Database reset"
