package url

import "testing"

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bare id", "dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"watch", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"watch with params", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s&list=PL123", "dQw4w9WgXcQ"},
		{"youtu.be", "https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"youtu.be with time", "https://youtu.be/dQw4w9WgXcQ?t=42", "dQw4w9WgXcQ"},
		{"mobile subdomain", "https://m.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"music subdomain", "https://music.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"other subdomain", "https://gaming.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"embed", "https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"shorts", "https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"live", "https://www.youtube.com/live/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"old v path", "https://www.youtube.com/v/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"without scheme", "www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"surrounding spaces", "  https://youtu.be/dQw4w9WgXcQ  ", "dQw4w9WgXcQ"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseVideoID(test.input)
			if err != nil {
				t.Fatalf("ParseVideoID(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("ParseVideoID(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestParseVideoIDRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"blank", "   "},
		{"short id", "dQw4w9WgXc"},
		{"long id", "dQw4w9WgXcQx"},
		{"playlist", "https://www.youtube.com/playlist?list=PL123"},
		{"channel", "https://www.youtube.com/@somechannel"},
		{"channel path", "https://www.youtube.com/channel/UC123"},
		{"watch without v", "https://www.youtube.com/watch?list=PL123"},
		{"wrong host", "https://example.com/watch?v=dQw4w9WgXcQ"},
		{"unsupported scheme", "ftp://youtube.com/watch?v=dQw4w9WgXcQ"},
		{"empty youtu.be", "https://youtu.be/"},
		{"invalid id in watch", "https://www.youtube.com/watch?v=short"},
		{"not a url", "not a url"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := ParseVideoID(test.input); err == nil {
				t.Fatalf("ParseVideoID(%q) = %q, want error", test.input, got)
			}
		})
	}
}
