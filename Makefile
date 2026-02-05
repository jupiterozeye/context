.PHONY: all build install clean test fmt lint nix-build

BINARY_NAME := context
INSTALL_PATH := /usr/local/bin
SHELL_PATH := /usr/local/share/context/shell

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/context

install: build
	mkdir -p $(INSTALL_PATH)
	mkdir -p $(SHELL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/
	cp -r shell/* $(SHELL_PATH)/
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo "Shell scripts installed to $(SHELL_PATH)/"
	@echo ""
	@echo "Add to your shell config:"
	@echo "  Bash:  source $(SHELL_PATH)/context.bash"
	@echo "  Zsh:   source $(SHELL_PATH)/context.zsh"
	@echo "  Fish:  source $(SHELL_PATH)/context.fish"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	rm -rf $(SHELL_PATH)

clean:
	rm -f $(BINARY_NAME)
	rm -rf result/

test:
	go test ./...

fmt:
	go fmt ./...
	gofumpt -l -w .

lint:
	golangci-lint run

run-dir:
	go run ./cmd/context dir

run-last:
	go run ./cmd/context last

nix-build:
	nix build

nix-run-dir:
	nix run . -- dir

dev-shell:
	nix develop