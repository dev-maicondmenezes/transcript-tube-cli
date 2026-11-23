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
//
// Quando Kind é KindVideo e a URL também carrega uma playlist
// (watch?v=ID&list=PL... ou youtu.be/ID?list=PL...), PlaylistID traz o ID da
// playlist para que o chamador decida entre o vídeo e a playlist inteira.
type Target struct {
	Kind       Kind
	ID         string
	PlaylistID string
}

var (
	playlistIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{10,64}$`)
	rawPlaylistPrefix = regexp.MustCompile(`^(PL|UU|FL|OLAK5uy_)`)
)

// ParseTarget resolve raw em um vídeo ou em uma playlist.
//
// Regras:
//   - tudo o que ParseVideoID aceita é um vídeo; se a URL também tiver um
//     parâmetro list= válido, o ID da playlist vem em Target.PlaylistID;
//   - youtube.com/playlist?list=ID e watch?list=ID (sem v) são playlists;
//   - um ID cru com prefixo PL, UU, FL ou OLAK5uy_ também é uma playlist.
//
// Quando a entrada não é nem vídeo nem playlist, devolve o erro de
// ParseVideoID.
func ParseTarget(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)

	videoID, videoErr := ParseVideoID(raw)
	if videoErr == nil {
		return Target{Kind: KindVideo, ID: videoID, PlaylistID: listParam(raw)}, nil
	}

	if rawPlaylistPrefix.MatchString(raw) && playlistIDPattern.MatchString(raw) {
		return Target{Kind: KindPlaylist, ID: raw}, nil
	}
	if id := playlistIDFromURL(raw); id != "" {
		return Target{Kind: KindPlaylist, ID: id}, nil
	}
	return Target{}, videoErr
}

func parseHTTPURL(raw string) *neturl.URL {
	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}
	parsed, err := neturl.Parse(candidate)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}
	return parsed
}

func isYouTubeHost(host string) bool {
	return host == "youtube.com" || strings.HasSuffix(host, ".youtube.com")
}

// listParam devolve o list= válido de uma URL de vídeo do YouTube (inclusive
// youtu.be), ou "" se não houver.
func listParam(raw string) string {
	parsed := parseHTTPURL(raw)
	if parsed == nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	if !isYouTubeHost(host) && host != "youtu.be" && host != "www.youtu.be" {
		return ""
	}
	id := strings.TrimSpace(parsed.Query().Get("list"))
	if !playlistIDPattern.MatchString(id) {
		return ""
	}
	return id
}

func playlistIDFromURL(raw string) string {
	parsed := parseHTTPURL(raw)
	if parsed == nil || !isYouTubeHost(strings.ToLower(parsed.Hostname())) {
		return ""
	}
	switch strings.TrimRight(parsed.Path, "/") {
	case "/playlist", "/watch", "":
	default:
		return ""
	}
	return listParam(raw)
}
