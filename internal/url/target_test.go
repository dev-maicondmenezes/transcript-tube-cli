package url

import "testing"

func TestParseTarget(t *testing.T) {
	const pl = "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"

	tests := []struct {
		name  string
		input string
		want  Target
	}{
		{"video id", "dQw4w9WgXcQ", Target{KindVideo, "dQw4w9WgXcQ"}},
		{"video url", "https://youtu.be/dQw4w9WgXcQ", Target{KindVideo, "dQw4w9WgXcQ"}},
		{"video com list tem prioridade", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=" + pl, Target{KindVideo, "dQw4w9WgXcQ"}},
		{"playlist url", "https://www.youtube.com/playlist?list=" + pl, Target{KindPlaylist, pl}},
		{"playlist sem esquema", "youtube.com/playlist?list=" + pl, Target{KindPlaylist, pl}},
		{"playlist music", "https://music.youtube.com/playlist?list=" + pl, Target{KindPlaylist, pl}},
		{"watch sem v com list", "https://www.youtube.com/watch?list=" + pl, Target{KindPlaylist, pl}},
		{"playlist com espaços", "  https://www.youtube.com/playlist?list=" + pl + "  ", Target{KindPlaylist, pl}},
		{"playlist id cru PL", pl, Target{KindPlaylist, pl}},
		{"playlist id cru UU", "UUuAXFkgsw1L7xaCfnd5JJOw", Target{KindPlaylist, "UUuAXFkgsw1L7xaCfnd5JJOw"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseTarget(test.input)
			if err != nil {
				t.Fatalf("ParseTarget(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("ParseTarget(%q) = %+v, want %+v", test.input, got, test.want)
			}
		})
	}
}

func TestParseTargetRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"vazio", ""},
		{"id curto", "dQw4w9WgXc"},
		{"canal", "https://www.youtube.com/@somechannel"},
		{"host errado", "https://example.com/playlist?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"},
		{"esquema errado", "ftp://youtube.com/playlist?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"},
		{"playlist id curto demais", "https://www.youtube.com/playlist?list=PL123"},
		{"playlist sem list", "https://www.youtube.com/playlist"},
		{"id cru sem prefixo", "XXrAXtmErZgOeiKm4sgNOknGvNjby9efdf"},
		{"caminho desconhecido", "https://www.youtube.com/channel/UC123?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := ParseTarget(test.input); err == nil {
				t.Fatalf("ParseTarget(%q) = %+v, want error", test.input, got)
			}
		})
	}
}
