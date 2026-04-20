package scraper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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
	if client.http.Timeout != 180*time.Second {
		t.Errorf("http timeout = %v, want %v", client.http.Timeout, 180*time.Second)
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
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	// Retries: 0 keeps the test fast and focused on the HTTP error surface.
	_, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{Retries: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "firecrawl: HTTP 500:"
	if len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", err.Error(), want)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (retries disabled)", got)
	}
}

func TestFetchHTML_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&map[string]any{
			"success": false,
			"data":    map[string]any{},
			// No `code` field -> not retryable, avoids slow retry path in tests.
			"error": "rate limit exceeded",
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

func TestFetchHTML_SendsTimeoutAndProxy(t *testing.T) {
	var body firecrawlRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{HTML: "<html></html>"},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{
		WaitFor: 8000,
		Timeout: 120000,
		Proxy:   "enhanced",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.Timeout != 120000 {
		t.Errorf("timeout = %d, want 120000", body.Timeout)
	}
	if body.Proxy != "enhanced" {
		t.Errorf("proxy = %q, want %q", body.Proxy, "enhanced")
	}
	if body.WaitFor != 8000 {
		t.Errorf("waitFor = %d, want 8000", body.WaitFor)
	}
}

func TestFetchHTML_OmitsTimeoutAndProxyWhenUnset(t *testing.T) {
	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Errorf("unmarshal body: %v", err)
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{HTML: "<html></html>"},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := raw["timeout"]; ok {
		t.Errorf("timeout should be omitted when zero, got %v", raw["timeout"])
	}
	if _, ok := raw["proxy"]; ok {
		t.Errorf("proxy should be omitted when empty, got %v", raw["proxy"])
	}
	if _, ok := raw["waitFor"]; ok {
		t.Errorf("waitFor should be omitted when zero, got %v", raw["waitFor"])
	}
}

func TestFetchHTML_RetryOn408(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			_, _ = w.Write([]byte(`{"success":false,"code":"SCRAPE_TIMEOUT","error":"timed out"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{HTML: "<html>ok</html>"},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	html, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{Retries: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<html>ok</html>" {
		t.Errorf("html = %q, want %q", html, "<html>ok</html>")
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestFetchHTML_RetryOnScrapeTimeoutCode(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// 200 OK with success=false and a retryable code inside.
			_ = json.NewEncoder(w).Encode(&map[string]any{
				"success": false,
				"code":    "SCRAPE_TIMEOUT",
				"error":   "timed out",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{HTML: "<html>retried</html>"},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	html, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{Retries: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<html>retried</html>" {
		t.Errorf("html = %q, want retried", html)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestFetchHTML_NoRetryOn401(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{Retries: 3})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (non-retryable 401)", got)
	}
}

func TestFetchHTML_RetryExhaustion(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(`{"success":false,"code":"SCRAPE_TIMEOUT","error":"timed out"}`))
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTMLWithOptions(context.Background(), "https://example.com", FetchOptions{Retries: 1})
	if err == nil {
		t.Fatal("expected error after retry exhaustion, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2 (initial + 1 retry)", got)
	}
}

func TestFetchHTML_BackwardsCompatSendsTimeout(t *testing.T) {
	var body firecrawlRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(firecrawlResponse{
			Success: true,
			Data: struct {
				HTML     string         `json:"html"`
				Metadata map[string]any `json:"metadata"`
			}{HTML: "<html></html>"},
		})
	}))
	defer srv.Close()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTML(context.Background(), "https://example.com", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The legacy FetchHTML path should now apply a generous timeout so heavy
	// SPA pages do not time out on Firecrawl's default.
	if body.Timeout != 90000 {
		t.Errorf("timeout = %d, want 90000 (legacy default)", body.Timeout)
	}
	if body.WaitFor != 5000 {
		t.Errorf("waitFor = %d, want 5000", body.WaitFor)
	}
}

func TestFetchHTML_RetryRespectsContext(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(`{"success":false,"code":"SCRAPE_TIMEOUT"}`))
	}))
	defer srv.Close()

	// Context cancels before the 2s backoff elapses, so no retry should fire.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	client := NewFirecrawlClient("key", srv.URL)
	_, err := client.FetchHTMLWithOptions(ctx, "https://example.com", FetchOptions{Retries: 5})
	if err == nil {
		t.Fatal("expected error (context deadline), got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (context killed retry backoff)", got)
	}
}
