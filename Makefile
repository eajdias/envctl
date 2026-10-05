.PHONY: build test coverage lint doctor doctor-fix run-all snapshot install clean

BINARY_NAME=envctl
SRC=./cmd/envctl

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "v1.2.0")

build:
	go build -ldflags "-s -w -X main.Version=$(VERSION)" -o $(BINARY_NAME) $(SRC)

test:
	go test -v ./...

coverage:
	go test -coverprofile=coverage.out ./...
	@echo "Coverage report: coverage.out"
	@go tool cover -func=coverage.out | tail -1

# Same verdict as CI (which gates only new issues): a bare `run ./...` shows
# legacy debt the CI never reports.
lint:
	golangci-lint run --new-from-rev=origin/main ./...

# go run: identical behavior, no ./envctl artifact churning the tree.
doctor:
	go run $(SRC) doctor

doctor-fix:
	go run $(SRC) doctor --fix

run-all:
	go run $(SRC) run all

snapshot:
	go run $(SRC) snapshot

install: build
	@mkdir -p $$HOME/.local/bin
	@cp $(BINARY_NAME) $$HOME/.local/bin/$(BINARY_NAME) 2>/dev/null || powershell.exe -NoProfile -Command "Copy-Item envctl.exe '$$HOME/.local/bin/envctl.exe' -Force"
	@echo "Installed $(BINARY_NAME) to ~/.local/bin"

clean:
	@rm -f $(BINARY_NAME) coverage.out
	@rm -rf dist/
