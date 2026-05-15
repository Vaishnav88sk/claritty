VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY      := clarctl
MODULE      := github.com/Vaishnav88sk/claritty/clarctl-go
BUILD_FLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
DIST        := dist

.PHONY: all build install clean test lint cross help

## help: Show this help message
help:
	@echo "clarctl — Claritty AI-SRE CLI (Go)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Build for the current platform
build:
	@mkdir -p $(DIST)
	go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY) .
	@echo "Built $(DIST)/$(BINARY)"

## install: Install clarctl to /usr/local/bin (requires sudo)
install: build
	install -m 755 $(DIST)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed /usr/local/bin/$(BINARY)"

## cross: Cross-compile for Linux, macOS (amd64 + arm64), Windows
cross:
	@mkdir -p $(DIST)
	GOOS=linux   GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-linux-arm64   .
	GOOS=darwin  GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-darwin-amd64  .
	GOOS=darwin  GOARCH=arm64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-darwin-arm64  .
	GOOS=windows GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe .
	@echo "Cross-compiled binaries in $(DIST)/:"
	@ls -lh $(DIST)/

## test: Run unit tests
test:
	go test ./... -v -race

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## clean: Remove built binaries
clean:
	rm -rf $(DIST)

## tidy: Tidy go.mod and go.sum
tidy:
	go mod tidy
