// Package playlist lista os vídeos de uma playlist do YouTube usando a
// YouTube Data API v3 (a mesma API key usada pelo core para metadados).
package playlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// ErrNotFound indica que a playlist não existe ou é privada.
var ErrNotFound = errors.New("playlist não encontrada")

// ErrEmpty indica que a playlist não possui nenhum vídeo listável.
var ErrEmpty = errors.New("playlist sem vídeos")

// Info descreve uma playlist e a ordem dos seus vídeos.
type Info struct {
	ID       string
	Title    string
	VideoIDs []string
}

// Lister é o seam de teste do CLI para listar playlists.
type Lister interface {
	List(ctx context.Context, playlistID string) (*Info, error)
}

const pageSize = 50

// Client é um Lister baseado na YouTube Data API v3.
type Client struct {
	service    *youtube.Service
	httpClient *http.Client
	baseURL    string
}

// Option customiza um Client. Existe principalmente para testes.
type Option func(*Client)

// WithHTTPClient injeta um http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

// WithBaseURL sobrescreve o endpoint da API (por exemplo, um httptest.Server).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// New cria um Client autenticado com apiKey.
func New(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("playlist: chave da API é obrigatória")
	}

	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}

	clientOptions := []option.ClientOption{option.WithAPIKey(apiKey)}
	if client.httpClient != nil {
		clientOptions = append(clientOptions, option.WithHTTPClient(client.httpClient))
	}
	service, err := youtube.NewService(context.Background(), clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("playlist: criar cliente do YouTube: %w", err)
	}
	if client.baseURL != "" {
		service.BasePath = strings.TrimRight(client.baseURL, "/") + "/"
	}
	client.service = service
	return client, nil
}

// List devolve o título e todos os IDs de vídeo da playlist, na ordem em que
// aparecem, paginando a API. IDs repetidos são mantidos apenas uma vez.
func (c *Client) List(ctx context.Context, playlistID string) (*Info, error) {
	title, err := c.title(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	info := &Info{ID: playlistID, Title: title}
	seen := map[string]struct{}{}
	pageToken := ""
	for {
		call := c.service.PlaylistItems.
			List([]string{"contentDetails"}).
			PlaylistId(playlistID).
			MaxResults(pageSize).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, mapAPIError(err)
		}
		for _, item := range response.Items {
			if item.ContentDetails == nil {
				continue
			}
			id := strings.TrimSpace(item.ContentDetails.VideoId)
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			info.VideoIDs = append(info.VideoIDs, id)
		}
		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	if len(info.VideoIDs) == 0 {
		return nil, ErrEmpty
	}
	return info, nil
}

func (c *Client) title(ctx context.Context, playlistID string) (string, error) {
	response, err := c.service.Playlists.
		List([]string{"snippet"}).
		Id(playlistID).
		Context(ctx).
		Do()
	if err != nil {
		return "", mapAPIError(err)
	}
	if len(response.Items) == 0 || response.Items[0].Snippet == nil {
		return "", ErrNotFound
	}
	return response.Items[0].Snippet.Title, nil
}

func mapAPIError(err error) error {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return fmt.Errorf("listar playlist: %w", err)
}
