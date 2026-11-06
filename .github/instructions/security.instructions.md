---
applyTo: '**'
description: 'Segurança — app desktop Go + Fyne (execução do npm, SQLite, backup CSV, empacotamento e CI)'
---

# Segurança — packages-npm

## Execução do npm (`internal/adapters/installer`)

- O alvo do npm vem de `domain.Package.InstallTarget()` (`nome@versão`), montado só a partir de
  `PackageName`/`Version` já validados. Valor vindo da UI ou do CSV nunca chega ao npm sem
  passar por `NewPackageName`/`NewVersion`; valor começando com `-` seria lido como flag do npm.
- O npm é executado via `exec.Cmd` com `Path` absoluto e `Args` separados. Binários
  (npm, node, asdf) são escolhidos por caminho absoluto e limpo (`absolutePath`) e checados com
  `isExecutable`; a busca usa o `PATH` do processo e, depois, locais fixos de instalação e
  de gerenciadores de versão no diretório do usuário — nunca o diretório corrente.
- O processo do npm herda `os.Environ()` (pode conter `NPM_TOKEN` e afins) e lê o `~/.npmrc`
  do usuário: nunca exibir na UI nem logar o ambiente ou o conteúdo do `.npmrc`.
- Nunca enfraquecer a verificação do registry: sem `--strict-ssl=false`,
  `NODE_TLS_REJECT_UNAUTHORIZED=0` ou registry em `http://` nos comandos do npm.

## Persistência (`internal/adapters/persistence`)

- Valores entram no SQL sempre como parâmetro `?`. Partes dinâmicas da query são só
  identificadores de whitelist (`sortColumn`) e a lista de placeholders `?` — nunca
  `fmt.Sprintf` com valor vindo da UI, do CSV ou do npm.
- O banco fica em `os.UserConfigDir()/packages-npm/packages.db`.

## Backup CSV (`internal/adapters/backup`)

- O arquivo importado é entrada externa: lido com `encoding/csv`, cada linha vira
  `domain.Package` validado antes de qualquer INSERT, e `ImportReplace` roda em transação
  (`ReplaceAll`) — falha em qualquer linha não pode deixar a tabela parcialmente substituída.

## Empacotamento e release

- `make package-macos` gera um `.pkg` não assinado que instala só no domínio do usuário
  (`~/Applications`, `enable_currentUserHome`), com bundle não relocável e sem scripts de
  pré/pós-instalação. Mudar para o domínio do sistema ou adicionar scripts exige justificativa
  na PR.
- Linux e Windows são gerados pelo `release.yml` a partir de tag `v*.*.*`.

## Esteira e dependências

- `ci.yml` roda com `contents: read`; `release.yml` com `contents: write` (necessário para
  publicar a release) usando o `GITHUB_TOKEN`. Não ampliar `permissions`.
- Actions de terceiros (`msys2/setup-msys2`, `softprops/action-gh-release`,
  `golangci/golangci-lint-action`) estão fixadas por tag de major; action nova ou troca de
  versão deve ser fixada por SHA de commit.
- Dependências Go ficam em `go.mod`/`go.sum` versionados; dependência nova exige
  justificativa na PR. Ferramenta instalada no CI que gera artefato (`fyne`) não deve ganhar
  novos usos com `@latest`.
