package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

type fakeExtractor struct {
	data   *core.VideoData
	err    error
	block  bool
	noise  bool
	called bool
	id     string
	opts   core.Options
}

func (f *fakeExtractor) Extract(ctx context.Context, videoID string, opts core.Options) (*core.VideoData, error) {
	f.called = true
	f.id = videoID
	f.opts = opts
	if f.noise {
		fmt.Println("ruído da biblioteca InnerTube")
	}
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.data, f.err
}

func sampleData() *core.VideoData {
	return &core.VideoData{
		ID:          "dQw4w9WgXcQ",
		Title:       "Test Video",
		Channel:     "Test Channel",
		Description: "Descrição de teste.",
		PublishedAt: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC),
		Duration:    "03:34",
		ViewCount:   123,
		LikeCount:   4,
		Transcript: []core.TranscriptSegment{
			{OffsetMs: 0, Text: "primeira linha"},
			{OffsetMs: 2000, Text: "segunda linha"},
		},
		Chapters: []core.Chapter{
			{OffsetMs: 0, Title: "Intro"},
			{OffsetMs: 1000, Title: "Meio"},
		},
	}
}

func testGetenv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func envWithKey() map[string]string {
	return map[string]string{"YOUTUBE_API_KEY": "test-key"}
}

func newTestApp(t *testing.T, extractor *fakeExtractor, env map[string]string) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := &App{
		In:        strings.NewReader(""),
		Out:       out,
		Err:       errOut,
		InIsTTY:   false,
		Getenv:    testGetenv(env),
		Extractor: extractor,
		Now:       time.Now,
	}
	return app, out, errOut
}

func runTestApp(t *testing.T, app *App, args ...string) (int, string, string) {
	t.Helper()
	code := app.Run(args)
	return code, app.Out.(*bytes.Buffer).String(), app.Err.(*bytes.Buffer).String()
}

func TestHelpWithoutEnv(t *testing.T) {
	extractor := &fakeExtractor{}
	app, out, _ := newTestApp(t, extractor, nil)
	code, stdout, _ := runTestApp(t, app, "--help")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "Uso:") {
		t.Fatalf("stdout = %q, want usage", stdout)
	}
	if !strings.Contains(stdout, "YOUTUBE_API_KEY") {
		t.Fatalf("stdout = %q, want env requirement", stdout)
	}
	if extractor.called {
		t.Fatal("extractor was called for --help")
	}
	if out.Len() == 0 {
		t.Fatal("stdout is empty")
	}
}

func TestVersionWithoutEnv(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	Version, Commit, Date = "v0.1.0", "abc123", "2026-11-08T00:00:00Z"
	t.Cleanup(func() { Version, Commit, Date = oldVersion, oldCommit, oldDate })

	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, stdout, _ := runTestApp(t, app, "--version")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	for _, want := range []string{"tyot v0.1.0", "commit: abc123", "built: 2026-11-08T00:00:00Z"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
	if extractor.called {
		t.Fatal("extractor was called for --version")
	}
}

func TestVersionAllowsExtraArgs(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, stdout, _ := runTestApp(t, app, "--version", "ignored", "also-ignored")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "tyot ") {
		t.Fatalf("stdout = %q, want version", stdout)
	}
}

func TestMissingEnvWinsOverURL(t *testing.T) {
	extractor := &fakeExtractor{}
	app, out, _ := newTestApp(t, extractor, nil)
	code, stdout, stderr := runTestApp(t, app, "dQw4w9WgXcQ")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "YOUTUBE_API_KEY") || !strings.Contains(stderr, "console.cloud.google.com") {
		t.Fatalf("stderr = %q, want rich env error", stderr)
	}
	if extractor.called {
		t.Fatal("extractor was called without env")
	}
	if out.Len() != 0 {
		t.Fatal("stdout buffer is not empty")
	}
}

func TestMissingEnvWinsOverOutputFlag(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, _, stderr := runTestApp(t, app, "-o", filepath.Join(t.TempDir(), "out.md"))
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "YOUTUBE_API_KEY") {
		t.Fatalf("stderr = %q, want env error", stderr)
	}
	if extractor.called {
		t.Fatal("extractor was called without env")
	}
}

func TestMissingEnvWinsOverInvalidURL(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, _, stderr := runTestApp(t, app, "not-a-url")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "YOUTUBE_API_KEY") {
		t.Fatalf("stderr = %q, want env error", stderr)
	}
}

func TestUnknownFlagWithoutEnv(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, _, stderr := runTestApp(t, app, "--bogus")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "YOUTUBE_API_KEY") {
		t.Fatalf("stderr = %q, want env error", stderr)
	}
}

func TestUnknownFlagWithEnv(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "--bogus")
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d", code, ExitArgument)
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Fatalf("stderr = %q, want unknown flag", stderr)
	}
}

func TestInvalidURLWithEnv(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "https://example.com/watch?v=dQw4w9WgXcQ")
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d", code, ExitArgument)
	}
	if !strings.Contains(stderr, "URL não reconhecida") {
		t.Fatalf("stderr = %q, want URL error", stderr)
	}
	if extractor.called {
		t.Fatal("extractor was called for invalid URL")
	}
}

func TestSuccessWritesOnlyMarkdownToStdout(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, errOut := newTestApp(t, extractor, envWithKey())
	code, stdout, stderr := runTestApp(t, app, "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.HasPrefix(stdout, "# Test Video") {
		t.Fatalf("stdout = %q, want markdown", stdout)
	}
	if strings.Contains(stdout, "tyot:") {
		t.Fatalf("stdout contains progress log: %q", stdout)
	}
	if !strings.Contains(stderr, "tyot: extraindo") {
		t.Fatalf("stderr = %q, want progress log", stderr)
	}
	if !extractor.called {
		t.Fatal("extractor was not called")
	}
	if extractor.id != "dQw4w9WgXcQ" {
		t.Fatalf("videoID = %q", extractor.id)
	}
	if extractor.opts.Lang != "pt" {
		t.Fatalf("Lang = %q, want pt", extractor.opts.Lang)
	}
	if errOut.Len() == 0 {
		t.Fatal("stderr buffer is empty")
	}
}

func TestQuietSuppressesProgress(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, stdout, stderr := runTestApp(t, app, "--quiet", "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.HasPrefix(stdout, "# Test Video") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestOutputFileCreatesDirectories(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	path := filepath.Join(t.TempDir(), "nested", "dir", "out.md")
	code, stdout, stderr := runTestApp(t, app, "--output", path, "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitOK, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != core.ToMarkdown(sampleData()) {
		t.Fatalf("file content = %q, want markdown", string(content))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %o, want 644", info.Mode().Perm())
	}
}

func TestOutputFileInvalidPath(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	code, _, stderr := runTestApp(t, app, "--output", filepath.Join(blocker, "out.md"), "dQw4w9WgXcQ")
	if code != ExitFileWrite {
		t.Fatalf("exit = %d, want %d", code, ExitFileWrite)
	}
	if !strings.Contains(stderr, "erro:") {
		t.Fatalf("stderr = %q, want error", stderr)
	}
}

func TestOutputDashMeansStdout(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, stdout, _ := runTestApp(t, app, "-o", "-", "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.HasPrefix(stdout, "# Test Video") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestNoChapters(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, withChapters, _ := runTestApp(t, app, "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(withChapters, "### ") {
		t.Fatalf("stdout without --no-chapters = %q, want chapter heading", withChapters)
	}

	extractor = &fakeExtractor{data: sampleData()}
	app, _, _ = newTestApp(t, extractor, envWithKey())
	code, withoutChapters, _ := runTestApp(t, app, "--no-chapters", "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if strings.Contains(withoutChapters, "### ") {
		t.Fatalf("stdout with --no-chapters = %q, want no chapter heading", withoutChapters)
	}
}

func TestPartialTranscriptAllowed(t *testing.T) {
	extractor := &fakeExtractor{
		data: sampleData(),
		err:  fmt.Errorf("transcript: %w", core.ErrTranscriptUnavailable),
	}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, stdout, stderr := runTestApp(t, app, "--allow-partial", "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitOK, stderr)
	}
	if !strings.HasPrefix(stdout, "# Test Video") {
		t.Fatalf("stdout = %q, want markdown", stdout)
	}
	if !strings.Contains(stderr, "aviso:") {
		t.Fatalf("stderr = %q, want warning", stderr)
	}
	if !extractor.opts.AllowPartialTranscript {
		t.Fatal("AllowPartialTranscript = false, want true")
	}
}

func TestPartialTranscriptWithoutFlag(t *testing.T) {
	extractor := &fakeExtractor{err: fmt.Errorf("transcript: %w", core.ErrTranscriptUnavailable)}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "dQw4w9WgXcQ")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "--allow-partial") {
		t.Fatalf("stderr = %q, want --allow-partial hint", stderr)
	}
}

func TestExtractionError(t *testing.T) {
	extractor := &fakeExtractor{err: errors.New("boom")}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "dQw4w9WgXcQ")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "falha na extração") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestExtractionTimeout(t *testing.T) {
	extractor := &fakeExtractor{block: true}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "--timeout", "1ms", "dQw4w9WgXcQ")
	if code != ExitExtraction {
		t.Fatalf("exit = %d, want %d", code, ExitExtraction)
	}
	if !strings.Contains(stderr, "tempo limite") {
		t.Fatalf("stderr = %q, want timeout message", stderr)
	}
}

func TestStdinInput(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	app.In = strings.NewReader("\n\nhttps://youtu.be/dQw4w9WgXcQ\nignored\n")
	code, stdout, _ := runTestApp(t, app)
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.HasPrefix(stdout, "# Test Video") {
		t.Fatalf("stdout = %q", stdout)
	}
	if extractor.id != "dQw4w9WgXcQ" {
		t.Fatalf("videoID = %q, want dQw4w9WgXcQ", extractor.id)
	}
}

func TestNoArgsTTYShowsHelpAndExitsOne(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	app.InIsTTY = true
	code, stdout, stderr := runTestApp(t, app)
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d", code, ExitArgument)
	}
	if !strings.Contains(stdout, "Uso:") {
		t.Fatalf("stdout = %q, want help", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestNoArgsEmptyStdin(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app)
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d", code, ExitArgument)
	}
	if !strings.Contains(stderr, "stdin vazio") {
		t.Fatalf("stderr = %q, want stdin empty error", stderr)
	}
}

func TestMultipleArgsRejected(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "dQw4w9WgXcQ", "dQw4w9WgXcQ")
	if code != ExitArgument {
		t.Fatalf("exit = %d, want %d", code, ExitArgument)
	}
	if stderr == "" {
		t.Fatal("stderr is empty")
	}
}

func TestConfigAPIKeyRejected(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("youtube_api_key: secret\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "--config", configPath, "dQw4w9WgXcQ")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr, "proibido") || !strings.Contains(stderr, "YOUTUBE_API_KEY") {
		t.Fatalf("stderr = %q", stderr)
	}
	if extractor.called {
		t.Fatal("extractor was called with invalid config")
	}
}

func TestConfigUnknownField(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("unknown_field: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "--config", configPath, "dQw4w9WgXcQ")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr, "unknown_field") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestConfigInvalidYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("default_lang: [unterminated\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app, "--config", configPath, "dQw4w9WgXcQ")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(stderr, "inválido") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestConfigPrecedenceThroughCLI(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("default_lang: en\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tests := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{
			name: "file",
			env:  envWithKey(),
			args: []string{"--config", configPath, "dQw4w9WgXcQ"},
			want: "en",
		},
		{
			name: "env overrides file",
			env:  map[string]string{"YOUTUBE_API_KEY": "test-key", "TYOT_LANG": "es"},
			args: []string{"--config", configPath, "dQw4w9WgXcQ"},
			want: "es",
		},
		{
			name: "flag overrides env and file",
			env:  map[string]string{"YOUTUBE_API_KEY": "test-key", "TYOT_LANG": "es"},
			args: []string{"--config", configPath, "--lang", "fr", "dQw4w9WgXcQ"},
			want: "fr",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extractor := &fakeExtractor{data: sampleData()}
			app, _, _ := newTestApp(t, extractor, test.env)
			code, _, stderr := runTestApp(t, app, test.args...)
			if code != ExitOK {
				t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitOK, stderr)
			}
			if extractor.opts.Lang != test.want {
				t.Fatalf("Lang = %q, want %q", extractor.opts.Lang, test.want)
			}
		})
	}
}

func TestConfigFileMissingUsesDefaults(t *testing.T) {
	extractor := &fakeExtractor{data: sampleData()}
	app, _, _ := newTestApp(t, extractor, envWithKey())
	code, _, stderr := runTestApp(t, app,
		"--config", filepath.Join(t.TempDir(), "missing.yaml"), "dQw4w9WgXcQ")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitOK, stderr)
	}
	if extractor.opts.Lang != "pt" {
		t.Fatalf("Lang = %q, want pt", extractor.opts.Lang)
	}
}

func TestCompletionWithoutEnv(t *testing.T) {
	extractor := &fakeExtractor{}
	app, _, _ := newTestApp(t, extractor, nil)
	code, stdout, stderr := runTestApp(t, app, "completion", "bash")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitOK, stderr)
	}
	if !strings.Contains(stdout, "tyot") {
		t.Fatalf("stdout = %q, want completion script", stdout)
	}
}
