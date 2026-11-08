# tyot

`tyot` (*Transcript YOuTube*) extrai metadados, capítulos e a transcrição
completa de um vídeo do YouTube e os formata como Markdown estruturado.

O binário é um adaptador fino do
[`transcript-tube-core`](https://github.com/dev-maicondmenezes/transcript-tube-core):
toda a extração e formatação vivem no core; o CLI cuida de argumentos,
configuração, validação de ambiente, escrita de arquivo e distribuição.

## Instalação

### GitHub Releases (Linux, macOS e Windows)

```bash
gh release download --repo dev-maicondmenezes/transcript-tube-cli \
  --pattern 'tyot_*_linux_amd64.tar.gz'
tar -xzf tyot_*_linux_amd64.tar.gz
sudo install -m 0755 tyot /usr/local/bin/tyot
```

### Debian/Ubuntu — pacote `.deb`

```bash
gh release download --repo dev-maicondmenezes/transcript-tube-cli \
  --pattern 'tyot_*_amd64.deb'
sudo apt install ./tyot_*.deb
man tyot
```

### Debian/Ubuntu — repositório APT (GitHub Pages)

Assinado com a chave do projeto:

```bash
curl -fsSL https://dev-maicondmenezes.github.io/transcript-tube-cli/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/tyot.gpg
echo "deb [signed-by=/usr/share/keyrings/tyot.gpg] https://dev-maicondmenezes.github.io/transcript-tube-cli stable main" \
  | sudo tee /etc/apt/sources.list.d/tyot.list
sudo apt update
sudo apt install tyot
```

> O repositório APT é publicado pelo workflow `release.yml` no branch
> `gh-pages` a partir dos `.deb` anexados à GitHub Release.

### macOS — Homebrew

```bash
brew tap dev-maicondmenezes/tap
brew install tyot
man tyot
```

### Go install

```bash
go install github.com/dev-maicondmenezes/transcript-tube-cli/cmd/tyot@latest
```

O binário é instalado como `tyot` porque o pacote principal fica em
`cmd/tyot/`.

## Requisito: `YOUTUBE_API_KEY`

O `tyot` usa a YouTube Data API v3 para metadados. A API key é lida
**exclusivamente** da variável de ambiente `YOUTUBE_API_KEY`:

```bash
export YOUTUBE_API_KEY="sua-chave"
```

Obtenha uma chave em <https://console.cloud.google.com/apis/credentials>:

1. Ative a YouTube Data API v3 no projeto.
2. Crie uma credencial do tipo "Chave de API".
3. Restrinja a chave à YouTube Data API v3.

Não existe `--api-key` e o arquivo de configuração **não pode** conter a chave.
Se um campo como `youtube_api_key` for encontrado, o CLI falha com exit code 4
e instrui a usar a variável de ambiente.

`tyot --help`, `tyot --version` e `tyot completion` funcionam sem a chave.
Todas as outras invocações validam `YOUTUBE_API_KEY` antes de qualquer outra
lógica — inclusive erros de URL ou flags desconhecidas.

## Uso

```bash
# ID cru
tyot dQw4w9WgXcQ

# URL curta
tyot "https://youtu.be/dQw4w9WgXcQ"

# URL completa, salvando em arquivo (cria diretórios)
tyot "https://www.youtube.com/watch?v=dQw4w9WgXcQ" -o out/rick.md

# Idioma preferido e timeout
tyot dQw4w9WgXcQ --lang en --timeout 90s

# Markdown parcial se a transcrição falhar
tyot dQw4w9WgXcQ --allow-partial

# Sem capítulos no Markdown
tyot dQw4w9WgXcQ --no-chapters

# URL via stdin (quando não é TTY)
echo "https://youtu.be/dQw4w9WgXcQ" | tyot
```

O `stdout` contém **apenas Markdown**. Logs de progresso e avisos vão para
`stderr`; use `--quiet`/`-q` para suprimir o progresso.

### Flags

| Flag | Curto | Default | Descrição |
| :--- | :--- | :--- | :--- |
| `--help` | `-h` | — | Mostra ajuda resumida. |
| `--version` | `-V` | — | Mostra versão, commit e data de build. |
| `--config` | — | `$XDG_CONFIG_HOME/tyot/config.yaml` | Arquivo de configuração. |
| `--lang` | `-l` | `pt` | Idioma preferido da transcrição. |
| `--output` | `-o` | `stdout` | Arquivo Markdown de saída. `-` significa stdout. |
| `--timeout` | — | `60s` | Timeout total da extração. |
| `--allow-partial` | — | `false` | Gera Markdown parcial se a transcrição falhar. |
| `--no-chapters` | — | `false` | Remove headings de capítulos do Markdown. |
| `--quiet` | `-q` | `false` | Suprime logs de progresso em `stderr`. |

### Variáveis de ambiente

| Variável | Precedência | Uso |
| :--- | :--- | :--- |
| `YOUTUBE_API_KEY` | única fonte | API key da YouTube Data API v3. Obrigatória para extração. |
| `TYOT_LANG` | 2 | Equivale a `--lang`. |
| `TYOT_TIMEOUT` | 2 | Equivale a `--timeout`. |
| `TYOT_CONFIG` | 2 | Caminho alternativo do arquivo de configuração. |
| `NO_COLOR` | — | Desabilita cores (reservado para o futuro). |

### Arquivo de configuração

```yaml
# $XDG_CONFIG_HOME/tyot/config.yaml
default_lang: "pt"
default_timeout: "60s"
allow_partial: false
no_chapters: false
```

Precedência: **flag > env > arquivo > default**. Arquivo ausente não é erro.
Campos desconhecidos e campos de API key são rejeitados com exit code 4.
Exemplo completo em [`docs/examples/config.yaml`](docs/examples/config.yaml).

### Códigos de saída

| Código | Significado |
| :--- | :--- |
| `0` | Sucesso. |
| `1` | Erro de argumento (URL/ID/flag inválidos). |
| `2` | Erro de extração, API key ausente, rede, API, timeout ou transcrição. |
| `3` | Erro ao criar diretório/arquivo de saída. |
| `4` | Arquivo de configuração inválido (inclui tentativa de API key no arquivo). |

## Documentação

- [`docs/cli.md`](docs/cli.md) — manual completo: flags, formatos de URL,
  configuração, API key, exit codes, troubleshooting.
- [`docs/man/tyot.1`](docs/man/tyot.1) — man page gerada.
- [`docs/completions/`](docs/completions/) — completions para bash, zsh e fish
  (`tyot completion powershell` gera a versão para PowerShell).
- [`CHANGELOG.md`](CHANGELOG.md) — histórico de versões.
- [`RELEASING.md`](RELEASING.md) — fluxo de release com PR, tag, GoReleaser,
  APT e Homebrew.

## Desenvolvimento

Requisitos: Go 1.23.6+, GoReleaser v2 (para `release-dry-run`), `dpkg-deb`,
`man` e `git-cliff` (opcional).

```bash
make setup
make check          # gofmt, vet, build, test e man page atualizada
make test-race
make man            # regenera docs/man e docs/completions
make release-dry-run
```

Durante o desenvolvimento, o CLI usa um `go.work` local apontando para o
checkout irmão `../transcript-tube-core` (o arquivo é ignorado pelo Git). O
`go.mod` versionado **não** contém `replace`; o workflow de release fixa o core
na tag publicada.

Testes padrão são offline e usam fakes. Testes com a API real exigem
`YOUTUBE_API_KEY`:

```bash
source .env
tyot dQw4w9WgXcQ > /tmp/video.md
```

## Licença

MIT — veja [`LICENSE`](LICENSE).
