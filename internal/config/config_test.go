package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func getenv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func isolateConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func TestLoadDefaults(t *testing.T) {
	isolateConfigDir(t)
	cfg, err := Load(Flags{}, getenv(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != DefaultLang {
		t.Fatalf("Lang = %q, want %q", cfg.Lang, DefaultLang)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", cfg.Timeout, DefaultTimeout)
	}
	if cfg.AllowPartial || cfg.NoChapters || cfg.Quiet {
		t.Fatalf("boolean defaults = %+v, want false", cfg)
	}
	if cfg.Output != "" {
		t.Fatalf("Output = %q, want empty", cfg.Output)
	}
}

func TestLoadFile(t *testing.T) {
	path := writeConfig(t, `
default_lang: "en"
default_timeout: "30s"
allow_partial: true
no_chapters: true
`)
	cfg, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != "en" || cfg.Timeout != 30*time.Second || !cfg.AllowPartial || !cfg.NoChapters {
		t.Fatalf("Load() = %+v", cfg)
	}
}

func TestLoadPrecedence(t *testing.T) {
	path := writeConfig(t, `default_lang: "en"`)
	env := getenv(map[string]string{"TYOT_LANG": "es"})

	cfg, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != "es" {
		t.Fatalf("env should override file: Lang = %q", cfg.Lang)
	}

	cfg, err = Load(Flags{
		ConfigPath:    path,
		ConfigPathSet: true,
		Lang:          "fr",
		LangSet:       true,
	}, env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != "fr" {
		t.Fatalf("flag should override env/file: Lang = %q", cfg.Lang)
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	isolateConfigDir(t)
	cfg, err := Load(Flags{
		ConfigPath:    filepath.Join(t.TempDir(), "missing.yaml"),
		ConfigPathSet: true,
	}, getenv(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != DefaultLang || cfg.Timeout != DefaultTimeout {
		t.Fatalf("Load() = %+v", cfg)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeConfig(t, "default_lang: [unterminated")
	_, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	var cfgErr *Error
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error type = %T, want *config.Error", err)
	}
}

func TestLoadUnknownField(t *testing.T) {
	path := writeConfig(t, "default_lang: en\nunknown_field: true\n")
	_, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Fatalf("error = %v, want unknown_field mention", err)
	}
}

func TestLoadRejectsAPIKeyField(t *testing.T) {
	for _, key := range []string{"youtube_api_key", "api_key", "yt_api_key", "youtubeApiKey"} {
		t.Run(key, func(t *testing.T) {
			path := writeConfig(t, key+": \"secret\"\n")
			_, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			message := err.Error()
			if !strings.Contains(message, "proibido") {
				t.Fatalf("error = %v, want proibido", err)
			}
			if !strings.Contains(message, "YOUTUBE_API_KEY") {
				t.Fatalf("error = %v, want YOUTUBE_API_KEY instruction", err)
			}
		})
	}
}

func TestLoadRejectsNestedAPIKeyField(t *testing.T) {
	path := writeConfig(t, "nested:\n  youtube_api_key: \"secret\"\n")
	_, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
	if err == nil || !strings.Contains(err.Error(), "proibido") {
		t.Fatalf("Load() error = %v, want proibido", err)
	}
}

func TestLoadInvalidEnvTimeout(t *testing.T) {
	isolateConfigDir(t)
	_, err := Load(Flags{}, getenv(map[string]string{"TYOT_TIMEOUT": "not-a-duration"}))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "TYOT_TIMEOUT") {
		t.Fatalf("error = %v, want TYOT_TIMEOUT", err)
	}
}

func TestLoadInvalidFileTimeout(t *testing.T) {
	path := writeConfig(t, "default_timeout: not-a-duration\n")
	_, err := Load(Flags{ConfigPath: path, ConfigPathSet: true}, getenv(nil))
	if err == nil || !strings.Contains(err.Error(), "default_timeout") {
		t.Fatalf("Load() error = %v, want default_timeout", err)
	}
}

func TestLoadRejectsNonPositiveTimeout(t *testing.T) {
	isolateConfigDir(t)
	_, err := Load(Flags{Timeout: 0, TimeoutSet: true}, getenv(nil))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadUsesTYOTConfigPath(t *testing.T) {
	path := writeConfig(t, "default_lang: de\n")
	cfg, err := Load(Flags{}, getenv(map[string]string{"TYOT_CONFIG": path}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Lang != "de" {
		t.Fatalf("Lang = %q, want de", cfg.Lang)
	}
}

func TestLoadRejectsEmptyExplicitConfigPath(t *testing.T) {
	_, err := Load(Flags{ConfigPath: "  ", ConfigPathSet: true}, getenv(nil))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
