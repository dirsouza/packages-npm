.PHONY: run build package tidy clean fmt lint

BIN     := bin/packages-npm
MAIN    := ./cmd/app
VERSION   := $(shell grep -E '^Version\s*=' FyneApp.toml | sed 's/.*"\(.*\)".*/\1/')
BUILD     := $(shell grep -E '^Build\s*=' FyneApp.toml | sed 's/.*"\(.*\)".*/\1/')
BUILD_INT := $(shell echo $(BUILD) | sed 's/^0*//;s/^$$/0/')
LDFLAGS   := -X main.Version=$(VERSION) -X main.Build=$(BUILD)

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(MAIN)

package: clean
	@echo "Empacotando com fyne package..."
	@mkdir -p dist
	fyne package --src $(MAIN)
	@find . -maxdepth 1 -name "*.app"    -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.tar.gz" -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.exe"    -exec mv {} dist/ \;
	@echo "Artefatos em dist/"

run: clean build
	$(BIN)

tidy:
	go mod tidy

clean:
	rm -rf bin/ dist/
