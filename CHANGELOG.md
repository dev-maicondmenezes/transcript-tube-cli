# Changelog

Todas as mudanças notáveis deste projeto são documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto adere a [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

### Added

- Suporte a playlists: URLs `playlist?list=`, `watch?list=` (sem `v=`) e IDs
  crus `PL`/`UU`/`FL`/`OLAK5uy_` extraem todos os vídeos da playlist pela
  YouTube Data API v3.
- `--playlist-mode single|split` (também `TYOT_PLAYLIST_MODE` e `playlist_mode`
  no arquivo de configuração): um único Markdown ou um arquivo por vídeo em
  `<output>/<titulo-da-playlist>/<titulo-do-video>.md`.
- `watch?v=ID&list=...` extrai só o vídeo; com `--playlist-mode` explícito na
  linha de comando, extrai a playlist inteira.
- Nomes de pastas e arquivos normalizados em minúsculas, com espaços trocados
  por `-` e desambiguação de títulos repetidos.
- Vídeos que falham numa playlist são avisados e pulados; os demais são
  gerados e o comando sai com código `2`.

## [0.1.0] - 2026-11-08

Primeira versão pública do CLI `tyot`.

### Added

- Comando `tyot` para extrair metadados, capítulos e transcrição de vídeos do
  YouTube em Markdown.
- Aceita ID cru, `youtube.com/watch?v=`, `youtu.be/`, `embed`, `shorts` e `live`.
- Configuração por flags, variáveis de ambiente (`TYOT_LANG`, `TYOT_TIMEOUT`,
  `TYOT_CONFIG`) e arquivo YAML, com precedência flag > env > file > default.
- Validação obrigatória de `YOUTUBE_API_KEY` em todas as invocações que não
  sejam de ajuda/versão/completion, com mensagem rica e exit code 2.
- Escrita atômica em arquivo com `--output`/`-o`, criando diretórios.
- `--allow-partial` para gerar Markdown parcial quando a transcrição falha.
- `--no-chapters`, `--lang`, `--timeout`, `--quiet` e `--config`.
- `--version` com versão, commit e data de build.
- Man page completa (`docs/man/tyot.1`) e completions para bash, zsh, fish e
  PowerShell.
- Códigos de saída estáveis: 0 (sucesso), 1 (argumento), 2 (extração),
  3 (arquivo) e 4 (configuração).
- Workflows de CI e release, GoReleaser com `.deb`, fórmula Homebrew e
  repositório APT via GitHub Pages.
- CHANGELOG gerenciado por Conventional Commits/git-cliff.

### Security

- A API key é lida exclusivamente de `YOUTUBE_API_KEY`; não existe `--api-key`
  nem campo de API key no arquivo de configuração. O loader rejeita
  explicitamente campos como `youtube_api_key`, `api_key` e `yt_api_key`.
