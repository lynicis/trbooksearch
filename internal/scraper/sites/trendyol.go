package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"trbooksearch/internal/scraper"
)

const (
	trendyolBaseURL   = "https://www.trendyol.com"
	trendyolSearchURL = trendyolBaseURL + "/sr?q=%s&wc=91"
	trendyolCargoFee  = 29.99
)

type Trendyol struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewTrendyol(limit int) *Trendyol {
	return &Trendyol{limit: limit}
}

func (t *Trendyol) Name() string                                 { return "trendyol.com" }
func (t *Trendyol) SiteCategory() scraper.Category               { return scraper.NewBook }
func (t *Trendyol) SetFirecrawl(client *scraper.FirecrawlClient) { t.firecrawl = client }

func (t *Trendyol) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(trendyolSearchURL, url.QueryEscape(query))

	var pageHTML string
	var err error
	if t.firecrawl != nil {
		pageHTML, err = t.firecrawl.FetchHTML(ctx, searchURL, 5000)
	} else {
		// Use FetchPageWithWait — Trendyol is an SPA that needs time to render product cards
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, "a.product-card")
	}
	if err != nil {
		return nil, fmt.Errorf("trendyol: %w", err)
	}
	if pageHTML == "" {
		return nil, nil // no products found
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("trendyol: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0

	// Use the confirmed working selector — a.product-card is the current structure
	productCards := doc.Find("a.product-card")

	productCards.Each(func(i int, card *goquery.Selection) {
		if productCount >= t.limit {
			return
		}

		// Publisher/brand
		publisher := strings.TrimSpace(card.Find("span.product-brand").Text())

		// Product name
		name := strings.TrimSpace(card.Find("span.product-name").Text())
		if name == "" || !scraper.MatchesQuery(name, query) {
			return
		}

		// Price: try discounted sale-price first, then normal price-section
		priceText := firstNonEmpty(
			card.Find("div.sale-price").First().Text(),
			card.Find("div.price-section").First().Text(),
		)
		if priceText == "" {
			return
		}
		price := scraper.ParsePrice(priceText)
		if price <= 0 {
			return
		}

		productURL := ""
		if href, exists := card.Attr("href"); exists && href != "" {
			productURL = buildTrendyolURL(href)
		}
		if productURL == "" {
			if href, exists := card.Find("a").First().Attr("href"); exists && href != "" {
				productURL = buildTrendyolURL(href)
			}
		}

		results = append(results, scraper.BookResult{
			Title:      name,
			Publisher:  publisher,
			Price:      price,
			CargoFee:   trendyolCargoFee,
			TotalPrice: price + trendyolCargoFee,
			Condition:  scraper.NewBook.String(),
			URL:        productURL,
			Site:       "trendyol.com",
			Category:   scraper.NewBook,
		})

		results = append(results, scraper.BookResult{
			Title:       name,
			Publisher:   publisher,
			Price:       price,
			CargoFee:    0,
			TotalPrice:  price,
			FreeCargo:   true,
			LoyaltyNote: "Trendyol Elite: ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			URL:         productURL,
			Site:        "trendyol.com",
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}

func buildTrendyolURL(href string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "/") {
		href = trendyolBaseURL + href
	}
	if idx := strings.Index(href, "?"); idx != -1 {
		href = href[:idx]
	}
	return href
}

// firstNonEmpty returns the first non-empty trimmed string from the arguments.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
