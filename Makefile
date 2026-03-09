.PHONY: all test lint

APP_NAME=ultqd
CMD_PATH=./cmd/ultqd
BUILD_DIR=./bin

all: lint test

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run
