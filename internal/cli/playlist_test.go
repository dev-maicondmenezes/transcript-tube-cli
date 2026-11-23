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

	code, stdout, stderr := run(testPlaylistURL, "--quiet")
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

	code, stdout, stderr := run(testPlaylistURL, "-o", out, "-q")
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

	if code, _, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-q"); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if got := listFiles(t, dir); len(got) != 2 || !strings.HasPrefix(got[0], "curso-de-go-2026-básico/") {
		t.Fatalf("files = %v", got)
	}
}

func TestPlaylistSplitRejectsStdoutOutput(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	code, _, stderr := run(testPlaylistURL, "--playlist-mode", "split", "-o", "-")
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d (stderr %q)", code, ExitArgument, stderr)
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
	env["TYOT_PLAYLIST_MODE"] = "split"
	app.Getenv = testGetenv(env)

	// env define split -> arquivos
	if code, _, stderr := run(testPlaylistURL, "-o", dir, "-q"); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if got := listFiles(t, dir); len(got) != 2 {
		t.Fatalf("env split: files = %v", got)
	}

	// flag sobrepõe env -> stdout
	app.Out.(interface{ Reset() }).Reset()
	code, stdout, _ := run(testPlaylistURL, "--playlist-mode", "single", "-q")
	if code != ExitOK || !strings.Contains(stdout, "**Vídeos:** 2") {
		t.Fatalf("flag single: exit = %d, stdout = %q", code, stdout)
	}
}

func TestVideoURLWithListStaysSingleVideo(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	url := "https://www.youtube.com/watch?v=aaaaaaaaaaa&list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"
	code, stdout, stderr := run(url, "-q")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if lister.called {
		t.Fatal("watch?v=...&list=... deve extrair só o vídeo")
	}
	if len(extractor.calls) != 1 || extractor.calls[0] != "aaaaaaaaaaa" {
		t.Fatalf("calls = %v", extractor.calls)
	}
	if !strings.Contains(stdout, "# Aula 01 - Introdução") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestPlaylistRawIDIsRecognized(t *testing.T) {
	lister, extractor := defaultPlaylist()
	_, _, run := newPlaylistApp(t, lister, extractor)

	if code, _, stderr := run("PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf", "-q"); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if !lister.called {
		t.Fatal("ID cru de playlist deveria listar a playlist")
	}
}
