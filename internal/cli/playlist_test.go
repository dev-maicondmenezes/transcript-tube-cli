package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/playlist"
	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

const testPlaylistURL = "https://www.youtube.com/playlist?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"

type fakeLister struct {
	info   *playlist.Info
	err    error
	called bool
	id     string
}

func (f *fakeLister) List(_ context.Context, playlistID string) (*playlist.Info, error) {
	f.called = true
	f.id = playlistID
	return f.info, f.err
}

// multiExtractor devolve dados/erros por ID de vídeo.
type multiExtractor struct {
	videos map[string]*core.VideoData
	errs   map[string]error
	calls  []string
}

func (m *multiExtractor) Extract(_ context.Context, videoID string, _ core.Options) (*core.VideoData, error) {
	m.calls = append(m.calls, videoID)
	if err := m.errs[videoID]; err != nil {
		return nil, err
	}
	return m.videos[videoID], nil
}

func video(id, title string) *core.VideoData {
	data := sampleData()
	data.ID = id
	data.Title = title
	return data
}

func newPlaylistApp(t *testing.T, lister *fakeLister, extractor *multiExtractor) (*App, string, func(args ...string) (int, string, string)) {
	t.Helper()
	app, _, _ := newTestApp(t, &fakeExtractor{}, envWithKey())
	app.Extractor = extractor
	app.Playlists = lister
	dir := t.TempDir()
	run := func(args ...string) (int, string, string) {
		return runTestApp(t, app, args...)
	}
	return app, dir, run
}

func defaultPlaylist() (*fakeLister, *multiExtractor) {
	lister := &fakeLister{info: &playlist.Info{
		ID:       "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
		Title:    "Curso de Go  2026: Básico",
		VideoIDs: []string{"aaaaaaaaaaa", "bbbbbbbbbbb"},
	}}
	extractor := &multiExtractor{videos: map[string]*core.VideoData{
		"aaaaaaaaaaa": video("aaaaaaaaaaa", "Aula 01 - Introdução"),
		"bbbbbbbbbbb": video("bbbbbbbbbbb", "Aula 02 / Variáveis"),
	}}
	return lister, extractor
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(root, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func TestPlaylistSingleToStdout(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	code, stdout, stderr := run(testPlaylistURL, "--playlist-mode", "single", "--quiet")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if lister.id != "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf" {
		t.Fatalf("playlist id = %q", lister.id)
	}
	for _, want := range []string{
		"# Curso de Go  2026: Básico",
		"**Vídeos:** 2",
		"# Aula 01 - Introdução",
		"# Aula 02 / Variáveis",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Index(stdout, "Aula 01") > strings.Index(stdout, "Aula 02") {
		t.Fatal("vídeos fora de ordem")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty with --quiet", stderr)
	}
}

func TestPlaylistSingleToFile(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)
	out := filepath.Join(dir, "sub", "playlist.md")

	code, stdout, stderr := run(testPlaylistURL, "--playlist-mode", "single", "-o", out, "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Aula 01 - Introdução") || !strings.Contains(string(data), "# Aula 02 / Variáveis") {
		t.Fatalf("arquivo incompleto:\n%s", data)
	}
}

func TestPlaylistSplitCreatesNormalizedFiles(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)

	code, stdout, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-o", dir)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty in split mode", stdout)
	}

	got := listFiles(t, dir)
	want := []string{
		"curso-de-go-2026-básico/aula-01-introdução.md",
		"curso-de-go-2026-básico/aula-02-variáveis.md",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("files = %v, want %v", got, want)
	}
	content, err := os.ReadFile(filepath.Join(dir, want[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "# Aula 01 - Introdução") {
		t.Fatalf("conteúdo inesperado:\n%s", content)
	}
}

func TestPlaylistSplitDuplicateTitles(t *testing.T) {
	lister, extractor := defaultPlaylist()
	extractor.videos["bbbbbbbbbbb"].Title = "AULA 01 - introdução"
	_, dir, run := newPlaylistApp(t, lister, extractor)

	if code, _, stderr := run(testPlaylistURL, "--playlist-mode=split", "-o", dir, "-q"); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	got := listFiles(t, dir)
	want := []string{
		"curso-de-go-2026-básico/aula-01-introdução-2.md",
		"curso-de-go-2026-básico/aula-01-introdução.md",
	}
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("files = %v, want %v", got, want)
	}
}

func TestPlaylistSplitDefaultsToCurrentDirectory(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	// Sem nenhuma flag de playlist: padrão é um arquivo por vídeo no diretório atual.
	code, stdout, stderr := run(testPlaylistURL, "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if got := listFiles(t, dir); len(got) != 2 || !strings.HasPrefix(got[0], "curso-de-go-2026-básico/") {
		t.Fatalf("files = %v", got)
	}
}

func TestPlaylistSplitRejectsStdoutOutput(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	// Vale tanto com split explícito quanto com o padrão (split).
	for _, args := range [][]string{
		{testPlaylistURL, "--playlist-mode", "split", "-o", "-"},
		{testPlaylistURL, "-o", "-"},
	} {
		code, _, stderr := run(args...)
		if code != ExitArgument {
			t.Fatalf("%v: exit = %d, want %d (stderr %q)", args, code, ExitArgument, stderr)
		}
	}
	if lister.called {
		t.Fatal("não deveria listar a playlist com argumentos inválidos")
	}
}

func TestPlaylistSplitRejectsMarkdownFileOutput(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)

	code, _, stderr := run(testPlaylistURL, "-o", filepath.Join(dir, "playlist.md"))
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d (stderr %q)", code, ExitArgument, stderr)
	}
	if !strings.Contains(stderr, "--playlist-mode single") {
		t.Fatalf("stderr = %q, want dica de --playlist-mode single", stderr)
	}
	if lister.called {
		t.Fatal("não deveria listar a playlist com argumentos inválidos")
	}
}

func TestPlaylistPartialFailureKeepsOthers(t *testing.T) {
	lister, extractor := defaultPlaylist()
	extractor.errs = map[string]error{"aaaaaaaaaaa": core.ErrVideoNotFound}
	_, dir, run := newPlaylistApp(t, lister, extractor)

	code, _, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-o", dir, "-q")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "aaaaaaaaaaa ignorado") || !strings.Contains(stderr, "1 de 2") {
		t.Fatalf("stderr = %q", stderr)
	}
	got := listFiles(t, dir)
	if len(got) != 1 || got[0] != "curso-de-go-2026-básico/aula-02-variáveis.md" {
		t.Fatalf("files = %v", got)
	}
}

func TestPlaylistAllVideosFail(t *testing.T) {
	lister, extractor := defaultPlaylist()
	extractor.errs = map[string]error{
		"aaaaaaaaaaa": core.ErrVideoNotFound,
		"bbbbbbbbbbb": core.ErrTranscriptUnavailable,
	}
	_, dir, run := newPlaylistApp(t, lister, extractor)

	code, stdout, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-o", dir, "-q")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if stdout != "" || len(listFiles(t, dir)) != 0 {
		t.Fatalf("nada deveria ser gerado (stdout=%q)", stdout)
	}
	if !strings.Contains(stderr, "nenhum dos 2 vídeos") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestPlaylistListingErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"not found", playlist.ErrNotFound, "playlist não encontrada ou privada"},
		{"empty", playlist.ErrEmpty, "playlist sem vídeos"},
		{"timeout", context.DeadlineExceeded, "tempo limite ao listar"},
		{"other", errors.New("boom"), "falha ao listar a playlist"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lister := &fakeLister{err: test.err}
			_, _, run := newPlaylistApp(t, lister, &multiExtractor{})
			code, _, stderr := run(testPlaylistURL, "-q")
			if code != ExitExtraction {
				t.Fatalf("exit = %d, want %d", code, ExitExtraction)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
		})
	}
}

func TestPlaylistInvalidModeIsConfigError(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	code, _, stderr := run(testPlaylistURL, "--playlist-mode", "zip")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr, "modo de playlist inválido") {
		t.Fatalf("stderr = %q", stderr)
	}
	if lister.called {
		t.Fatal("não deveria listar com configuração inválida")
	}
}

func TestPlaylistModeFromEnvAndFlagPrecedence(t *testing.T) {
	lister, extractor := defaultPlaylist()
	app, dir, run := newPlaylistApp(t, lister, extractor)
	env := envWithKey()
	env["TYOT_PLAYLIST_MODE"] = "single"
	app.Getenv = testGetenv(env)

	// env define single (o padrão é split) -> Markdown em stdout
	code, stdout, stderr := run(testPlaylistURL, "-q")
	if code != ExitOK || !strings.Contains(stdout, "**Vídeos:** 2") {
		t.Fatalf("env single: exit = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}

	// flag sobrepõe env -> arquivos
	app.Out.(interface{ Reset() }).Reset()
	if code, _, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-o", dir, "-q"); code != ExitOK {
		t.Fatalf("flag split: exit = %d, stderr = %q", code, stderr)
	}
	if got := listFiles(t, dir); len(got) != 2 {
		t.Fatalf("flag split: files = %v", got)
	}
}

const testVideoInPlaylistURL = "https://www.youtube.com/watch?v=aaaaaaaaaaa&list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"

func TestVideoInPlaylistNoPlaylistFlag(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	code, stdout, stderr := run(testVideoInPlaylistURL, "--no-playlist", "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if lister.called {
		t.Fatal("--no-playlist deve extrair só o vídeo")
	}
	if len(extractor.calls) != 1 || extractor.calls[0] != "aaaaaaaaaaa" {
		t.Fatalf("calls = %v", extractor.calls)
	}
	if !strings.Contains(stdout, "# Aula 01 - Introdução") {
		t.Fatalf("stdout = %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty (sem aviso nem pergunta)", stderr)
	}
}

func TestVideoInPlaylistYesPlaylistFlag(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)

	code, _, stderr := run(testVideoInPlaylistURL, "--yes-playlist", "-o", dir, "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if !lister.called || lister.id != "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf" || len(extractor.calls) != 2 {
		t.Fatalf("lister.called=%v id=%q calls=%v, want playlist com 2 vídeos", lister.called, lister.id, extractor.calls)
	}
	if got := listFiles(t, dir); len(got) != 2 {
		t.Fatalf("files = %v", got)
	}
}

func TestVideoInPlaylistConflictingFlags(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	// Mesmo numa URL sem playlist, a combinação é inválida.
	for _, url := range []string{testVideoInPlaylistURL, "dQw4w9WgXcQ"} {
		code, _, stderr := run(url, "--yes-playlist", "--no-playlist")
		if code != ExitArgument || !strings.Contains(stderr, "não podem ser usadas juntas") {
			t.Fatalf("%s: exit = %d, stderr = %q", url, code, stderr)
		}
	}
	if lister.called || len(extractor.calls) != 0 {
		t.Fatal("nada deveria ser extraído")
	}
}

func TestVideoInPlaylistWithoutTTYExtractsVideoAndWarns(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor) // InIsTTY = false

	code, stdout, stderr := run(testVideoInPlaylistURL, "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if lister.called || len(extractor.calls) != 1 {
		t.Fatalf("lister.called=%v calls=%v, want só o vídeo", lister.called, extractor.calls)
	}
	if !strings.Contains(stderr, "faz parte de uma playlist") || !strings.Contains(stderr, "--yes-playlist") {
		t.Fatalf("stderr = %q, want aviso com dica de --yes-playlist", stderr)
	}
	if !strings.Contains(stdout, "# Aula 01 - Introdução") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestVideoInPlaylistPrompt(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantCode     int
		wantPlaylist bool
		wantAsked    int // quantas vezes a pergunta apareceu
	}{
		{"p escolhe a playlist", "p\n", ExitOK, true, 1},
		{"playlist por extenso, maiúsculas", " PLAYLIST \n", ExitOK, true, 1},
		{"v escolhe o vídeo", "v\n", ExitOK, false, 1},
		{"enter assume o vídeo", "\n", ExitOK, false, 1},
		{"EOF assume o vídeo", "", ExitOK, false, 1},
		{"resposta inválida repete e depois aceita", "x\np\n", ExitOK, true, 2},
		{"3 respostas inválidas falham", "x\ny\nz\n", ExitArgument, false, 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lister, extractor := defaultPlaylist()
			app, dir, run := newPlaylistApp(t, lister, extractor)
			app.InIsTTY = true
			app.In = strings.NewReader(test.input)

			args := []string{testVideoInPlaylistURL}
			if test.wantPlaylist {
				args = append(args, "-o", dir)
			}
			code, stdout, stderr := run(args...)
			if code != test.wantCode {
				t.Fatalf("exit = %d, want %d (stderr %q)", code, test.wantCode, stderr)
			}
			if got := strings.Count(stderr, "faz parte de uma playlist."); got != test.wantAsked {
				t.Fatalf("pergunta apareceu %d vez(es), want %d:\n%s", got, test.wantAsked, stderr)
			}
			if strings.Contains(stdout, "playlist") && strings.Contains(stdout, "Escolha") {
				t.Fatalf("a pergunta não pode ir para stdout: %q", stdout)
			}
			if test.wantCode != ExitOK {
				return
			}
			if lister.called != test.wantPlaylist {
				t.Fatalf("lister.called = %v, want %v", lister.called, test.wantPlaylist)
			}
			wantCalls := 1
			if test.wantPlaylist {
				wantCalls = 2
			}
			if len(extractor.calls) != wantCalls {
				t.Fatalf("calls = %v, want %d", extractor.calls, wantCalls)
			}
		})
	}
}

func TestPlainVideoNeverPrompts(t *testing.T) {
	lister, extractor := defaultPlaylist()
	app, _, run := newPlaylistApp(t, lister, extractor)
	app.InIsTTY = true
	app.In = strings.NewReader("")

	code, _, stderr := run("aaaaaaaaaaa", "-q")
	if code != ExitOK || stderr != "" || lister.called {
		t.Fatalf("exit = %d, stderr = %q, lister.called = %v", code, stderr, lister.called)
	}
}

func TestPlaylistRawIDIsRecognized(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, dir, run := newPlaylistApp(t, lister, extractor)

	if code, _, stderr := run("PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf", "-o", dir, "-q"); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if !lister.called {
		t.Fatal("ID cru de playlist deveria listar a playlist")
	}
}
