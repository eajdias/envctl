.PHONY: build test doctor doctor-fix run-all snapshot install clean

BINARY_NAME=envctl
SRC=./cmd/envctl

build:
	go build -ldflags "-s -w -X main.Version=v1.1.0" -o $(BINARY_NAME) $(SRC)

build-windows:
	go build -ldflags "-s -w -X main.Version=v1.1.0" -o envctl.exe $(SRC)

test:
	go test -v ./...

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
	@rm -f $(BINARY_NAME) envctl.exe
	@rm -rf dist/
