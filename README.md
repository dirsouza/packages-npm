# packages-npm

[![CI](https://github.com/dirsouza/packages-npm/actions/workflows/ci.yml/badge.svg)](https://github.com/dirsouza/packages-npm/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/dirsouza/packages-npm)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> Gerenciador visual de pacotes npm globais — desktop app em **Go + Fyne**.

Permite instalar, desinstalar e gerenciar pacotes npm globais de forma centralizada, com persistência local, backup/restauração e detecção automática de status.

---

## Funcionalidades

| Funcionalidade | Descrição |
| --- | --- |
| 📦 Listagem | Exibe todos os pacotes cadastrados com status de instalação em tempo real |
| ➕ Adicionar | Cadastra novo pacote com nome de exibição, nome npm e versão opcional |
| ✏️ Editar | Atualiza os dados de um pacote existente |
| 🗑️ Remover | Remove o registro do banco de dados (não desinstala o pacote) |
| ⬇️ Instalar | Executa `npm install -g` nos pacotes selecionados com progress dialog |
| 🚫 Desinstalar | Executa `npm uninstall -g` nos pacotes instalados selecionados |
| ✅ Status | Detecta automaticamente se cada pacote está instalado, com versão incorreta ou ausente |
| 🟢 Node.js | Exibe no rodapé as versões do Node.js e do npm usados pelo app |
| ☑️ Seleção em massa | Selecionar todos / Desmarcar todos com um clique |
| 🔀 Ordenação | Ordena a lista por ID (ordem de inserção, padrão) ou por Nome |
| 💾 Backup | Exporta todos os pacotes para arquivo `.csv` |
| 📥 Restaurar | Importa de `.csv` em modo Mesclar (preserva existentes) ou Substituir tudo |

---

## Pré-requisitos

| Ferramenta | Versão mínima | Observação |
| --- | --- | --- |
| [Go](https://go.dev/dl/) | 1.26+ (conforme `go.mod`) | Compilação do projeto |
| [Node.js / npm](https://nodejs.org/) | qualquer LTS | Necessário em runtime para instalar/verificar pacotes |
| **macOS** | Xcode Command Line Tools | `xcode-select --install` |
| **Linux** | `gcc` e libs OpenGL/X11/Wayland | `sudo apt install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev` |
| **Windows** | MinGW-w64 ou TDM-GCC | Fyne usa CGo e requer compilador C |

> **Fyne** usa OpenGL/Metal para renderização — é necessário um display gráfico ou ambiente com GPU virtual.

### Detecção do Node.js / npm

Aberto pelo Finder ou pelo menu do sistema, o app não herda o `PATH` do terminal. Por isso ele procura o npm, nesta ordem:

| SO | Onde procura |
| --- | --- |
| macOS / Linux | `PATH` do processo → Homebrew (`/opt/homebrew`, `/usr/local`) / `/usr/bin` / `n` → [Volta](https://volta.sh/) → [asdf](https://asdf-vm.com/) → [fnm](https://github.com/Schniz/fnm) → [nvm](https://github.com/nvm-sh/nvm) |
| Windows | `PATH` do processo → instalador oficial (`Program Files\nodejs`) → Volta → nvm-windows → fnm → Scoop → Chocolatey |

- **asdf:** usa a versão global do `~/.tool-versions` (ou `ASDF_NODEJS_VERSION`); se ela não estiver instalada, usa a maior versão instalada. Após instalar/desinstalar, o app roda `asdf reshim nodejs` para os comandos aparecerem no terminal.
- **nvm:** usa o alias `default`; sem ele, a versão instalada mais recente.

> Com o Corepack habilitado (`corepack enable`), os atalhos `yarn`/`pnpm` não pertencem ao npm, e `npm install -g yarn` falha com `EEXIST`. Para instalar o Yarn pelo app, rode antes `corepack disable yarn`.

---

## Instalação

### 🐧 Linux / 🪟 Windows — binários prontos

Baixe na página de [Releases](https://github.com/dirsouza/packages-npm/releases):

| Plataforma | Arquivo |
| --- | --- |
| Linux | `packages-npm-linux-amd64.tar.xz` — extraia e rode `sudo make install` |
| Windows | `packages-npm-windows-amd64.exe` — execute diretamente |

### 🍎 macOS — build manual

O app **não** é distribuído pré-compilado para macOS: sem assinatura Developer ID / notarização da Apple, o Gatekeeper bloqueia binários baixados da internet. Compilando localmente, o app não recebe o atributo de quarentena e abre sem avisos.

Pré-requisitos: Go 1.26+ e Xcode Command Line Tools (veja tabela acima).

```bash
git clone https://github.com/dirsouza/packages-npm.git
cd packages-npm
go install fyne.io/tools/cmd/fyne@latest
make package-macos
```

O comando apenas gera o instalador `dist/packages-npm-<versão>.pkg` (para a arquitetura do Mac que fez o build) — nada é instalado. Ao abrir o `.pkg`, o `Packages NPM.app` é instalado em `~/Applications` (somente para o usuário atual), sem exigir permissão de administrador — funciona também em Macs gerenciados.

Para instalar pelo terminal:

```bash
installer -pkg dist/packages-npm-<versão>.pkg -target CurrentUserHomeDirectory
```

---

## Execução em desenvolvimento

```bash
# 1. Clone o repositório
git clone https://github.com/dirsouza/packages-npm.git
cd packages-npm

# 2. Baixe as dependências
go mod download

# 3. Execute em modo desenvolvimento
make run

# 4. Ou compile o binário
make build
./bin/packages-npm
```

### Comandos disponíveis

Rode `make` (ou `make help`) para ver o menu com todos os alvos.

| Comando | Descrição |
| --- | --- |
| `make help` | Mostra o menu de comandos (alvo padrão) |
| `make run` | Limpa, compila e executa |
| `make build` | Gera binário em `bin/packages-npm` com versão/build do `FyneApp.toml` |
| `make fmt` | Formata o código (`go fmt`) |
| `make lint` | Roda o `golangci-lint` v2 (config em `.golangci.yml`) |
| `make package` | Empacota com `fyne package` e move os artefatos para `dist/` |
| `make package-macos` | Gera o instalador `.pkg` em `dist/` (instala em `~/Applications`) — macOS |
| `make tidy` | Sincroniza `go.mod` / `go.sum` |
| `make clean` | Remove os diretórios `bin/` e `dist/` |

---

## Banco de dados

O banco SQLite é criado automaticamente na primeira execução:

| OS | Caminho |
| --- | --- |
| macOS | `~/Library/Application Support/packages-npm/packages.db` |
| Linux | `~/.config/packages-npm/packages.db` |
| Windows | `%APPDATA%\packages-npm\packages.db` |

---

## Backup / Restauração

### Exportar

1. Clique em **Exportar** na toolbar
2. Escolha o local e nome do arquivo (padrão: `packages-npm-backup.csv`)
3. O arquivo gerado contém: `id`, `display_name`, `package_name`, `version`

```csv
id,display_name,package_name,version
1,TypeScript,typescript,
2,NestJS CLI,@nestjs/cli,
3,HTTP Server,http-server,14.1.1
```

### Importar

1. Clique em **Importar** na toolbar
2. Selecione o arquivo `.csv`
3. Escolha o modo:
   - **Substituir tudo** — apaga os registros atuais e restaura com os IDs originais
   - **Mesclar** — mantém os existentes, adiciona apenas os ausentes

---

## Versões de pacotes suportadas

O campo "Versão" aceita os seguintes formatos:

| Formato | Exemplo | Comportamento |
| --- | --- | --- |
| Vazio | *(vazio)* | Instala a versão `latest` |
| Versão exata | `5.0.0`, `1.2.3-beta.1` | Instala exatamente essa versão |
| Range `^` | `^5.0.0` | Compatível com minor/patch |
| Range `~` | `~1.2.0` | Compatível com patch |
| Comparadores | `>=2.0.0`, `<3` | Instala versão que satisfaz |
| Wildcard | `1.x`, `*` | Qualquer versão no major |
| Dist-tag | `latest`, `next`, `beta` | Tag npm |

---

## Arquitetura e contribuição

Clean Architecture + Hexagonal (Ports & Adapters) com DDD. Regras de camadas, convenções, comandos de verificação, fluxo de branches (`feat/*`/`fix/*` → `develop` → `main`), versionamento e padrão de commits estão no [AGENTS.md](./AGENTS.md).

---

## Tecnologias

- **[Go 1.26+](https://go.dev/)** — linguagem principal
- **[Fyne v2.8](https://fyne.io/)** — framework de UI cross-platform
- **[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)** — driver SQLite em Go puro

---

## Licença

MIT © [dirsouza](https://github.com/dirsouza)
