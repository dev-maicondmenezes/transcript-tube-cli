package url

import "testing"

const testPL = "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf"

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Target
	}{
		{"video id", "dQw4w9WgXcQ", Target{Kind: KindVideo, ID: "dQw4w9WgXcQ"}},
		{"video url", "https://youtu.be/dQw4w9WgXcQ", Target{Kind: KindVideo, ID: "dQw4w9WgXcQ"}},
		{
			"watch com list traz a playlist embutida",
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=" + testPL,
			Target{Kind: KindVideo, ID: "dQw4w9WgXcQ", PlaylistID: testPL},
		},
		{
			"youtu.be com list traz a playlist embutida",
			"https://youtu.be/dQw4w9WgXcQ?list=" + testPL,
			Target{Kind: KindVideo, ID: "dQw4w9WgXcQ", PlaylistID: testPL},
		},
		{
			"list inválido é ignorado",
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123",
			Target{Kind: KindVideo, ID: "dQw4w9WgXcQ"},
		},
		{"playlist url", "https://www.youtube.com/playlist?list=" + testPL, Target{Kind: KindPlaylist, ID: testPL}},
		{"playlist sem esquema", "youtube.com/playlist?list=" + testPL, Target{Kind: KindPlaylist, ID: testPL}},
		{"playlist music", "https://music.youtube.com/playlist?list=" + testPL, Target{Kind: KindPlaylist, ID: testPL}},
		{"watch sem v com list", "https://www.youtube.com/watch?list=" + testPL, Target{Kind: KindPlaylist, ID: testPL}},
		{"playlist com espaços", "  https://www.youtube.com/playlist?list=" + testPL + "  ", Target{Kind: KindPlaylist, ID: testPL}},
		{"playlist id cru PL", testPL, Target{Kind: KindPlaylist, ID: testPL}},
		{"playlist id cru UU", "UUuAXFkgsw1L7xaCfnd5JJOw", Target{Kind: KindPlaylist, ID: "UUuAXFkgsw1L7xaCfnd5JJOw"}},
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
		{"host errado", "https://example.com/playlist?list=" + testPL},
		{"esquema errado", "ftp://youtube.com/playlist?list=" + testPL},
		{"playlist id curto demais", "https://www.youtube.com/playlist?list=PL123"},
		{"playlist sem list", "https://www.youtube.com/playlist"},
		{"id cru sem prefixo", "XXrAXtmErZgOeiKm4sgNOknGvNjby9efdf"},
		{"caminho desconhecido", "https://www.youtube.com/channel/UC123?list=" + testPL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := ParseTarget(test.input); err == nil {
				t.Fatalf("ParseTarget(%q) = %+v, want error", test.input, got)
			}
		})
	}
}
