package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

// DR (D&R) is a Firecrawl-only scraper because dr.com.tr aggressively blocks
// automated requests, returning 403 even with stealth browsers.
type DR struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewDR(limit int) *DR {
	return &DR{limit: limit}
}

func (d *DR) Name() string                                 { return "dr.com.tr" }
func (d *DR) SiteCategory() scraper.Category               { return scraper.NewBook }
func (d *DR) SetFirecrawl(client *scraper.FirecrawlClient) { d.firecrawl = client }

func (d *DR) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	if d.firecrawl == nil {
		return nil, fmt.Errorf("dr.com.tr Firecrawl gerektirir (--firecrawl bayrağını kullanın)")
	}

	searchURL := fmt.Sprintf(
		"https://www.dr.com.tr/search?q=%s&redirect=search&ProductType=Kitap",
		url.QueryEscape(query),
	)

	pageHTML, err := d.firecrawl.FetchHTML(ctx, searchURL, 8000)
	if err != nil {
		return nil, fmt.Errorf("dr: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("dr: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0
	cargoFee := 24.99

	// D&R uses product card classes. Try multiple selectors since the site
	// may render differently through Firecrawl.
	selectors := []string{
		".prd-list-item",
		".product-card",
		".search-result-item",
		"[data-product-id]",
		".product-item",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			if productCount >= d.limit {
				return
			}

			title := strings.TrimSpace(s.Find(".prd-name, .product-name, .product-title, h3 a, .item-name").First().Text())
			author := strings.TrimSpace(s.Find(".prd-author, .product-author, .author-name, .item-author").First().Text())
			publisher := strings.TrimSpace(s.Find(".prd-publisher, .product-publisher, .publisher-name, .item-publisher").First().Text())

			var price float64
			s.Find(".prd-price, .product-price, .price, .current-price, .sale-price").Each(func(_ int, p *goquery.Selection) {
				if price == 0 {
					price = scraper.ParsePrice(p.Text())
				}
			})

			productURL := ""
			s.Find("a[href]").Each(func(j int, a *goquery.Selection) {
				if productURL != "" {
					return
				}
				if href, exists := a.Attr("href"); exists {
					href = strings.TrimSpace(href)
					if href != "" && href != "#" {
						if !strings.HasPrefix(href, "http") {
							href = "https://www.dr.com.tr" + href
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
				Author:     author,
				Publisher:  publisher,
				Price:      price,
				CargoFee:   cargoFee,
				TotalPrice: price + cargoFee,
				Condition:  scraper.NewBook.String(),
				Site:       "dr.com.tr",
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
				LoyaltyNote: "D&R Premium ile ücretsiz kargo",
				Condition:   scraper.NewBook.String(),
				Site:        "dr.com.tr",
				URL:         productURL,
				Category:    scraper.NewBook,
			})
			productCount++
		})

		if productCount > 0 {
			break // Found products with this selector, stop trying others
		}
	}

	return results, nil
}
