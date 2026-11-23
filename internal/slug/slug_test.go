package slug

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"espaços e maiúsculas", "Aula 01 Introdução ao Go", "aula-01-introdução-ao-go"},
		{"espaços repetidos", "Hello    World", "hello-world"},
		{"tabs e quebras", "Hello\t\nWorld", "hello-world"},
		{"bordas aparadas", "  Hello World  ", "hello-world"},
		{"remove inválidos", `Go: "Tips/Tricks" <2026>?`, "go-tipstricks-2026"},
		{"preserva hífen existente", "Pré-requisitos do Curso", "pré-requisitos-do-curso"},
		{"aparas pontos e hífens", "...Título - ", "título"},
		{"hífen solto entre palavras", "Aula 01 - Introdução", "aula-01-introdução"},
		{"emoji preservado", "Go 🚀 rápido", "go-🚀-rápido"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Normalize(test.title, "fb"); got != test.want {
				t.Fatalf("Normalize(%q) = %q, want %q", test.title, got, test.want)
			}
		})
	}
}

func TestNormalizeFallback(t *testing.T) {
	for _, title := range []string{"", "   ", `/\:*?"<>|`, "..."} {
		if got := Normalize(title, "dQw4w9WgXcQ"); got != "dQw4w9WgXcQ" {
			t.Fatalf("Normalize(%q) = %q, want fallback", title, got)
		}
	}
}

func TestNormalizeTruncates(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "é"
	}
	got := Normalize(long, "fb")
	if n := len([]rune(got)); n != MaxRunes {
		t.Fatalf("runes = %d, want %d", n, MaxRunes)
	}
}

func TestUnique(t *testing.T) {
	used := map[string]struct{}{}
	got := []string{
		Unique("aula", used),
		Unique("aula", used),
		Unique("aula", used),
		Unique("outra", used),
	}
	want := []string{"aula", "aula-2", "aula-3", "outra"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Unique[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
