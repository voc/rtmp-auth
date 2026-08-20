package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/voc/rtmp-auth/storage"
	"github.com/voc/rtmp-auth/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	streamStore, err := store.NewStore(store.StoreConfig{
		Backend: "file",
		File:    store.FileBackendConfig{Path: t.TempDir() + "/state.pb"},
	})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return streamStore
}

func addTestStream(t *testing.T, streamStore *store.Store) {
	t.Helper()

	err := streamStore.AddStream(&storage.Stream{
		Application: "live",
		Name:        "camera",
		AuthKey:     "secret",
	})
	if err != nil {
		t.Fatalf("add stream: %v", err)
	}
}

func streamIsActive(t *testing.T, streamStore *store.Store) bool {
	t.Helper()

	state, err := streamStore.Get()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(state.Streams) != 1 {
		t.Fatalf("got %d streams, want 1", len(state.Streams))
	}
	return state.Streams[0].Active
}

func TestPublishHandlerActivatesAuthorizedStream(t *testing.T) {
	streamStore := newTestStore(t)
	addTestStream(t, streamStore)

	form := url.Values{"app": {"live"}, "name": {"camera"}, "auth": {"secret"}, "call": {"publish"}}
	request := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	PublishHandler(streamStore)(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "0" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "0")
	}
	if !streamIsActive(t, streamStore) {
		t.Fatal("stream is inactive after publish")
	}
}

func TestPublishHandlerRejectsUnauthorizedStream(t *testing.T) {
	streamStore := newTestStore(t)
	addTestStream(t, streamStore)

	form := url.Values{"app": {"live"}, "name": {"camera"}, "auth": {"incorrect"}, "call": {"publish"}}
	request := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	PublishHandler(streamStore)(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if streamIsActive(t, streamStore) {
		t.Fatal("stream is active after unauthorized publish")
	}
}

func TestUnpublishHandlerDeactivatesStream(t *testing.T) {
	streamStore := newTestStore(t)
	addTestStream(t, streamStore)
	state, err := streamStore.Get()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if !streamStore.SetActive(state.Streams[0].Id) {
		t.Fatal("activate stream")
	}

	form := url.Values{"app": {"live"}, "name": {"camera"}, "call": {"unpublish"}}
	request := httptest.NewRequest(http.MethodPost, "/unpublish", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	UnpublishHandler(streamStore)(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "0" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "0")
	}
	if streamIsActive(t, streamStore) {
		t.Fatal("stream is active after unpublish")
	}
}
