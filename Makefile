.PHONY: build test doctor doctor-fix run-all snapshot install clean

BINARY_NAME=win11-new.exe
SRC=./cmd/win11-new

build:
	go build -ldflags "-s -w -X main.Version=v1.0.0" -o $(BINARY_NAME) $(SRC)

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
	@powershell.exe -NoProfile -Command "Copy-Item $(BINARY_NAME) '$$HOME/.local/bin/$(BINARY_NAME)' -Force"
	@echo "Installed $(BINARY_NAME) to ~/.local/bin"

clean:
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
