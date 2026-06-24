.PHONY: build install clean run test css tidy dev

# Build the pharos CLI binary (includes embedded CSS)
build: css
	go build -o pharos ./cmd/pharos

# Build Tailwind CSS from source (scans Go files for classes directly)
css:
	./.bin/tailwindcss --input web/input.css --output web/app.css --content "**/*.go" --minify
	cp web/app.css internal/web/app.css

# Install to GOPATH/bin
install: css
	go install ./cmd/pharos

# Clean build artifacts
clean:
	rm -f pharos
	go clean

# Run directly
run: css
	go run ./cmd/pharos

# Test
test:
	go test ./...

# Tidy dependencies
tidy:
	go mod tidy

# Hot-reload dev server (rebuilds Go + CSS on change)
dev:
	go run ./cmd/pharos dev
