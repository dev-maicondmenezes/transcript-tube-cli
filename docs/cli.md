# Manual do `tyot`

`tyot` (*Transcript YOuTube*) extrai metadados, capítulos e transcrição de um
vídeo do YouTube e escreve **Markdown puro** em `stdout` ou em um arquivo.

O binário é um adaptador fino do
[`transcript-tube-core`](https://github.com/dev-maicondmenezes/transcript-tube-core):
a extração, os fallbacks de transcrição e a formatação vivem no core. O CLI
cuida de argumentos, configuração, validação de ambiente, escrita de arquivo e
distribuição.

## Índice

- [Instalação](#instalação)
- [Uso rápido](#uso-rápido)
- [Formatos de URL e ID](#formatos-de-url-e-id)
- [Flags](#flags)
- [Variáveis de ambiente](#variáveis-de-ambiente)
- [Arquivo de configuração](#arquivo-de-configuração)
- [A API key `YOUTUBE_API_KEY`](#a-api-key-youtube_api_key)
- [Saída Markdown](#saída-markdown)
- [Códigos de saída](#códigos-de-saída)
- [Completions e man page](#completions-e-man-page)
- [Troubleshooting](#troubleshooting)
- [Segurança](#segurança)

## Instalação

### GitHub Releases

```bash
gh release download --repo dev-maicondmenezes/transcript-tube-cli \
  --pattern 'tyot_*_linux_amd64.tar.gz'
tar -xzf tyot_*_linux_amd64.tar.gz
sudo install -m 0755 tyot /usr/local/bin/tyot
```

### Debian/Ubuntu — `.deb`

```bash
gh release download --repo dev-maicondmenezes/transcript-tube-cli \
  --pattern 'tyot_*_amd64.deb'
sudo apt install ./tyot_*.deb
man tyot
```

### Debian/Ubuntu — repositório APT

```bash
curl -fsSL https://dev-maicondmenezes.github.io/transcript-tube-cli/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/tyot.gpg
echo "deb [signed-by=/usr/share/keyrings/tyot.gpg] https://dev-maicondmenezes.github.io/transcript-tube-cli stable main" \
  | sudo tee /etc/apt/sources.list.d/tyot.list
sudo apt update
sudo apt install tyot
```

### macOS — Homebrew

```bash
brew tap dev-maicondmenezes/tap
brew install tyot
man tyot
```

### Go

```bash
go install github.com/dev-maicondmenezes/transcript-tube-cli/cmd/tyot@latest
```

## Uso rápido

```bash
# ID cru
tyot dQw4w9WgXcQ

# URL curta
tyot "https://youtu.be/dQw4w9WgXcQ"

# URL completa, salvando em arquivo (cria diretórios)
tyot "https://www.youtube.com/watch?v=dQw4w9WgXcQ" -o out/rick.md

# Idioma preferido e timeout
tyot dQw4w9WgXcQ --lang en --timeout 90s

# Markdown parcial quando a transcrição não estiver disponível
tyot dQw4w9WgXcQ --allow-partial

# URL via stdin (quando stdin não é um TTY)
echo "https://youtu.be/dQw4w9WgXcQ" | tyot

# Ajuda e versão funcionam sem YOUTUBE_API_KEY
tyot --help
tyot --version
```

Sem `--output`, o Markdown vai para `stdout`. O `stderr` recebe apenas logs de
progresso, avisos e erros; use `--quiet`/`-q` para suprimir o progresso.

Se nenhum argumento for informado e `stdin` for um TTY, o `tyot` mostra a ajuda
e sai com código `1`. Se `stdin` **não** for um TTY, a primeira linha não vazia
é lida como URL/ID.

## Formatos de URL e ID

O `tyot` aceita:

| Formato | Exemplo |
| :--- | :--- |
| ID cru (11 caracteres `[A-Za-z0-9_-]`) | `dQw4w9WgXcQ` |
| `watch?v=` | `https://www.youtube.com/watch?v=dQw4w9WgXcQ` |
| URL curta | `https://youtu.be/dQw4w9WgXcQ` |
| Embed | `https://www.youtube.com/embed/dQw4w9WgXcQ` |
| Shorts | `https://www.youtube.com/shorts/dQw4w9WgXcQ` |
| Live | `https://www.youtube.com/live/dQw4w9WgXcQ` |
| Formato antigo `/v/` | `https://www.youtube.com/v/dQw4w9WgXcQ` |
| Sem esquema | `youtube.com/watch?v=dQw4w9WgXcQ` |
| Subdomínio `*.youtube.com` | `https://music.youtube.com/watch?v=dQw4w9WgXcQ` |

São rejeitados com código `1`: URLs de playlist/canal, hosts desconhecidos,
IDs com tamanho diferente de 11 e esquemas que não sejam `http`/`https`.

## Flags

| Flag | Curto | Default | Descrição |
| :--- | :--- | :--- | :--- |
| `--help` | `-h` | — | Mostra a ajuda resumida. |
| `--version` | `-V` | — | Mostra versão, commit, data de build e versão do Go. |
| `--config` | — | `$XDG_CONFIG_HOME/tyot/config.yaml` | Arquivo de configuração YAML. |
| `--lang` | `-l` | `pt` | Idioma preferido da transcrição. |
| `--output` | `-o` | `stdout` | Arquivo Markdown de saída. `-` significa `stdout`. |
| `--timeout` | — | `60s` | Timeout total da extração (formato Go: `30s`, `2m`, `1m30s`). |
| `--allow-partial` | — | `false` | Gera Markdown parcial quando a transcrição falha. |
| `--no-chapters` | — | `false` | Remove os headings de capítulos do Markdown. |
| `--quiet` | `-q` | `false` | Suprime logs de progresso em `stderr`. |

A precedência é **flag > variável de ambiente > arquivo > default**. Apenas
flags informadas explicitamente sobrepõem as camadas inferiores.

`--output` e `--quiet` existem somente como flags; não há equivalente no arquivo
de configuração nem em variáveis de ambiente.

### Saída atômica

Com `--output`, o `tyot`:

1. cria os diretórios necessários com permissão `0755`;
2. escreve em um arquivo temporário no mesmo diretório;
3. sincroniza o conteúdo para o disco;
4. renomeia o temporário para o destino.

O arquivo final tem permissão `0644`. Uma falha de diretório/permissão/disco
resulta em código `3` e não deixa Markdown parcial no destino.

## Variáveis de ambiente

| Variável | Precedência | Uso |
| :--- | :--- | :--- |
| `YOUTUBE_API_KEY` | única fonte | API key da YouTube Data API v3. Obrigatória para extração. |
| `TYOT_LANG` | 2 | Equivale a `--lang`. |
| `TYOT_TIMEOUT` | 2 | Equivale a `--timeout` (formato Go). |
| `TYOT_CONFIG` | 2 | Caminho alternativo do arquivo de configuração. |
| `NO_COLOR` | — | Desabilita cores (reservado para versões futuras). |

`TYOT_TIMEOUT` inválido resulta em código `4`.

## Arquivo de configuração

Localização padrão (via `os.UserConfigDir()`):

| Sistema | Caminho |
| :--- | :--- |
| Linux | `$XDG_CONFIG_HOME/tyot/config.yaml` (normalmente `~/.config/tyot/config.yaml`) |
| macOS | `~/Library/Application Support/tyot/config.yaml` |
| Windows | `%AppData%\tyot\config.yaml` |

O caminho pode ser trocado por `TYOT_CONFIG` ou `--config`. Um arquivo ausente
não é erro.

```yaml
# Exemplo completo: docs/examples/config.yaml
default_lang: "pt"
default_timeout: "60s"
allow_partial: false
no_chapters: false
```

Regras:

- Campos desconhecidos são **erro** (fail-fast, código `4`).
- O arquivo **não pode** conter API key. Campos como `youtube_api_key`,
  `api_key`, `yt_api_key`, `youtube-key` ou `YouTubeApiKey` são rejeitados com
  código `4` e uma mensagem que instrui a usar `YOUTUBE_API_KEY`.
- YAML inválido ou `default_timeout` inválido resultam em código `4`.

## A API key `YOUTUBE_API_KEY`

O `tyot` usa a YouTube Data API v3 para metadados. A chave é lida
**exclusivamente** da variável de ambiente `YOUTUBE_API_KEY`.

```bash
export YOUTUBE_API_KEY="sua-chave"
```

Obtenha uma chave em <https://console.cloud.google.com/apis/credentials>:

1. Crie ou selecione um projeto no Google Cloud.
2. Ative a **YouTube Data API v3**.
3. Crie uma credencial do tipo **Chave de API**.
4. Restrinja a chave à YouTube Data API v3.

### Regra de validação

Toda invocação que **não seja do tipo ajuda** valida `YOUTUBE_API_KEY` antes de
qualquer outra lógica — inclusive antes de reclamar de URL inválida ou flag
desconhecida.

São isentos da validação:

- `tyot --help` / `tyot -h`
- `tyot --version` / `tyot -V`
- `tyot help`
- `tyot completion <shell>`

Não são isentos (a env é validada primeiro):

- `tyot <URL-ou-ID>`
- `tyot` sem argumentos
- `tyot -o arquivo.md`
- qualquer combinação de flags que leve à extração
- argumentos inválidos

Uma chave ausente, vazia ou composta apenas de espaços resulta em código `2` e
na mensagem:

```text
erro: variável de ambiente YOUTUBE_API_KEY não está definida.

O tyot precisa da chave da YouTube Data API v3 para extrair metadados.
Defina-a antes de executar:

  export YOUTUBE_API_KEY="sua-chave"
  tyot <URL-ou-ID>

Como obter uma chave:
  1. Acesse https://console.cloud.google.com/apis/credentials
  2. Ative a YouTube Data API v3 no projeto.
  3. Crie uma credencial do tipo "Chave de API".
  4. Restrinja a chave à YouTube Data API v3.

A ajuda (tyot --help) e a versão (tyot --version) funcionam sem a chave.

Código de saída: 2
```

O valor da chave **nunca** é impresso, nem em logs de progresso, nem em erros.

## Saída Markdown

O Markdown segue o template do core:

```markdown
# Nunca Vou Te Deixar

**Canal:** Rick Astley
**URL:** https://www.youtube.com/watch?v=dQw4w9WgXcQ
**Publicado em:** 25/10/2009 07:00 UTC
**Duração:** 03:33
**Visualizações:** 1000000
**Likes:** 12345

---

## Descrição

Descrição do vídeo.

---

## Transcrição

### [00:00] Introdução

[00:00] Primeiro trecho
[00:05] Segundo trecho
```

- `--lang` escolhe o idioma preferido; se indisponível, o core usa o fallback
  definido por ele.
- Capítulos oficiais/automáticos aparecem como headings `###` antes dos
  segmentos correspondentes.
- `--no-chapters` remove os headings, mantendo os timestamps da transcrição.
- `--allow-partial` mantém o Markdown com metadados e descrição quando a
  transcrição falha, emitindo um aviso em `stderr` e saindo com código `0`.

Nenhum log é escrito em `stdout`: ele contém apenas Markdown.

## Códigos de saída

| Código | Significado |
| :--- | :--- |
| `0` | Sucesso. |
| `1` | Erro de argumento: URL/ID inválido, argumento vazio, flag inválida ou uso incorreto. |
| `2` | Erro de extração: `YOUTUBE_API_KEY` ausente, rede, API, vídeo não encontrado, timeout, cancelamento ou transcrição indisponível. |
| `3` | Erro ao criar diretório ou escrever o arquivo de saída. |
| `4` | Erro de configuração: YAML inválido, campo desconhecido, timeout inválido ou tentativa de API key no arquivo. |

## Completions e man page

O `.deb` instala a man page e as completions em:

| Shell | Caminho |
| :--- | :--- |
| man | `/usr/share/man/man1/tyot.1` |
| bash | `/usr/share/bash-completion/completions/tyot` |
| zsh | `/usr/share/zsh/vendor-completions/_tyot` |
| fish | `/usr/share/fish/vendor_completions.d/tyot.fish` |

Instalação manual:

```bash
# bash
tyot completion bash | sudo tee /etc/bash_completion.d/tyot >/dev/null

# zsh (com oh-my-zsh ou diretório no fpath)
tyot completion zsh > "${fpath[1]}/_tyot"

# fish
tyot completion fish > ~/.config/fish/completions/tyot.fish

# man page a partir do arquivo do repositório
man -l docs/man/tyot.1
```

O comando `tyot completion powershell` também está disponível.

## Troubleshooting

### `YOUTUBE_API_KEY` não está definida

Exporte a variável **no mesmo shell** que executa o `tyot`:

```bash
export YOUTUBE_API_KEY="sua-chave"
tyot dQw4w9WgXcQ
```

Em `cron`, systemd, IDEs ou apps gráficos, a variável precisa ser definida no
ambiente do processo — não basta estar no `.bashrc` do terminal.

### `403` / `quotaExceeded` / `keyInvalid`

- Confirme que a YouTube Data API v3 está ativada no projeto da chave.
- Verifique as restrições da chave (API e IP/referrer).
- A cota padrão é diária; aguarde a renovação ou use outro projeto.

### Transcrição indisponível

Vídeos sem legendas, com legendas desabilitadas, privados ou com restrição de
idade podem não ter transcrição. Use `--allow-partial` para obter pelo menos
metadados e descrição:

```bash
tyot dQw4w9WgXcQ --allow-partial
```

### Timeout

Aumente o limite:

```bash
tyot dQw4w9WgXcQ --timeout 2m
```

### Arquivo de configuração rejeitado

O `tyot` falha em campos desconhecidos e em qualquer campo de API key. Remova o
campo e use `YOUTUBE_API_KEY`:

```bash
tyot dQw4w9WgXcQ --config /caminho/config.yaml
```

### `stdout` com conteúdo inesperado

Use `--quiet` e redirecione o `stderr` para inspecionar o que é log:

```bash
tyot dQw4w9WgXcQ --quiet > video.md
```

## Segurança

- A API key existe apenas em `YOUTUBE_API_KEY`, nunca em flags, arquivos
  versionados, `manifest.json`, `localStorage` ou logs.
- O arquivo de configuração é validado para rejeitar campos de API key.
- O `tyot` não imprime o valor da chave em nenhuma mensagem.
- Prefira `export YOUTUBE_API_KEY="$(cat ~/.config/tyot/api-key)"` em vez de
  colocar a chave diretamente no histórico do shell.
- Em CI, armazene a chave como secret e injete apenas no passo que executa o
  `tyot`.

## Ver também

- [`README.md`](../README.md) — instalação e visão geral.
- [`docs/man/tyot.1`](man/tyot.1) — man page gerada.
- [`RELEASING.md`](../RELEASING.md) — processo de release.
