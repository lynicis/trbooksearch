package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
			Timeout: 60 * time.Second,
		},
	}
}

type firecrawlRequest struct {
	URL     string   `json:"url"`
	Formats []string `json:"formats"`
	WaitFor int      `json:"waitFor,omitempty"`
}

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		HTML     string         `json:"html"`
		Metadata map[string]any `json:"metadata"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// FetchHTML fetches the HTML content of a URL using Firecrawl's scrape API.
// waitFor is the time in milliseconds to wait for JS rendering (0 to skip).
func (fc *FirecrawlClient) FetchHTML(ctx context.Context, url string, waitFor int) (string, error) {
	reqBody := firecrawlRequest{
		URL:     url,
		Formats: []string{"html"},
	}
	if waitFor > 0 {
		reqBody.WaitFor = waitFor
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("firecrawl: marshal request: %w", err)
	}

	endpoint := fc.apiURL + "/v1/scrape"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("firecrawl: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fc.apiKey)

	resp, err := fc.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("firecrawl: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("firecrawl: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firecrawl: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var fcResp firecrawlResponse
	if err := json.Unmarshal(body, &fcResp); err != nil {
		return "", fmt.Errorf("firecrawl: parse response: %w", err)
	}

	if !fcResp.Success {
		return "", fmt.Errorf("firecrawl: API error: %s", fcResp.Error)
	}

	if fcResp.Data.HTML == "" {
		return "", nil
	}

	return fcResp.Data.HTML, nil
}
