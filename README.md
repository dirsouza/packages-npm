# packages-npm

[![CI](https://github.com/dirsouza/packages-npm/actions/workflows/ci.yml/badge.svg)](https://github.com/dirsouza/packages-npm/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/dirsouza/packages-npm)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> Gerenciador visual de pacotes npm globais — desktop app em **Go + Fyne**.

Permitindo instalar, desinstalar e gerenciar pacotes npm globais de forma centralizada, com persistência local, backup/restauração e detecção automática de status.

---

## Funcionalidades

| Funcionalidade | Descrição |
|---|---|
| 📦 Listagem | Exibe todos os pacotes cadastrados com status de instalação em tempo real |
| ➕ Adicionar | Cadastra novo pacote com nome de exibição, nome npm e versão opcional |
| ✏️ Editar | Atualiza os dados de um pacote existente |
| 🗑️ Remover | Remove o registro do banco de dados |
| ⬇️ Instalar | Executa `npm install -g` nos pacotes selecionados com progress dialog |
| 🚫 Desinstalar | Executa `npm uninstall -g` nos pacotes instalados selecionados |
| ✅ Status | Detecta automaticamente se cada pacote está instalado, com versão incorreta ou ausente |
| ☑️ Seleção em massa | Selecionar todos / Desmarcar todos com um clique |
| 🔀 Ordenação | Ordena a lista por Nome de Exibição ou por ID (ordem de inserção) |
| 💾 Backup | Exporta todos os pacotes para arquivo `.csv` |
| 📥 Restaurar | Importa de `.csv` em modo Mesclar (preserva existentes) ou Substituir tudo |

---

## Pré-requisitos

| Ferramenta | Versão mínima | Observação |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.21+ | Compilação do projeto |
| [Node.js / npm](https://nodejs.org/) | qualquer LTS | Necessário em runtime para instalar/verificar pacotes |
| **macOS** | Xcode Command Line Tools | `xcode-select --install` |
| **Linux** | `gcc`, `libgl1`, `xorg-dev` | `sudo apt install gcc libgl1-mesa-dev xorg-dev` |
| **Windows** | MinGW-w64 ou TDM-GCC | Fyne requer compilador C |

> **Fyne** usa OpenGL/Metal para renderização — é necessário um display gráfico ou ambiente com GPU virtual.

---

## Instalação

### 🐧 Linux / 🪟 Windows — binários prontos

Baixe na página de [Releases](https://github.com/dirsouza/packages-npm/releases):

| Plataforma | Arquivo |
|---|---|
| Linux | `packages-npm-linux-amd64.tar.xz` — extraia e rode `sudo make install` |
| Windows | `packages-npm-windows-amd64.exe` — execute diretamente |

### 🍎 macOS — build manual

O app **não** é distribuído pré-compilado para macOS: sem assinatura Developer ID / notarização da Apple, o Gatekeeper bloqueia binários baixados da internet. Compilando localmente, o app não recebe o atributo de quarentena e abre sem avisos.

Pré-requisitos: Go 1.21+ e Xcode Command Line Tools (veja tabela acima).

```bash
git clone https://github.com/dirsouza/packages-npm.git
cd packages-npm
go install fyne.io/tools/cmd/fyne@latest
make package-macos
```

O comando gera o `Packages NPM.app`, empacota um instalador `dist/packages-npm-<versão>.pkg` (que instala em `/Applications`) e já instala o `.app` em `/Applications` — ou em `~/Applications` caso o usuário não tenha permissão de escrita (Macs gerenciados). Ao final, o `.app` intermediário é removido e apenas o `.pkg` permanece em `dist/`.

> O `.app` precisa estar em uma pasta Applications (`/Applications` ou `~/Applications`) para funcionar corretamente.

---

## Execução em desenvolvimento

```bash
# 1. Clone o repositório
git clone https://github.com/dirsouza/packages-npm.git
cd packages-npm

# 2. Baixe as dependências
go mod tidy

# 3. Execute em modo desenvolvimento
make run

# 4. Ou compile o binário
make build
./bin/packages-npm
```

### Comandos disponíveis

| Comando | Descrição |
|---|---|
| `make run` | Limpa, compila e executa |
| `make build` | Gera binário em `bin/packages-npm` |
| `make package` | Empacota com `fyne package` e move os artefatos para `dist/` |
| `make package-macos` | Gera `.app` + instalador `.pkg` e instala em Applications (macOS) |
| `make tidy` | Sincroniza `go.mod` / `go.sum` |
| `make clean` | Remove o diretório `bin/` |

---

## Banco de dados

O banco SQLite é criado automaticamente na primeira execução:

| OS | Caminho |
|---|---|
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
|---|---|---|
| Vazio | *(vazio)* | Instala a versão `latest` |
| Versão exata | `5.0.0`, `1.2.3-beta.1` | Instala exatamente essa versão |
| Range `^` | `^5.0.0` | Compatível com minor/patch |
| Range `~` | `~1.2.0` | Compatível com patch |
| Comparadores | `>=2.0.0`, `<3` | Instala versão que satisfaz |
| Wildcard | `1.x`, `*` | Qualquer versão no major |
| Dist-tag | `latest`, `next`, `beta` | Tag npm |

---

## Arquitetura

Veja [INSTRUCTIONS.md](./INSTRUCTIONS.md) para detalhes da arquitetura e guia de contribuição.

---

## Tecnologias

- **[Go 1.21+](https://go.dev/)** — linguagem principal
- **[Fyne v2.8](https://fyne.io/)** — framework de UI cross-platform
- **[modernc/sqlite](https://pkg.go.dev/modernc.org/sqlite)** — SQLite puro Go (sem CGo)

---

## Licença

MIT © [dirsouza](https://github.com/dirsouza)
