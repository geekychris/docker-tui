.PHONY: build run clean install build-tui build-menu

TUI_BINARY=docker-tui
MENU_BINARY=docker-menu
BUILD_DIR=./bin

build: build-tui build-menu

build-tui:
	go build -o $(BUILD_DIR)/$(TUI_BINARY) main.go docker_ops.go helpers.go

build-menu:
	go build -o $(BUILD_DIR)/$(MENU_BINARY) docker-menu.go

run: build-menu
	$(BUILD_DIR)/$(MENU_BINARY)

run-tui: build-tui
	$(BUILD_DIR)/$(TUI_BINARY)

clean:
	go clean
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

deps:
	go mod tidy
	go mod download

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

dev: fmt vet build

all: clean deps fmt vet build
