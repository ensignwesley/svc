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

install-hooks:
	cp .git/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."
