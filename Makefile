.PHONY: help run build package package-macos tidy clean fmt lint

.DEFAULT_GOAL := help

BIN     := bin/packages-npm
MAIN    := ./cmd/app
VERSION   := $(shell grep -E '^Version\s*=' FyneApp.toml | sed 's/.*"\(.*\)".*/\1/')
BUILD     := $(shell grep -E '^Build\s*=' FyneApp.toml | sed 's/[^0-9]*//g')
BUILD_INT := $(shell echo $(BUILD) | sed 's/^0*//;s/^$$/0/')
LDFLAGS   := -X main.Version=$(VERSION) -X main.Build=$(BUILD)
APP_NAME  := Packages NPM
APP_ID    := com.dirsouza.packages-npm

##@ Ajuda

help: ## Mostra este menu
	@printf "\033[1m$(APP_NAME)\033[0m v$(VERSION) (build $(BUILD))\n"
	@printf "\nUso: make \033[36m<alvo>\033[0m\n"
	@awk 'BEGIN {FS = ":.*## "} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} \
		/^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Desenvolvimento

run: clean build ## Limpa, compila e executa o app
	$(BIN)

build: ## Compila o binário em bin/packages-npm
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(MAIN)

fmt: ## Formata o código (go fmt)
	go fmt ./...

lint: ## Roda o golangci-lint
	golangci-lint run ./...

##@ Empacotamento

package: clean ## Empacota com fyne package para o SO atual (artefatos em dist/)
	@echo "Empacotando com fyne package..."
	@mkdir -p dist
	@cp FyneApp.toml $(MAIN)/
	fyne package --src $(MAIN)
	@find . -maxdepth 1 -name "*.app"    -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.tar.gz" -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.tar.xz" -exec mv {} dist/ \;
	@find . -maxdepth 1 -name "*.exe"    -exec mv {} dist/ \;
	@echo "Artefatos em dist/"

# Gera o instalador dist/packages-npm-<versão>.pkg — apenas macOS.
# Não instala nada: o .pkg instala o .app em ~/Applications (domínio do
# usuário, sem permissão de administrador). O bundle não é relocável para
# o Installer não sobrescrever cópias do app em outras pastas.
package-macos: clean ## Gera o instalador .pkg do macOS (instala em ~/Applications)
	@if [ "$$(go env GOOS)" != "darwin" ]; then \
		echo "❌ package-macos é suportado apenas no macOS"; exit 1; \
	fi
	@command -v fyne >/dev/null 2>&1 || { \
		echo "❌ fyne CLI não encontrado. Instale com: go install fyne.io/tools/cmd/fyne@v1.7.4"; exit 1; }
	@command -v pkgbuild >/dev/null 2>&1 || { echo "❌ pkgbuild não encontrado."; exit 1; }
	@command -v productbuild >/dev/null 2>&1 || { echo "❌ productbuild não encontrado."; exit 1; }
	@echo "🍎 Empacotando $(APP_NAME).app..."
	@mkdir -p dist/pkg-root
	@cp FyneApp.toml $(MAIN)/
	@rm -rf "$(APP_NAME).app"
	fyne package --src $(MAIN)
	@mv -f "$(APP_NAME).app" "dist/pkg-root/$(APP_NAME).app"
	@echo "📦 Gerando instalador .pkg..."
	@pkgbuild --analyze --root dist/pkg-root dist/component.plist >/dev/null
	@plutil -replace 0.BundleIsRelocatable -bool NO dist/component.plist
	@pkgbuild \
		--root dist/pkg-root \
		--component-plist dist/component.plist \
		--identifier "$(APP_ID)" \
		--version "$(VERSION)" \
		--install-location "/Applications" \
		dist/component.pkg >/dev/null
	@printf '%s\n' \
		'<?xml version="1.0" encoding="utf-8"?>' \
		'<installer-gui-script minSpecVersion="2">' \
		'  <title>$(APP_NAME)</title>' \
		'  <domains enable_anywhere="false" enable_currentUserHome="true" enable_localSystem="false"/>' \
		'  <options customize="never" require-scripts="false" hostArchitectures="$(shell uname -m)"/>' \
		'  <choices-outline><line choice="$(APP_ID)"/></choices-outline>' \
		'  <choice id="$(APP_ID)" visible="false"><pkg-ref id="$(APP_ID)"/></choice>' \
		'  <pkg-ref id="$(APP_ID)" version="$(VERSION)">component.pkg</pkg-ref>' \
		'</installer-gui-script>' > dist/distribution.xml
	@productbuild \
		--distribution dist/distribution.xml \
		--package-path dist \
		"dist/packages-npm-$(VERSION).pkg" >/dev/null
	@rm -rf dist/pkg-root dist/component.plist dist/component.pkg dist/distribution.xml
	@echo "✓ dist/packages-npm-$(VERSION).pkg (instala em ~/Applications)"

##@ Manutenção

tidy: ## Sincroniza dependências (go mod tidy)
	go mod tidy

clean: ## Remove bin/ e dist/
	rm -rf bin/ dist/
