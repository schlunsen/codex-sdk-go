.PHONY: help build test test-race fmt fmt-check vet lint coverage examples clean

help:
	@echo "Codex SDK for Go - development tasks"
	@echo ""
	@echo "  make build      - Compile all packages"
	@echo "  make test       - Run unit tests"
	@echo "  make test-race  - Run unit tests with the race detector"
	@echo "  make fmt        - gofmt all sources"
	@echo "  make fmt-check  - Fail if any file is not gofmt'd"
	@echo "  make vet        - go vet"
	@echo "  make lint       - go vet + golangci-lint (if installed)"
	@echo "  make coverage   - Tests with HTML coverage report"
	@echo "  make examples   - Compile every example"
	@echo "  make clean      - Remove build/test artifacts"

build:
	go build ./...

test:
	go test -count=1 ./...

test-race:
	go test -race -count=1 ./...

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "not gofmt'd:"; echo "$$files"; exit 1; fi

vet:
	go vet ./...

lint: vet
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed, skipping"; fi

coverage:
	go test -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "report: coverage.html"

examples:
	@for dir in examples/*/; do \
		echo "building $$dir"; \
		go build -o /dev/null "./$$dir" || exit 1; \
	done

clean:
	go clean -testcache
	rm -f coverage.out coverage.html

.DEFAULT_GOAL := help
