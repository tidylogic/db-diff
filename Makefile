.PHONY: all ui build clean dev-ui

# Build everything: frontend → embed → Go binary
all: ui build

# Install Node deps and build the React frontend into web/static/
ui:
	cd ui && npm install && npm run build

# Build the Go binary (requires ui to have been built)
build:
	go build -o db-diff ./cmd/db-diff

# Build the Go binary only (skip UI build — use if static files are already built)
build-go:
	go build -o db-diff ./cmd/db-diff

# Start Vite dev server for UI-only development
dev-ui:
	cd ui && npm run dev

# Remove build artifacts
clean:
	rm -f db-diff
	rm -rf web/static/assets
	find web/static -maxdepth 1 -type f ! -name '.gitkeep' -delete

# Run the web server (builds everything first)
serve: all
	./db-diff web
