package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

type Idefix struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewIdefix(limit int) *Idefix {
	return &Idefix{limit: limit}
}

func (ix *Idefix) Name() string                                 { return "idefix.com" }
func (ix *Idefix) SiteCategory() scraper.Category               { return scraper.NewBook }
func (ix *Idefix) SetFirecrawl(client *scraper.FirecrawlClient) { ix.firecrawl = client }

func (ix *Idefix) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	// Filter by category 3307 (Kitap/Books) to avoid non-book results
	searchURL := fmt.Sprintf(
		"https://www.idefix.com/arama?q=%s&kategori=3307",
		url.QueryEscape(query),
	)

	var pageHTML string
	var err error
	if ix.firecrawl != nil {
		pageHTML, err = ix.firecrawl.FetchHTML(ctx, searchURL, 5000)
	} else {
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, "div.justify-self-start")
	}
	if err != nil {
		return nil, fmt.Errorf("idefix: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("idefix: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0
	cargoFee := 24.99

	// Each product card is wrapped in div.justify-self-start within the grid section
	doc.Find("div.justify-self-start").Each(func(i int, s *goquery.Selection) {
		if productCount >= ix.limit {
			return
		}

		// Publisher from first span in h3 (font-semibold)
		publisher := strings.TrimSpace(s.Find("h3 span.font-semibold").First().Text())

		// Title from second span in h3 (font-medium)
		// The title typically includes publisher name suffix like "Dune - İthaki Yayınları"
		fullTitle := strings.TrimSpace(s.Find("h3 span.font-medium").First().Text())
		title := fullTitle
		// Remove publisher suffix if present (e.g., "Dune - İthaki Yayınları" -> "Dune")
		if publisher != "" && strings.HasSuffix(title, " - "+publisher) {
			title = strings.TrimSuffix(title, " - "+publisher)
		}

		// Price: prefer the "Sepette" (cart) price, which is the lowest
		// Cart price: span.text-secondary-500
		var price float64
		cartPriceText := strings.TrimSpace(s.Find("span.text-secondary-500").First().Text())
		if cartPriceText != "" {
			// Remove "Sepette" prefix
			cartPriceText = strings.ReplaceAll(cartPriceText, "Sepette", "")
			price = scraper.ParsePrice(cartPriceText)
		}

		// Fallback: try the list price
		if price <= 0 {
			listPriceEl := s.Find("div.min-h-\\[1\\.25rem\\] > span").First()
			price = scraper.ParsePrice(listPriceEl.Text())
		}

		// Product URL from the first <a> tag in the card
		productURL := ""
		s.Find("a[href]").Each(func(j int, a *goquery.Selection) {
			if productURL != "" {
				return
			}
			if href, exists := a.Attr("href"); exists {
				href = strings.TrimSpace(href)
				if strings.Contains(href, "-p-") {
					if !strings.HasPrefix(href, "http") {
						href = "https://www.idefix.com" + href
					}
					// Remove fragment like #sortId=...
					if idx := strings.Index(href, "#"); idx != -1 {
						href = href[:idx]
					}
					productURL = href
				}
			}
		})

		if title == "" || price <= 0 {
			return
		}

		results = append(results, scraper.BookResult{
			Title:      title,
			Author:     "", // Author not available on listing page
			Publisher:  publisher,
			Price:      price,
			CargoFee:   cargoFee,
			TotalPrice: price + cargoFee,
			Condition:  scraper.NewBook.String(),
			Site:       "idefix.com",
			URL:        productURL,
			Category:   scraper.NewBook,
		})

		results = append(results, scraper.BookResult{
			Title:       title,
			Author:      "", // Author not available on listing page
			Publisher:   publisher,
			Price:       price,
			CargoFee:    0,
			TotalPrice:  price,
			FreeCargo:   true,
			LoyaltyNote: "200 TL üzeri siparişlerde ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			Site:        "idefix.com",
			URL:         productURL,
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}
