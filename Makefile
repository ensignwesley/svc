.PHONY: build test check-version lint all

all: check-version test build

build:
	go build -o svc ./cmd/svc/

test:
	go test ./...

# Verify version constant in main.go matches ## Status version in README.md
check-version:
	@VERSION_GO=$$(grep 'const version' cmd/svc/main.go | grep -o '"[^"]*"' | tr -d '"'); \
	VERSION_README=$$(grep '^\*\*v' README.md | head -1 | grep -o 'v[0-9][0-9.]*' | head -1 | tr -d 'v'); \
	if [ "$$VERSION_GO" != "$$VERSION_README" ]; then \
		echo "❌ Version mismatch: main.go=$$VERSION_GO README=$$VERSION_README"; \
		exit 1; \
	fi; \
	echo "✅ Version consistent: $$VERSION_GO"

# Warn when non-test Go source exceeds the cognitive-overhead ceiling.
# This is a warning, not a hard failure — the ceiling is a heuristic, not a law.
loc-check:
	@LINES=$$(find . -name "*.go" -not -path "*/vendor/*" -not -name "*_test.go" | xargs wc -l | tail -1 | awk '{print $$1}'); \
	echo "Non-test Go: $$LINES lines"; \
	if [ "$$LINES" -gt 3500 ]; then \
		echo "⚠️  LOC ceiling exceeded ($$LINES > 3500). Consider: splitting main.go, cutting features, or raising the ceiling consciously."; \
	elif [ "$$LINES" -gt 3000 ]; then \
		echo "⚠️  Approaching LOC ceiling ($$LINES / 3500). Review before adding features."; \
	else \
		echo "✅ Within ceiling."; \
	fi

install-hooks:
	cp .git/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."

# Build release binaries for common platforms
release:
	@mkdir -p dist
	@VERSION=$$(grep 'const version' cmd/svc/main.go | grep -o '"[^"]*"' | tr -d '"'); \
	echo "Building svc v$$VERSION..."; \
	GOOS=linux  GOARCH=amd64 go build -ldflags="-s -w -X main.version=$$VERSION" -o dist/svc-linux-amd64  ./cmd/svc/; \
	GOOS=linux  GOARCH=arm64 go build -ldflags="-s -w -X main.version=$$VERSION" -o dist/svc-linux-arm64  ./cmd/svc/; \
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$$VERSION" -o dist/svc-darwin-arm64 ./cmd/svc/; \
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$$VERSION" -o dist/svc-darwin-amd64 ./cmd/svc/
	@echo "Binaries in dist/:"
	@ls -lh dist/
