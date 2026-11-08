package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/config"
	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/url"
	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

func (a *App) runExtract(cmd *cobra.Command, args []string, flags *rootFlags) error {
	raw, err := a.readInput(cmd, args)
	if err != nil {
		return err
	}

	videoID, err := url.ParseVideoID(raw)
	if err != nil {
		return &codedError{code: ExitArgument, err: err}
	}

	cfg, err := config.Load(a.configFlags(cmd, flags), a.Getenv)
	if err != nil {
		return &codedError{code: ExitConfig, err: err}
	}

	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: extraindo metadados e transcrição de %s...\n", videoID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	start := a.Now()
	data, extractErr := a.extract(ctx, videoID, core.Options{
		Lang:                   cfg.Lang,
		AllowPartialTranscript: cfg.AllowPartial,
	}, cfg.Quiet)

	if extractErr != nil {
		if errors.Is(extractErr, core.ErrTranscriptUnavailable) && cfg.AllowPartial && data != nil {
			fmt.Fprintf(a.Err, "aviso: transcrição indisponível; gerando Markdown parcial: %v\n", extractErr)
		} else {
			return mapExtractError(extractErr)
		}
	}
	if data == nil {
		return &codedError{code: ExitExtraction, err: errors.New("a extração não retornou dados")}
	}

	if cfg.NoChapters {
		data.Chapters = nil
	}
	markdown := core.ToMarkdown(data)

	if cfg.Output == "" || cfg.Output == "-" {
		if _, err := io.WriteString(a.Out, markdown); err != nil {
			return &codedError{code: ExitFileWrite, err: fmt.Errorf("escrever Markdown em stdout: %w", err)}
		}
	} else if err := writeFileAtomic(cfg.Output, []byte(markdown)); err != nil {
		return &codedError{code: ExitFileWrite, err: err}
	}

	if !cfg.Quiet {
		fmt.Fprintf(a.Err, "tyot: concluído em %s\n", a.Now().Sub(start).Round(time.Millisecond))
	}
	return nil
}

func (a *App) extract(ctx context.Context, videoID string, opts core.Options, quiet bool) (*core.VideoData, error) {
	if !a.SilenceCoreStdout {
		return a.Extractor.Extract(ctx, videoID, opts)
	}
	restore := silenceStdout(quiet)
	defer restore()
	return a.Extractor.Extract(ctx, videoID, opts)
}

func (a *App) configFlags(cmd *cobra.Command, flags *rootFlags) config.Flags {
	changed := func(name string) bool {
		return cmd.Flags().Changed(name)
	}
	return config.Flags{
		ConfigPath:    flags.configPath,
		ConfigPathSet: changed("config"),

		Lang:    flags.lang,
		LangSet: changed("lang"),

		Timeout:    flags.timeout,
		TimeoutSet: changed("timeout"),

		AllowPartial:    flags.allowPartial,
		AllowPartialSet: changed("allow-partial"),

		NoChapters:    flags.noChapters,
		NoChaptersSet: changed("no-chapters"),

		Output: flags.output,
		Quiet:  flags.quiet,
	}
}

func (a *App) readInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) == 1 {
		raw := strings.TrimSpace(args[0])
		if raw == "" {
			return "", &codedError{code: ExitArgument, err: errors.New("URL ou ID de vídeo vazio")}
		}
		return raw, nil
	}

	if a.InIsTTY {
		if err := cmd.Help(); err != nil {
			return "", &codedError{code: ExitArgument, err: err}
		}
		return "", &codedError{code: ExitArgument, silent: true}
	}

	if a.In == nil {
		return "", &codedError{code: ExitArgument, err: errors.New("stdin indisponível")}
	}
	data, err := io.ReadAll(io.LimitReader(a.In, 1<<20))
	if err != nil {
		return "", &codedError{code: ExitArgument, err: fmt.Errorf("ler stdin: %w", err)}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line, nil
		}
	}
	return "", &codedError{code: ExitArgument, err: errors.New("nenhuma URL ou ID de vídeo informado (stdin vazio)")}
}

func mapExtractError(err error) error {
	switch {
	case errors.Is(err, core.ErrInvalidVideoID):
		return &codedError{code: ExitArgument, err: fmt.Errorf("ID de vídeo inválido: %w", err)}
	case errors.Is(err, core.ErrMissingAPIKey):
		return &codedError{code: ExitExtraction, err: errors.New(missingAPIKeyMessage)}
	case errors.Is(err, context.DeadlineExceeded):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("tempo limite de extração excedido: %w", err)}
	case errors.Is(err, context.Canceled):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("extração cancelada: %w", err)}
	case errors.Is(err, core.ErrVideoNotFound):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("vídeo não encontrado: %w", err)}
	case errors.Is(err, core.ErrTranscriptUnavailable):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("transcrição indisponível (use --allow-partial para gerar Markdown parcial): %w", err)}
	case errors.Is(err, core.ErrMetadataUnavailable):
		return &codedError{code: ExitExtraction, err: fmt.Errorf("metadados indisponíveis: %w", err)}
	default:
		return &codedError{code: ExitExtraction, err: fmt.Errorf("falha na extração: %w", err)}
	}
}
