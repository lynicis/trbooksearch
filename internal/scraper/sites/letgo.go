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
	letgoBaseURL   = "https://www.letgo.com"
	letgoSearchURL = letgoBaseURL + "/arama?q=%s&category=Kitap%%2C+Dergi+%%26+Film"
)

type Letgo struct {
	limit int
}

func NewLetgo(limit int) *Letgo {
	return &Letgo{limit: limit}
}

func (l *Letgo) Name() string                   { return "letgo.com" }
func (l *Letgo) SiteCategory() scraper.Category { return scraper.UsedBook }

func (l *Letgo) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(letgoSearchURL, url.QueryEscape(query))

	// Use FetchPageWithWait to ensure item cards are rendered before extracting HTML.
	pageHTML, err := scraper.FetchPageWithWait(ctx, searchURL, `[data-testid="item-card"]`)
	if err != nil {
		return nil, fmt.Errorf("letgo: %w", err)
	}
	if pageHTML == "" {
		return nil, nil // no items rendered
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("letgo: parsing HTML: %w", err)
	}

	var results []scraper.BookResult

	doc.Find(`[data-testid="item-card"]`).Each(func(i int, card *goquery.Selection) {
		if len(results) >= l.limit {
			return
		}

		itemURL := ""
		if href, exists := card.Find("a").First().Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" {
				itemURL = resolveLetgoURL(href)
			}
		}

		body := card.Find(`[data-slot="item-card-body"]`)

		// Title is in a <div> with line-clamp-1 inside the item-card-body.
		// The price <p> also has line-clamp-1 but is a <p> element, so div.line-clamp-1
		// correctly targets only the title. Also try img alt as fallback.
		title := strings.TrimSpace(body.Find("div.line-clamp-1").First().Text())
		if title == "" {
			// Fallback: try the item image alt text
			title = strings.TrimSpace(card.Find(`[data-slot="item-card-image"] img`).First().AttrOr("alt", ""))
		}

		var price float64
		body.Find("p").Each(func(_ int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if strings.Contains(text, "TL") && price == 0 {
				price = scraper.ParsePrice(text)
			}
		})

		location := strings.TrimSpace(card.Find(`[data-slot="item-card-body"] .text-secondary-600 span`).First().Text())
		seller := location
		if seller == "" {
			seller = "letgo satıcı"
		}

		// Post-filter: only keep results relevant to the query.
		// NOTE: Letgo's search often returns irrelevant items (popular/recent listings
		// rather than query-matched results). This filter is essential but may result
		// in 0 results for niche queries like specific book titles.
		if title == "" || price <= 0 {
			return
		}

		results = append(results, scraper.BookResult{
			Title:        title,
			Price:        price,
			TotalPrice:   price,
			CargoFee:     0,
			CargoUnknown: true,
			Condition:    "İkinci El",
			Seller:       seller,
			URL:          itemURL,
			Site:         "letgo.com",
			Category:     scraper.UsedBook,
		})
	})

	return results, nil
}

func resolveLetgoURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	if !strings.HasPrefix(href, "/") {
		href = "/" + href
	}
	return letgoBaseURL + href
}
