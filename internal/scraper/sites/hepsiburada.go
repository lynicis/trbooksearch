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
	hepsiburadaBaseURL   = "https://www.hepsiburada.com"
	hepsiburadaSearchURL = "https://www.hepsiburada.com/ara?q=%s"
	hepsiburadaCargoFee  = 34.90
)

type Hepsiburada struct {
	limit     int
	firecrawl *scraper.FirecrawlClient
}

func NewHepsiburada(limit int) *Hepsiburada {
	return &Hepsiburada{limit: limit}
}

func (h *Hepsiburada) Name() string                                 { return "hepsiburada.com" }
func (h *Hepsiburada) SiteCategory() scraper.Category               { return scraper.NewBook }
func (h *Hepsiburada) SetFirecrawl(client *scraper.FirecrawlClient) { h.firecrawl = client }

func (h *Hepsiburada) Search(ctx context.Context, query string, searchType scraper.SearchType) ([]scraper.BookResult, error) {
	searchURL := fmt.Sprintf(hepsiburadaSearchURL, url.QueryEscape(query))

	var pageHTML string
	var err error
	if h.firecrawl != nil {
		pageHTML, err = h.firecrawl.FetchHTMLWithOptions(ctx, searchURL, scraper.FetchOptions{
			WaitFor: 8000,
			Timeout: 120000,
			Proxy:   "enhanced",
			Retries: 1,
		})
	} else {
		// Use FetchPageWithWait — Hepsiburada needs JS to render product cards
		pageHTML, err = scraper.FetchPageWithWait(ctx, searchURL, `[class*="productCard-module_article"]`)
	}
	if err != nil {
		return nil, fmt.Errorf("hepsiburada: %w", err)
	}
	if pageHTML == "" {
		return nil, nil // no products found
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("hepsiburada: parsing HTML: %w", err)
	}

	var results []scraper.BookResult
	productCount := 0

	doc.Find(`[class*="productCard-module_article"]`).Each(func(i int, card *goquery.Selection) {
		if productCount >= h.limit {
			return
		}

		titleEl := card.Find(`[class*="title-module_titleText"]`)
		title := strings.TrimSpace(titleEl.AttrOr("title", titleEl.Text()))
		if title == "" {
			title = strings.TrimSpace(titleEl.Text())
		}
		if title == "" {
			return
		}

		linkEl := card.Find(`a[class*="productCardLink-module"]`)
		href := linkEl.AttrOr("href", "")
		if href == "" {
			return
		}
		productURL := resolveHepsiburadaURL(href)

		priceIntEl := card.Find(`div[class*="price-module_finalPrice"]`)
		priceFracEl := card.Find(`span[class*="price-module_finalPriceFraction"]`)

		priceInt := strings.TrimSpace(priceIntEl.Text())
		priceFrac := strings.TrimSpace(priceFracEl.Text())

		if priceFrac != "" {
			priceInt = strings.TrimSuffix(priceInt, priceFrac)
			priceInt = strings.TrimSpace(priceInt)
		}

		priceStr := buildPriceString(priceInt, priceFrac)
		price := scraper.ParsePrice(priceStr)
		if price <= 0 {
			return
		}

		results = append(results, scraper.BookResult{
			Title:      title,
			Price:      price,
			CargoFee:   hepsiburadaCargoFee,
			TotalPrice: price + hepsiburadaCargoFee,
			Condition:  scraper.NewBook.String(),
			URL:        productURL,
			Site:       "hepsiburada.com",
			Category:   scraper.NewBook,
		})

		results = append(results, scraper.BookResult{
			Title:       title,
			Price:       price,
			CargoFee:    0,
			TotalPrice:  price,
			FreeCargo:   true,
			LoyaltyNote: "Hepsiburada Premium: ücretsiz kargo",
			Condition:   scraper.NewBook.String(),
			URL:         productURL,
			Site:        "hepsiburada.com",
			Category:    scraper.NewBook,
		})
		productCount++
	})

	return results, nil
}

func resolveHepsiburadaURL(href string) string {
	if strings.HasPrefix(href, "https://adservice.hepsiburada.com") {
		parsed, err := url.Parse(href)
		if err == nil {
			redirect := parsed.Query().Get("redirect")
			if redirect != "" {
				return redirect
			}
		}
	}
	if strings.HasPrefix(href, "/") {
		return hepsiburadaBaseURL + href
	}
	return href
}

func buildPriceString(intPart, fracPart string) string {
	intPart = strings.TrimSpace(intPart)
	fracPart = strings.TrimSpace(fracPart)
	if intPart == "" {
		return ""
	}
	if fracPart == "" {
		return intPart
	}
	return intPart + fracPart
}
