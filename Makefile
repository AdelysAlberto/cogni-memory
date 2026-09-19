.PHONY: build install test run clean release-patch release-minor release-major

BINARY_NAME=cogni
INSTALL_DIR=$(HOME)/.local/bin
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v2.0.3")

build:
	@mkdir -p bin
	go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/cogni
	@echo "✅ Binario compilado en bin/$(BINARY_NAME) ($(VERSION))"

install: build
	@mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Instalado globalmente en $(INSTALL_DIR)/$(BINARY_NAME)"

release-patch:
	./release.sh patch

release-minor:
	./release.sh minor

release-major:
	./release.sh major

test:
	go test -v ./...

run:
	go run ./cmd/cogni ui

macos-app:
	@chmod +x macos/build.sh
	./macos/build.sh --install

macos-package:
	@chmod +x macos/package.sh
	./macos/package.sh

clean:
	rm -rf bin/ macos/build/ macos/dist/

