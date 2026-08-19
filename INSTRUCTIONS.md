# INSTRUCTIONS — packages-npm

Guia técnico de arquitetura, desenvolvimento e contribuição.

---

## Índice

1. [Arquitetura](#arquitetura)
2. [Estrutura de diretórios](#estrutura-de-diretórios)
3. [Fluxo de dados](#fluxo-de-dados)
4. [Domínio](#domínio)
5. [Ports & Adapters](#ports--adapters)
6. [UI Layer](#ui-layer)
7. [Como adicionar um novo pacote padrão](#como-adicionar-um-novo-pacote-padrão)
8. [Como adicionar uma nova funcionalidade](#como-adicionar-uma-nova-funcionalidade)
9. [Convenções de código](#convenções-de-código)
10. [Testes](#testes)
11. [Build multiplataforma](#build-multiplataforma)

---

## Arquitetura

O projeto segue **Clean Architecture + Hexagonal (Ports & Adapters)** com elementos de **DDD**.

```
┌─────────────────────────────────────────────────────────────┐
│  UI Layer  (fyne — adaptador de entrada)                    │
│  ui/window · ui/components · ui/viewmodel                   │
└────────────────────────┬────────────────────────────────────┘
                         │  inbound port (interface)
┌────────────────────────▼────────────────────────────────────┐
│  Application Core                                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Use Cases  (internal/core/usecases)                 │   │
│  │  PackageService — orquestra todos os casos de uso    │   │
│  └──────────┬──────────────────────────────┬────────────┘   │
│             │ outbound ports               │                 │
│  ┌──────────▼──────────┐  ┌───────────────▼────────────┐   │
│  │  Domain             │  │  Ports/Outbound (interfaces)│   │
│  │  Package · Values   │  │  Repository · Installer     │   │
│  │  Errors · Types     │  │  Checker · BackupStorage    │   │
│  └─────────────────────┘  └───────────────┬────────────┘   │
└──────────────────────────────────────────-─┼────────────────┘
                                             │  implementações
┌────────────────────────────────────────────▼────────────────┐
│  Adapters  (internal/adapters)                              │
│  SQLitePackageRepository · NpmInstaller · NpmPackageChecker │
│  CSVBackupStorage                                           │
└─────────────────────────────────────────────────────────────┘
```

**Regra de dependência:** as setas apontam sempre para dentro (domínio). Nenhuma camada interna importa camadas externas.

---

## Estrutura de diretórios

```
packages-npm/
├── cmd/
│   └── app/
│       ├── main.go          # Composition root — wiring de todas as dependências
│       └── seed.go          # Bootstrap de pacotes padrão (política de produto)
│
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   │   ├── package.go        # Entidade Package + comportamento de negócio
│   │   │   ├── value_objects.go  # PackageName, Version (imutáveis, validados)
│   │   │   ├── types.go          # SortOrder, ImportMode (tipos de aplicação)
│   │   │   └── errors.go         # Erros de domínio tipados
│   │   │
│   │   ├── ports/
│   │   │   ├── inbound/
│   │   │   │   └── package_use_case.go  # Contrato público dos casos de uso
│   │   │   └── outbound/
│   │   │       ├── package_repository.go  # Persistência
│   │   │       ├── package_installer.go   # Instalação npm
│   │   │       ├── package_checker.go     # Verificação de status
│   │   │       └── backup_storage.go      # Backup/restauração
│   │   │
│   │   └── usecases/
│   │       └── package_service.go  # Implementação de todos os casos de uso
│   │
│   ├── adapters/
│   │   ├── persistence/
│   │   │   └── sqlite_package_repository.go  # SQLite via modernc/sqlite
│   │   ├── installer/
│   │   │   ├── npm_installer.go   # npm install/uninstall -g
│   │   │   └── npm_checker.go     # npm list -g --json
│   │   └── backup/
│   │       └── csv_backup_storage.go  # encoding/csv (stdlib)
│   │
│   └── infrastructure/
│       └── database/
│           └── connection.go  # Abre e fecha conexão SQLite
│
├── ui/
│   ├── app.go                    # Factory do app Fyne + tema
│   ├── uitheme/
│   │   └── dark_theme.go         # Tema escuro customizado (Navy + Indigo)
│   ├── viewmodel/
│   │   └── package_vm.go         # PackageVM — desacopla UI do domínio
│   ├── components/
│   │   ├── badge.go              # Badge de versão
│   │   ├── header.go             # Cabeçalho da aplicação
│   │   ├── package_list.go       # Lista scrollável de cards
│   │   ├── dialogs.go            # Toolbar, SelectionBar, StatsBar, forms
│   │   └── install_progress.go   # Dialog de progresso genérico
│   └── window/
│       └── main_window.go        # Presenter — orquestra layout e eventos
│
├── Makefile
├── go.mod
├── go.sum
├── README.md
├── INSTRUCTIONS.md
└── .gitignore
```

---

## Fluxo de dados

### Instalar pacotes selecionados

```
MainWindow.onInstall()
  → useCase.InstallSelected(ids, progressFn)    [inbound port]
    → repo.FindByIDs(ids)                        [outbound port → SQLite]
    → installer.Install(target)                  [outbound port → npm]
    → progressFn(target, done, total, err)       [callback → UI via fyne.Do]
  → refreshInstallStatus()                       [goroutine → fyne.Do]
```

### Verificar status de instalação

```
refreshInstallStatus()  [goroutine]
  → useCase.CheckInstalled()                     [inbound port]
    → checker.ListGlobalInstalled()              [outbound → npm list -g --json]
    → repo.FindAll(SortByDisplayName)            [outbound → SQLite]
    → cross-reference por package_name
  → fyne.Do → pkgList.SetItems(items)            [thread-safe UI update]
```

### Exportar backup

```
MainWindow.onExport()
  → dialog.NewFileSave (Fyne)
  → useCase.ExportBackup(path)                   [inbound port]
    → repo.FindAll(SortByID)                     [outbound → SQLite]
    → backup.Write(path, entries)                [outbound → CSV]
```

---

## Domínio

### Entidade `Package`

```go
type Package struct {
    ID          int64
    DisplayName string      // Nome exibido na UI
    Name        PackageName // Value Object validado
    Version     Version     // Value Object — aceita semver, ranges, dist-tags
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func (p *Package) InstallTarget() string  // "typescript" ou "typescript@5.0.0"
func (p *Package) Validate() error        // valida DisplayName e Name obrigatórios
```

### Value Objects

| Tipo | Restrições | Exemplos válidos |
|---|---|---|
| `PackageName` | não vazio, string npm válida | `typescript`, `@nestjs/cli` |
| `Version` | semver, range, dist-tag ou vazio | `5.0.0`, `^5`, `~1.2`, `latest`, *(vazio)* |

### Erros de domínio

```go
ErrPackageNotFound    // ID não existe no banco
ErrDuplicatePackage   // package_name já cadastrado
ErrNoPackageSelected  // nenhum ID passado para install/uninstall
ErrInstallFailed      // npm install retornou erro
ErrUninstallFailed    // npm uninstall retornou erro
```

---

## Ports & Adapters

### Inbound (o que a UI pode pedir ao core)

```go
type PackageUseCase interface {
    ListAll(order domain.SortOrder) ([]*domain.Package, error)
    Add(dto PackageDTO) (*domain.Package, error)
    Update(dto PackageDTO) error
    Delete(id int64) error
    InstallSelected(ids []int64, onProgress outbound.ProgressFn) error
    UninstallSelected(ids []int64, onProgress outbound.ProgressFn) error
    CheckInstalled() (map[string]InstalledInfo, error)
    ExportBackup(path string) error
    ImportBackup(path string, mode domain.ImportMode) (imported int, err error)
}
```

### Outbound (o que o core precisa de infraestrutura)

```go
PackageRepository  // FindAll, FindByID, FindByIDs, Save, Update, Delete,
                   // DeleteAll, Restore, ReplaceAll
PackageInstaller   // Install(target string) error
                   // Uninstall(pkgName string) error
PackageChecker     // ListGlobalInstalled() (map[string]InstalledPackage, error)
BackupStorage      // Write(path, entries) error
                   // Read(path) ([]BackupEntry, error)
```

---

## UI Layer

A UI é um **adaptador de entrada** — nunca contém lógica de negócio.

| Componente | Responsabilidade |
|---|---|
| `MainWindow` | Presenter: layout, event handlers, estado de seleção |
| `PackageList` | Renderiza cards com status, checkbox, ações |
| `PackageVM` | ViewModel: mapeamento domínio → UI, lógica de status visual |
| `DarkTheme` | Tema customizado: paleta Navy + Indigo |
| `ProgressDialog` | Dialog genérico de progresso para install/uninstall |
| `Badge` | Componente visual de versão com cor por status |

### Thread safety na UI

O Fyne exige que criação de diálogos e mutações visuais ocorram na goroutine principal.

**Padrão obrigatório para operações assíncronas:**

```go
// ✅ Correto
progressDlg := components.NewProgressDialog(...)  // cria antes da goroutine
go func() {
    result := doBlockingWork()         // trabalho pesado na goroutine
    fyne.Do(func() {                   // atualização na goroutine principal
        progressDlg.Finish(...)
        m.reload()
    })
}()

// ❌ Errado — mutação de UI fora de fyne.Do
go func() {
    m.items[0].Selected = true         // data race
    m.pkgList.SetItems(m.items)        // pode crashar
}()
```

---

## Como adicionar um novo pacote padrão

Edite `cmd/app/seed.go` e adicione uma entrada à slice `defaults`:

```go
{displayName: "Meu Pacote", packageName: "meu-pacote", version: ""},
```

O seed só executa quando o banco está vazio — não duplica em execuções posteriores.

---

## Como adicionar uma nova funcionalidade

### Exemplo: exportar para JSON

1. **Novo adapter** em `internal/adapters/backup/json_backup_storage.go`
   - Implemente `outbound.BackupStorage`
2. **Nenhuma mudança no domínio ou use cases** — já existe `ExportBackup(path string)`
3. **Troque o wiring** em `cmd/app/main.go`:
   ```go
   // csvBackup := backup.NewCSVBackupStorage()
   jsonBackup := backup.NewJSONBackupStorage()
   packageService := usecases.NewPackageService(repo, npmInstaller, npmChecker, jsonBackup)
   ```

### Exemplo: novo caso de uso

1. Adicione o método à interface em `internal/core/ports/inbound/package_use_case.go`
2. Implemente em `internal/core/usecases/package_service.go`
3. Se precisar de nova capacidade externa, crie nova interface em `ports/outbound/`
4. Crie o adaptador concreto em `internal/adapters/`
5. Wire em `cmd/app/main.go`
6. Adicione handler na UI em `ui/window/main_window.go`

---

## Convenções de código

| Convenção | Regra |
|---|---|
| **Errors** | Erros de domínio em `domain/errors.go`; erros de infra com `fmt.Errorf("contexto: %w", err)` |
| **Interfaces** | Definidas no lado do consumidor (core), implementadas fora (adapters) |
| **Value Objects** | Sempre imutáveis; construtores retornam `(T, error)` |
| **Goroutines** | Apenas em handlers de UI; toda mutação de UI dentro de `fyne.Do` |
| **Comentários** | Somente onde o código não é auto-explicativo; sem comentários óbvios |
| **Compile-time assertions** | `var _ Interface = (*Type)(nil)` em todo adaptador |
| **Imports** | Ordem: stdlib → externos → internos (separados por linha em branco) |

---

## Testes

Estrutura recomendada:

```
internal/core/usecases/package_service_test.go  # unit tests com mocks
internal/adapters/persistence/sqlite_test.go    # integration tests com DB em memória
internal/adapters/installer/npm_checker_test.go # unit com saída npm mockada
```

Para rodar (quando existirem):

```bash
go test ./...
go test -race ./...        # detecta data races
go test -cover ./...       # cobertura
```

SQLite em memória para testes de integração:

```go
conn, _ := database.Open(":memory:")
repo, _ := persistence.NewSQLitePackageRepository(conn)
```

---

## Build multiplataforma

### macOS (nativo)

```bash
make build
```

### Linux (via Docker)

```bash
docker run --rm -v $(pwd):/app -w /app \
  golang:1.21 bash -c "apt-get install -y libgl1-mesa-dev xorg-dev && go build -o bin/packages-npm-linux ./cmd/app"
```

### Windows (via fyne-cross)

```bash
go install github.com/fyne-io/fyne-cross@latest
fyne-cross windows -arch amd64
```

> **fyne-cross** requer Docker instalado na máquina host.
