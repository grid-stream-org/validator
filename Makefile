BINARY=validator

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

LDFLAGS=-ldflags "-s -w"

ifeq ($(OS),Windows_NT)
    RM_DIR := rmdir /s /q
    RM_FILE := del /f /q
    RUN_BINARY := build\$(BINARY)
else
    RM_DIR := rm -rf
    RM_FILE := rm -f
    RUN_BINARY := ./build/$(BINARY)
endif

.PHONY: build test clean run fmt vet tidy coverage help

download: ## Download project dependencies
	$(GOMOD) download

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o build/$(BINARY) ./cmd/$(BINARY)

test: ## Run tests
	$(GOTEST) -v ./...

clean: ## Clean build files
	$(RM_DIR) build
	$(RM_FILE) coverage.out

run: build ## Build and run the binary
	$(RUN_BINARY)

fmt: ## Run go fmt
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOVET) ./...

tidy: ## Tidy up module files
	$(GOMOD) tidy

coverage: ## Run tests with coverage
	$(GOTEST) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

help: ## Display this help message
	@cat $(MAKEFILE_LIST) | grep -e "^[a-zA-Z_-]*: *.*## *" | \
      awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help