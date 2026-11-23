package playlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client
}

func TestNewRequiresKey(t *testing.T) {
	if _, err := New("  "); err == nil {
		t.Fatal("New() without key should fail")
	}
}

func TestListPaginatesAndDedups(t *testing.T) {
	var itemCalls int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/playlists"):
			if got := r.URL.Query().Get("id"); got != "PL1" {
				t.Errorf("playlists id = %q", got)
			}
			fmt.Fprint(w, `{"items":[{"snippet":{"title":"Minha Playlist"}}]}`)
		case strings.HasSuffix(r.URL.Path, "/playlistItems"):
			itemCalls++
			if r.URL.Query().Get("pageToken") == "" {
				fmt.Fprint(w, `{"items":[{"contentDetails":{"videoId":"aaaaaaaaaaa"}},{"contentDetails":{"videoId":"bbbbbbbbbbb"}}],"nextPageToken":"p2"}`)
				return
			}
			fmt.Fprint(w, `{"items":[{"contentDetails":{"videoId":"bbbbbbbbbbb"}},{"contentDetails":{"videoId":"ccccccccccc"}}]}`)
		default:
			http.NotFound(w, r)
		}
	})

	info, err := client.List(context.Background(), "PL1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if info.Title != "Minha Playlist" {
		t.Fatalf("Title = %q", info.Title)
	}
	want := []string{"aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"}
	if strings.Join(info.VideoIDs, ",") != strings.Join(want, ",") {
		t.Fatalf("VideoIDs = %v, want %v", info.VideoIDs, want)
	}
	if itemCalls != 2 {
		t.Fatalf("playlistItems calls = %d, want 2", itemCalls)
	}
}

func TestListNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[]}`)
	})
	if _, err := client.List(context.Background(), "PLmissing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListHTTP404IsNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":404,"message":"nope"}}`)
	})
	if _, err := client.List(context.Background(), "PL1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListEmpty(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/playlists") {
			fmt.Fprint(w, `{"items":[{"snippet":{"title":"Vazia"}}]}`)
			return
		}
		fmt.Fprint(w, `{"items":[]}`)
	})
	if _, err := client.List(context.Background(), "PL1"); !errors.Is(err, ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}
