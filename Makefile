.PHONY: build test cross-compile clean

# Build for current platform
build:
	go build -o bin/charon-key ./cmd/charon-key

# Run tests
test:
	go test ./...

# Cross-compile for all target platforms
cross-compile: clean
	@echo "Cross-compiling for Linux x86-64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/charon-key-linux-amd64 ./cmd/charon-key
	@echo "Cross-compiling for Linux ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/charon-key-linux-arm64 ./cmd/charon-key
	@echo "Cross-compiling for macOS x86-64 (Intel)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/charon-key-darwin-amd64 ./cmd/charon-key
	@echo "Cross-compiling for macOS ARM64 (Apple Silicon)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/charon-key-darwin-arm64 ./cmd/charon-key
	@echo "Creating checksums..."
	cd bin && sha256sum charon-key-* > checksums.txt

clean:
	rm -rf bin/
	mkdir -p bin

