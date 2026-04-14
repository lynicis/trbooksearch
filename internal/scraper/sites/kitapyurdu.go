package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"trbooksearch/internal/scraper"
)

type Kitapyurdu struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewKitapyurdu(limit int) *Kitapyurdu {
	return &Kitapyurdu{limit: limit}
}

func (k *Kitapyurdu) Name() string                                 { return "kitapyurdu.com" }
func (k *Kitapyurdu) SiteCategory() scraper.Category               { return scraper.NewBook }
func (k *Kitapyurdu) SetFirecrawl(client *scraper.FirecrawlClient) { k.firecrawl = client }

func (k *Kitapyurdu) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(
		"https://www.kitapyurdu.com/index.php?route=product/search&filter_name=%s",
		url.QueryEscape(query),
	)

	var pageHTML string
	var err error
	if k.firecrawl != nil {
		pageHTML, err = k.firecrawl.FetchHTML(ctx, searchURL, 5000)
	} else {
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, ".ky-product")
	}
	if err != nil {
		return nil, fmt.Errorf("kitapyurdu: %w", err)
	}
	if pageHTML == "" {
		return nil, nil // no products found
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("kitapyurdu: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0
	cargoFee := 24.90

	doc.Find(".ky-product").Each(func(i int, s *goquery.Selection) {
		if productCount >= k.limit {
			return
		}

		title := strings.TrimSpace(s.Find(".ky-product-title").First().Text())
		author := strings.TrimSpace(s.Find(".ky-product-author a").First().Text())
		publisher := strings.TrimSpace(s.Find(".ky-product-publisher a").First().Text())

		priceText := strings.TrimSpace(s.Find(".ky-product-sell-price").First().Text())
		price := scraper.ParsePrice(priceText)

		productURL := ""
		if href, exists := s.Find("a").Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" && !strings.HasPrefix(href, "http") {
				href = "https://www.kitapyurdu.com" + href
			}
			productURL = href
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
			Site:       "kitapyurdu.com",
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
			Site:        "kitapyurdu.com",
			URL:         productURL,
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}
