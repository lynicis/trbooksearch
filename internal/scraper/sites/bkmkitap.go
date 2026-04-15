package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

type Bkmkitap struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewBkmkitap(limit int) *Bkmkitap {
	return &Bkmkitap{limit: limit}
}

func (b *Bkmkitap) Name() string                                 { return "bkmkitap.com" }
func (b *Bkmkitap) SiteCategory() scraper.Category               { return scraper.NewBook }
func (b *Bkmkitap) SetFirecrawl(client *scraper.FirecrawlClient) { b.firecrawl = client }

func (b *Bkmkitap) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(
		"https://www.bkmkitap.com/arama?q=%s",
		url.QueryEscape(query),
	)

	var pageHTML string
	var err error
	if b.firecrawl != nil {
		pageHTML, err = b.firecrawl.FetchHTML(ctx, searchURL, 5000)
	} else {
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, ".waw-product")
	}
	if err != nil {
		return nil, fmt.Errorf("bkmkitap: %w", err)
	}
	if pageHTML == "" {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("bkmkitap: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0
	cargoFee := 19.90

	doc.Find("div.waw-product").Each(func(i int, s *goquery.Selection) {
		if productCount >= b.limit {
			return
		}

		title := strings.TrimSpace(s.Find("p.product-title").First().Text())

		// Publisher is the first p.txt-title, author is the second
		var publisher, author string
		s.Find("p.txt-title").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			switch j {
			case 0:
				publisher = text
			case 1:
				author = text
			}
		})

		// Discounted/sale price from the basket button text or data-price attribute
		var price float64
		if dataPrice, exists := s.Attr("data-price"); exists {
			price = scraper.ParsePrice(dataPrice)
		}
		if price <= 0 {
			// Fallback: try the basket button text
			basketText := strings.TrimSpace(s.Find("a.waw-basket").First().Text())
			// Remove "Sepete Ekle" text
			basketText = strings.ReplaceAll(basketText, "Sepete Ekle", "")
			price = scraper.ParsePrice(basketText)
		}

		// Product URL
		productURL := ""
		if href, exists := s.Find(".waw-product-item-area > a").First().Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" && !strings.HasPrefix(href, "http") {
				href = "https://www.bkmkitap.com" + href
			}
			productURL = href
		}

		// Skip out-of-stock items (they show "Tükendi" instead of price)
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
			Site:       "bkmkitap.com",
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
			LoyaltyNote: "100 TL üzeri siparişlerde ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			Site:        "bkmkitap.com",
			URL:         productURL,
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}
