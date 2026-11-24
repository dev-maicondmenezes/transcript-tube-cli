package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// resolvePlaylistScope decide, para uma URL de vídeo que também carrega uma
// playlist (watch?v=ID&list=PL...), se o usuário quer só o vídeo ou a playlist
// inteira. Devolve true para a playlist.
//
//   - --yes-playlist / --no-playlist decidem sem perguntar;
//   - com stdin interativo, pergunta em stderr (stdout é reservado ao Markdown);
//   - sem TTY não há como perguntar: extrai só o vídeo e avisa em stderr.
func (a *App) resolvePlaylistScope(videoID string, flags *rootFlags) (bool, error) {
	switch {
	case flags.yesPlaylist && flags.noPlaylist:
		return false, &codedError{code: ExitArgument, err: errors.New(
			"--yes-playlist e --no-playlist não podem ser usadas juntas")}
	case flags.yesPlaylist:
		return true, nil
	case flags.noPlaylist:
		return false, nil
	}

	if !a.InIsTTY || a.In == nil {
		fmt.Fprintf(a.Err, "aviso: o link do vídeo %s faz parte de uma playlist; extraindo só o vídeo "+
			"(use --yes-playlist para a playlist inteira ou --no-playlist para silenciar este aviso)\n", videoID)
		return false, nil
	}
	return a.askPlaylistScope(videoID)
}

const maxScopeAttempts = 3

func (a *App) askPlaylistScope(videoID string) (bool, error) {
	reader := bufio.NewReader(a.In)
	for attempt := 0; attempt < maxScopeAttempts; attempt++ {
		fmt.Fprintf(a.Err, "O link do vídeo %s faz parte de uma playlist.\n", videoID)
		fmt.Fprint(a.Err, "Baixar a transcrição de:\n"+
			"  [v] somente este vídeo\n"+
			"  [p] a playlist inteira\n"+
			"Escolha (v/p) [v]: ")

		line, err := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if err != nil && !errors.Is(err, io.EOF) {
			return false, &codedError{code: ExitArgument, err: fmt.Errorf("ler resposta: %w", err)}
		}
		if err != nil && answer == "" { // EOF sem resposta (Ctrl-D)
			fmt.Fprintln(a.Err)
			return false, nil
		}

		switch answer {
		case "", "v", "video", "vídeo":
			return false, nil
		case "p", "playlist":
			return true, nil
		}
		fmt.Fprintf(a.Err, "Resposta inválida %q: digite v ou p.\n\n", answer)
	}
	return false, &codedError{code: ExitArgument, err: errors.New(
		"resposta inválida; use --yes-playlist ou --no-playlist para evitar a pergunta")}
}
