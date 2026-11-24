package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/config"
	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/playlist"
	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/slug"
	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

// youtubePlaylists é o playlist.Lister de produção: lê YOUTUBE_API_KEY do
// ambiente (já validada antes de qualquer lógica) e consulta a Data API v3.
type youtubePlaylists struct{}

func (youtubePlaylists) List(ctx context.Context, playlistID string) (*playlist.Info, error) {
	client, err := playlist.New(strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY")))
	if err != nil {
		return nil, err
	}
	return client.List(ctx, playlistID)
}

const videoSeparator = "\n\n***\n\n"

// runPlaylist extrai todos os vídeos de uma playlist.
//
// O timeout (--timeout) vale para a listagem e, separadamente, para cada
// vídeo. Vídeos que falham (privados, removidos, sem transcrição sem
// --allow-partial) são avisados em stderr e pulados; os demais são gravados e,
// ao final, o comando sai com código 2 se algum falhou.
func (a *App) runPlaylist(playlistID string, cfg config.Config) error {
	split := cfg.PlaylistMode == config.PlaylistModeSplit
	if split && cfg.Output == "-" {
		return &codedError{code: ExitArgument, err: errors.New(
			"--playlist-mode split grava arquivos; --output - (stdout) não é suportado")}
	}

	if split && strings.HasSuffix(strings.ToLower(cfg.Output), ".md") {
		return &codedError{code: ExitArgument, err: fmt.Errorf(
			"--output %q parece um arquivo, mas com --playlist-mode split ele é o diretório base; "+
				"para gerar um único arquivo use --playlist-mode single", cfg.Output)}
	}

	start := a.Now()
	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: listando playlist %s...\n", playlistID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	info, err := a.Playlists.List(ctx, playlistID)
	cancel()
	if err != nil {
		return mapPlaylistError(err)
	}

	total := len(info.VideoIDs)
	title := strings.TrimSpace(info.Title)
	if title == "" {
		title = info.ID
	}
	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: playlist %q com %d vídeo(s)\n", title, total)
	}

	var videos []*core.VideoData
	usedNames := make(map[string]struct{}, total)
	failed := 0
	succeeded := 0
	for i, videoID := range info.VideoIDs {
		if !cfg.Quiet {
			fmt.Fprintf(a.Err, "tyot: [%d/%d] extraindo %s...\n", i+1, total, videoID)
		}
		data, err := a.fetchVideo(videoID, cfg)
		if err != nil {
			failed++
			fmt.Fprintf(a.Err, "aviso: vídeo %s ignorado: %v\n", videoID, err)
			continue
		}
		succeeded++
		if split {
			if err := a.writePlaylistVideoFile(cfg, info, title, data, usedNames); err != nil {
				return err
			}
		} else {
			videos = append(videos, data)
		}
	}
	if succeeded == 0 {
		return &codedError{code: ExitExtraction, err: fmt.Errorf(
			"nenhum dos %d vídeos da playlist pôde ser extraído", total)}
	}

	if !split {
		if err := a.writePlaylistSingle(cfg, info, title, videos); err != nil {
			return err
		}
	}

	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: concluído em %s\n", a.Now().Sub(start).Round(time.Millisecond))
	}
	if failed > 0 {
		return &codedError{code: ExitExtraction, err: fmt.Errorf(
			"%d de %d vídeos da playlist falharam (os demais foram gerados)", failed, total)}
	}
	return nil
}

// writePlaylistVideoFile grava um vídeo individual imediatamente, sem manter
// as transcrições anteriores em memória durante a extração da playlist.
func (a *App) writePlaylistVideoFile(cfg config.Config, info *playlist.Info, title string, data *core.VideoData, used map[string]struct{}) error {
	base := cfg.Output
	if base == "" {
		base = "."
	}
	dir := filepath.Join(base, slug.Normalize(title, slug.Normalize(info.ID, "playlist")))
	name := slug.Unique(slug.Normalize(data.Title, data.ID), used)
	path := filepath.Join(dir, name+".md")
	if err := writeFileAtomic(path, []byte(core.ToMarkdown(data))); err != nil {
		return &codedError{code: ExitFileWrite, err: err}
	}
	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: gravado %s\n", path)
	}
	return nil
}

// writePlaylistSingle junta todos os vídeos em um único Markdown, com um
// cabeçalho da playlist, em stdout ou no arquivo de --output.
func (a *App) writePlaylistSingle(cfg config.Config, info *playlist.Info, title string, videos []*core.VideoData) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "**Playlist:** https://www.youtube.com/playlist?list=%s\n", info.ID)
	fmt.Fprintf(&b, "**Vídeos:** %d\n", len(videos))
	for _, data := range videos {
		b.WriteString(videoSeparator)
		b.WriteString(strings.TrimRight(core.ToMarkdown(data), "\n"))
		b.WriteString("\n")
	}
	markdown := b.String()

	if cfg.Output == "" || cfg.Output == "-" {
		if _, err := io.WriteString(a.Out, markdown); err != nil {
			return &codedError{code: ExitFileWrite, err: fmt.Errorf("escrever Markdown em stdout: %w", err)}
		}
		return nil
	}
	if err := writeFileAtomic(cfg.Output, []byte(markdown)); err != nil {
		return &codedError{code: ExitFileWrite, err: err}
	}
	return nil
}

func mapPlaylistError(err error) error {
	switch {
	case errors.Is(err, playlist.ErrNotFound):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("playlist não encontrada ou privada: %w", err)}
	case errors.Is(err, playlist.ErrEmpty):
		return &codedError{code: ExitExtraction, err: err}
	case errors.Is(err, context.DeadlineExceeded):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("tempo limite ao listar a playlist excedido: %w", err)}
	case errors.Is(err, context.Canceled):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("listagem da playlist cancelada: %w", err)}
	default:
		return &codedError{code: ExitExtraction, err: fmt.Errorf("falha ao listar a playlist: %w", err)}
	}
}
