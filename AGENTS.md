# AGENTS.md

Instruções para agentes que trabalham no repositório `transcript-tube-cli`.

## Stack

- Go 1.23.6+.
- `cmd/tyot/` — binário de usuário `tyot`.
- `cmd/gen-man/` — gera man page e completions (não é binário de usuário).
- `internal/cli/`, `internal/config/`, `internal/url/` — adaptador de linha de
  comando sobre o `transcript-tube-core`.
- CI/CD: GitHub Actions, GoReleaser v2, `nfpm` (`.deb`), Homebrew e APT em
  GitHub Pages.
- Node.js não é usado neste repositório.

## Regras de arquitetura

- O CLI é um adaptador fino: extração e formatação vivem no
  `github.com/dev-maicondmenezes/transcript-tube-core`.
- `stdout` é reservado para Markdown. Logs, avisos e erros vão para `stderr`.
- Toda invocação que não seja `help`/`version`/`completion` valida
  `YOUTUBE_API_KEY` antes de qualquer outra lógica.
- A API key é lida **exclusivamente** de `YOUTUBE_API_KEY`. Não existe
  `--api-key`, campo no YAML ou fallback em `.env` carregado automaticamente.
- O arquivo de configuração rejeita campos desconhecidos e qualquer chave
  relacionada a API key.
- `go.mod` versionado **não** contém `replace`. O `go.work` local é ignorado
  pelo Git.
- `cmd/tyot/main.go` apenas chama `cli.Run`; não coloque regra de negócio ali.
- Toda flag precisa de help curto e aparição na man page gerada.
- O binário se chama `tyot`; o pacote principal fica em `cmd/tyot/`.

## Anti-padrões

- Escrever log em `stdout`.
- Carregar `.env` automaticamente.
- Aceitar API key em arquivo, flag, `localStorage` ou log.
- Adicionar `replace` local no `go.mod` da `main`.
- Editar `docs/man/` ou `docs/completions/` à mão (rode `make man`).
- Fazer scraping do YouTube no CLI; isso pertence ao core.
- Publicar tag/release manualmente fora do fluxo de merge.
- Reutilizar uma tag ou versão já publicada.

## Comandos de verificação

```bash
go build ./...
go vet ./...
gofmt -l .
go test ./...
go test -race ./...
make check
make release-dry-run
```

Testes padrão são offline e usam fakes. Testes com a API real exigem
`YOUTUBE_API_KEY` e não fazem parte de `go test ./...`:

```bash
source .env
go run ./cmd/tyot dQw4w9WgXcQ --quiet
```

Validação da man page e do pacote:

```bash
make man
man --warnings -l docs/man/tyot.1
goreleaser release --snapshot --clean
dpkg-deb --info dist/tyot_*_amd64.deb
```

## Política de testes

- Table-driven por padrão.
- Fakes para extração e rede; a suíte padrão nunca acessa a internet.
- Cubra os códigos de saída `0`–`4` e a precedência flag > env > file > default.
- `cmd/gen-man` deve permanecer determinístico (sem timestamp variável).
- Man page e completions versionadas devem bater com `make man`.

## Release

- `VERSION` contém a versão do CLI (SemVer sem `v`).
- `CORE_VERSION` contém a tag exata do core (por exemplo `v0.2.0`).
- O merge de um PR em `main` dispara `.github/workflows/release.yml`, que:
  - espera a tag do core existir;
  - valida `go.mod`/`go.sum`;
  - cria a tag `vX.Y.Z` e publica a GitHub Release com GoReleaser;
  - publica os `.deb` no repositório APT (`gh-pages`);
  - atualiza a formula no `homebrew-tap` (se o secret existir).
- Nunca crie tags antes do merge, exceto em recuperação documentada.

Consulte [`README.md`](README.md), [`docs/cli.md`](docs/cli.md) e
[`RELEASING.md`](RELEASING.md).
