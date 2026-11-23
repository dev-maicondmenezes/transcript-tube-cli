// Package slug normaliza títulos do YouTube em nomes seguros de arquivo e
// pasta: minúsculos, com espaços entre palavras substituídos por "-".
package slug

import (
	"strconv"
	"strings"
	"unicode"
)

// MaxRunes limita o tamanho de um nome normalizado para ficar bem abaixo do
// limite de 255 bytes dos sistemas de arquivos comuns (já considerando a
// extensão e sufixos de desambiguação).
const MaxRunes = 100

// Normalize converte title em um nome de arquivo/pasta:
//
//   - letras em minúsculo (acentos são preservados);
//   - qualquer sequência de espaços vira um único "-" (e sequências de "-"
//     são reduzidas a um só);
//   - caracteres inválidos em nomes de arquivo (/ \ : * ? " < > | e controles)
//     são removidos;
//   - "-" e "." nas pontas são aparados.
//
// Se o resultado ficar vazio, devolve fallback.
func Normalize(title, fallback string) string {
	var b strings.Builder
	pendingDash := false

	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case unicode.IsSpace(r):
			pendingDash = b.Len() > 0
		case isForbidden(r):
			// removido sem inserir separador
		default:
			if pendingDash {
				b.WriteRune('-')
				pendingDash = false
			}
			b.WriteRune(r)
		}
	}

	name := collapseDashes(b.String())
	name = truncate(name, MaxRunes)
	name = strings.Trim(name, "-.")
	if name == "" {
		return fallback
	}
	return name
}

// collapseDashes reduz sequências de "-" a um único "-" (por exemplo, o
// título "Aula 01 - Intro" vira "aula-01-intro", não "aula-01---intro").
func collapseDashes(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if r == '-' {
			if prevDash {
				continue
			}
			prevDash = true
		} else {
			prevDash = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isForbidden(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}
	return strings.ContainsRune(`/\:*?"<>|`, r)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// Unique devolve base, ou base-2, base-3... caso o nome já esteja em used, e
// registra o nome escolhido em used. Os nomes já são minúsculos, então a
// comparação é naturalmente case-insensitive.
func Unique(base string, used map[string]struct{}) string {
	name := base
	for i := 2; ; i++ {
		if _, taken := used[name]; !taken {
			used[name] = struct{}{}
			return name
		}
		name = base + "-" + strconv.Itoa(i)
	}
}
