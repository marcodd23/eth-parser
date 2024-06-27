all: help

## clean: Clean up all build artifacts
.PHONY: clean
clean:
	@echo "🚀 Cleaning up old artifacts MAIN"

## test: Runs all tests
.PHONY: test
test:
	@echo "🚀 Running tests"
	@go test -cover -count=1 ./internal/...

## build: Build the application artifacts. Linting can be skipped by setting env variable IGNORE_LINTING.
.PHONY: build
build: test
	@go mod tidy
	@echo "🚀 Building artifacts"
	@go build -race -ldflags="-s -w" -o bin ./cmd

.PHONY: run
run:
	@echo "🚀 Running the app"
	@go run ./cmd/main.go

help: Makefile
	@echo
	@echo "📗 Choose a command run in "${REPO_NAME}":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo