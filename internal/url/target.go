package url

import (
	neturl "net/url"
	"regexp"
	"strings"
)

// Kind diferencia o tipo de recurso apontado pela entrada do usuário.
type Kind int

const (
	// KindVideo é um vídeo único.
	KindVideo Kind = iota
	// KindPlaylist é uma playlist.
	KindPlaylist
)

// Target é o recurso resolvido a partir de uma URL ou ID.
type Target struct {
	Kind Kind
	ID   string
}

var (
	playlistIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{10,64}$`)
	rawPlaylistPrefix = regexp.MustCompile(`^(PL|UU|FL|OLAK5uy_)`)
)

// ParseTarget resolve raw em um vídeo ou em uma playlist.
//
// Regras:
//   - tudo o que ParseVideoID aceita continua sendo um vídeo, inclusive
//     watch?v=ID&list=PL..., em que o vídeo explícito tem prioridade;
//   - youtube.com/playlist?list=ID e watch?list=ID (sem v) são playlists;
//   - um ID cru com prefixo PL, UU, FL ou OLAK5uy_ também é uma playlist.
//
// Quando a entrada não é nem vídeo nem playlist, devolve o erro de
// ParseVideoID.
func ParseTarget(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)

	videoID, videoErr := ParseVideoID(raw)
	if videoErr == nil {
		return Target{Kind: KindVideo, ID: videoID}, nil
	}

	if rawPlaylistPrefix.MatchString(raw) && playlistIDPattern.MatchString(raw) {
		return Target{Kind: KindPlaylist, ID: raw}, nil
	}
	if id := playlistIDFromURL(raw); id != "" {
		return Target{Kind: KindPlaylist, ID: id}, nil
	}
	return Target{}, videoErr
}

func playlistIDFromURL(raw string) string {
	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}
	parsed, err := neturl.Parse(candidate)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "youtube.com" && !strings.HasSuffix(host, ".youtube.com") {
		return ""
	}
	switch strings.TrimRight(parsed.Path, "/") {
	case "/playlist", "/watch", "":
	default:
		return ""
	}

	id := strings.TrimSpace(parsed.Query().Get("list"))
	if !playlistIDPattern.MatchString(id) {
		return ""
	}
	return id
}
