// Package config carrega e mescla a configuração do tyot.
//
// Precedência: flag > variável de ambiente > arquivo YAML > default.
// A API key nunca é aceita no arquivo: use exclusivamente YOUTUBE_API_KEY.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultLang    = "pt"
	DefaultTimeout = 60 * time.Second
)

// Config é a configuração efetiva após aplicar todas as camadas.
type Config struct {
	Lang         string
	Timeout      time.Duration
	AllowPartial bool
	NoChapters   bool
	Output       string
	Quiet        bool
	ConfigPath   string
}

// Flags representa apenas os valores das flags e se foram alterados. Apenas
// flags explicitamente definidas pelo usuário sobrepõem env/file/default.
type Flags struct {
	ConfigPath    string
	ConfigPathSet bool

	Lang    string
	LangSet bool

	Timeout    time.Duration
	TimeoutSet bool

	AllowPartial    bool
	AllowPartialSet bool

	NoChapters    bool
	NoChaptersSet bool

	Output string
	Quiet  bool
}

// Error é um erro de configuração (exit code 4).
type Error struct {
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Default devolve a configuração base.
func Default() Config {
	return Config{
		Lang:    DefaultLang,
		Timeout: DefaultTimeout,
	}
}

// DefaultPath devolve o caminho padrão do arquivo de configuração.
// TYOT_CONFIG tem precedência; depois usamos os.UserConfigDir().
func DefaultPath(getenv func(string) string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	if path := strings.TrimSpace(getenv("TYOT_CONFIG")); path != "" {
		return path
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "tyot", "config.yaml")
	}
	return filepath.Join(".config", "tyot", "config.yaml")
}

// Load carrega defaults, arquivo, env e flags, nesta ordem.
func Load(flags Flags, getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := Default()
	path := DefaultPath(getenv)
	if flags.ConfigPathSet {
		if p := strings.TrimSpace(flags.ConfigPath); p == "" {
			return Config{}, &Error{Message: "--config não pode ser vazio"}
		} else {
			path = p
		}
	}
	cfg.ConfigPath = path

	if err := applyFile(&cfg, path); err != nil {
		return Config{}, err
	}
	if err := applyEnv(&cfg, getenv); err != nil {
		return Config{}, err
	}
	applyFlags(&cfg, flags)

	if strings.TrimSpace(cfg.Lang) == "" {
		cfg.Lang = DefaultLang
	}
	if cfg.Timeout <= 0 {
		return Config{}, &Error{Message: fmt.Sprintf("timeout deve ser maior que zero (recebido %s)", cfg.Timeout)}
	}
	return cfg, nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return &Error{Message: fmt.Sprintf("não foi possível ler o arquivo de configuração %q", path), Err: err}
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}

	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return &Error{Message: fmt.Sprintf("arquivo de configuração inválido (%s)", path), Err: err}
	}
	if key := findAPIKey(raw); key != "" {
		return &Error{Message: fmt.Sprintf(
			"o arquivo de configuração %q contém o campo proibido %q.\nRemova o campo e defina a API key apenas na variável de ambiente YOUTUBE_API_KEY.",
			path, key,
		)}
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var file fileConfig
	if err := decoder.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return &Error{Message: fmt.Sprintf("arquivo de configuração inválido (%s)", path), Err: err}
	}

	if file.DefaultLang != nil {
		cfg.Lang = strings.TrimSpace(*file.DefaultLang)
	}
	if file.DefaultTimeout != nil {
		value := strings.TrimSpace(*file.DefaultTimeout)
		if value != "" {
			timeout, err := time.ParseDuration(value)
			if err != nil {
				return &Error{Message: fmt.Sprintf("default_timeout inválido em %q", path), Err: err}
			}
			cfg.Timeout = timeout
		}
	}
	if file.AllowPartial != nil {
		cfg.AllowPartial = *file.AllowPartial
	}
	if file.NoChapters != nil {
		cfg.NoChapters = *file.NoChapters
	}
	return nil
}

func applyEnv(cfg *Config, getenv func(string) string) error {
	if value := strings.TrimSpace(getenv("TYOT_LANG")); value != "" {
		cfg.Lang = value
	}
	if value := strings.TrimSpace(getenv("TYOT_TIMEOUT")); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return &Error{Message: fmt.Sprintf("TYOT_TIMEOUT inválido (%q)", value), Err: err}
		}
		cfg.Timeout = timeout
	}
	return nil
}

func applyFlags(cfg *Config, flags Flags) {
	if flags.LangSet {
		cfg.Lang = strings.TrimSpace(flags.Lang)
	}
	if flags.TimeoutSet {
		cfg.Timeout = flags.Timeout
	}
	if flags.AllowPartialSet {
		cfg.AllowPartial = flags.AllowPartial
	}
	if flags.NoChaptersSet {
		cfg.NoChapters = flags.NoChapters
	}
	cfg.Output = flags.Output
	cfg.Quiet = flags.Quiet
}

type fileConfig struct {
	DefaultLang    *string `yaml:"default_lang"`
	DefaultTimeout *string `yaml:"default_timeout"`
	AllowPartial   *bool   `yaml:"allow_partial"`
	NoChapters     *bool   `yaml:"no_chapters"`
}

func findAPIKey(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isAPIKeyKey(key) {
				return key
			}
			if found := findAPIKey(child); found != "" {
				return found
			}
		}
	case map[any]any:
		for key, child := range typed {
			name := fmt.Sprint(key)
			if isAPIKeyKey(name) {
				return name
			}
			if found := findAPIKey(child); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findAPIKey(child); found != "" {
				return found
			}
		}
	}
	return ""
}

func isAPIKeyKey(key string) bool {
	normalized := strings.ToLower(key)
	normalized = strings.NewReplacer("_", "", "-", "", " ", "", ".", "").Replace(normalized)
	return strings.Contains(normalized, "apikey") ||
		normalized == "youtubekey" ||
		normalized == "ytkey"
}
