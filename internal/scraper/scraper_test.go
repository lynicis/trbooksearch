package scraper

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// ParsePrice
// ---------------------------------------------------------------------------

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		// Standard Turkish format: thousands dot, decimal comma, TL suffix
		{name: "simple with TL", input: "100,00 TL", want: 100.0},
		{name: "thousands separator with TL", input: "1.234,56 TL", want: 1234.56},
		{name: "lira sign suffix", input: "99,90₺", want: 99.90},

		// No comma means the dot is a decimal point (direct parse path)
		{name: "dot decimal no comma", input: "34.90", want: 34.90},

		// Edge cases
		{name: "empty string", input: "", want: 0},
		{name: "surrounding whitespace", input: "  100,00 TL  ", want: 100.0},
		{name: "comma decimal no suffix", input: "29,99", want: 29.99},
		{name: "large thousands", input: "1.000,00", want: 1000.0},
		{name: "non-numeric", input: "abc", want: 0},

		// More realistic prices
		{name: "no decimal part", input: "50 TL", want: 50.0},
		{name: "zero", input: "0", want: 0},
		{name: "lira sign with space", input: "45,00 ₺", want: 45.0},
		{name: "large price", input: "12.345,67 TL", want: 12345.67},
		{name: "only TL", input: "TL", want: 0},
		{name: "only lira sign", input: "₺", want: 0},
		{name: "whitespace only", input: "   ", want: 0},
		{name: "comma only decimal", input: "0,99", want: 0.99},
		{name: "multiple thousands separators", input: "1.000.000,50 TL", want: 1000000.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePrice(tt.input)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("ParsePrice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Category.String
// ---------------------------------------------------------------------------

func TestCategoryString(t *testing.T) {
	tests := []struct {
		name string
		cat  Category
		want string
	}{
		{name: "UsedBook", cat: UsedBook, want: "İkinci El"},
		{name: "NewBook", cat: NewBook, want: "Yeni"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cat.String()
			if got != tt.want {
				t.Errorf("Category(%d).String() = %q, want %q", tt.cat, got, tt.want)
			}
		})
	}
}

func TestCategoryStringUnknownValue(t *testing.T) {
	// Any value other than UsedBook (0) falls through to "Yeni"
	unknown := Category(99)
	if got := unknown.String(); got != "Yeni" {
		t.Errorf("Category(99).String() = %q, want %q", got, "Yeni")
	}
}

// ---------------------------------------------------------------------------
// SearchType constants
// ---------------------------------------------------------------------------

func TestSearchTypeConstants(t *testing.T) {
	if TitleSearch != 0 {
		t.Errorf("TitleSearch = %d, want 0", TitleSearch)
	}
	if ISBNSearch != 1 {
		t.Errorf("ISBNSearch = %d, want 1", ISBNSearch)
	}
}

func TestSearchTypeOrdering(t *testing.T) {
	// Confirm iota ordering: Title < ISBN
	if TitleSearch >= ISBNSearch {
		t.Error("expected TitleSearch < ISBNSearch")
	}
}

// ---------------------------------------------------------------------------
// Category constants
// ---------------------------------------------------------------------------

func TestCategoryConstants(t *testing.T) {
	if UsedBook != 0 {
		t.Errorf("UsedBook = %d, want 0", UsedBook)
	}
	if NewBook != 1 {
		t.Errorf("NewBook = %d, want 1", NewBook)
	}
}

// ---------------------------------------------------------------------------
// RandomUserAgent
// ---------------------------------------------------------------------------

func TestRandomUserAgentNonEmpty(t *testing.T) {
	for i := 0; i < 50; i++ {
		ua := RandomUserAgent()
		if ua == "" {
			t.Fatal("RandomUserAgent() returned empty string")
		}
	}
}

func TestRandomUserAgentFromKnownSet(t *testing.T) {
	known := map[string]bool{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36":       true,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36": true,
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36":                 true,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0":                                      true,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0":                                  true,
	}

	for i := 0; i < 100; i++ {
		ua := RandomUserAgent()
		if !known[ua] {
			t.Errorf("RandomUserAgent() returned unknown UA: %q", ua)
		}
	}
}

func TestRandomUserAgentVariety(t *testing.T) {
	// With 5 possible agents and 200 calls the probability of seeing
	// only one distinct value is negligible.
	seen := make(map[string]struct{})
	for i := 0; i < 200; i++ {
		seen[RandomUserAgent()] = struct{}{}
	}
	if len(seen) < 2 {
		t.Errorf("expected at least 2 distinct user agents in 200 calls, got %d", len(seen))
	}
}

// ---------------------------------------------------------------------------
// BookResult struct
// ---------------------------------------------------------------------------

func TestBookResultFields(t *testing.T) {
	br := BookResult{
		Title:        "Sefiller",
		Author:       "Victor Hugo",
		Publisher:    "İş Bankası Kültür Yayınları",
		Price:        85.50,
		CargoFee:     9.99,
		TotalPrice:   95.49,
		FreeCargo:    false,
		LoyaltyNote:  "",
		Condition:    "Yeni",
		Seller:       "kitapyurdu",
		URL:          "https://www.kitapyurdu.com/kitap/sefiller/12345.html",
		Site:         "kitapyurdu.com",
		Category:     NewBook,
		CargoUnknown: false,
	}

	if br.Title != "Sefiller" {
		t.Errorf("Title = %q, want %q", br.Title, "Sefiller")
	}
	if br.Author != "Victor Hugo" {
		t.Errorf("Author = %q, want %q", br.Author, "Victor Hugo")
	}
	if br.Publisher != "İş Bankası Kültür Yayınları" {
		t.Errorf("Publisher = %q", br.Publisher)
	}
	if math.Abs(br.Price-85.50) > 0.001 {
		t.Errorf("Price = %v, want 85.50", br.Price)
	}
	if math.Abs(br.CargoFee-9.99) > 0.001 {
		t.Errorf("CargoFee = %v, want 9.99", br.CargoFee)
	}
	if math.Abs(br.TotalPrice-95.49) > 0.001 {
		t.Errorf("TotalPrice = %v, want 95.49", br.TotalPrice)
	}
	if br.FreeCargo {
		t.Error("FreeCargo should be false")
	}
	if br.LoyaltyNote != "" {
		t.Errorf("LoyaltyNote = %q, want empty", br.LoyaltyNote)
	}
	if br.Condition != "Yeni" {
		t.Errorf("Condition = %q, want %q", br.Condition, "Yeni")
	}
	if br.Seller != "kitapyurdu" {
		t.Errorf("Seller = %q", br.Seller)
	}
	if br.URL != "https://www.kitapyurdu.com/kitap/sefiller/12345.html" {
		t.Errorf("URL = %q", br.URL)
	}
	if br.Site != "kitapyurdu.com" {
		t.Errorf("Site = %q", br.Site)
	}
	if br.Category != NewBook {
		t.Errorf("Category = %d, want %d (NewBook)", br.Category, NewBook)
	}
	if br.CargoUnknown {
		t.Error("CargoUnknown should be false")
	}
}

func TestBookResultZeroValue(t *testing.T) {
	var br BookResult

	if br.Title != "" {
		t.Errorf("zero-value Title = %q, want empty", br.Title)
	}
	if br.Price != 0 {
		t.Errorf("zero-value Price = %v, want 0", br.Price)
	}
	if br.Category != UsedBook {
		t.Errorf("zero-value Category = %d, want %d (UsedBook)", br.Category, UsedBook)
	}
	if br.FreeCargo {
		t.Error("zero-value FreeCargo should be false")
	}
	if br.CargoUnknown {
		t.Error("zero-value CargoUnknown should be false")
	}
}

func TestBookResultFreeCargo(t *testing.T) {
	br := BookResult{
		Title:       "Test Kitabı",
		Price:       50.0,
		CargoFee:    0,
		TotalPrice:  50.0,
		FreeCargo:   true,
		LoyaltyNote: "Hepsiburada Premium ile ücretsiz kargo",
		Category:    NewBook,
	}

	if !br.FreeCargo {
		t.Error("FreeCargo should be true")
	}
	if br.LoyaltyNote != "Hepsiburada Premium ile ücretsiz kargo" {
		t.Errorf("LoyaltyNote = %q", br.LoyaltyNote)
	}
	if br.TotalPrice != br.Price {
		t.Errorf("TotalPrice (%v) should equal Price (%v) when cargo is free",
			br.TotalPrice, br.Price)
	}
}
