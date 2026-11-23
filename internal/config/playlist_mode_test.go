package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPlaylistModeDefaultsToSingle(t *testing.T) {
	cfg, err := Load(Flags{ConfigPath: filepath.Join(t.TempDir(), "nope.yaml"), ConfigPathSet: true}, getenv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PlaylistMode != PlaylistModeSingle {
		t.Fatalf("PlaylistMode = %q, want %q", cfg.PlaylistMode, PlaylistModeSingle)
	}
}

func TestPlaylistModePrecedence(t *testing.T) {
	path := writeConfig(t, "playlist_mode: split\n")
	base := Flags{ConfigPath: path, ConfigPathSet: true}

	tests := []struct {
		name  string
		flags Flags
		env   map[string]string
		want  string
	}{
		{"arquivo", base, nil, PlaylistModeSplit},
		{"env sobrepõe arquivo", base, map[string]string{"TYOT_PLAYLIST_MODE": "single"}, PlaylistModeSingle},
		{
			"flag sobrepõe env",
			Flags{ConfigPath: path, ConfigPathSet: true, PlaylistMode: "single", PlaylistModeSet: true},
			map[string]string{"TYOT_PLAYLIST_MODE": "split"},
			PlaylistModeSingle,
		},
		{
			"valor normalizado",
			Flags{ConfigPath: path, ConfigPathSet: true, PlaylistMode: " SPLIT ", PlaylistModeSet: true},
			nil,
			PlaylistModeSplit,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load(test.flags, getenv(test.env))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PlaylistMode != test.want {
				t.Fatalf("PlaylistMode = %q, want %q", cfg.PlaylistMode, test.want)
			}
		})
	}
}

func TestPlaylistModeInvalid(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.yaml")
	tests := []struct {
		name  string
		flags Flags
		env   map[string]string
	}{
		{"flag", Flags{ConfigPath: missing, ConfigPathSet: true, PlaylistMode: "zip", PlaylistModeSet: true}, nil},
		{"env", Flags{ConfigPath: missing, ConfigPathSet: true}, map[string]string{"TYOT_PLAYLIST_MODE": "zip"}},
		{"arquivo", Flags{ConfigPath: writeConfig(t, "playlist_mode: zip\n"), ConfigPathSet: true}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(test.flags, getenv(test.env))
			var cfgErr *Error
			if !errors.As(err, &cfgErr) {
				t.Fatalf("err = %v, want *config.Error", err)
			}
		})
	}
}
