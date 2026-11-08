package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateEnv(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "present", value: "abc"},
		{name: "present with spaces", value: "  abc  "},
		{name: "empty", value: "", wantErr: true},
		{name: "spaces", value: "   ", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateEnv(func(key string) string {
				if key == "YOUTUBE_API_KEY" {
					return test.value
				}
				return ""
			})
			if test.wantErr && err == nil {
				t.Fatal("ValidateEnv() error = nil, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateEnv() error = %v", err)
			}
			if test.wantErr {
				var envErr *EnvError
				if !errors.As(err, &envErr) {
					t.Fatalf("error type = %T, want *EnvError", err)
				}
				if !strings.Contains(envErr.Message, "YOUTUBE_API_KEY") {
					t.Fatalf("message = %q, want YOUTUBE_API_KEY", envErr.Message)
				}
				if !strings.Contains(envErr.Message, "console.cloud.google.com") {
					t.Fatalf("message = %q, want console URL", envErr.Message)
				}
				if strings.Contains(envErr.Message, test.value) && test.value != "" {
					t.Fatalf("message leaked the env value")
				}
			}
		})
	}
}

func TestIsHelpType(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "help long", args: []string{"--help"}, want: true},
		{name: "help short", args: []string{"-h"}, want: true},
		{name: "version long", args: []string{"--version"}, want: true},
		{name: "version short", args: []string{"-V"}, want: true},
		{name: "help command", args: []string{"help"}, want: true},
		{name: "help command args", args: []string{"help", "completion"}, want: true},
		{name: "completion", args: []string{"completion", "bash"}, want: true},
		{name: "short cluster", args: []string{"-qV"}, want: true},
		{name: "help equals true", args: []string{"--help=true"}, want: true},
		{name: "help equals false", args: []string{"--help=false"}, want: false},
		{name: "no args", args: nil, want: false},
		{name: "url", args: []string{"dQw4w9WgXcQ"}, want: false},
		{name: "output flag", args: []string{"-o", "file.md"}, want: false},
		{name: "unknown flag", args: []string{"--bogus"}, want: false},
		{name: "after terminator", args: []string{"--", "--help"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsHelpType(test.args); got != test.want {
				t.Fatalf("IsHelpType(%v) = %v, want %v", test.args, got, test.want)
			}
		})
	}
}
