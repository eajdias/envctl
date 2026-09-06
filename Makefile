.PHONY: build test coverage lint doctor doctor-fix run-all snapshot install clean

BINARY_NAME=envctl
SRC=./cmd/envctl

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "v1.2.0")

build:
	go build -ldflags "-s -w -X main.Version=$(VERSION)" -o $(BINARY_NAME) $(SRC)

build-windows:
	go build -ldflags "-s -w -X main.Version=$(VERSION)" -o envctl.exe $(SRC)

test:
	go test -v ./...

coverage:
	go test -coverprofile=coverage.out ./...
	@echo "Coverage report: coverage.out"
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

doctor: build
	./$(BINARY_NAME) doctor

doctor-fix: build
	./$(BINARY_NAME) doctor --fix

run-all: build
	./$(BINARY_NAME) run all

snapshot: build
	./$(BINARY_NAME) snapshot

install: build
	@mkdir -p $$HOME/.local/bin
	@cp $(BINARY_NAME) $$HOME/.local/bin/$(BINARY_NAME) 2>/dev/null || powershell.exe -NoProfile -Command "Copy-Item envctl.exe '$$HOME/.local/bin/envctl.exe' -Force"
	@echo "Installed $(BINARY_NAME) to ~/.local/bin"

clean:
	@rm -f $(BINARY_NAME) envctl.exe coverage.out
	@rm -rf dist/
