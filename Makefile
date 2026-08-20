.PHONY: run build package package-macos tidy clean fmt lint

BIN     := bin/packages-npm
MAIN    := ./cmd/app
VERSION   := $(shell grep -E '^Version\s*=' FyneApp.toml | sed 's/.*"\(.*\)".*/\1/')
BUILD     := $(shell grep -E '^Build\s*=' FyneApp.toml | sed 's/[^0-9]*//g')
BUILD_INT := $(shell echo $(BUILD) | sed 's/^0*//;s/^$$/0/')
LDFLAGS   := -X main.Version=$(VERSION) -X main.Build=$(BUILD)
APP_NAME  := Packages NPM
APP_ID    := com.dirsouza.packages-npm

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(MAIN)

package: clean
	@echo "Empacotando com fyne package..."
	@mkdir -p dist
	@cp FyneApp.toml $(MAIN)/
	fyne package --src $(MAIN)
	@find . -maxdepth 1 -name "*.app"    -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.tar.gz" -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.tar.xz" -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.exe"    -exec mv {} dist/ \;
	@echo "Artefatos em dist/"

# Gera o .app, empacota em instalador .pkg (instala em /Applications)
# e instala o .app em /Applications — apenas macOS
package-macos: clean
	@if [ "$$(go env GOOS)" != "darwin" ]; then \
		echo "❌ package-macos é suportado apenas no macOS"; exit 1; \
	fi
	@command -v fyne >/dev/null 2>&1 || { \
		echo "❌ fyne CLI não encontrado. Instale com: go install fyne.io/tools/cmd/fyne@latest"; exit 1; }
	@command -v pkgbuild >/dev/null 2>&1 || { echo "❌ pkgbuild não encontrado."; exit 1; }
	@echo "🍎 Empacotando $(APP_NAME).app..."
	@mkdir -p dist
	@cp FyneApp.toml $(MAIN)/
	@rm -rf "$(APP_NAME).app"
	fyne package --src $(MAIN)
	@mv -f "$(APP_NAME).app" "dist/$(APP_NAME).app"
	@echo "✓ dist/$(APP_NAME).app"
	@echo "📦 Gerando instalador .pkg..."
	@mkdir -p "dist/pkg-root/Applications"
	@cp -R "dist/$(APP_NAME).app" "dist/pkg-root/Applications/$(APP_NAME).app"
	@pkgbuild \
		--root dist/pkg-root \
		--identifier "$(APP_ID)" \
		--version "$(VERSION)" \
		--install-location "/" \
		"dist/packages-npm-$(VERSION).pkg" >/dev/null
	@rm -rf dist/pkg-root
	@echo "✓ dist/packages-npm-$(VERSION).pkg (instala em /Applications)"
	@if [ -w /Applications ]; then APPS_DIR="/Applications"; \
	else APPS_DIR="$$HOME/Applications"; \
		echo "⚠️  Sem permissão em /Applications — instalando em $$APPS_DIR"; \
	fi; \
	mkdir -p "$$APPS_DIR"; \
	echo "🚚 Instalando em $$APPS_DIR..."; \
	rm -rf "$$APPS_DIR/$(APP_NAME).app"; \
	cp -R "dist/$(APP_NAME).app" "$$APPS_DIR/$(APP_NAME).app"; \
	echo "✓ $$APPS_DIR/$(APP_NAME).app"
	@rm -rf "dist/$(APP_NAME).app"
	@echo "🧹 dist/$(APP_NAME).app removido — artefato final: dist/packages-npm-$(VERSION).pkg"

run: clean build
	$(BIN)

tidy:
	go mod tidy

clean:
	rm -rf bin/ dist/
