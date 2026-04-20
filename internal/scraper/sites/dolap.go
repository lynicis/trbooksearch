package sites

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

const (
	dolapBaseURL   = "https://dolap.com"
	dolapSearchURL = dolapBaseURL + "/ara?q=%s"
)

type Dolap struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewDolap(limit int) *Dolap {
	return &Dolap{limit: limit}
}

func (d *Dolap) Name() string                                 { return "dolap.com" }
func (d *Dolap) SiteCategory() scraper.Category               { return scraper.UsedBook }
func (d *Dolap) SetFirecrawl(client *scraper.FirecrawlClient) { d.firecrawl = client }

func (d *Dolap) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	if d.firecrawl == nil {
		return nil, fmt.Errorf("dolap.com Firecrawl gerektirir (--firecrawl bayrağını kullanın)")
	}

	searchURL := fmt.Sprintf(dolapSearchURL, url.QueryEscape(query))

	pageHTML, err := d.firecrawl.FetchHTMLWithOptions(ctx, searchURL, scraper.FetchOptions{
		WaitFor: 5000,
		Timeout: 90000,
		Proxy:   "enhanced",
		Retries: 1,
	})
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
