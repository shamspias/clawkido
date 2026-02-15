# Variables
BINARY_NAME=clawkido
MAIN_PATH=cmd/clawkido/main.go

# Default target (runs when you just type 'make')
all: build

# Build the binary
build:
	@echo "🦞 Building Clawkido..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Build and Run immediately
run: build
	@echo "🚀 Running Clawkido..."
	./$(BINARY_NAME)

# Remove binary and cleanup
clean:
	@echo "🧹 Cleaning up..."
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe

# Update dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
