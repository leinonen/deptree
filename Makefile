.PHONY: build install clean test help

BINARY_NAME=deptree
INSTALL_PATH=/usr/local/bin

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  install  - Build and install to $(INSTALL_PATH)"
	@echo "  clean    - Remove built binaries"
	@echo "  test     - Run tests"
	@echo "  help     - Show this help message"

build:
	go build -o $(BINARY_NAME)

install: build
	cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_PATH)"

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test -v ./...

.DEFAULT_GOAL := build
