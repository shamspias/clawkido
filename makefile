BINARY_NAME := clawkido
MAIN_PATH   := cmd/clawkido/main.go
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X main.version=$(VERSION)

.PHONY: all build run clean deps lint vet test fmt release

all: build

build:
	@echo "🦞 Building Clawkido $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Built: ./$(BINARY_NAME)"

run: build
	@echo "🚀 Starting Clawkido..."
	./$(BINARY_NAME)

fmt:
	@echo "📐 Formatting..."
	gofmt -s -w .

vet:
	@echo "🔍 Vetting..."
	go vet ./...

lint: vet
	@echo "🧹 Linting..."
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "  (golangci-lint not installed, skipping)"

test:
	@echo "🧪 Running tests..."
	go test -race -cover ./...

deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy

clean:
	@echo "🧹 Cleaning..."
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

release:
	@echo "📦 Building release binaries..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64   $(MAIN_PATH)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64  $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows.exe  $(MAIN_PATH)
	@echo "✅ Release binaries in ./dist/"
