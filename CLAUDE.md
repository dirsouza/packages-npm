# CLAUDE.md — Instruções de Design para IA

Este arquivo orienta assistentes de IA (GitHub Copilot, Claude, ChatGPT, etc.) sobre as convenções, regras arquiteturais e padrões de código deste projeto. **Leia e respeite tudo aqui antes de gerar ou sugerir qualquer código.**

---

## Visão geral do projeto

`packages-npm` é um **app desktop em Go + Fyne** para gerenciar pacotes npm globais.

- **Linguagem:** Go 1.21+
- **UI:** Fyne v2.8 (OpenGL/Metal, cross-platform)
- **Banco:** SQLite via `modernc.org/sqlite` (pure Go, sem CGo)
- **Módulo Go:** `github.com/dirsouza/packages-npm`

---

## Arquitetura — regras invioláveis

O projeto segue **Clean Architecture + Hexagonal (Ports & Adapters)** com **DDD**.

### Direção das dependências

```
UI  →  inbound port  →  core (domain + usecases)  ←  outbound ports  ←  adapters
```

**Regras:**
1. **O domínio não importa nada** do projeto — apenas stdlib
2. **Use cases (`usecases/`)** importam somente `domain` e `ports`
3. **Inbound ports (`ports/inbound/`)** importam somente `domain` e `ports/outbound`
4. **Adapters** (`adapters/`, `infrastructure/`) são os únicos que importam libs externas (SQLite, Fyne, etc.)
5. **A UI** (`ui/`) importa somente `ports/inbound`, `domain` e seus próprios pacotes
6. **`cmd/app/`** é o único lugar que conhece todas as camadas — é o composition root

### O que NUNCA fazer

- ❌ Importar `ui/` de qualquer pacote em `internal/`
- ❌ Importar `adapters/` do `core/`
- ❌ Colocar lógica de negócio na UI (`ui/window/`, `ui/components/`)
- ❌ Colocar lógica de apresentação no domínio ou use cases
- ❌ Usar `sql.DB` diretamente fora de `internal/adapters/persistence/`
- ❌ Chamar `exec.Command` fora de `internal/adapters/installer/`
- ❌ Usar estado global (variáveis de pacote mutáveis, singletons)
- ❌ Criar uma nova interface outbound sem implementar assertion `var _ Interface = (*Type)(nil)`

---

## Estrutura de pacotes

```
cmd/app/           → composition root (main.go + seed.go)
internal/
  core/
    domain/        → entidades, value objects, erros, tipos de aplicação
    ports/
      inbound/     → contratos que a UI usa (PackageUseCase)
      outbound/    → contratos que o core precisa (Repository, Installer, etc.)
    usecases/      → PackageService — implementa PackageUseCase
  adapters/
    persistence/   → SQLitePackageRepository
    installer/     → NpmInstaller, NpmPackageChecker
    backup/        → CSVBackupStorage
  infrastructure/
    database/      → Connection (abre/fecha sql.DB)
ui/
  app.go           → factory do app Fyne
  uitheme/         → DarkTheme (Navy + Indigo)
  viewmodel/       → PackageVM (mapeia domínio → UI)
  components/      → widgets reutilizáveis
  window/          → MainWindow (presenter)
```

---

## Domínio

### Entidade principal: `domain.Package`

```go
type Package struct {
    ID          int64
    DisplayName string      // obrigatório, validado por Validate()
    Name        PackageName // value object — não vazio, npm-válido
    Version     Version     // value object — semver, range, dist-tag ou vazio
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Nunca** construa `Package` diretamente com `Package{...}` fora do domínio sem chamar `Validate()`. Use sempre `packageFromDTO` no use case.

### Value Objects

- `PackageName`: imutável, validado em `NewPackageName(raw string)`
- `Version`: aceita versão exata (`5.0.0`), ranges (`^5`, `~1.2`, `>=2`), wildcards (`1.x`, `*`) e dist-tags (`latest`, `beta`). Vazio = `latest`.

### Tipos de aplicação (`domain/types.go`)

```go
type SortOrder uint8   // SortByDisplayName | SortByID
type ImportMode uint8  // ImportMerge | ImportReplace
```

Estes tipos vivem no domínio para evitar que `inbound` importe `outbound`.

### Erros de domínio (`domain/errors.go`)

Sempre use erros tipados do domínio. Nunca retorne `errors.New("package not found")` — use `domain.ErrPackageNotFound`.

---

## Ports

### Inbound — `ports/inbound/PackageUseCase`

É o **único contrato** que a UI conhece do core. Toda nova funcionalidade exposta à UI deve ser adicionada aqui primeiro.

### Outbound — `ports/outbound/`

| Interface | Responsabilidade |
|---|---|
| `PackageRepository` | CRUD + ReplaceAll + Restore |
| `PackageInstaller` | Install / Uninstall via npm |
| `PackageChecker` | ListGlobalInstalled via npm list |
| `BackupStorage` | Write / Read de arquivo de backup |
| `ProgressFn` | Callback de progresso (tipo, não interface) |

**Ao criar novo adaptador:** implemente a interface outbound e adicione `var _ outbound.X = (*MyAdapter)(nil)`.

---

## Use Cases (`PackageService`)

- Única implementação de `PackageUseCase`
- Recebe todas as dependências por **injeção no construtor** (`NewPackageService`)
- Nunca acessa banco, npm ou sistema de arquivos diretamente
- Função `packageFromDTO(dto)` centraliza criação de `Package` a partir de DTO — **não duplique essa lógica**

---

## UI

### Regras de thread safety (Fyne)

O Fyne requer que toda criação e mutação de widgets ocorra na **goroutine principal**.

```go
// ✅ CORRETO — operação bloqueante em goroutine, UI via fyne.Do
go func() {
    result := doBlockingWork()
    fyne.Do(func() {
        m.reload()
        m.setStatus("concluído")
    })
}()

// ❌ ERRADO — mutação de UI fora de fyne.Do
go func() {
    m.items[0].Selected = true    // data race
    m.pkgList.SetItems(m.items)   // pode crashar
}()
```

**Regra:** qualquer código que chame `.SetText()`, `.SetItems()`, `.Show()`, `.Hide()`, `dialog.Show*`, `m.reload()`, `m.deselectAll()`, `m.setStatus()` a partir de uma goroutine **deve** estar dentro de `fyne.Do(func() { ... })`.

### ViewModel (`ui/viewmodel/PackageVM`)

- Mapeia `domain.Package` → dados planos para a UI
- Contém lógica de apresentação: `StatusLabel()`, `StatusColorName()`, `ResolveInstallStatus()`
- **Nunca** coloque lógica de negócio aqui — apenas apresentação

### Components (`ui/components/`)

- Widgets puros — recebem dados e callbacks, não dependem do use case
- Não fazem chamadas I/O
- `MainWindow` é o único presenter que orquestra tudo

### Tema (`ui/uitheme/DarkTheme`)

Paleta definida:
- Background: `#0F172A` (Navy escuro)
- Card: `#27374F`
- Primary/Accent: `#6366F1` (Indigo)
- `InnerPadding=6`, `Padding=6`, `InlineIcon=20`, `Text=13`

---

## Banco de dados (SQLite)

- Driver: `modernc.org/sqlite` registrado como `"sqlite"` (não `"sqlite3"`)
- Conexão gerenciada em `infrastructure/database/Connection`; o composition root (`main.go`) controla o lifecycle com `defer conn.Close()`
- Migrations via `ALTER TABLE ADD COLUMN` com tratamento de "duplicate column name" (idempotente)
- Timestamps: `created_at` e `updated_at` preenchidos em `Save` e `Update`; `FindAll` ordena por `domain.SortOrder`

---

## Seed de dados padrão

- **Localização:** `cmd/app/seed.go` — política de produto, **não** no repositório
- Executa apenas quando a tabela está vazia
- Usa os mesmos value objects e `Validate()` do domínio

---

## Backup / Restauração

- Formato: **CSV** com cabeçalho `id,display_name,package_name,version`
- Adapter: `CSVBackupStorage` em `internal/adapters/backup/`
- `ImportReplace`: `ReplaceAll()` transacional (BEGIN → DELETE → INSERT → COMMIT/ROLLBACK)
- `ImportMerge`: adiciona apenas os ausentes por `package_name`
- Cada entrada é validada como `domain.Package` antes de qualquer INSERT

---

## Comandos de build e verificação

```bash
go build ./...          # compila todos os pacotes
go vet ./...            # análise estática
go build -o bin/packages-npm ./cmd/app   # gera o binário
make run                # limpa, compila e executa
make build              # apenas compila
make tidy               # go mod tidy
make clean              # remove bin/
```

**Após qualquer alteração de código:** sempre rode `go build ./...` e `go vet ./...` antes de considerar a tarefa concluída.

---

## Convenções de nomenclatura

| Contexto | Convenção | Exemplo |
|---|---|---|
| Interfaces | Substantivo + sufixo semântico | `PackageRepository`, `BackupStorage` |
| Implementações | Prefixo tecnológico + nome da interface | `SQLitePackageRepository`, `CSVBackupStorage` |
| Construtores | `New` + tipo | `NewPackageService`, `NewMainWindow` |
| Erros de domínio | `Err` + PascalCase | `ErrPackageNotFound`, `ErrDuplicatePackage` |
| Value Objects | Construtores retornam `(T, error)` | `NewPackageName`, `NewVersion` |
| Handlers de UI | `on` + ação | `onAdd`, `onInstall`, `onExport` |
| Helpers privados | camelCase descritivo | `packageFromDTO`, `sortColumn`, `seedDefaultPackages` |

---

## O que fazer ao receber uma tarefa

1. **Leia** os arquivos relevantes antes de editar
2. **Identifique a camada correta**: domínio? porta? use case? adapter? UI?
3. **Respeite a direção de dependências** — nunca inverta
4. **Adicione à interface inbound** antes de implementar no service
5. **Crie o adapter** implementando a interface outbound
6. **Wire em `cmd/app/main.go`** — composition root é o único lugar com visão total
7. **Propague `fyne.Do`** em qualquer mutação de UI vinda de goroutine
8. **Valide com** `go build ./...` e `go vet ./...` antes de encerrar

---

## O que NÃO fazer (resumo rápido)

| ❌ Proibido | ✅ Correto |
|---|---|
| Lógica de negócio na UI | Use case ou domínio |
| `sql.DB` fora de adapters | `PackageRepository` via porta |
| `exec.Command` fora de adapters | `PackageInstaller` / `PackageChecker` via porta |
| Estado global mutável | Injeção de dependências |
| Mutação de UI em goroutine sem `fyne.Do` | Sempre `fyne.Do(func(){...})` |
| `errors.New("mensagem genérica")` no domínio | `domain.ErrXxx` tipado |
| Duplicar `NewPackageName + NewVersion + Validate` | `packageFromDTO(dto)` |
| Seed no repositório | `cmd/app/seed.go` |
| Aliases de tipo sem uso externo real | Remova (YAGNI) |
