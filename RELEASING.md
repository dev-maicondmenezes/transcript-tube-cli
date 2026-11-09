# Processo de release

Este repositório publica uma versão quando um PR é mesclado em `main`. Não crie
tags nem releases manualmente: o workflow faz tudo.

## Pré-requisitos

- `ci/test` e `ci/release-dry-run` verdes no PR.
- `VERSION` atualizado (SemVer sem `v`).
- `CORE_VERSION` apontando para uma tag **já publicada** do
  `transcript-tube-core` (por exemplo `v0.2.0`).
- `CHANGELOG.md` atualizado.
- `go.mod` sem `replace` local.
- `go.sum` sincronizado com o `CORE_VERSION`.
- Secret `HOMEBREW_TAP_GITHUB_TOKEN` configurado (write no repositório
  `homebrew-tap`).
- Secret opcional `APT_GPG_PRIVATE_KEY` para assinar o repositório APT.

## Ordem obrigatória

O CLI depende do core como módulo versionado. A ordem é:

1. Abra e mescle o PR do `transcript-tube-core`.
2. Aguarde o workflow do core criar a tag `vX.Y.Z`.
3. Atualize `CORE_VERSION`/`go.sum` no PR do CLI, se necessário.
4. Aguarde `ci/test` e `ci/release-dry-run` no PR do CLI.
5. Mescle o PR do CLI.

O workflow do CLI espera a tag do core por até 30 minutos. Se ela não existir,
o release falha com mensagem explícita em vez de publicar um binário quebrado.

## Atualizar o core no CLI

```bash
# no repositório transcript-tube-cli
go get github.com/dev-maicondmenezes/transcript-tube-core@v0.2.0
go mod tidy
go test ./...
```

Atualize `CORE_VERSION` com a mesma tag. O `go.sum` versionado precisa conter o
checksum do módulo do core. O workflow de release executa `go mod tidy` e falha
se `go.mod`/`go.sum` ficarem desatualizados — isso evita publicar com uma
dependência inconsistente.

## Fluxo do workflow

O merge em `main` dispara `.github/workflows/release.yml`:

1. Lê `VERSION` e `CORE_VERSION`.
2. Verifica se a tag `vX.Y.Z` já existe.
3. Se a tag não existir, aguarda a tag do core.
4. Executa `go mod tidy` e valida que `go.mod`/`go.sum` não mudaram.
5. Cria a tag anotada `vX.Y.Z` e a envia ao GitHub.
6. Executa o GoReleaser:
   - binários Linux/macOS/Windows (`amd64`, `arm64`, `armv7`);
   - arquivos `.tar.gz`/`.zip` com LICENSE, README, CHANGELOG, man page e
     completions;
   - pacotes `.deb` (`amd64`, `arm64`, `armv7`);
   - `checksums.txt`;
   - GitHub Release.
7. A publicação da GitHub Release dispara `publish/apt.yml`, que publica os
   `.deb` no branch `gh-pages` como repositório APT:
   - preserva versões antigas em `pool/main`;
   - regenera `Packages`, `Packages.gz` e `Release`;
   - assina com `Release.gpg`/`InRelease` quando `APT_GPG_PRIVATE_KEY` existir;
   - publica `key.gpg` para o `signed-by` do APT.
8. A mesma publicação dispara `publish/brew.yml`, que baixa os archives macOS,
   calcula seus checksums e atualiza `Formula/tyot.rb` no `homebrew-tap`.

## Secrets

| Secret | Obrigatório | Uso |
| :--- | :--- | :--- |
| `HOMEBREW_TAP_GITHUB_TOKEN` | sim, para Homebrew | PAT com `contents: write` no `homebrew-tap`. |
| `APT_GPG_PRIVATE_KEY` | não | Chave GPG privada (ASCII armored) para assinar o APT. |

O `GITHUB_TOKEN` padrão do workflow publica a GitHub Release e o `gh-pages`.

Configure com:

```bash
gh secret set HOMEBREW_TAP_GITHUB_TOKEN \
  --repo dev-maicondmenezes/transcript-tube-cli

gh secret set APT_GPG_PRIVATE_KEY \
  --repo dev-maicondmenezes/transcript-tube-cli \
  < private-key.asc
```

Prefira um token fine-grained restrito ao repositório `homebrew-tap`, com
permissão de conteúdo de leitura/escrita. Tokens clássicos e OAuth do `gh`
funcionam, mas devem ser rotacionados.

## Idempotência

- Se a tag `vX.Y.Z` já existir, o workflow não recria a release; o job APT ainda
  pode sincronizar os `.deb` existentes.
- Para republicar uma versão, incremente `VERSION` em um novo PR. Nunca reutilize
  uma tag publicada.
- O GoReleaser usa `mode: replace`; uma re-execução manual consciente substitui
  os assets da release da mesma tag.

## Recuperação

Se o release falhar depois de criar a tag:

```bash
# reexecutar o workflow no commit da tag
gh workflow run release.yml \
  --repo dev-maicondmenezes/transcript-tube-cli \
  --ref main

# ou recriar apenas a release a partir da tag
git fetch --tags
git checkout v0.1.0
goreleaser release --clean
```

Se for necessário remover uma release/tag publicada por engano, faça isso de
forma explícita e documente no PR seguinte; o próximo merge não reutiliza a
mesma versão.

## Verificação pós-release

```bash
gh release view v0.1.0 --repo dev-maicondmenezes/transcript-tube-cli
gh release download v0.1.0 --repo dev-maicondmenezes/transcript-tube-cli
gh run list --repo dev-maicondmenezes/transcript-tube-cli --limit 5
```

No APT:

```bash
curl -fsSL https://dev-maicondmenezes.github.io/transcript-tube-cli/dists/stable/Release
```

No Homebrew:

```bash
brew tap dev-maicondmenezes/tap
brew install tyot
tyot --version
```
