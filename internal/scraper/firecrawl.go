package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FirecrawlClient wraps the Firecrawl /v1/scrape API.
type FirecrawlClient struct {
	apiKey string
	apiURL string
	http   *http.Client
}

// NewFirecrawlClient creates a new Firecrawl client.
func NewFirecrawlClient(apiKey, apiURL string) *FirecrawlClient {
	return &FirecrawlClient{
		apiKey: apiKey,
		apiURL: apiURL,
		http: &http.Client{
			// Generous client timeout so we can actually receive a response from
			// a long Firecrawl scrape (e.g. 120s timeout + retry). The real
			// scrape deadline is enforced via the `timeout` parameter below.
			Timeout: 180 * time.Second,
		},
	}
}

// FetchOptions configures a single call to FetchHTMLWithOptions.
type FetchOptions struct {
	// WaitFor is the delay in milliseconds to wait after page load before
	// extracting the DOM. Useful for JS-rendered pages. 0 to skip.
	WaitFor int
	// Timeout is the Firecrawl-side scrape timeout in milliseconds.
	// Firecrawl's API default is 60000ms. For heavy SPAs we often need more.
	// 0 means "use Firecrawl's default".
	Timeout int
	// Proxy selects the Firecrawl proxy tier: "basic", "auto", "enhanced".
	// Empty string lets Firecrawl pick the default.
	Proxy string
	// Retries is the number of additional attempts on transient failures
	// (HTTP 408, 429, 5xx, or API SCRAPE_TIMEOUT). 0 disables retries.
	Retries int
}

type firecrawlRequest struct {
	URL     string   `json:"url"`
	Formats []string `json:"formats"`
	WaitFor int      `json:"waitFor,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
	Proxy   string   `json:"proxy,omitempty"`
}

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		HTML     string         `json:"html"`
		Metadata map[string]any `json:"metadata"`
	} `json:"data"`
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

// FetchHTML fetches the HTML content of a URL using Firecrawl's scrape API.
// waitFor is the time in milliseconds to wait for JS rendering (0 to skip).
//
// This is a convenience wrapper around FetchHTMLWithOptions that applies
// sensible defaults (90s scrape timeout and one retry on transient errors).
func (fc *FirecrawlClient) FetchHTML(ctx context.Context, url string, waitFor int) (string, error) {
	return fc.FetchHTMLWithOptions(ctx, url, FetchOptions{
		WaitFor: waitFor,
		Timeout: 90000,
		Retries: 1,
	})
}

// FetchHTMLWithOptions fetches the HTML content of a URL with explicit control
// over timeout, proxy tier, and retry behavior. Use this for heavy SPA sites
// that time out under the default Firecrawl configuration.
func (fc *FirecrawlClient) FetchHTMLWithOptions(ctx context.Context, url string, opts FetchOptions) (string, error) {
	reqBody := firecrawlRequest{
		URL:     url,
		Formats: []string{"html"},
		WaitFor: opts.WaitFor,
		Timeout: opts.Timeout,
		Proxy:   opts.Proxy,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("firecrawl: marshal request: %w", err)
	}

	attempts := opts.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Fixed 2s backoff between attempts. Keeps total worst-case time
			// bounded so the outer search context can still complete.
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}

		html, retryable, err := fc.doScrape(ctx, jsonBody)
		if err == nil {
			return html, nil
		}

		lastErr = err
		if !retryable {
			return "", err
		}
		// Don't keep retrying if the caller's context is already gone.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	return "", lastErr
}

// doScrape performs a single HTTP request to the Firecrawl scrape endpoint.
// It returns (html, retryable, err). When retryable is true and err is
// non-nil, the caller may retry the same request.
func (fc *FirecrawlClient) doScrape(ctx context.Context, jsonBody []byte) (string, bool, error) {
	endpoint := fc.apiURL + "/v1/scrape"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", false, fmt.Errorf("firecrawl: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fc.apiKey)

	resp, err := fc.http.Do(req)
	if err != nil {
		// Network-level error: retry only if caller's context is still alive.
		if ctx.Err() != nil {
			return "", false, fmt.Errorf("firecrawl: request failed: %w", err)
		}
		return "", true, fmt.Errorf("firecrawl: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("firecrawl: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		httpErr := fmt.Errorf("firecrawl: HTTP %d: %s", resp.StatusCode, string(body))
		return "", isRetryableStatus(resp.StatusCode) || bodyIndicatesTimeout(body), httpErr
	}

	var fcResp firecrawlResponse
	if err := json.Unmarshal(body, &fcResp); err != nil {
		return "", false, fmt.Errorf("firecrawl: parse response: %w", err)
	}

	if !fcResp.Success {
		apiErr := fmt.Errorf("firecrawl: API error: %s", fcResp.Error)
		// SCRAPE_TIMEOUT and similar transient codes are worth retrying.
		return "", isRetryableCode(fcResp.Code), apiErr
	}

	return fcResp.Data.HTML, false, nil
}

func isRetryableStatus(code int) bool {
	if code == http.StatusRequestTimeout || code == http.StatusTooManyRequests {
		return true
	}
	if code >= 500 && code <= 599 {
		return true
	}
	return false
}

func isRetryableCode(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "SCRAPE_TIMEOUT", "RATE_LIMITED", "TIMEOUT":
		return true
	}
	return false
}

// bodyIndicatesTimeout handles the case where Firecrawl surfaces a
// SCRAPE_TIMEOUT inside a non-2xx response body (as seen in HTTP 408 replies).
func bodyIndicatesTimeout(body []byte) bool {
	return bytes.Contains(body, []byte("SCRAPE_TIMEOUT")) ||
		bytes.Contains(body, []byte("\"timeout\"")) ||
		bytes.Contains(body, []byte("timed out"))
}
