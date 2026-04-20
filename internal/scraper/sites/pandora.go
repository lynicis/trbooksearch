package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

// Pandora is a Firecrawl-only scraper because pandora.com.tr uses a heavily
// JavaScript-rendered Next.js SPA that doesn't render search results with
// standard headless browser automation.
type Pandora struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewPandora(limit int) *Pandora {
	return &Pandora{limit: limit}
}

func (p *Pandora) Name() string                                 { return "pandora.com.tr" }
func (p *Pandora) SiteCategory() scraper.Category               { return scraper.NewBook }
func (p *Pandora) SetFirecrawl(client *scraper.FirecrawlClient) { p.firecrawl = client }

func (p *Pandora) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	if p.firecrawl == nil {
		return nil, fmt.Errorf("pandora.com.tr Firecrawl gerektirir (--firecrawl bayrağını kullanın)")
	}

	// Pandora's real search endpoint is /Arama/Keyword?sozcuk=<query>. The
	// previously used /Arama?q=<query> route returns a "Geçersiz arama türü"
	// error page and produces zero results.
	searchURL := fmt.Sprintf(
		"https://www.pandora.com.tr/Arama/Keyword?sozcuk=%s",
		url.QueryEscape(query),
	)

	pageHTML, err := p.firecrawl.FetchHTMLWithOptions(ctx, searchURL, scraper.FetchOptions{
		WaitFor: 8000,
		Timeout: 90000,
		Proxy:   "auto",
		Retries: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("pandora: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("pandora: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0
	const cargoFee = 24.99

	// Search result cards are anchors pointing to /kitap/{slug}/{id}. The
	// anchor wraps the whole card — title in <h3>, author in a <span>,
	// publisher in a <p>, and price in a <div> containing a <span>…TL.
	doc.Find(`a[href*="/kitap/"]`).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if productCount >= p.limit {
			return false
		}

		href, exists := s.Attr("href")
		if !exists {
			return true
		}
		href = strings.TrimSpace(href)
		if href == "" {
			return true
		}

		// Must look like a product link: /kitap/<slug>/<id>
		if !isPandoraProductHref(href) {
			return true
		}

		productURL := href
		if !strings.HasPrefix(productURL, "http") {
			productURL = "https://www.pandora.com.tr" + productURL
		}

		title := strings.TrimSpace(s.Find("h3").First().Text())
		if title == "" {
			return true
		}

		// Author is the first inline <span> sibling under the title block.
		author := strings.TrimSpace(s.Find("h3").First().NextFiltered("span").Text())
		// Publisher sits in a <p> right after the author span.
		publisher := strings.TrimSpace(s.Find("h3").First().Parent().Find("p").First().Text())

		// Price lives in a <span> inside a price <div>. Pick the first span
		// whose text contains "TL" or the ₺ symbol.
		var price float64
		s.Find("span").EachWithBreak(func(_ int, sp *goquery.Selection) bool {
			text := strings.TrimSpace(sp.Text())
			if (strings.Contains(text, "TL") || strings.Contains(text, "₺")) && len(text) < 40 {
				if v := scraper.ParsePrice(text); v > 0 {
					price = v
					return false
				}
			}
			return true
		})

		if price <= 0 {
			return true
		}

		results = append(results, scraper.BookResult{
			Title:      title,
			Author:     author,
			Publisher:  publisher,
			Price:      price,
			CargoFee:   cargoFee,
			TotalPrice: price + cargoFee,
			Condition:  scraper.NewBook.String(),
			Site:       "pandora.com.tr",
			URL:        productURL,
			Category:   scraper.NewBook,
		})

		results = append(results, scraper.BookResult{
			Title:       title,
			Author:      author,
			Publisher:   publisher,
			Price:       price,
			CargoFee:    0,
			TotalPrice:  price,
			FreeCargo:   true,
			LoyaltyNote: "150 TL üzeri siparişlerde ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			Site:        "pandora.com.tr",
			URL:         productURL,
			Category:    scraper.NewBook,
		})
		productCount++
		return true
	})

	return results, nil
}

// isPandoraProductHref reports whether an href looks like a Pandora book
// detail URL of the form /kitap/<slug>/<numeric-id>. Menu links such as
// /kitap/arama-gecmisi/935336 pass this check too, which is fine — the
// parent context (search results grid) ensures we only see relevant cards.
func isPandoraProductHref(href string) bool {
	idx := strings.Index(href, "/kitap/")
	if idx < 0 {
		return false
	}
	rest := href[idx+len("/kitap/"):]
	// Must have at least "<slug>/<id>"
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return false
	}
	slug := parts[0]
	id := parts[1]
	if slug == "" || id == "" {
		return false
	}
	// Drop any query string on the id
	if q := strings.IndexAny(id, "?#"); q >= 0 {
		id = id[:q]
	}
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
