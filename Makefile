# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOWORK=$(GOCMD) work
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
BINARY_NAME=glu

# UI parameters
NPM=npm
UI_DIR=ui

.PHONY: all build test clean dev help fmt lint

all: test build

help:
	@echo "Available targets:"
	@echo "  build       - Build the Go backend and UI"
	@echo "  test        - Run Go tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  fmt         - Format code"
	@echo "  lint        - Run code linting"
	@echo "  dev         - Run both backend and UI in development mode"
	@echo "  init        - Initialize the project"
	@echo "  check       - Run all checks"

build:
	cd $(UI_DIR) && $(NPM) run build
	$(GOBUILD) -o bin/$(BINARY_NAME) -tags ui ./cmd/glu

test:
	$(GOTEST) -v -race ./...

clean:
	rm -f bin/$(BINARY_NAME)
	rm -rf $(UI_DIR)/dist
	rm -rf $(UI_DIR)/node_modules

fmt:
	$(GOFMT) ./...
	cd $(UI_DIR) && $(NPM) run format

lint:
	$(GOVET) ./...
	cd $(UI_DIR) && $(NPM) run lint

# Run both backend and UI in development mode
dev:
	@echo "Starting backend and frontend in development mode..."
	@(trap 'kill 0' SIGINT; \
		go run ./cmd/glu -dev & \
		cd $(UI_DIR) && $(NPM) run start & \
		wait)

# Initialize the project
init: 
	$(GOWORK) init
	$(GOWORK) use .
	$(GOWORK) use ./ui
	$(GOMOD) tidy
	cd $(UI_DIR) && $(NPM) install

# Run all checks
check: fmt lint test 
