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
	amazonBaseURL      = "https://www.amazon.com.tr"
	amazonSearchURL    = "https://www.amazon.com.tr/s?k=%s&i=stripbooks"
	amazonDefaultCargo = 34.90
)

type Amazon struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewAmazon(limit int) *Amazon {
	return &Amazon{limit: limit}
}

func (a *Amazon) Name() string                                 { return "amazon.com.tr" }
func (a *Amazon) SiteCategory() scraper.Category               { return scraper.NewBook }
func (a *Amazon) SetFirecrawl(client *scraper.FirecrawlClient) { a.firecrawl = client }

func (a *Amazon) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(amazonSearchURL, url.QueryEscape(query))

	var pageHTML string
	var err error
	if a.firecrawl != nil {
		pageHTML, err = a.firecrawl.FetchHTML(ctx, searchURL, 5000)
	} else {
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, `[data-component-type="s-search-result"]`)
	}
	if err != nil {
		return nil, fmt.Errorf("amazon.com.tr: %w", err)
	}
	if pageHTML == "" {
		return nil, fmt.Errorf("amazon.com.tr erişim engellendi")
	}

	htmlLower := strings.ToLower(pageHTML)
	if strings.Contains(htmlLower, "captcha") ||
		(strings.Contains(htmlLower, "robot") && strings.Contains(htmlLower, "sorry")) ||
		strings.Contains(htmlLower, "automated access") {
		return nil, fmt.Errorf("amazon.com.tr erişim engellendi")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("amazon.com.tr: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0

	doc.Find(`[data-component-type="s-search-result"]`).Each(func(i int, s *goquery.Selection) {
		if productCount >= a.limit {
			return
		}

		asin, _ := s.Attr("data-asin")
		if asin == "" {
			return
		}

		sponsoredText := strings.ToLower(s.Find(".a-color-secondary .a-text-bold").Text())
		if strings.Contains(sponsoredText, "sponsor") {
			return
		}

		title := strings.TrimSpace(s.Find("h2 span").Text())
		if title == "" {
			return
		}

		productURL := ""
		if href, exists := s.Find("h2 a").Attr("href"); exists {
			productURL = resolveAmazonURL(strings.TrimSpace(href))
		}
		if productURL == "" {
			if href, exists := s.Find("a.a-link-normal.s-no-outline").Attr("href"); exists {
				productURL = resolveAmazonURL(strings.TrimSpace(href))
			}
		}

		price := 0.0
		priceText := strings.TrimSpace(s.Find(".a-price .a-offscreen").First().Text())
		if priceText != "" {
			price = scraper.ParsePrice(priceText)
		} else {
			whole := strings.TrimSpace(s.Find(".a-price-whole").First().Text())
			fraction := strings.TrimSpace(s.Find(".a-price-fraction").First().Text())
			if whole != "" {
				price = scraper.ParsePrice(whole + "," + fraction + " TL")
			}
		}

		if price <= 0 {
			return
		}

		author := ""
		infoRow := s.Find(".a-row.a-size-base.a-color-secondary")
		if infoRow.Length() > 0 {
			authorLink := infoRow.Find("a.a-size-base").First()
			if authorLink.Length() > 0 {
				author = strings.TrimSpace(authorLink.Text())
			} else {
				infoRow.Find("span.a-size-base").Each(func(_ int, span *goquery.Selection) {
					text := strings.TrimSpace(span.Text())
					if author == "" && text != "" && !strings.Contains(text, "|") && text != "–" {
						author = text
					}
				})
			}
		}

		results = append(results, scraper.BookResult{
			Title:      title,
			Author:     author,
			Price:      price,
			CargoFee:   amazonDefaultCargo,
			TotalPrice: price + amazonDefaultCargo,
			Condition:  scraper.NewBook.String(),
			Site:       "amazon.com.tr",
			URL:        productURL,
			Category:   scraper.NewBook,
		})

		results = append(results, scraper.BookResult{
			Title:       title,
			Author:      author,
			Price:       price,
			CargoFee:    0,
			TotalPrice:  price,
			FreeCargo:   true,
			LoyaltyNote: "Amazon Prime: ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			Site:        "amazon.com.tr",
			URL:         productURL,
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}

func resolveAmazonURL(href string) string {
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
	return amazonBaseURL + href
}
