// Package url converte URLs do YouTube e IDs crus em um videoID validado.
package url

import (
	"fmt"
	neturl "net/url"
	"strings"

	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

// ParseVideoID aceita um ID cru ou uma URL do YouTube e devolve o ID
// canônico (11 caracteres [A-Za-z0-9_-]).
func ParseVideoID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("URL ou ID de vídeo vazio")
	}

	if core.ValidateVideoID(raw) == nil {
		return raw, nil
	}

	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}

	parsed, err := neturl.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("URL inválida: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL inválida: esquema %q não suportado", parsed.Scheme)
	}

	host := strings.ToLower(parsed.Hostname())
	var videoID string

	switch {
	case host == "youtu.be" || host == "www.youtu.be":
		videoID = firstPathSegment(parsed.Path)
	case host == "youtube.com" || strings.HasSuffix(host, ".youtube.com"):
		videoID, err = parseYouTubePath(parsed)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("URL não reconhecida: host %q", parsed.Hostname())
	}

	videoID = strings.TrimSpace(videoID)
	if err := core.ValidateVideoID(videoID); err != nil {
		return "", fmt.Errorf("ID de vídeo inválido %q: %w", videoID, err)
	}
	return videoID, nil
}

func parseYouTubePath(parsed *neturl.URL) (string, error) {
	path := strings.TrimSpace(parsed.Path)
	switch {
	case path == "/watch":
		return parsed.Query().Get("v"), nil
	case strings.HasPrefix(path, "/embed/"):
		return firstPathSegment(strings.TrimPrefix(path, "/embed")), nil
	case strings.HasPrefix(path, "/shorts/"):
		return firstPathSegment(strings.TrimPrefix(path, "/shorts")), nil
	case strings.HasPrefix(path, "/live/"):
		return firstPathSegment(strings.TrimPrefix(path, "/live")), nil
	case strings.HasPrefix(path, "/v/"):
		return firstPathSegment(strings.TrimPrefix(path, "/v")), nil
	case path == "" || path == "/":
		return parsed.Query().Get("v"), nil
	default:
		return "", fmt.Errorf("URL não reconhecida: caminho %q", parsed.Path)
	}
}

func firstPathSegment(path string) string {
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if segment = strings.TrimSpace(segment); segment != "" {
			return segment
		}
	}
	return ""
}
