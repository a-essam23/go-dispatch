# Makefile for the go-dispatch project

# Set the name of your final binary
BINARY_NAME=go-dispatch
# Set the path to the main package
CMD_PATH=./cmd/go-dispatch

# Go parameters
GO=go
GO_BUILD=$(GO) build
GO_RUN=$(GO) run
GO_TEST=$(GO) test
GO_MOD=$(GO) mod

# Build flags
# -v: print the names of packages as they are compiled.
# -ldflags="-s -w": link-time flags to strip debug symbols and DWARF info, making the binary smaller.
BUILD_FLAGS=-v -ldflags="-s -w"

# Default command to run when you just type "make"
.DEFAULT_GOAL := help

## --------------------------------------
## Build Commands
## --------------------------------------

# Build the application for your current OS and place it in ./bin/
build:
	@echo "==> Building binary for development..."
	@mkdir -p ./bin
	$(GO_BUILD) $(BUILD_FLAGS) -o ./bin/$(BINARY_NAME) $(CMD_PATH)

# Cross-compile a lean, static Linux binary for production deployment
build-linux:
	@echo "==> Building static Linux binary for production..."
	@mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) $(BUILD_FLAGS) -o ./bin/$(BINARY_NAME)-linux $(CMD_PATH)

## --------------------------------------
## Development & Testing
## --------------------------------------

# Run the application locally, using the .env file if it exists
run:
	@echo "==> Running application..."
	$(GO_RUN) $(CMD_PATH)

# Run all tests, including the race detector for concurrency safety
test:
	@echo "==> Running tests with race detector..."
	$(GO_TEST) -v -race ./...

# Run tests and generate a code coverage report
coverage:
	@echo "==> Running tests and generating coverage report..."
	$(GO_TEST) -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report generated at coverage.html"

## --------------------------------------
## Dependency Management
## --------------------------------------

# Tidy up go.mod and go.sum files
tidy:
	@echo "==> Tidying module dependencies..."
	$(GO_MOD) tidy

# Ensure all dependencies are downloaded to the local module cache
deps:
	@echo "==> Downloading dependencies..."
	$(GO_MOD) download

## --------------------------------------
## Housekeeping
## --------------------------------------

# Remove all build artifacts and coverage reports
clean:
	@echo "==> Cleaning up..."
	@rm -rf ./bin
	@rm -f coverage.out coverage.html

# Display a helpful list of available commands
help:
	@echo ""
	@echo "Usage: make <command>"
	@echo ""
	@echo "Available commands:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
