package sites

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"trbooksearch/internal/scraper"
)

const (
	gardropsBaseURL   = "https://www.gardrops.com"
	gardropsSearchURL = gardropsBaseURL + "/search?q=%s"
)

var gardropsSlugRe = regexp.MustCompile(`^/(.+)-[a-f0-9]{16}-p-(\d+)-(\d+)$`)

type Gardrops struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewGardrops(limit int) *Gardrops {
	return &Gardrops{limit: limit}
}

func (g *Gardrops) Name() string                                 { return "gardrops.com" }
func (g *Gardrops) SiteCategory() scraper.Category               { return scraper.UsedBook }
func (g *Gardrops) SetFirecrawl(client *scraper.FirecrawlClient) { g.firecrawl = client }

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
