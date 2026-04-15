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

	searchURL := fmt.Sprintf(
		"https://www.pandora.com.tr/Arama?q=%s",
		url.QueryEscape(query),
	)

	pageHTML, err := p.firecrawl.FetchHTML(ctx, searchURL, 8000)
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
	cargoFee := 24.99

	// Pandora uses various product card selectors depending on rendering.
	// Try multiple approaches since it's a Next.js SPA.
	selectors := []string{
		"a[href*='/kitap/']",
		".product-card",
		".book-card",
		"[data-testid='product-card']",
		".search-result-item",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			if productCount >= p.limit {
				return
			}

			var title, author, publisher, productURL string
			var price float64

			if sel == "a[href*='/kitap/']" {
				// Extract from link-based cards
				if href, exists := s.Attr("href"); exists {
					href = strings.TrimSpace(href)
					if href != "" {
						if !strings.HasPrefix(href, "http") {
							productURL = "https://www.pandora.com.tr" + href
						} else {
							productURL = href
						}
					}
					// Try to extract title from URL slug
					parts := strings.Split(href, "/")
					for j, part := range parts {
						if part == "kitap" && j+1 < len(parts) {
							title = strings.ReplaceAll(parts[j+1], "-", " ")
							break
						}
					}
				}

				// Look for title text in card
				cardTitle := strings.TrimSpace(s.Find("h2, h3, .title, .book-title, .product-title").First().Text())
				if cardTitle != "" {
					title = cardTitle
				}
				if title == "" {
					title = strings.TrimSpace(s.Text())
					// Limit to reasonable title length
					if len(title) > 200 {
						title = ""
					}
				}

				// Look for price
				s.Find("span, p, div").Each(func(_ int, el *goquery.Selection) {
					text := strings.TrimSpace(el.Text())
					if (strings.Contains(text, "TL") || strings.Contains(text, "₺")) && price == 0 && len(text) < 30 {
						price = scraper.ParsePrice(text)
					}
				})

				// Look for author/publisher in child elements
				author = strings.TrimSpace(s.Find(".author, .writer, .book-author").First().Text())
				publisher = strings.TrimSpace(s.Find(".publisher, .book-publisher, .yayinevi").First().Text())
			} else {
				// Generic card extraction
				title = strings.TrimSpace(s.Find("h2, h3, .title, .product-title, .book-title").First().Text())
				author = strings.TrimSpace(s.Find(".author, .writer").First().Text())
				publisher = strings.TrimSpace(s.Find(".publisher, .yayinevi").First().Text())

				s.Find("span, p, div").Each(func(_ int, el *goquery.Selection) {
					text := strings.TrimSpace(el.Text())
					if (strings.Contains(text, "TL") || strings.Contains(text, "₺")) && price == 0 && len(text) < 30 {
						price = scraper.ParsePrice(text)
					}
				})

				if href, exists := s.Find("a[href]").First().Attr("href"); exists {
					href = strings.TrimSpace(href)
					if href != "" {
						if !strings.HasPrefix(href, "http") {
							productURL = "https://www.pandora.com.tr" + href
						} else {
							productURL = href
						}
					}
				}
			}

			if title == "" || price <= 0 {
				return
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
		})

		if productCount > 0 {
			break
		}
	}

	return results, nil
}
