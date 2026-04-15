package scraper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewFirecrawlClient(t *testing.T) {
	client := NewFirecrawlClient("test-api-key", "https://api.firecrawl.dev")

	if client.apiKey != "test-api-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-api-key")
	}
	if client.apiURL != "https://api.firecrawl.dev" {
		t.Errorf("apiURL = %q, want %q", client.apiURL, "https://api.firecrawl.dev")
	}
	if client.http == nil {
		t.Fatal("http client is nil")
	}
	if client.http.Timeout != 60*time.Second {
		t.Errorf("http timeout = %v, want %v", client.http.Timeout, 60*time.Second)
	}
}

func TestFetchHTML_Success(t *testing.T) {
	const wantHTML = "<html><body><h1>Hello</h1></body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{
				HTML: wantHTML,
			},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	got, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantHTML {
		t.Errorf("got %q, want %q", got, wantHTML)
	}
}

func TestFetchHTML_SuccessWithWaitFor(t *testing.T) {
	var receivedBody firecrawlRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &receivedBody); err != nil {
			t.Errorf("unmarshaling request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{
				HTML: "<html></html>",
			},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody.WaitFor != 5000 {
		t.Errorf("waitFor = %d, want 5000", receivedBody.WaitFor)
	}
	if receivedBody.URL != "https://example.com" {
		t.Errorf("url = %q, want %q", receivedBody.URL, "https://example.com")
	}
	if len(receivedBody.Formats) != 1 || receivedBody.Formats[0] != "html" {
		t.Errorf("formats = %v, want [html]", receivedBody.Formats)
	}
}

func TestFetchHTML_NotIncludeWaitForWhenZero(t *testing.T) {
	var rawBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &rawBody); err != nil {
			t.Errorf("unmarshaling request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{
				HTML: "<html></html>",
			},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := rawBody["waitFor"]; exists {
		t.Errorf("waitFor should be omitted when zero, but found %v", rawBody["waitFor"])
	}
}

func TestFetchHTML_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "firecrawl: HTTP 500:"
	if len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", err.Error(), want)
	}
}

func TestFetchHTML_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&map[string]any{
			"success": false,
			"data":    map[string]any{},
			"error":   "rate limit exceeded",
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "firecrawl: API error: rate limit exceeded"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestFetchHTML_EmptyHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{
				HTML: "",
			},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	got, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestFetchHTML_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not json at all"))
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "firecrawl: parse response:"
	if len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", err.Error(), want)
	}
}

func TestFetchHTML_ContextCanceled(t *testing.T) {
	// Use a mutex-guarded flag so the handler can block until we cancel.
	var mu sync.Mutex
	handlerReached := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		handlerReached = true
		mu.Unlock()
		// Block long enough for the context to be canceled.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	client := NewFirecrawlClient("key", srv.URL)

	errCh := make(chan error, 1)
	go func() {
		_, err := client.FetchHTML(ctx, "https://example.com", 0)
		errCh <- err
	}()

	// Wait for the handler to be reached, then cancel.
	for {
		mu.Lock()
		reached := handlerReached
		mu.Unlock()
		if reached {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error after context cancellation, got nil")
		}
		if ctx.Err() != context.Canceled {
			t.Errorf("context error = %v, want context.Canceled", ctx.Err())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for FetchHTML to return after context cancellation")
	}
}

func TestFetchHTML_VerifyHeaders(t *testing.T) {
	const apiKey = "fc-test-key-12345"
	var gotContentType, gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")

		// Also verify the HTTP method and path.
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/scrape" {
			t.Errorf("path = %q, want /v1/scrape", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{
				HTML: "<html></html>",
			},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient(apiKey, srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "application/json")
	}
	wantAuth := "Bearer " + apiKey
	if gotAuth != wantAuth {
		t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
	}
}
