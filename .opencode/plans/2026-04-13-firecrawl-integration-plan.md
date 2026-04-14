# Firecrawl Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Firecrawl as an optional global scraping backend, plus dolap.com and gardrops.com scrapers.

**Architecture:** When `--firecrawl` flag is set and API key is configured, all scrapers use Firecrawl's `/v1/scrape` API to fetch HTML instead of go-rod/HTTP. Two new scrapers (dolap.com, gardrops.com) are only available in Firecrawl mode. The goquery parsing layer is unchanged.

**Tech Stack:** Go, Firecrawl REST API (raw HTTP, no SDK), `gopkg.in/yaml.v3` for config

---

### Task 1: Add yaml dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add gopkg.in/yaml.v3**

Run:
```bash
go get gopkg.in/yaml.v3
```

**Step 2: Verify**

Run: `go mod tidy`
Expected: no errors

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add yaml.v3 dependency for config file support"
```

---

### Task 2: Create config package

**Files:**
- Create: `internal/config/config.go`

**Step 1: Create the config package**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Firecrawl FirecrawlConfig `yaml:"firecrawl"`
}

// FirecrawlConfig holds Firecrawl-specific settings.
type FirecrawlConfig struct {
	APIKey string `yaml:"api_key"`
	APIURL string `yaml:"api_url"`
}

const appName = "trbooksearch"

// Load reads the config file from the XDG config directory.
// Returns a zero Config if the file doesn't exist (not an error).
func Load() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("config dir: %w", err)
	}

	path := filepath.Join(configDir, appName, "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil // no config file is fine
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Default API URL
	if cfg.Firecrawl.APIURL == "" {
		cfg.Firecrawl.APIURL = "https://api.firecrawl.dev"
	}

	return cfg, nil
}

// ConfigPath returns the expected config file path for display in error messages.
func ConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "~/.config/" + appName + "/config.yaml"
	}
	return filepath.Join(configDir, appName, "config.yaml")
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/config/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add config package for reading ~/.config/trbooksearch/config.yaml"
```

---

### Task 3: Create Firecrawl client

**Files:**
- Create: `internal/scraper/firecrawl.go`

**Step 1: Create the Firecrawl HTTP client**

```go
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
		HTML     string            `json:"html"`
		Metadata map[string]any    `json:"metadata"`
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
	defer resp.Body.Close()

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
```

**Step 2: Verify it compiles**

Run: `go build ./internal/scraper/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/scraper/firecrawl.go
git commit -m "feat: add Firecrawl client for HTML fetching via /v1/scrape API"
```

---

### Task 4: Add SetFirecrawl to Scraper interface and update all existing scrapers

**Files:**
- Modify: `internal/scraper/scraper.go:57-66` (Scraper interface)
- Modify: `internal/scraper/sites/nadirkitap.go:22-31`
- Modify: `internal/scraper/sites/kitapyurdu.go:15-26`
- Modify: `internal/scraper/sites/trendyol.go:21-30`
- Modify: `internal/scraper/sites/hepsiburada.go:21-30`
- Modify: `internal/scraper/sites/amazon.go:21-30`
- Modify: `internal/scraper/sites/letgo.go:20-29`

**Step 1: Add SetFirecrawl to the Scraper interface**

In `internal/scraper/scraper.go`, add to the Scraper interface:

```go
type Scraper interface {
	Name() string
	Search(ctx context.Context, query string, searchType SearchType) ([]BookResult, error)
	SiteCategory() Category
	SetBrowser(browser *rod.Browser)
	SetFirecrawl(client *FirecrawlClient)
}
```

**Step 2: Add firecrawl field and SetFirecrawl method to each existing scraper**

For each of the 6 scrapers, add:
- A `firecrawl *scraper.FirecrawlClient` field to the struct
- A `SetFirecrawl` method

Example for Nadirkitap (repeat pattern for all 6):

```go
type Nadirkitap struct {
	limit     int
	browser   *rod.Browser
	firecrawl *scraper.FirecrawlClient
}

func (n *Nadirkitap) SetFirecrawl(client *scraper.FirecrawlClient) { n.firecrawl = client }
```

**Step 3: Update each scraper's Search method to use Firecrawl when available**

In each scraper's `Search` method, replace the fetch call with a conditional:

For **nadirkitap.go** (currently uses `scraper.FetchPage`):
```go
var pageHTML string
var err error
if n.firecrawl != nil {
	pageHTML, err = n.firecrawl.FetchHTML(ctx, searchURL, 0)
} else {
	pageHTML, err = scraper.FetchPage(ctx, n.browser, searchURL)
}
```

For **kitapyurdu.go** (currently uses `FetchPageWithWait`):
```go
var pageHTML string
var err error
if k.firecrawl != nil {
	pageHTML, err = k.firecrawl.FetchHTML(ctx, searchURL, 5000)
} else {
	pageHTML, err = scraper.FetchPageWithWait(ctx, k.browser, searchURL, ".ky-product")
}
```

For **trendyol.go** (currently uses `FetchPageWithWait`):
```go
var pageHTML string
var err error
if t.firecrawl != nil {
	pageHTML, err = t.firecrawl.FetchHTML(ctx, searchURL, 5000)
} else {
	pageHTML, err = scraper.FetchPageWithWait(ctx, t.browser, searchURL, "a.product-card")
}
```

For **hepsiburada.go** (currently uses `FetchPageWithWait`):
```go
var pageHTML string
var err error
if h.firecrawl != nil {
	pageHTML, err = h.firecrawl.FetchHTML(ctx, searchURL, 5000)
} else {
	pageHTML, err = scraper.FetchPageWithWait(ctx, h.browser, searchURL, `[class*="productCard-module_article"]`)
}
```

For **amazon.go** (currently uses `FetchPage`):
```go
var pageHTML string
var err error
if a.firecrawl != nil {
	pageHTML, err = a.firecrawl.FetchHTML(ctx, searchURL, 5000)
} else {
	pageHTML, err = scraper.FetchPage(ctx, a.browser, searchURL)
}
```

For **letgo.go** (currently uses `FetchPage`):
```go
var pageHTML string
var err error
if l.firecrawl != nil {
	pageHTML, err = l.firecrawl.FetchHTML(ctx, searchURL, 0)
} else {
	pageHTML, err = scraper.FetchPage(ctx, l.browser, searchURL)
}
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: no errors

**Step 5: Commit**

```bash
git add internal/scraper/scraper.go internal/scraper/sites/
git commit -m "feat: add SetFirecrawl to Scraper interface, all scrapers support dual fetch mode"
```

---

### Task 5: Update engine to support Firecrawl mode

**Files:**
- Modify: `internal/engine/engine.go:30-37` (Engine struct)
- Modify: `internal/engine/engine.go:50-75` (Search method)

**Step 1: Add FirecrawlClient to Engine**

```go
type Engine struct {
	scrapers  []scraper.Scraper
	firecrawl *scraper.FirecrawlClient
}

func NewEngine(firecrawl *scraper.FirecrawlClient, scrapers ...scraper.Scraper) *Engine {
	return &Engine{scrapers: scrapers, firecrawl: firecrawl}
}
```

**Step 2: Update Search method to conditionally launch browser**

Replace the browser launch block in `Search()`:

```go
func (e *Engine) Search(ctx context.Context, query string, searchType scraper.SearchType, statusCh chan<- SiteStatus) SearchResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []scraper.BookResult
		errors  []SearchError
	)

	if e.firecrawl != nil {
		// Firecrawl mode: pass client to all scrapers, no browser needed.
		for _, s := range e.scrapers {
			s.SetFirecrawl(e.firecrawl)
		}
	} else {
		// Browser mode: launch shared headless browser.
		browser, browserErr := scraper.LaunchBrowser()
		if browserErr != nil {
			for _, s := range e.scrapers {
				errors = append(errors, SearchError{Site: s.Name(), Err: browserErr})
				if statusCh != nil {
					statusCh <- SiteStatus{Site: s.Name(), Status: "error", Err: browserErr}
				}
			}
			if statusCh != nil {
				close(statusCh)
			}
			return SearchResult{Errors: errors}
		}
		defer browser.MustClose()

		for _, s := range e.scrapers {
			s.SetBrowser(browser)
		}
	}

	// ... rest of parallel dispatch is unchanged ...
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors (may need to update callers of NewEngine first, see Task 7)

**Step 4: Commit**

```bash
git add internal/engine/engine.go
git commit -m "feat: engine supports Firecrawl mode, skips browser launch when client provided"
```

---

### Task 6: Create dolap.com and gardrops.com scrapers

**Files:**
- Create: `internal/scraper/sites/dolap.go`
- Create: `internal/scraper/sites/gardrops.go`

**Step 1: Create dolap.go**

```go
package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"

	"trbooksearch/internal/scraper"
)

const (
	dolapBaseURL   = "https://dolap.com"
	dolapSearchURL = dolapBaseURL + "/ara?q=%s"
)

type Dolap struct {
	limit     int
	browser   *rod.Browser
	firecrawl *scraper.FirecrawlClient
}

func NewDolap(limit int) *Dolap {
	return &Dolap{limit: limit}
}

func (d *Dolap) SetBrowser(browser *rod.Browser)              { d.browser = browser }
func (d *Dolap) SetFirecrawl(client *scraper.FirecrawlClient) { d.firecrawl = client }
func (d *Dolap) Name() string                                 { return "dolap.com" }
func (d *Dolap) SiteCategory() scraper.Category               { return scraper.UsedBook }

func (d *Dolap) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	if d.firecrawl == nil {
		return nil, fmt.Errorf("dolap.com Firecrawl gerektirir (--firecrawl bayrağını kullanın)")
	}

	searchURL := fmt.Sprintf(dolapSearchURL, url.QueryEscape(query))

	pageHTML, err := d.firecrawl.FetchHTML(ctx, searchURL, 0)
	if err != nil {
		return nil, fmt.Errorf("dolap: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("dolap: parsing HTML: %w", err)
	}

	var results []scraper.BookResult

	doc.Find("div.col-xs-6.col-md-4").Each(func(i int, card *goquery.Selection) {
		if len(results) >= d.limit {
			return
		}

		// Title (brand/publisher from card)
		title := strings.TrimSpace(card.Find(".detail-footer .title-info-block .title").Text())
		// Category detail (e.g. "Edebiyat", "Diğer Kitap & Dergi")
		detail := strings.TrimSpace(card.Find(".detail-footer .title-info-block .detail").Text())

		// Build a combined title
		displayTitle := title
		if detail != "" && detail != title {
			displayTitle = title + " - " + detail
		}

		// Price
		priceText := strings.TrimSpace(card.Find(".price-detail .price").Text())
		price := scraper.ParsePrice(priceText)

		// Seller
		seller := strings.TrimSpace(card.Find(".detail-head .title-stars-block .title").Text())

		// Product URL
		productURL := ""
		if href, exists := card.Find(".img-block a").First().Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" && !strings.HasPrefix(href, "http") {
				href = dolapBaseURL + href
			}
			productURL = href
		}

		// Condition from label badge
		condition := "İkinci El"
		labelText := strings.TrimSpace(card.Find(".label-block span").Text())
		if strings.Contains(strings.ToLower(labelText), "yeni") {
			condition = labelText
		}

		if displayTitle == "" || price <= 0 {
			return
		}

		// Post-filter relevance
		if !scraper.MatchesQuery(displayTitle, query) && !scraper.MatchesQuery(productURL, query) {
			return
		}

		results = append(results, scraper.BookResult{
			Title:        displayTitle,
			Price:        price,
			TotalPrice:   price,
			CargoFee:     0,
			CargoUnknown: true,
			Condition:    condition,
			Seller:       seller,
			URL:          productURL,
			Site:         "dolap.com",
			Category:     scraper.UsedBook,
		})
	})

	return results, nil
}
```

**Step 2: Create gardrops.go**

```go
package sites

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"

	"trbooksearch/internal/scraper"
)

const (
	gardropsBaseURL   = "https://www.gardrops.com"
	gardropsSearchURL = gardropsBaseURL + "/search?q=%s"
)

var gardropsSlugRe = regexp.MustCompile(`^/(.+)-[a-f0-9]{16}-p-(\d+)-(\d+)$`)

type Gardrops struct {
	limit     int
	browser   *rod.Browser
	firecrawl *scraper.FirecrawlClient
}

func NewGardrops(limit int) *Gardrops {
	return &Gardrops{limit: limit}
}

func (g *Gardrops) SetBrowser(browser *rod.Browser)              { g.browser = browser }
func (g *Gardrops) SetFirecrawl(client *scraper.FirecrawlClient) { g.firecrawl = client }
func (g *Gardrops) Name() string                                 { return "gardrops.com" }
func (g *Gardrops) SiteCategory() scraper.Category               { return scraper.UsedBook }

func (g *Gardrops) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	if g.firecrawl == nil {
		return nil, fmt.Errorf("gardrops.com Firecrawl gerektirir (--firecrawl bayrağını kullanın)")
	}

	searchURL := fmt.Sprintf(gardropsSearchURL, url.QueryEscape(query))

	pageHTML, err := g.firecrawl.FetchHTML(ctx, searchURL, 5000)
	if err != nil {
		return nil, fmt.Errorf("gardrops: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("gardrops: parsing HTML: %w", err)
	}

	var results []scraper.BookResult

	// Product cards are direct children of the grid container
	doc.Find("div.grid.grid-cols-2 > div.flex.flex-col").Each(func(i int, card *goquery.Selection) {
		if len(results) >= g.limit {
			return
		}

		// Product URL and title from image link
		linkEl := card.Find("a.relative.block").First()
		productURL := ""
		title := ""

		if href, exists := linkEl.Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" {
				if !strings.HasPrefix(href, "http") {
					productURL = gardropsBaseURL + href
				} else {
					productURL = href
				}
				// Try to extract a better title from the URL slug
				title = extractGardropsTitle(href)
			}
		}

		// Fallback: use img alt
		if title == "" {
			imgEl := linkEl.Find("img").First()
			title = strings.TrimSpace(imgEl.AttrOr("alt", ""))
		}

		// Price
		priceText := strings.TrimSpace(card.Find("p.text-smd.font-medium").First().Text())
		price := scraper.ParsePrice(priceText)

		// Condition: check badges
		condition := "İkinci El"
		card.Find("div.absolute.inset-x-0.bottom-0 p").Each(func(_ int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if strings.Contains(strings.ToLower(text), "yeni") {
				condition = text
			}
		})

		// Cargo: check for free shipping badge
		cargoFee := 0.0
		cargoUnknown := true
		card.Find("div.absolute.inset-x-0.bottom-0 p").Each(func(_ int, p *goquery.Selection) {
			text := strings.ToLower(strings.TrimSpace(p.Text()))
			if strings.Contains(text, "kargo bedava") || strings.Contains(text, "ücretsiz kargo") {
				cargoFee = 0
				cargoUnknown = false
			}
		})

		if title == "" || price <= 0 {
			return
		}

		results = append(results, scraper.BookResult{
			Title:        title,
			Price:        price,
			TotalPrice:   price + cargoFee,
			CargoFee:     cargoFee,
			CargoUnknown: cargoUnknown,
			Condition:    condition,
			URL:          productURL,
			Site:         "gardrops.com",
			Category:     scraper.UsedBook,
		})
	})

	return results, nil
}

// extractGardropsTitle extracts a human-readable title from a gardrops URL slug.
// URL format: /{slug}-{16-char-hex}-p-{productId}-{sellerId}
func extractGardropsTitle(href string) string {
	// Parse just the path
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	path := parsed.Path

	matches := gardropsSlugRe.FindStringSubmatch(path)
	if len(matches) < 2 {
		return ""
	}

	slug := matches[1]
	// Replace hyphens with spaces and title-case
	title := strings.ReplaceAll(slug, "-", " ")
	title = strings.TrimSpace(title)

	return title
}
```

**Step 3: Verify it compiles**

Run: `go build ./internal/scraper/sites/`
Expected: no errors

**Step 4: Commit**

```bash
git add internal/scraper/sites/dolap.go internal/scraper/sites/gardrops.go
git commit -m "feat: add dolap.com and gardrops.com scrapers (Firecrawl-only)"
```

---

### Task 7: Update registry to support Firecrawl mode

**Files:**
- Modify: `internal/scraper/sites/registry.go`

**Step 1: Update AllScrapers signature**

```go
package sites

import "trbooksearch/internal/scraper"

// AllScrapers returns all available scrapers with the given result limit.
// When firecrawlEnabled is true, Firecrawl-only sites (dolap, gardrops) are included.
func AllScrapers(limit int, firecrawlEnabled bool) []scraper.Scraper {
	scrapers := []scraper.Scraper{
		NewNadirkitap(limit),
		NewKitapyurdu(limit),
		NewTrendyol(limit),
		NewHepsiburada(limit),
		NewAmazon(limit),
		NewLetgo(limit),
	}
	if firecrawlEnabled {
		scrapers = append(scrapers,
			NewDolap(limit),
			NewGardrops(limit),
		)
	}
	return scrapers
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/scraper/sites/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/scraper/sites/registry.go
git commit -m "feat: registry includes dolap/gardrops when Firecrawl is enabled"
```

---

### Task 8: Update CLI (search.go) and root.go to wire everything together

**Files:**
- Modify: `cmd/search.go`
- Modify: `cmd/root.go:13-18`

**Step 1: Add --firecrawl flag and config loading to search.go**

Add a new flag variable:
```go
var flagFirecrawl bool
```

Register it in `init()`:
```go
searchCmd.Flags().BoolVar(&flagFirecrawl, "firecrawl", false, "Firecrawl API ile tüm siteleri tara (API anahtarı gerektirir)")
```

Update `runSearch` to load config and create FirecrawlClient:

```go
func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	searchType := scraper.TitleSearch
	if flagISBN {
		searchType = scraper.ISBNSearch
	}

	// Load config for Firecrawl
	var firecrawlClient *scraper.FirecrawlClient
	if flagFirecrawl {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("yapılandırma hatası: %w", err)
		}
		if cfg.Firecrawl.APIKey == "" {
			return fmt.Errorf("Firecrawl API anahtarı bulunamadı.\n\nYapılandırma dosyası oluşturun: %s\n\nİçerik:\nfirecrawl:\n  api_key: \"fc-...\"", config.ConfigPath())
		}
		firecrawlClient = scraper.NewFirecrawlClient(cfg.Firecrawl.APIKey, cfg.Firecrawl.APIURL)
	}

	// Build scraper list
	allScrapers := sites.AllScrapers(flagLimit, flagFirecrawl)

	// ... site filtering unchanged ...

	eng := engine.NewEngine(firecrawlClient, filtered...)

	// ... rest unchanged ...
}
```

Add import for `"trbooksearch/internal/config"`.

**Step 2: Update root.go description to include new sites**

In `cmd/root.go`, update the Long description:

```go
Long: `trbooksearch - Türkiye'deki kitap sitelerinde arama yaparak
ikinci el ve yeni kitap fiyatlarını karşılaştırmanızı sağlar.

Desteklenen siteler:
  İkinci El: nadirkitap.com, letgo.com
  Yeni:      kitapyurdu.com, trendyol.com, hepsiburada.com, amazon.com.tr

Firecrawl ile ek siteler (--firecrawl):
  İkinci El: dolap.com, gardrops.com`,
```

**Step 3: Verify the full build**

Run: `go build ./...`
Expected: no errors

**Step 4: Commit**

```bash
git add cmd/search.go cmd/root.go
git commit -m "feat: add --firecrawl flag, wire config loading and Firecrawl client"
```

---

### Task 9: Manual smoke test

**Step 1: Build the binary**

Run: `go build -o trbooksearch .`
Expected: binary created

**Step 2: Test without --firecrawl (should work exactly as before)**

Run: `./trbooksearch search "Suç ve Ceza" --limit 3`
Expected: results from 6 core sites, no mention of dolap/gardrops

**Step 3: Test --firecrawl without config (should show helpful error)**

Run: `./trbooksearch search --firecrawl "Suç ve Ceza"`
Expected: error message with config file path and setup instructions

**Step 4: Create config file and test with --firecrawl**

Create `~/.config/trbooksearch/config.yaml`:
```yaml
firecrawl:
  api_key: "fc-YOUR-KEY-HERE"
```

Run: `./trbooksearch search --firecrawl "Suç ve Ceza" --limit 3`
Expected: results from all 8 sites including dolap.com and gardrops.com

**Step 5: Commit any fixes**

Fix any issues found during testing and commit.
