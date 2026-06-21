.PHONY: build install clean run test css tidy

# Build the learn CLI binary (includes embedded CSS)
build: css
	go build -o learn ./cmd/learn

# Build Tailwind CSS from source (scans Go files for classes directly)
css:
	./.local/bin/tailwindcss --input web/input.css --output web/app.css --content "**/*.go" --minify
	cp web/app.css internal/web/app.css

# Install to GOPATH/bin
install: css
	go install ./cmd/learn

# Clean build artifacts
clean:
	rm -f learn
	go clean

# Run directly
run: css
	go run ./cmd/learn

# Test
test:
	go test ./...

# Tidy dependencies
tidy:
	go mod tidy
