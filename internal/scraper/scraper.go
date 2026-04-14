package scraper

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

// SearchType determines how to search for a book.
type SearchType int

const (
	TitleSearch SearchType = iota
	ISBNSearch
)

// Category determines whether a book is used or new.
type Category int

const (
	UsedBook Category = iota
	NewBook
)

func (c Category) String() string {
	if c == UsedBook {
		return "İkinci El"
	}
	return "Yeni"
}

// BookResult represents a single book listing from any site.
type BookResult struct {
	Title        string
	Author       string
	Publisher    string
	Price        float64  // Book price in TL
	CargoFee     float64  // Shipping cost in TL (0 if free or unknown)
	TotalPrice   float64  // Price + CargoFee
	FreeCargo    bool     // Whether loyalty program covers cargo
	LoyaltyNote  string   // e.g. "Hepsiburada Premium ile ücretsiz kargo"
	Condition    string   // "Yeni" / "İkinci El" / specific condition text
	Seller       string   // Seller/shop name
	URL          string   // Direct link to the listing
	Site         string   // e.g. "kitapyurdu.com"
	Category     Category // Used or New
	CargoUnknown bool     // True if cargo fee couldn't be determined
}

// Scraper is the interface each website scraper must implement.
type Scraper interface {
	Name() string
	Search(ctx context.Context, query string, searchType SearchType) ([]BookResult, error)
	SiteCategory() Category
	SetFirecrawl(client *FirecrawlClient)
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
}

// RandomUserAgent returns a random browser User-Agent string.
func RandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// launchBrowser starts a fresh isolated headless Chromium instance.
func launchBrowser() (*rod.Browser, error) {
	l, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(l)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to browser: %w", err)
	}

	return browser, nil
}

// FetchPage launches a fresh isolated browser, creates a stealth page,
// navigates to the URL, waits for page load, extracts HTML, and closes
// the browser. Each call gets a completely fresh browser fingerprint.
func FetchPage(ctx context.Context, url string) (string, error) {
	browser, err := launchBrowser()
	if err != nil {
		return "", err
	}
	defer browser.MustClose()

	page, err := stealth.Page(browser)
	if err != nil {
		return "", fmt.Errorf("creating stealth page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx)

	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigating to %s: %w", url, err)
	}

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("waiting for page load: %w", err)
	}

	// Brief pause to let JS rendering complete after load event.
	time.Sleep(3 * time.Second)

	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("getting page HTML: %w", err)
	}

	return html, nil
}

// FetchPageWithWait launches a fresh isolated browser, creates a stealth page,
// navigates, waits for page load, then waits up to 20s for a specific CSS
// selector to appear before extracting HTML.
func FetchPageWithWait(ctx context.Context, url string, waitSelector string) (string, error) {
	browser, err := launchBrowser()
	if err != nil {
		return "", err
	}
	defer browser.MustClose()

	page, err := stealth.Page(browser)
	if err != nil {
		return "", fmt.Errorf("creating stealth page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx)

	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigating to %s: %w", url, err)
	}

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("waiting for page load: %w", err)
	}

	// Wait for the specific selector to appear in the DOM.
	if waitSelector != "" {
		if _, err := page.Timeout(20 * time.Second).Element(waitSelector); err != nil {
			return "", nil
		}
	}

	// Brief pause to let remaining JS rendering settle.
	time.Sleep(1 * time.Second)

	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("getting page HTML: %w", err)
	}

	return html, nil
}

// ParsePrice extracts a float64 price from a Turkish-formatted price string.
func ParsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "TL", "")
	s = strings.ReplaceAll(s, "₺", "")
	s = strings.TrimSpace(s)

	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}

	var price float64
	_, _ = fmt.Sscanf(s, "%f", &price)
	return price
}

// MatchesQuery checks if a title contains the search phrase (case-insensitive).
// Used to filter out irrelevant results from sites with broad search (nadirkitap, trendyol).
func MatchesQuery(title, query string) bool {
	return strings.Contains(strings.ToLower(title), strings.ToLower(query))
}
