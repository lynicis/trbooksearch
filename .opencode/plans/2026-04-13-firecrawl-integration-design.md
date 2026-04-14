# Firecrawl Integration Design

**Date:** 2026-04-13
**Status:** Approved

## Goal

Add Firecrawl as an optional global scraping backend that replaces the current
go-rod/HTTP fetch layer when enabled. This also adds two new sites (dolap.com,
gardrops.com) that are only available when Firecrawl is active.

## How it works

When the user passes `--firecrawl` and has a Firecrawl API key configured:

1. No headless browser is launched (go-rod is skipped entirely)
2. All scrapers fetch HTML via the Firecrawl `/v1/scrape` API
3. dolap.com and gardrops.com scrapers are included in the search
4. All goquery parsing code remains identical

When `--firecrawl` is NOT set:

1. Current behavior is unchanged (go-rod + HTTP)
2. dolap.com and gardrops.com are excluded

## Configuration

Config file: `~/.config/trbooksearch/config.yaml`

```yaml
firecrawl:
  api_key: "fc-..."
  api_url: "https://api.firecrawl.dev"  # optional, for self-hosted
```

New package: `internal/config/config.go`

- Uses `os.UserConfigDir()` for cross-platform XDG support
- Reads YAML with `gopkg.in/yaml.v3`
- Returns typed `Config` struct

## Firecrawl Client

New file: `internal/scraper/firecrawl.go`

```go
type FirecrawlClient struct {
    apiKey string
    apiURL string
    http   *http.Client
}

func (fc *FirecrawlClient) FetchHTML(ctx context.Context, url string, waitFor int) (string, error)
```

Calls `POST /v1/scrape` with body:
```json
{
  "url": "<target>",
  "formats": ["html"],
  "waitFor": 5000
}
```

Returns the HTML string for goquery parsing.

## Scraper Interface Changes

```go
type Scraper interface {
    Name() string
    Search(ctx context.Context, query string, searchType SearchType) ([]BookResult, error)
    SiteCategory() Category
    SetBrowser(browser *rod.Browser)
    SetFirecrawl(client *FirecrawlClient)
}
```

Each scraper checks: if `firecrawl != nil`, use it to fetch HTML; otherwise use
the current method (go-rod or HTTP). dolap/gardrops return an error if Firecrawl
is nil.

## Engine Changes

`engine.go` `Search()` method:

- If Firecrawl client is provided: skip `LaunchBrowser()`, call
  `SetFirecrawl(client)` on all scrapers
- If no Firecrawl client: launch browser as before, call `SetBrowser(browser)`

## Registry Changes

```go
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
        scrapers = append(scrapers, NewDolap(limit), NewGardrops(limit))
    }
    return scrapers
}
```

## CLI Changes

New flag: `--firecrawl` on the `search` command.

Validation: if `--firecrawl` is set but no API key in config, print error with
setup instructions and exit.

## Site replacement table

| Site             | Current fetch         | With Firecrawl            |
|------------------|-----------------------|---------------------------|
| nadirkitap.com   | go-rod FetchPage      | firecrawl.FetchHTML       |
| kitapyurdu.com   | go-rod FetchPageWait  | firecrawl.FetchHTML(5000) |
| trendyol.com     | go-rod FetchPage      | firecrawl.FetchHTML       |
| hepsiburada.com  | go-rod FetchPage      | firecrawl.FetchHTML       |
| amazon.com.tr    | go-rod FetchPage      | firecrawl.FetchHTML       |
| letgo.com        | go-rod FetchPage      | firecrawl.FetchHTML       |
| dolap.com (new)  | N/A                   | firecrawl.FetchHTML       |
| gardrops.com (new)| N/A                  | firecrawl.FetchHTML(5000) |

## New Scrapers

### dolap.com

- Search URL: `https://dolap.com/ara?q={query}`
- Category: UsedBook
- Cards: `div.col-xs-6.col-md-4`
- Title: `.detail-footer .title-info-block .title`
- Price: `.price-detail .price` (format: "70 TL")
- Seller: `.detail-head .title-stars-block .title`
- Product URL: `.img-block a[rel="nofollow"]` href
- Condition: `.label-block span` ("Yeni & Etiketli") or parse from URL slug
- Cargo: unknown (not shown on cards), `CargoUnknown: true`
- Default cargo: 0 (unknown)

### gardrops.com

- Search URL: `https://www.gardrops.com/search?q={query}`
- Category: UsedBook
- Cards: `div.grid.grid-cols-2 > div.flex.flex-col`
- Title: `a.relative.block img` alt attribute (brand + size, limited)
- Price: `p.text-smd.font-medium` (format: "70 ₺")
- Seller: not on cards (ID only in URL)
- Product URL: `a.relative.block` href
- Condition: badge "Yeni & Etiketli" or default "İkinci El"
- Cargo: free ("Kargo Bedava" badge on all items observed)
- Default cargo: 0 (free shipping)

### Gardrops limitation

Product cards show brand + size in alt text, not proper book titles. For book
searches the title may show as "Diğer" (Other). URL slugs contain more info.
We parse the slug to construct a more useful title when the alt text is too
generic.

## Dependencies added

- `gopkg.in/yaml.v3` (config file parsing)
- No Firecrawl SDK -- we use raw HTTP to avoid unnecessary dependencies
