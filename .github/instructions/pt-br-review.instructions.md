---
applyTo: '**'
excludeAgent: 'cloud-agent'
description: 'Idioma, deduplicação e contexto de risco do review — packages-npm'
---

# Idioma

- Escreva **todo** comentário de review em **português do Brasil**. Termos técnicos (use case,
  port, adapter, value object, widget, goroutine, shim) podem ficar em inglês.

# Achados duplicados e threads existentes

- Antes de comentar, confira as threads da PR sobre o mesmo trecho. Se o autor já respondeu
  ou corrigiu, **não repita** o achado.
- **Uma causa raiz = um comentário**, mesmo que ela apareça em vários pontos.
- Em re-review, priorize: (1) mudanças novas; (2) riscos ainda não cobertos; (3) regressões.

# Contexto de risco (app desktop Go + Fyne)

- App desktop local de um único usuário. As entradas externas são a UI, o CSV de backup
  importado, a saída do `npm` e o ambiente (`HOME`, `PATH`, `ASDF_DATA_DIR` etc.); os dados
  ficam em um SQLite local.
- **Direção das dependências é `blocking`** (regras do `AGENTS.md`): `core/domain` só importa
  stdlib; `usecases` só `domain` e `ports`; `ui/` só `ports/inbound`, `domain` e pacotes
  próprios; `exec.Cmd`/`exec.Command` só em `internal/adapters/installer`; `sql.DB` só em
  `internal/adapters/persistence`; `cmd/app` é o único composition root. Novo adapter sem
  `var _ outbound.X = (*T)(nil)` também é achado.
- **Thread safety do Fyne é `blocking`**: qualquer mutação de widget, `dialog.Show*`,
  `m.reload()` ou `m.setStatus()` chamada de goroutine fora de `fyne.Do(func(){...})` é data
  race / crash.
- **Execução do npm**: o app é aberto pelo Finder/menu sem o `PATH` do shell. Toda resolução
  de binário (npm, node, asdf, gerenciadores de versão) deve funcionar sem depender do `PATH`,
  checar `isExecutable` antes de escolher um candidato, comparar versões numericamente
  (24.14 > 24.9) e comparar caminhos já normalizados (absolutos e limpos). O npm é executado
  via `exec.Cmd` com `Args`; nome e versão vindos da UI ou do CSV só chegam ao npm depois de
  `NewPackageName`/`NewVersion` (ex.: valor começando com `-` viraria flag do npm).
- **Cross-platform**: o código roda em macOS, Linux e Windows. Aponte caminhos montados com
  `/` fixo em vez de `filepath`, `:` em vez de `os.PathListSeparator` e ausência do `npm.cmd`
  no Windows. Builds de Linux/Windows saem do CI (`release.yml`); o `.pkg` do macOS é gerado
  manualmente com `make package-macos`.
- **Backup/restauração**: `ImportReplace` usa `ReplaceAll` transacional e cada linha do CSV
  vira `domain.Package` validado antes de qualquer INSERT. Quebrar a transação ou inserir sem
  validar é `blocking`; criar `Package` fora de `packageFromDTO` é duplicação a apontar.
- O projeto ainda não tem suíte de testes: ausência de teste não é `blocking`, mas sugira
  teste quando a mudança for lógica pura (domínio, resolução de versão/caminho).
- Seja específico: diga qual camada/pacote (`core/domain`, `core/usecases`,
  `adapters/installer`, `adapters/persistence`, `adapters/backup`, `ui/window`, `cmd/app`)
  e qual regra ou contrato é afetado.
