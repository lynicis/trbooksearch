package sites

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"trbooksearch/internal/scraper"

	"github.com/PuerkitoBio/goquery"
)

const (
	nadirkitapBaseURL      = "https://www.nadirkitap.com"
	nadirkitapSearchURL    = "https://www.nadirkitap.com/kitapara_sonuc.php?kelime=%s&kategori=kitap"
	nadirkitapDefaultCargo = 40.0
)

type Nadirkitap struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewNadirkitap(limit int) *Nadirkitap {
	return &Nadirkitap{limit: limit}
}

func (n *Nadirkitap) Name() string                                 { return "nadirkitap.com" }
func (n *Nadirkitap) SiteCategory() scraper.Category               { return scraper.UsedBook }
func (n *Nadirkitap) SetFirecrawl(client *scraper.FirecrawlClient) { n.firecrawl = client }

func (n *Nadirkitap) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(nadirkitapSearchURL, url.QueryEscape(query))

	var pageHTML string
	var err error
	if n.firecrawl != nil {
		pageHTML, err = n.firecrawl.FetchHTML(ctx, searchURL, 0)
	} else {
		pageHTML, err = scraper.FetchPage(ctx, searchURL)
	}
	if err != nil {
		return nil, fmt.Errorf("nadirkitap: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("nadirkitap: parsing HTML: %w", err)
	}

	var results []scraper.BookResult

	doc.Find(".productsListHover").Each(func(i int, card *goquery.Selection) {
		if len(results) >= n.limit {
			return
		}

		result := scraper.BookResult{
			Site:     "nadirkitap.com",
			Category: scraper.UsedBook,
		}

		titleLink := card.Find("a.nadirBook").First()
		if titleLink.Length() > 0 {
			result.Title = strings.TrimSpace(titleLink.Text())
			if href, exists := titleLink.Attr("href"); exists {
				result.URL = resolveNadirkitapURL(href)
			}
		}

		priceEl := card.Find("a.price").First()
		if priceEl.Length() > 0 {
			result.Price = scraper.ParsePrice(priceEl.Text())
		}

		sellerEl := card.Find("a.sellerName").First()
		if sellerEl.Length() > 0 {
			result.Seller = strings.TrimSpace(sellerEl.Text())
		}

		condEl := card.Find(".text-nadir").First()
		if condEl.Length() > 0 {
			result.Condition = strings.TrimSpace(condEl.Text())
		}
		if result.Condition == "" {
			result.Condition = "İkinci El"
		}

		result.Publisher = extractNadirkitapPublisher(card)
		result.CargoFee, result.CargoUnknown = extractNadirkitapCargo(card)
		result.TotalPrice = result.Price + result.CargoFee

		// Only keep results with title containing the search phrase
		if result.Title != "" && result.Price > 0 && scraper.MatchesQuery(result.Title, query) {
			results = append(results, result)
		}
	})

	return results, nil
}

func resolveNadirkitapURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	if !strings.HasPrefix(href, "/") {
		href = "/" + href
	}
	return nadirkitapBaseURL + href
}

func extractNadirkitapPublisher(card *goquery.Selection) string {
	uls := card.Find("ul")
	if uls.Length() >= 2 {
		secondUL := uls.Eq(1)
		var parts []string
		secondUL.Find("li").Each(func(_ int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				parts = append(parts, text)
			}
		})
		if len(parts) > 0 {
			return strings.Join(parts, " / ")
		}
	}
	return ""
}

var cargoAmountRe = regexp.MustCompile(`(\d+[\.,]?\d*)\s*TL`)

func extractNadirkitapCargo(card *goquery.Selection) (float64, bool) {
	cardHTML, _ := card.Html()
	cardHTMLLower := strings.ToLower(cardHTML)

	if strings.Contains(cardHTMLLower, "ücretsiz kargo") || strings.Contains(cardHTMLLower, "kargo bedava") {
		return 0, false
	}

	var cargoFee float64
	found := false
	card.Find("img").Each(func(_ int, img *goquery.Selection) {
		if found {
			return
		}
		title, exists := img.Attr("title")
		if !exists {
			return
		}
		if strings.Contains(strings.ToLower(title), "kargo") {
			matches := cargoAmountRe.FindStringSubmatch(title)
			if len(matches) >= 2 {
				cargoFee = scraper.ParsePrice(matches[1] + " TL")
				found = true
			}
		}
	})
	if found {
		return cargoFee, false
	}

	if strings.Contains(cardHTMLLower, "kargo") {
		if strings.Contains(cardHTMLLower, "alıcıya ait") || strings.Contains(cardHTMLLower, "alıcıya") {
			return nadirkitapDefaultCargo, false
		}
		matches := cargoAmountRe.FindStringSubmatch(cardHTML)
		if len(matches) >= 2 {
			fee := scraper.ParsePrice(matches[1] + " TL")
			if fee > 0 && fee < 500 {
				return fee, false
			}
		}
	}

	return 0, true
}
