# AGENTS.md — Instruções de Design para IA

Este arquivo orienta assistentes de IA sobre as convenções, regras arquiteturais e padrões de código deste projeto. **Leia e respeite tudo aqui antes de gerar ou sugerir qualquer código.**

---

## Visão geral do projeto

`packages-npm` é um **app desktop em Go + Fyne** para gerenciar pacotes npm globais.

- **Linguagem:** Go 1.26+ (conforme `go.mod`)
- **UI:** Fyne v2.8 (OpenGL/Metal, cross-platform: macOS, Linux e Windows)
- **Banco:** SQLite via `modernc.org/sqlite` (driver pure Go)
- **Módulo Go:** `github.com/dirsouza/packages-npm`

---

## Preparação do ambiente

- **Go** na versão do `go.mod` (a diretiva `toolchain` fixa o patch usado no build e no CI) e **Node.js/npm** (necessário em runtime para instalar e verificar pacotes)
- **Compilador C e libs gráficas** (o Fyne usa CGo e OpenGL/Metal):
  - macOS: `xcode-select --install`
  - Linux: `sudo apt-get install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev`
  - Windows: GCC do MinGW-w64 (o CI usa `mingw-w64-x86_64-gcc` via MSYS2)
- **fyne CLI** (para `make package*`): `go install fyne.io/tools/cmd/fyne@v1.7.4`
- **golangci-lint v2** (para `make lint`; o `.golangci.yml` usa `version: "2"`): `brew install golangci-lint` ou `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`

---

## Arquitetura — regras invioláveis

O projeto segue **Clean Architecture + Hexagonal (Ports & Adapters)** com **DDD**.

### Direção das dependências

```text
UI  →  inbound port  →  core (domain + usecases)  ←  outbound ports  ←  adapters
```

**Regras:**

1. **O domínio não importa nada** do projeto — apenas stdlib
2. **Use cases (`usecases/`)** importam somente `domain` e `ports`
3. **Inbound ports (`ports/inbound/`)** importam somente `domain` e `ports/outbound`
4. **Libs externas ficam nas bordas**: o driver SQLite só em `infrastructure/database/` (e `database/sql` só em `adapters/persistence/`); o Fyne só em `ui/`
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

```text
cmd/app/           → composition root (main.go, seed.go, version.go)
internal/
  core/
    domain/        → entidades, value objects, erros, tipos de aplicação
    ports/
      inbound/     → contratos que a UI usa (PackageUseCase)
      outbound/    → contratos que o core precisa (Repository, Installer, etc.)
    usecases/      → PackageService — implementa PackageUseCase
  adapters/
    persistence/   → SQLitePackageRepository
    installer/     → NpmInstaller, NpmPackageChecker, resolução do npm (npm_resolver.go, asdf.go)
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
    Name        PackageName // value object — segue as regras de nome do npm
    Version     Version     // value object — semver, range, dist-tag ou vazio
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Nunca** construa `Package` diretamente com `Package{...}` fora do domínio sem chamar `Validate()`. Use sempre `packageFromDTO` no use case.

### Value Objects

- `PackageName`: imutável, criado por `NewPackageName(raw string)`. Faz trim e aceita só nomes válidos no npm — letras, dígitos, `-`, `.`, `_`, `~`, escopo `@x/` opcional, até 214 caracteres, sem começar com `-`. Isso barra flags, URLs, caminhos e metacaracteres de shell (o `npm.cmd` do Windows passa pelo `cmd.exe`)
- `Version`: aceita versão exata (`5.0.0`), ranges (`^5`, `~1.2`, `>=2`), wildcards (`1.x`, `*`) e dist-tags (`latest`, `beta`). Vazio = `latest`. `IsExact()` diz se fixa uma única versão — só versão exata pode ficar "diferente da instalada"
- `Package.Validate()` rejeita nome de exibição vazio ou só com espaços

### Tipos de aplicação (`domain/types.go`)

```go
type SortOrder uint8   // SortByDisplayName | SortByID
type ImportMode uint8  // ImportMerge | ImportReplace
```

Estes tipos vivem no domínio porque são usados tanto pela UI (via `inbound`) quanto pelos adapters (via `outbound`).

### Erros de domínio (`domain/errors.go`)

Sempre use erros tipados do domínio (`ErrEmptyDisplayName`, `ErrEmptyPackageName`, `ErrInvalidPackageName`, `ErrInvalidVersion`, `ErrPackageNotFound`, `ErrDuplicatePackage`, `ErrInstallFailed`, `ErrUninstallFailed`, `ErrNoPackageSelected`). Nunca retorne `errors.New(...)` avulso no domínio; adapters adicionam contexto com `fmt.Errorf("%w: ...", domain.ErrXxx)`.

---

## Ports

### Inbound — `ports/inbound/PackageUseCase`

É o **único contrato** que a UI conhece do core. Toda nova funcionalidade exposta à UI deve ser adicionada aqui primeiro.

### Outbound — `ports/outbound/`

| Interface | Responsabilidade |
| --- | --- |
| `PackageRepository` | CRUD + ReplaceAll |
| `PackageInstaller` | Install / Uninstall via npm |
| `PackageChecker` | ListGlobalInstalled via `npm list`; CheckRuntime (versões do Node.js e npm) via `npm version` |
| `BackupStorage` | Write / Read de arquivo de backup |
| `ProgressFn` | Callback de progresso (tipo, não interface) |

**Ao criar novo adaptador:** implemente a interface outbound e adicione `var _ outbound.X = (*MyAdapter)(nil)`.

---

## Use Cases (`PackageService`)

- Única implementação de `PackageUseCase`
- Recebe todas as dependências por **injeção no construtor** (`NewPackageService`)
- Nunca acessa banco, npm ou sistema de arquivos diretamente
- Função `packageFromDTO(dto)` centraliza criação de `Package` a partir de DTO — **não duplique essa lógica**
- `InstallSelected` e `UninstallSelected` compartilham `runOnSelected` (validação dos IDs, busca e progresso); só mudam o argumento do npm e a operação
- `ImportBackup` devolve `inbound.ImportResult` (`Imported`, `Skipped`): no modo Mesclar, só entradas já cadastradas ou inválidas (erros de validação/duplicidade do domínio) contam como ignoradas; qualquer outro erro (ex.: banco indisponível) interrompe e volta com o resultado parcial

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

## Execução do npm (`internal/adapters/installer/`)

O app é aberto pelo Finder/menu do sistema e **não recebe o `PATH` do shell**. Por isso `newNpmCmd` resolve o npm por caminho absoluto e adiciona o diretório dele ao `PATH` do processo filho (o npm precisa achar o `node`).

- Ordem de busca (`resolveNpm`): `PATH` do processo → Homebrew/sistema/n → Volta → asdf → fnm → nvm (alias `default`, depois a maior versão); no Windows: instalador oficial, Volta, nvm-windows, fnm, Scoop, Chocolatey
- Caminhos fixos (Homebrew/sistema) vêm antes dos gerenciadores de versão: com Node do Homebrew e do asdf, o app aberto pelo Finder usa o do Homebrew mesmo que o terminal use o asdf. O rodapé mostra qual está em uso
- **asdf** (`asdf.go`): usa `ASDF_NODEJS_VERSION` ou o `~/.tool-versions` global e aponta direto para `installs/nodejs/<versão>/bin/npm` (os shims dependem do binário `asdf` no `PATH`); fallback para a maior versão executável. Após install/uninstall, roda `asdf reshim nodejs`
- Todo candidato passa por `isExecutable`; nos gerenciadores com várias versões (asdf, fnm, nvm, nvm-windows) vale a maior versão executável, comparada numericamente (`globNewest`/`compareVersions`, ignorando o prefixo `v`); caminhos comparados já normalizados (`absolutePath`)
- O alvo do npm é `Package.InstallTarget()` (`nome@versão`), montado a partir dos value objects
- Falha de install/uninstall volta como `domain.ErrInstallFailed`/`ErrUninstallFailed` com as primeiras linhas `npm error` da saída (`npmErrorSummary`); a UI mostra esse motivo, não só o nome do pacote

---

## Banco de dados (SQLite)

- Driver: `modernc.org/sqlite`, aberto com o nome `"sqlite"`
- Conexão gerenciada em `infrastructure/database/Connection`; o composition root (`main.go`) controla o lifecycle com `defer conn.Close()`
- Migrations via `ALTER TABLE ADD COLUMN` com tratamento de "duplicate column name" (idempotente)
- Timestamps: `created_at` e `updated_at` preenchidos em `Save` e `Update`; `FindAll` ordena por `domain.SortOrder`
- `FindAll`/`FindByIDs` ignoram (com log) registros que não passam mais na validação do domínio — um dado legado nunca pode impedir o app de abrir

---

## Seed de dados padrão

- **Localização:** `cmd/app/seed.go` — política de produto, **não** no repositório
- Executa apenas quando a tabela está vazia
- Usa os mesmos value objects e `Validate()` do domínio

---

## Backup / Restauração

- Formato: **CSV** com cabeçalho `id,display_name,package_name,version` — a leitura exige esse cabeçalho (aceita o BOM do Excel) e rejeita arquivo sem cabeçalho ou com colunas em outra ordem
- Adapter: `CSVBackupStorage` em `internal/adapters/backup/`
- `ImportReplace`: `ReplaceAll()` transacional (BEGIN → DELETE → INSERT → COMMIT/ROLLBACK)
- `ImportMerge`: adiciona apenas os ausentes por `package_name` e informa quantos foram ignorados; erro que não seja de validação/duplicidade interrompe a importação e devolve o resultado parcial
- Cada entrada é validada como `domain.Package` antes de qualquer INSERT

---

## Segurança

Resumo das regras; o detalhamento está em `.github/instructions/security.instructions.md`.

- **npm:** nome e versão vindos da UI ou do CSV só chegam ao npm via `NewPackageName`/`NewVersion` → `InstallTarget()`. As regras de nome do npm e o regex de versão são a defesa contra flag injection e contra injeção de comando no Windows (`npm.cmd` é executado pelo `cmd.exe`, que interpreta `&`, `|`, `%`); não afrouxe essas validações. Execução sempre com `exec.Cmd` (`Path` absoluto + `Args`), nunca pelo diretório corrente
- **Ambiente do usuário:** o npm herda `os.Environ()` e lê o `~/.npmrc` (podem conter tokens) — nunca exibir na UI nem logar esse conteúdo; nunca desligar a verificação TLS do registry (`--strict-ssl=false`, `NODE_TLS_REJECT_UNAUTHORIZED=0`)
- **SQL:** valores sempre como parâmetro `?`; partes dinâmicas da query só via whitelist (`sortColumn`) ou lista de placeholders
- **CSV importado** é entrada externa: validar cada linha antes de gravar e manter o `ReplaceAll` transacional
- **Empacotamento:** o `.pkg` instala só no domínio do usuário, sem scripts de pré/pós-instalação
- **Dependências:** `govulncheck` roda no CI e não pode acusar vulnerabilidade alcançável; corrija atualizando o módulo ou o patch do `toolchain`
- **CI:** não ampliar `permissions` dos workflows; actions de terceiros fixadas por SHA (com a tag em comentário) e ferramentas com versão fixa (`fyne` v1.7.4, `golangci-lint` v2.15.0); dependência ou action nova exige justificativa na PR

---

## Comandos de build e verificação

```bash
go build ./...          # compila todos os pacotes
go vet ./...            # análise estática
go test -race ./...     # testes
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...   # vulnerabilidades alcançáveis
make                    # menu com todos os alvos (make help)
make fmt                # go fmt ./...
make lint               # golangci-lint (config em .golangci.yml: gosec, cyclop, gocognit, revive...)
make run                # limpa, compila e executa
make build              # compila bin/packages-npm com Version/Build do FyneApp.toml
make package            # fyne package para o SO atual (artefatos em dist/)
make package-macos      # gera o instalador .pkg em dist/ (instala em ~/Applications)
make tidy               # go mod tidy
make clean              # remove bin/ e dist/
```

**Após qualquer alteração de código:** rode `go build ./...`, `go vet ./...`, `go test -race ./...`, `make fmt`, `make lint` e o `govulncheck` — o CI (`ci.yml`) reprova código não formatado, com achados do golangci-lint, com teste falhando ou com vulnerabilidade alcançável.

**Comandos que agentes não devem rodar para verificar:** `make run` abre a janela do app e não termina sozinho; `make run`, `make package` e `make package-macos` executam `clean` e apagam `bin/` e `dist/`. Use-os só quando o usuário pedir.

**Testes:** ficam ao lado do código (`*_test.go`, mesmo pacote), em tabela, e não dependem do Node da máquina:

- domínio: value objects e `Validate()`
- use cases: fakes à mão das portas outbound (struct que embute a interface e implementa só o que o teste usa)
- installer: resolução de versão/caminho com `t.TempDir` e `t.Setenv`; leitura da saída do npm
- persistência: SQLite real em memória (`database.Open(":memory:")`)
- backup: arquivo CSV em `t.TempDir`
- viewmodel: regras de apresentação

Mudança de comportamento vem com teste que falharia sem ela. A UI (`ui/window`, `ui/components`) e a execução real do npm ficam fora dos testes automatizados.

---

## Fluxo de branches, versão e commits

- **Branches:** PRs para `develop` só de `feat/*` ou `fix/*`; PRs para `main` só de `develop` ou `hotfix/*` (validado no `ci.yml`)
- **Versão:** `Version` e `Build` ficam no `FyneApp.toml` (lidos pelo `Makefile` e pelo `fyne package`; injetados em `cmd/app/version.go` via `-ldflags`). A cada release, suba `Version` (semver) e incremente `Build`
- **Release:** a tag `v*.*.*` dispara o `release.yml` (Linux e Windows); o `.pkg` do macOS é gerado manualmente com `make package-macos`
- **Commits:** Conventional Commits com descrição em pt-BR — `tipo(escopo): descrição` (escopos usados: `domain`, `usecases`, `persistence`, `installer`, `backup`, `ui`, `deps`, `release`, `build`, `agents`, `readme`, `review`, `security`)

---

## Instruções do Copilot

Lidas apenas pelo GitHub Copilot (code review e cloud agent); as regras essenciais já estão neste arquivo.

- `.github/instructions/pt-br-review.instructions.md` — idioma, deduplicação e contexto de risco do code review
- `.github/instructions/security.instructions.md` — detalhamento das regras de segurança

---

## Convenções de nomenclatura

| Contexto | Convenção | Exemplo |
| --- | --- | --- |
| Interfaces | Substantivo + sufixo semântico | `PackageRepository`, `BackupStorage` |
| Implementações | Prefixo tecnológico + nome da interface | `SQLitePackageRepository`, `CSVBackupStorage` |
| Construtores | `New` + tipo | `NewPackageService`, `NewMainWindow` |
| Erros de domínio | `Err` + PascalCase | `ErrPackageNotFound`, `ErrDuplicatePackage` |
| Value Objects | Construtores retornam `(T, error)` | `NewPackageName`, `NewVersion` |
| Handlers de UI | `on` + ação | `onAdd`, `onInstall`, `onExport` |
| Helpers privados | camelCase descritivo | `packageFromDTO`, `sortColumn`, `seedDefaultPackages` |

---

## Boas práticas de código

Princípios valem pela consequência que evitam neste código — não por si. Quando dois puxarem para lados opostos (ex.: DRY pede abstração, YAGNI pede esperar), escolha o mais simples que resolve o problema de hoje e registre o motivo na PR.

### Código Limpo

- Nomes que dizem o que a coisa é no domínio (`InstallTarget`, `ImportResult`, `npmErrorSummary`), sem abreviações nem prefixos de tipo; siglas em caixa uniforme (`ID`, `URL`)
- Funções curtas, com um nível de abstração e retorno antecipado para erros — o caminho feliz sem indentação; `cyclop` e `gocognit` no `.golangci.yml` limitam a complexidade
- Constantes nomeadas no lugar de números e textos mágicos (`maxPackageNameLength`, `maxErrorLines`, `utf8BOM`)
- Erros nunca engolidos: propague com contexto (`fmt.Errorf("...: %w", err)`) ou, quando ignorar for a decisão certa, registre em log e explique o motivo no nome ou no doc da função
- Código sem uso é removido, não comentado

### Comentários

- Comente o **porquê**, nunca o **o quê**: decisão não óbvia, restrição externa (ex.: "o shim do asdf depende do binário no `PATH`"), armadilha de plataforma
- Identificador exportado tem doc comment começando pelo próprio nome (`revive`/`godot` cobram); helper privado autoexplicativo não precisa de comentário
- Não repita em comentário o que o nome ou o tipo já dizem, não narre o código linha a linha e não deixe comentário desatualizado — comentário que mente é bug
- Nada de `TODO` sem contexto nem código comentado

### Refatoração

- Refatore em passos pequenos, com os testes verdes antes e depois; refatoração não muda comportamento — se mudar, é outro commit (`fix`/`feat`)
- Regra do escoteiro: ao mexer num trecho, deixe-o um pouco melhor (nome, duplicação, função longa), sem reescrever o que não faz parte da tarefa
- Antes de refatorar código sem teste, escreva o teste que fixa o comportamento atual

### SOLID

- **S** — cada tipo tem um motivo para mudar: o use case orquestra, o adapter traduz tecnologia, o ViewModel formata, o presenter (`MainWindow`) liga UI e use case
- **O** — nova tecnologia entra como novo adapter de uma porta existente, sem alterar o core
- **L** — toda implementação de porta respeita o contrato documentado na interface (ex.: `ReplaceAll` é atômico em qualquer adapter)
- **I** — portas pequenas e focadas (`PackageInstaller`, `PackageChecker`, `BackupStorage`); não adicione métodos que só um consumidor hipotético usaria
- **D** — o core depende de interfaces declaradas em `ports/`; implementações concretas só aparecem no composition root (`cmd/app`)

### Design Patterns

Use o padrão que o projeto já adota antes de introduzir outro:

- **Ports & Adapters** para toda integração externa (npm, SQLite, CSV)
- **Repository** para persistência (`PackageRepository`)
- **Value Object** para regras de formato (`PackageName`, `Version`) — validação no construtor, valor imutável
- **Factory/Constructor** `NewX` com injeção de dependências
- **ViewModel + Presenter** na UI (`PackageVM`, `MainWindow`)
- **Composition Root** em `cmd/app/main.go`

Padrão novo só com problema concreto que os atuais não resolvem.

### DRY

- Uma regra de negócio tem um único lugar: validação em value objects e `Validate()`, criação de `Package` em `packageFromDTO`, laço de instalar/desinstalar em `runOnSelected`, execução do npm em `runGlobal`
- Duplicação de conhecimento é bug; duplicação acidental de duas linhas parecidas pode ficar — abstraia na terceira ocorrência ou quando as cópias precisarem mudar juntas

### KISS

- Prefira stdlib e código explícito a dependências, reflexão ou genéricos sem necessidade
- A solução mais simples que atende ao caso de hoje: struct com campos claros em vez de mapas genéricos, `switch`/`if` legível em vez de indireção

### YAGNI

- Não crie método de porta, parâmetro, flag ou camada "para o futuro": `Restore` e `DeleteAll` foram removidos do `PackageRepository` por falta de uso
- Dependência nova só com uso real e justificativa na PR

---

## O que fazer ao receber uma tarefa

1. **Leia** os arquivos relevantes antes de editar
2. **Identifique a camada correta**: domínio? porta? use case? adapter? UI?
3. **Respeite a direção de dependências** — nunca inverta
4. **Adicione à interface inbound** antes de implementar no service
5. **Crie o adapter** implementando a interface outbound
6. **Wire em `cmd/app/main.go`** — composition root é o único lugar com visão total
7. **Propague `fyne.Do`** em qualquer mutação de UI vinda de goroutine
8. **Escreva ou ajuste o teste** que falharia sem a mudança e aplique as boas práticas acima no código que tocar
9. **Valide com** `go build ./...`, `go vet ./...`, `go test -race ./...`, `make fmt`, `make lint` e `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` antes de encerrar
