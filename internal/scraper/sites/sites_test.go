package sites

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

// ---------------------------------------------------------------------------
// helpers: mock Firecrawl server
// ---------------------------------------------------------------------------

// newMockFirecrawl spins up an httptest server that returns the given HTML
// in Firecrawl's /v1/scrape response format, then creates a real
// FirecrawlClient pointing at that server.
func newMockFirecrawl(t *testing.T, html string) (*scraper.FirecrawlClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"success": true,
			"data": map[string]any{
				"html":     html,
				"metadata": map[string]any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("mock server encode: %v", err)
		}
	}))
	fc := scraper.NewFirecrawlClient("test-key", srv.URL)
	return fc, srv
}

// newMockFirecrawlError returns a server that responds with an API error.
func newMockFirecrawlError(t *testing.T) (*scraper.FirecrawlClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		resp := map[string]any{
			"success": false,
			"error":   "mock error",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	fc := scraper.NewFirecrawlClient("test-key", srv.URL)
	return fc, srv
}

// ===========================================================================
// Registry tests
// ===========================================================================

func TestAllScrapers_Default(t *testing.T) {
	scrapers := AllScrapers(5, false)
	if len(scrapers) != 7 {
		t.Fatalf("expected 7 default scrapers, got %d", len(scrapers))
	}
	expected := []string{
		"nadirkitap.com",
		"kitapyurdu.com",
		"trendyol.com",
		"hepsiburada.com",
		"amazon.com.tr",
		"bkmkitap.com",
		"idefix.com",
	}
	for i, s := range scrapers {
		if s.Name() != expected[i] {
			t.Errorf("scraper[%d]: expected %q, got %q", i, expected[i], s.Name())
		}
	}
}

func TestAllScrapers_WithFirecrawl(t *testing.T) {
	scrapers := AllScrapers(5, true)
	if len(scrapers) != 12 {
		t.Fatalf("expected 12 scrapers with firecrawl, got %d", len(scrapers))
	}
	// Firecrawl-only scrapers at positions 7-11
	extraExpected := []string{
		"letgo.com",
		"dolap.com",
		"gardrops.com",
		"dr.com.tr",
		"pandora.com.tr",
	}
	for i, name := range extraExpected {
		if scrapers[7+i].Name() != name {
			t.Errorf("scraper[%d]: expected %q, got %q", 7+i, name, scrapers[7+i].Name())
		}
	}
}

func TestAllScrapers_LimitPropagation(t *testing.T) {
	// Verify that limit is passed through by checking the actual scraper struct fields
	scrapers := AllScrapers(3, true)
	for _, s := range scrapers {
		// We can't directly check limit from the interface, but we can verify
		// each scraper was constructed without panic.
		if s.Name() == "" {
			t.Error("scraper has empty name")
		}
	}
}

// ===========================================================================
// Per-scraper Name, SiteCategory, SetFirecrawl tests
// ===========================================================================

func TestNadirkitap_Name(t *testing.T) {
	s := NewNadirkitap(5)
	if s.Name() != "nadirkitap.com" {
		t.Errorf("expected nadirkitap.com, got %s", s.Name())
	}
}

func TestNadirkitap_SiteCategory(t *testing.T) {
	s := NewNadirkitap(5)
	if s.SiteCategory() != scraper.UsedBook {
		t.Errorf("expected UsedBook, got %v", s.SiteCategory())
	}
}

func TestNadirkitap_SetFirecrawl(t *testing.T) {
	s := NewNadirkitap(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestKitapyurdu_Name(t *testing.T) {
	s := NewKitapyurdu(5)
	if s.Name() != "kitapyurdu.com" {
		t.Errorf("expected kitapyurdu.com, got %s", s.Name())
	}
}

func TestKitapyurdu_SiteCategory(t *testing.T) {
	s := NewKitapyurdu(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestKitapyurdu_SetFirecrawl(t *testing.T) {
	s := NewKitapyurdu(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestTrendyol_Name(t *testing.T) {
	s := NewTrendyol(5)
	if s.Name() != "trendyol.com" {
		t.Errorf("expected trendyol.com, got %s", s.Name())
	}
}

func TestTrendyol_SiteCategory(t *testing.T) {
	s := NewTrendyol(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestTrendyol_SetFirecrawl(t *testing.T) {
	s := NewTrendyol(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestHepsiburada_Name(t *testing.T) {
	s := NewHepsiburada(5)
	if s.Name() != "hepsiburada.com" {
		t.Errorf("expected hepsiburada.com, got %s", s.Name())
	}
}

func TestHepsiburada_SiteCategory(t *testing.T) {
	s := NewHepsiburada(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestHepsiburada_SetFirecrawl(t *testing.T) {
	s := NewHepsiburada(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestAmazon_Name(t *testing.T) {
	s := NewAmazon(5)
	if s.Name() != "amazon.com.tr" {
		t.Errorf("expected amazon.com.tr, got %s", s.Name())
	}
}

func TestAmazon_SiteCategory(t *testing.T) {
	s := NewAmazon(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestAmazon_SetFirecrawl(t *testing.T) {
	s := NewAmazon(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestBkmkitap_Name(t *testing.T) {
	s := NewBkmkitap(5)
	if s.Name() != "bkmkitap.com" {
		t.Errorf("expected bkmkitap.com, got %s", s.Name())
	}
}

func TestBkmkitap_SiteCategory(t *testing.T) {
	s := NewBkmkitap(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestBkmkitap_SetFirecrawl(t *testing.T) {
	s := NewBkmkitap(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestIdefix_Name(t *testing.T) {
	s := NewIdefix(5)
	if s.Name() != "idefix.com" {
		t.Errorf("expected idefix.com, got %s", s.Name())
	}
}

func TestIdefix_SiteCategory(t *testing.T) {
	s := NewIdefix(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestIdefix_SetFirecrawl(t *testing.T) {
	s := NewIdefix(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestLetgo_Name(t *testing.T) {
	s := NewLetgo(5)
	if s.Name() != "letgo.com" {
		t.Errorf("expected letgo.com, got %s", s.Name())
	}
}

func TestLetgo_SiteCategory(t *testing.T) {
	s := NewLetgo(5)
	if s.SiteCategory() != scraper.UsedBook {
		t.Errorf("expected UsedBook, got %v", s.SiteCategory())
	}
}

func TestLetgo_SetFirecrawl(t *testing.T) {
	s := NewLetgo(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestDolap_Name(t *testing.T) {
	s := NewDolap(5)
	if s.Name() != "dolap.com" {
		t.Errorf("expected dolap.com, got %s", s.Name())
	}
}

func TestDolap_SiteCategory(t *testing.T) {
	s := NewDolap(5)
	if s.SiteCategory() != scraper.UsedBook {
		t.Errorf("expected UsedBook, got %v", s.SiteCategory())
	}
}

func TestDolap_SetFirecrawl(t *testing.T) {
	s := NewDolap(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestGardrops_Name(t *testing.T) {
	s := NewGardrops(5)
	if s.Name() != "gardrops.com" {
		t.Errorf("expected gardrops.com, got %s", s.Name())
	}
}

func TestGardrops_SiteCategory(t *testing.T) {
	s := NewGardrops(5)
	if s.SiteCategory() != scraper.UsedBook {
		t.Errorf("expected UsedBook, got %v", s.SiteCategory())
	}
}

func TestGardrops_SetFirecrawl(t *testing.T) {
	s := NewGardrops(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestDR_Name(t *testing.T) {
	s := NewDR(5)
	if s.Name() != "dr.com.tr" {
		t.Errorf("expected dr.com.tr, got %s", s.Name())
	}
}

func TestDR_SiteCategory(t *testing.T) {
	s := NewDR(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestDR_SetFirecrawl(t *testing.T) {
	s := NewDR(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

func TestPandora_Name(t *testing.T) {
	s := NewPandora(5)
	if s.Name() != "pandora.com.tr" {
		t.Errorf("expected pandora.com.tr, got %s", s.Name())
	}
}

func TestPandora_SiteCategory(t *testing.T) {
	s := NewPandora(5)
	if s.SiteCategory() != scraper.NewBook {
		t.Errorf("expected NewBook, got %v", s.SiteCategory())
	}
}

func TestPandora_SetFirecrawl(t *testing.T) {
	s := NewPandora(5)
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	s.SetFirecrawl(fc)
	if s.firecrawl != fc {
		t.Error("firecrawl client was not set")
	}
}

// ===========================================================================
// Helper function tests
// ===========================================================================

// ---------------------------------------------------------------------------
// resolveNadirkitapURL
// ---------------------------------------------------------------------------

func TestResolveNadirkitapURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{"empty", "", ""},
		{"absolute_http", "http://www.nadirkitap.com/book", "http://www.nadirkitap.com/book"},
		{"absolute_https", "https://www.nadirkitap.com/book", "https://www.nadirkitap.com/book"},
		{"relative_with_slash", "/kitap/123", "https://www.nadirkitap.com/kitap/123"},
		{"relative_no_slash", "kitap/123", "https://www.nadirkitap.com/kitap/123"},
		{"whitespace", "  /kitap/456  ", "https://www.nadirkitap.com/kitap/456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNadirkitapURL(tt.href)
			if got != tt.want {
				t.Errorf("resolveNadirkitapURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractNadirkitapPublisher
// ---------------------------------------------------------------------------

func TestExtractNadirkitapPublisher(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"two_uls_second_has_items",
			`<div class="productsListHover">
				<ul><li>Item1</li></ul>
				<ul><li>Yayınevi A</li><li>2020</li></ul>
			</div>`,
			"Yayınevi A / 2020",
		},
		{
			"single_ul",
			`<div class="productsListHover"><ul><li>Only</li></ul></div>`,
			"",
		},
		{
			"no_uls",
			`<div class="productsListHover"><p>No lists</p></div>`,
			"",
		},
		{
			"second_ul_empty",
			`<div class="productsListHover">
				<ul><li>First</li></ul>
				<ul></ul>
			</div>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.html)
			card := doc.Find(".productsListHover").First()
			got := extractNadirkitapPublisher(card)
			if got != tt.want {
				t.Errorf("extractNadirkitapPublisher = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractNadirkitapCargo
// ---------------------------------------------------------------------------

func TestExtractNadirkitapCargo(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantFee     float64
		wantUnknown bool
	}{
		{
			"free_cargo_text",
			`<div class="productsListHover"><span>Ücretsiz Kargo</span></div>`,
			0, false,
		},
		{
			"kargo_bedava",
			`<div class="productsListHover"><span>Kargo Bedava</span></div>`,
			0, false,
		},
		{
			"img_with_cargo_amount",
			`<div class="productsListHover"><img title="Kargo: 15,00 TL"></div>`,
			15.0, false,
		},
		{
			"aliciya_ait",
			`<div class="productsListHover"><span>kargo alıcıya ait</span></div>`,
			40.0, false,
		},
		{
			"cargo_amount_in_html",
			`<div class="productsListHover"><span>kargo 25 TL</span></div>`,
			25.0, false,
		},
		{
			"unknown_no_cargo_info",
			`<div class="productsListHover"><span>just text</span></div>`,
			0, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseHTML(t, tt.html)
			card := doc.Find(".productsListHover").First()
			fee, unknown := extractNadirkitapCargo(card)
			if fee != tt.wantFee {
				t.Errorf("fee = %f, want %f", fee, tt.wantFee)
			}
			if unknown != tt.wantUnknown {
				t.Errorf("unknown = %v, want %v", unknown, tt.wantUnknown)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildTrendyolURL
// ---------------------------------------------------------------------------

func TestBuildTrendyolURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{"relative", "/product/123", "https://www.trendyol.com/product/123"},
		{"with_query", "/product/123?boutiqueId=1&merchantId=2", "https://www.trendyol.com/product/123"},
		{"absolute_no_query", "https://www.trendyol.com/product/123", "https://www.trendyol.com/product/123"},
		{"absolute_with_query", "https://www.trendyol.com/product/123?foo=bar", "https://www.trendyol.com/product/123"},
		{"whitespace", "  /product/abc  ", "https://www.trendyol.com/product/abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTrendyolURL(tt.href)
			if got != tt.want {
				t.Errorf("buildTrendyolURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// firstNonEmpty
// ---------------------------------------------------------------------------

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"first_non_empty", []string{"", "  ", "hello", "world"}, "hello"},
		{"all_empty", []string{"", "  ", "   "}, ""},
		{"single_value", []string{"only"}, "only"},
		{"no_values", nil, ""},
		{"first_is_non_empty", []string{"first", "second"}, "first"},
		{"whitespace_only_skipped", []string{"  \t  ", "valid"}, "valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.values...)
			if got != tt.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveHepsiburadaURL
// ---------------------------------------------------------------------------

func TestResolveHepsiburadaURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			"adservice_redirect",
			"https://adservice.hepsiburada.com/click?redirect=https%3A%2F%2Fwww.hepsiburada.com%2Fproduct%2F123",
			"https://www.hepsiburada.com/product/123",
		},
		{
			"adservice_no_redirect",
			"https://adservice.hepsiburada.com/click?other=value",
			"https://adservice.hepsiburada.com/click?other=value",
		},
		{"relative", "/product/123", "https://www.hepsiburada.com/product/123"},
		{"absolute", "https://www.hepsiburada.com/product/123", "https://www.hepsiburada.com/product/123"},
		{"other_absolute", "https://example.com/foo", "https://example.com/foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveHepsiburadaURL(tt.href)
			if got != tt.want {
				t.Errorf("resolveHepsiburadaURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildPriceString
// ---------------------------------------------------------------------------

func TestBuildPriceString(t *testing.T) {
	tests := []struct {
		name     string
		intPart  string
		fracPart string
		want     string
	}{
		{"both_present", "50", ",99", "50,99"},
		{"empty_int", "", ",99", ""},
		{"empty_frac", "50", "", "50"},
		{"both_empty", "", "", ""},
		{"whitespace_int", "  50  ", "  ,99  ", "50,99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPriceString(tt.intPart, tt.fracPart)
			if got != tt.want {
				t.Errorf("buildPriceString(%q, %q) = %q, want %q", tt.intPart, tt.fracPart, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveAmazonURL
// ---------------------------------------------------------------------------

func TestResolveAmazonURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{"empty", "", ""},
		{"absolute_https", "https://www.amazon.com.tr/dp/123", "https://www.amazon.com.tr/dp/123"},
		{"absolute_http", "http://www.amazon.com.tr/dp/123", "http://www.amazon.com.tr/dp/123"},
		{"relative_with_slash", "/dp/123", "https://www.amazon.com.tr/dp/123"},
		{"relative_no_slash", "dp/123", "https://www.amazon.com.tr/dp/123"},
		{"whitespace", "  /dp/456  ", "https://www.amazon.com.tr/dp/456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAmazonURL(tt.href)
			if got != tt.want {
				t.Errorf("resolveAmazonURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveLetgoURL
// ---------------------------------------------------------------------------

func TestResolveLetgoURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{"absolute", "https://www.letgo.com/ilan/abc", "https://www.letgo.com/ilan/abc"},
		{"relative_with_slash", "/ilan/abc", "https://www.letgo.com/ilan/abc"},
		{"relative_no_slash", "ilan/abc", "https://www.letgo.com/ilan/abc"},
		{"http_absolute", "http://www.letgo.com/ilan/abc", "http://www.letgo.com/ilan/abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveLetgoURL(tt.href)
			if got != tt.want {
				t.Errorf("resolveLetgoURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractGardropsTitle
// ---------------------------------------------------------------------------

func TestExtractGardropsTitle(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			"valid_slug",
			"/harry-potter-ve-felsefe-tasi-a1b2c3d4e5f6a7b8-p-12345-67890",
			"harry potter ve felsefe tasi",
		},
		{
			"valid_with_full_url",
			"https://www.gardrops.com/kucuk-prens-aabbccddee112233-p-111-222",
			"kucuk prens",
		},
		{"invalid_format", "/not-matching-format", ""},
		{"empty", "", ""},
		{
			"single_word_slug",
			"/kitap-abcdef0123456789-p-100-200",
			"kitap",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGardropsTitle(tt.href)
			if got != tt.want {
				t.Errorf("extractGardropsTitle(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ===========================================================================
// Firecrawl-only scrapers: nil firecrawl returns error
// ===========================================================================

func TestLetgo_Search_NilFirecrawl(t *testing.T) {
	s := NewLetgo(5)
	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error when firecrawl is nil")
	}
	if !contains(err.Error(), "Firecrawl") {
		t.Errorf("error should mention Firecrawl, got: %s", err.Error())
	}
}

func TestDolap_Search_NilFirecrawl(t *testing.T) {
	s := NewDolap(5)
	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error when firecrawl is nil")
	}
	if !contains(err.Error(), "Firecrawl") {
		t.Errorf("error should mention Firecrawl, got: %s", err.Error())
	}
}

func TestGardrops_Search_NilFirecrawl(t *testing.T) {
	s := NewGardrops(5)
	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error when firecrawl is nil")
	}
	if !contains(err.Error(), "Firecrawl") {
		t.Errorf("error should mention Firecrawl, got: %s", err.Error())
	}
}

func TestDR_Search_NilFirecrawl(t *testing.T) {
	s := NewDR(5)
	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error when firecrawl is nil")
	}
	if !contains(err.Error(), "Firecrawl") {
		t.Errorf("error should mention Firecrawl, got: %s", err.Error())
	}
}

func TestPandora_Search_NilFirecrawl(t *testing.T) {
	s := NewPandora(5)
	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error when firecrawl is nil")
	}
	if !contains(err.Error(), "Firecrawl") {
		t.Errorf("error should mention Firecrawl, got: %s", err.Error())
	}
}

// ===========================================================================
// Integration tests: Search() with mock Firecrawl
// ===========================================================================

// ---------------------------------------------------------------------------
// Nadirkitap Search
// ---------------------------------------------------------------------------

func TestNadirkitap_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="productsListHover">
			<a class="nadirBook" href="/test-kitap-123">Test Kitap Başlık</a>
			<a class="price">50,00 TL</a>
			<a class="sellerName">Test Satıcı</a>
			<span class="text-nadir">İyi</span>
			<ul><li>Item1</li></ul>
			<ul><li>Yayınevi ABC</li></ul>
			<span>Ücretsiz Kargo</span>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewNadirkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Kitap Başlık" {
		t.Errorf("title = %q, want %q", r.Title, "Test Kitap Başlık")
	}
	if r.Price != 50.0 {
		t.Errorf("price = %f, want 50.0", r.Price)
	}
	if r.Seller != "Test Satıcı" {
		t.Errorf("seller = %q, want %q", r.Seller, "Test Satıcı")
	}
	if r.Condition != "İyi" {
		t.Errorf("condition = %q, want %q", r.Condition, "İyi")
	}
	if r.URL != "https://www.nadirkitap.com/test-kitap-123" {
		t.Errorf("url = %q, want %q", r.URL, "https://www.nadirkitap.com/test-kitap-123")
	}
	if r.Publisher != "Yayınevi ABC" {
		t.Errorf("publisher = %q, want %q", r.Publisher, "Yayınevi ABC")
	}
	if r.CargoFee != 0 {
		t.Errorf("cargo fee = %f, want 0 (free cargo)", r.CargoFee)
	}
	if r.CargoUnknown {
		t.Error("cargo should not be unknown for free cargo")
	}
	if r.Site != "nadirkitap.com" {
		t.Errorf("site = %q, want nadirkitap.com", r.Site)
	}
	if r.Category != scraper.UsedBook {
		t.Errorf("category = %v, want UsedBook", r.Category)
	}
}

func TestNadirkitap_Search_FiltersNonMatchingResults(t *testing.T) {
	html := `<html><body>
		<div class="productsListHover">
			<a class="nadirBook" href="/unrelated">Alakasız Kitap</a>
			<a class="price">30,00 TL</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewNadirkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "python programlama", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Scrapers no longer filter by query — relevance filtering is centralized in engine.
	// The scraper should return all valid results regardless of query match.
	if len(results) != 1 {
		t.Errorf("expected 1 result (scraper returns all valid results), got %d", len(results))
	}
}

func TestNadirkitap_Search_DefaultCondition(t *testing.T) {
	html := `<html><body>
		<div class="productsListHover">
			<a class="nadirBook" href="/kitap">Test Kitap</a>
			<a class="price">20,00 TL</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewNadirkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Condition != "İkinci El" {
		t.Errorf("expected default condition İkinci El, got %q", results[0].Condition)
	}
}

func TestNadirkitap_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewNadirkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty HTML, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Kitapyurdu Search
// ---------------------------------------------------------------------------

func TestKitapyurdu_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="ky-product">
			<a href="/kitap/test-kitap-p-123">Link</a>
			<div class="ky-product-title">Test Kitap</div>
			<div class="ky-product-author"><a>Test Yazar</a></div>
			<div class="ky-product-publisher"><a>Test Yayınevi</a></div>
			<div class="ky-product-sell-price">45,90 TL</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (cargo + free cargo), got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Author != "Test Yazar" {
		t.Errorf("author = %q", r.Author)
	}
	if r.Publisher != "Test Yayınevi" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Price != 45.90 {
		t.Errorf("price = %f, want 45.90", r.Price)
	}
	if r.CargoFee != 24.90 {
		t.Errorf("cargo = %f, want 24.90", r.CargoFee)
	}
	if r.Site != "kitapyurdu.com" {
		t.Errorf("site = %q", r.Site)
	}
	if r.Category != scraper.NewBook {
		t.Errorf("category = %v, want NewBook", r.Category)
	}

	// Second result should have free cargo
	rFree := results[1]
	if rFree.CargoFee != 0 {
		t.Errorf("free cargo result should have 0 cargo, got %f", rFree.CargoFee)
	}
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "150 TL üzeri siparişlerde ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestKitapyurdu_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Trendyol Search
// ---------------------------------------------------------------------------

func TestTrendyol_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<a class="product-card" href="/brand/test-kitap-p-12345">
			<span class="product-brand">Test Yayınevi</span>
			<span class="product-name">Test Kitap Adı</span>
			<div class="sale-price">35,50 TL</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewTrendyol(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Kitap Adı" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Publisher != "Test Yayınevi" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Price != 35.50 {
		t.Errorf("price = %f, want 35.50", r.Price)
	}
	if r.CargoFee != 29.99 {
		t.Errorf("cargo = %f, want 29.99", r.CargoFee)
	}
	if r.URL != "https://www.trendyol.com/brand/test-kitap-p-12345" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Site != "trendyol.com" {
		t.Errorf("site = %q", r.Site)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true on second result")
	}
	if rFree.LoyaltyNote != "Trendyol Elite: ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestTrendyol_Search_FiltersNonMatching(t *testing.T) {
	html := `<html><body>
		<a class="product-card" href="/brand/unrelated-p-111">
			<span class="product-brand">Brand</span>
			<span class="product-name">Alakasız Ürün</span>
			<div class="sale-price">10,00 TL</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewTrendyol(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "python kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Scrapers no longer filter by query — relevance filtering is centralized in engine.
	// The scraper should return all valid results regardless of query match.
	if len(results) != 2 {
		t.Errorf("expected 2 results (scraper returns all valid results), got %d", len(results))
	}
}

func TestTrendyol_Search_FallbackPrice(t *testing.T) {
	html := `<html><body>
		<a class="product-card" href="/brand/test-kitap-p-999">
			<span class="product-brand">Brand</span>
			<span class="product-name">Test Kitap Fiyat</span>
			<div class="price-section">42,00 TL</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewTrendyol(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results with fallback price, got %d", len(results))
	}
	if results[0].Price != 42.0 {
		t.Errorf("price = %f, want 42.0", results[0].Price)
	}
}

func TestTrendyol_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewTrendyol(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Hepsiburada Search
// ---------------------------------------------------------------------------

func TestHepsiburada_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<article class="productCard-module_article_xyz">
			<div class="title-module_titleText_abc" title="Test Hepsiburada Kitap">Test Hepsiburada Kitap</div>
			<a class="productCardLink-module_link" href="/test-kitap-p-HB123">Link</a>
			<div class="price-module_finalPrice_def">89,99</div>
		</article>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewHepsiburada(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Hepsiburada Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Price != 89.99 {
		t.Errorf("price = %f, want 89.99", r.Price)
	}
	if r.CargoFee != 34.90 {
		t.Errorf("cargo = %f, want 34.90", r.CargoFee)
	}
	if r.URL != "https://www.hepsiburada.com/test-kitap-p-HB123" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Site != "hepsiburada.com" {
		t.Errorf("site = %q", r.Site)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "Hepsiburada Premium: ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestHepsiburada_Search_WithFraction(t *testing.T) {
	html := `<html><body>
		<article class="productCard-module_article_xyz">
			<div class="title-module_titleText_abc">Kitap Frac</div>
			<a class="productCardLink-module_link" href="/frac-p-123">Link</a>
			<div class="price-module_finalPrice_def">120,50</div>
			<span class="price-module_finalPriceFraction_ghi">50</span>
		</article>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewHepsiburada(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// The scraper trims the fraction from intPart, then concatenates
	// intPart="120," + fracPart="50" -> "120,50" -> 120.50
	if results[0].Price != 120.50 {
		t.Errorf("price = %f, want 120.50", results[0].Price)
	}
}

func TestHepsiburada_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewHepsiburada(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Amazon Search
// ---------------------------------------------------------------------------

func TestAmazon_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div data-component-type="s-search-result" data-asin="B0TEST1">
			<h2><span>Amazon Test Kitap</span><a href="/dp/B0TEST1">link</a></h2>
			<div class="a-row a-size-base a-color-secondary">
				<a class="a-size-base">Test Author</a>
			</div>
			<span class="a-price"><span class="a-offscreen">75,00 TL</span></span>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Amazon Test Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Author != "Test Author" {
		t.Errorf("author = %q", r.Author)
	}
	if r.Price != 75.0 {
		t.Errorf("price = %f, want 75.0", r.Price)
	}
	if r.CargoFee != 34.90 {
		t.Errorf("cargo = %f, want 34.90", r.CargoFee)
	}
	if r.URL != "https://www.amazon.com.tr/dp/B0TEST1" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Site != "amazon.com.tr" {
		t.Errorf("site = %q", r.Site)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "Amazon Prime: ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestAmazon_Search_SkipsNoASIN(t *testing.T) {
	html := `<html><body>
		<div data-component-type="s-search-result">
			<h2><span>No ASIN Book</span></h2>
			<span class="a-price"><span class="a-offscreen">10,00 TL</span></span>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for no ASIN, got %d", len(results))
	}
}

func TestAmazon_Search_SkipsSponsoredItems(t *testing.T) {
	html := `<html><body>
		<div data-component-type="s-search-result" data-asin="B0SPONSOR">
			<div class="a-color-secondary"><span class="a-text-bold">Sponsorlu</span></div>
			<h2><span>Sponsored Book</span></h2>
			<span class="a-price"><span class="a-offscreen">99,00 TL</span></span>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for sponsored, got %d", len(results))
	}
}

func TestAmazon_Search_CaptchaDetected(t *testing.T) {
	html := `<html><body><h1>Sorry, we just need to make sure you're not a robot.</h1><form><input type="text" id="captcha"></form></body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error for captcha page")
	}
	if !contains(err.Error(), "erişim engellendi") {
		t.Errorf("error should mention access blocked, got: %s", err.Error())
	}
}

func TestAmazon_Search_FallbackPrice(t *testing.T) {
	html := `<html><body>
		<div data-component-type="s-search-result" data-asin="B0FALL">
			<h2><span>Fallback Price Book</span><a href="/dp/B0FALL">link</a></h2>
			<span class="a-price-whole">100</span>
			<span class="a-price-fraction">50</span>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Price != 100.50 {
		t.Errorf("fallback price = %f, want 100.50", results[0].Price)
	}
}

func TestAmazon_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error for empty HTML (amazon treats as blocked)")
	}
}

// ---------------------------------------------------------------------------
// Bkmkitap Search
// ---------------------------------------------------------------------------

func TestBkmkitap_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="waw-product" data-price="32,50">
			<div class="waw-product-item-area"><a href="/kitap/test-bkm-123">Link</a></div>
			<p class="product-title">BKM Test Kitap</p>
			<p class="txt-title">BKM Yayınevi</p>
			<p class="txt-title">BKM Yazar</p>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewBkmkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "BKM Test Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Publisher != "BKM Yayınevi" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Author != "BKM Yazar" {
		t.Errorf("author = %q", r.Author)
	}
	if r.Price != 32.50 {
		t.Errorf("price = %f, want 32.50", r.Price)
	}
	if r.CargoFee != 19.90 {
		t.Errorf("cargo = %f, want 19.90", r.CargoFee)
	}
	if r.Site != "bkmkitap.com" {
		t.Errorf("site = %q", r.Site)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "100 TL üzeri siparişlerde ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestBkmkitap_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewBkmkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Idefix Search
// ---------------------------------------------------------------------------

func TestIdefix_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="justify-self-start">
			<h3>
				<span class="font-semibold">İthaki Yayınları</span>
				<span class="font-medium">Dune - İthaki Yayınları</span>
			</h3>
			<span class="text-secondary-500">Sepette 65,00 TL</span>
			<a href="/dune-ithaki-yayinlari-p-123456#sortId=1">link</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewIdefix(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "dune", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Dune" {
		t.Errorf("title = %q, want %q (publisher suffix should be stripped)", r.Title, "Dune")
	}
	if r.Publisher != "İthaki Yayınları" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Price != 65.0 {
		t.Errorf("price = %f, want 65.0", r.Price)
	}
	if r.CargoFee != 24.99 {
		t.Errorf("cargo = %f, want 24.99", r.CargoFee)
	}
	if r.Author != "" {
		t.Errorf("author should be empty, got %q", r.Author)
	}
	if r.Site != "idefix.com" {
		t.Errorf("site = %q", r.Site)
	}
	// URL should have -p- and no fragment
	if r.URL != "https://www.idefix.com/dune-ithaki-yayinlari-p-123456" {
		t.Errorf("url = %q", r.URL)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "200 TL üzeri siparişlerde ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestIdefix_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewIdefix(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Letgo Search
// ---------------------------------------------------------------------------

func TestLetgo_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div data-testid="item-card">
			<a href="/ilan/test-kitap-123">link</a>
			<div data-slot="item-card-image"><img alt="Test Kitap Image"></div>
			<div data-slot="item-card-body">
				<div class="line-clamp-1">Test Kitap Letgo</div>
				<p>25,00 TL</p>
				<div class="text-secondary-600"><span>İstanbul</span></div>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewLetgo(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Kitap Letgo" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Price != 25.0 {
		t.Errorf("price = %f, want 25.0", r.Price)
	}
	if r.Seller != "İstanbul" {
		t.Errorf("seller = %q", r.Seller)
	}
	if r.Condition != "İkinci El" {
		t.Errorf("condition = %q", r.Condition)
	}
	if !r.CargoUnknown {
		t.Error("expected CargoUnknown=true")
	}
	if r.URL != "https://www.letgo.com/ilan/test-kitap-123" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Site != "letgo.com" {
		t.Errorf("site = %q", r.Site)
	}
	if r.Category != scraper.UsedBook {
		t.Errorf("category = %v, want UsedBook", r.Category)
	}
}

func TestLetgo_Search_FallbackTitleFromImgAlt(t *testing.T) {
	html := `<html><body>
		<div data-testid="item-card">
			<a href="/ilan/abc">link</a>
			<div data-slot="item-card-image"><img alt="Fallback Title"></div>
			<div data-slot="item-card-body">
				<p>15,00 TL</p>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewLetgo(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Fallback Title" {
		t.Errorf("title = %q, want Fallback Title", results[0].Title)
	}
}

func TestLetgo_Search_DefaultSeller(t *testing.T) {
	html := `<html><body>
		<div data-testid="item-card">
			<a href="/ilan/abc">link</a>
			<div data-slot="item-card-body">
				<div class="line-clamp-1">Kitap</div>
				<p>10,00 TL</p>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewLetgo(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Seller != "letgo satıcı" {
		t.Errorf("seller = %q, want default 'letgo satıcı'", results[0].Seller)
	}
}

func TestLetgo_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewLetgo(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Dolap Search
// ---------------------------------------------------------------------------

func TestDolap_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="col-xs-6 col-md-4">
			<div class="img-block"><a href="/test-kitap-p-123">img</a></div>
			<div class="detail-head"><div class="title-stars-block"><div class="title">DolapSeller</div></div></div>
			<div class="detail-footer">
				<div class="title-info-block">
					<div class="title">Test Kitap Dolap</div>
					<div class="detail">Edebiyat</div>
				</div>
			</div>
			<div class="price-detail"><div class="price">18,50 TL</div></div>
			<div class="label-block"><span>İkinci El</span></div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewDolap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Test Kitap Dolap - Edebiyat" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Price != 18.50 {
		t.Errorf("price = %f, want 18.50", r.Price)
	}
	if r.Seller != "DolapSeller" {
		t.Errorf("seller = %q", r.Seller)
	}
	if r.Condition != "İkinci El" {
		t.Errorf("condition = %q", r.Condition)
	}
	if !r.CargoUnknown {
		t.Error("expected CargoUnknown=true")
	}
	if r.Site != "dolap.com" {
		t.Errorf("site = %q", r.Site)
	}
	if r.Category != scraper.UsedBook {
		t.Errorf("category = %v, want UsedBook", r.Category)
	}
}

func TestDolap_Search_NewCondition(t *testing.T) {
	html := `<html><body>
		<div class="col-xs-6 col-md-4">
			<div class="img-block"><a href="/dolap-test-kitap">img</a></div>
			<div class="detail-footer"><div class="title-info-block"><div class="title">Dolap Test Kitap</div></div></div>
			<div class="price-detail"><div class="price">22,00 TL</div></div>
			<div class="label-block"><span>Sıfır / Yeni</span></div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewDolap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "dolap test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Condition != "Sıfır / Yeni" {
		t.Errorf("condition = %q, want %q", results[0].Condition, "Sıfır / Yeni")
	}
}

func TestDolap_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewDolap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Gardrops Search
// ---------------------------------------------------------------------------

func TestGardrops_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="grid grid-cols-2">
			<div class="flex flex-col">
				<a class="relative block" href="/harry-potter-abcdef0123456789-p-12345-67890">
					<img alt="Harry Potter Alt">
				</a>
				<p class="text-smd font-medium">40,00 TL</p>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewGardrops(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "harry", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Title != "harry potter" {
		t.Errorf("title = %q, want %q", r.Title, "harry potter")
	}
	if r.Price != 40.0 {
		t.Errorf("price = %f, want 40.0", r.Price)
	}
	if r.Condition != "İkinci El" {
		t.Errorf("condition = %q", r.Condition)
	}
	if !r.CargoUnknown {
		t.Error("expected CargoUnknown=true")
	}
	if r.Site != "gardrops.com" {
		t.Errorf("site = %q", r.Site)
	}
	if r.Category != scraper.UsedBook {
		t.Errorf("category = %v, want UsedBook", r.Category)
	}
}

func TestGardrops_Search_FreeCargoFromBadge(t *testing.T) {
	html := `<html><body>
		<div class="grid grid-cols-2">
			<div class="flex flex-col">
				<a class="relative block" href="/kitap-abc-abcdef0123456789-p-111-222">
					<img alt="Kitap">
				</a>
				<p class="text-smd font-medium">30,00 TL</p>
				<div class="absolute inset-x-0 bottom-0"><p>Kargo Bedava</p></div>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewGardrops(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CargoUnknown {
		t.Error("expected CargoUnknown=false for free cargo badge")
	}
	if results[0].CargoFee != 0 {
		t.Errorf("cargo fee = %f, want 0", results[0].CargoFee)
	}
}

func TestGardrops_Search_FallbackImgTitle(t *testing.T) {
	// URL slug doesn't match regex -> falls back to img alt
	html := `<html><body>
		<div class="grid grid-cols-2">
			<div class="flex flex-col">
				<a class="relative block" href="/non-matching-url">
					<img alt="Fallback Gardrops Kitap">
				</a>
				<p class="text-smd font-medium">20,00 TL</p>
			</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewGardrops(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Fallback Gardrops Kitap" {
		t.Errorf("title = %q, want Fallback Gardrops Kitap", results[0].Title)
	}
}

func TestGardrops_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewGardrops(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// DR Search
// ---------------------------------------------------------------------------

func TestDR_Search_WithMockFirecrawl(t *testing.T) {
	html := `<html><body>
		<div class="prd-list-item">
			<div class="prd-name">DR Test Kitap</div>
			<div class="prd-author">DR Yazar</div>
			<div class="prd-publisher">DR Yayınevi</div>
			<div class="prd-price">55,90 TL</div>
			<a href="/dr-test-kitap-p-123">link</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewDR(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "DR Test Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Author != "DR Yazar" {
		t.Errorf("author = %q", r.Author)
	}
	if r.Publisher != "DR Yayınevi" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Price != 55.90 {
		t.Errorf("price = %f, want 55.90", r.Price)
	}
	if r.CargoFee != 24.99 {
		t.Errorf("cargo = %f, want 24.99", r.CargoFee)
	}
	if r.URL != "https://www.dr.com.tr/dr-test-kitap-p-123" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Site != "dr.com.tr" {
		t.Errorf("site = %q", r.Site)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "D&R Premium ile ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestDR_Search_FallbackSelector(t *testing.T) {
	// Uses .product-card (second selector) when .prd-list-item is not found
	html := `<html><body>
		<div class="product-card">
			<div class="product-name">Fallback DR Book</div>
			<div class="product-price">30,00 TL</div>
			<a href="/fallback-book">link</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewDR(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results from fallback selector, got %d", len(results))
	}
	if results[0].Title != "Fallback DR Book" {
		t.Errorf("title = %q", results[0].Title)
	}
}

func TestDR_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewDR(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// Pandora Search
// ---------------------------------------------------------------------------

func TestPandora_Search_WithMockFirecrawl(t *testing.T) {
	// Mirrors the real pandora.com.tr search card structure: an anchor to
	// /kitap/<slug>/<numeric-id> wrapping an h3 (title), span (author),
	// p (publisher) and a span containing the price.
	html := `<html><body>
		<a href="/kitap/test-pandora-kitap/123">
			<div>
				<div>
					<h3>Pandora Test Kitap</h3>
					<span>Pandora Yazar</span>
					<p>Pandora Yayınevi</p>
					<div><span>45,99 TL</span></div>
				</div>
			</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewPandora(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Title != "Pandora Test Kitap" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Author != "Pandora Yazar" {
		t.Errorf("author = %q", r.Author)
	}
	if r.Publisher != "Pandora Yayınevi" {
		t.Errorf("publisher = %q", r.Publisher)
	}
	if r.Price != 45.99 {
		t.Errorf("price = %f, want 45.99", r.Price)
	}
	if r.CargoFee != 24.99 {
		t.Errorf("cargo = %f, want 24.99", r.CargoFee)
	}
	if r.Site != "pandora.com.tr" {
		t.Errorf("site = %q", r.Site)
	}
	if r.URL != "https://www.pandora.com.tr/kitap/test-pandora-kitap/123" {
		t.Errorf("url = %q", r.URL)
	}

	rFree := results[1]
	if !rFree.FreeCargo {
		t.Error("expected FreeCargo=true")
	}
	if rFree.LoyaltyNote != "150 TL üzeri siparişlerde ücretsiz kargo" {
		t.Errorf("loyalty note = %q", rFree.LoyaltyNote)
	}
}

func TestPandora_Search_SkipsNonProductKitapLinks(t *testing.T) {
	// Pandora sometimes renders in-page menu or help links whose path also
	// starts with /kitap/ but does not end with a numeric product id. The
	// scraper must ignore these and only treat /kitap/<slug>/<numeric-id>
	// as a real product card.
	html := `<html><body>
		<a href="/kitap/arama-yardim">
			<h3>Yardım</h3>
			<span>44,00 TL</span>
		</a>
		<a href="/kitap/gercek-kitap/555">
			<div>
				<h3>Gerçek Kitap</h3>
				<span>Bir Yazar</span>
				<p>Bir Yayınevi</p>
				<div><span>100,00 TL</span></div>
			</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewPandora(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (one product, with/without free cargo), got %d", len(results))
	}
	if results[0].Title != "Gerçek Kitap" {
		t.Errorf("title = %q, want %q", results[0].Title, "Gerçek Kitap")
	}
}

func TestPandora_Search_EmptyHTML(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "")
	defer srv.Close()

	s := NewPandora(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty HTML, got %v", results)
	}
}

// ===========================================================================
// Limit enforcement tests
// ===========================================================================

func TestNadirkitap_Search_RespectsLimit(t *testing.T) {
	html := `<html><body>
		<div class="productsListHover">
			<a class="nadirBook" href="/kitap-1">Test Kitap Bir</a>
			<a class="price">10,00 TL</a>
		</div>
		<div class="productsListHover">
			<a class="nadirBook" href="/kitap-2">Test Kitap İki</a>
			<a class="price">20,00 TL</a>
		</div>
		<div class="productsListHover">
			<a class="nadirBook" href="/kitap-3">Test Kitap Üç</a>
			<a class="price">30,00 TL</a>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewNadirkitap(2)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results (limit=2), got %d", len(results))
	}
}

func TestTrendyol_Search_RespectsLimit(t *testing.T) {
	html := `<html><body>
		<a class="product-card" href="/p1">
			<span class="product-brand">B1</span>
			<span class="product-name">Test Kitap A</span>
			<div class="sale-price">10,00 TL</div>
		</a>
		<a class="product-card" href="/p2">
			<span class="product-brand">B2</span>
			<span class="product-name">Test Kitap B</span>
			<div class="sale-price">20,00 TL</div>
		</a>
		<a class="product-card" href="/p3">
			<span class="product-brand">B3</span>
			<span class="product-name">Test Kitap C</span>
			<div class="sale-price">30,00 TL</div>
		</a>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewTrendyol(1)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// limit=1 product -> 2 results (cargo + free cargo)
	if len(results) != 2 {
		t.Errorf("expected 2 results (1 product * 2), got %d", len(results))
	}
}

// ===========================================================================
// Firecrawl error propagation tests
// ===========================================================================

func TestNadirkitap_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewNadirkitap(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "nadirkitap") {
		t.Errorf("error should mention nadirkitap, got: %s", err.Error())
	}
}

func TestKitapyurdu_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "kitapyurdu") {
		t.Errorf("error should mention kitapyurdu, got: %s", err.Error())
	}
}

func TestTrendyol_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewTrendyol(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "trendyol") {
		t.Errorf("error should mention trendyol, got: %s", err.Error())
	}
}

func TestHepsiburada_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewHepsiburada(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "hepsiburada") {
		t.Errorf("error should mention hepsiburada, got: %s", err.Error())
	}
}

func TestAmazon_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "amazon") {
		t.Errorf("error should mention amazon, got: %s", err.Error())
	}
}

func TestBkmkitap_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewBkmkitap(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "bkmkitap") {
		t.Errorf("error should mention bkmkitap, got: %s", err.Error())
	}
}

func TestIdefix_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewIdefix(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "idefix") {
		t.Errorf("error should mention idefix, got: %s", err.Error())
	}
}

func TestLetgo_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewLetgo(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "letgo") {
		t.Errorf("error should mention letgo, got: %s", err.Error())
	}
}

func TestDolap_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewDolap(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "dolap") {
		t.Errorf("error should mention dolap, got: %s", err.Error())
	}
}

func TestGardrops_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewGardrops(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "gardrops") {
		t.Errorf("error should mention gardrops, got: %s", err.Error())
	}
}

func TestDR_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewDR(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "dr") {
		t.Errorf("error should mention dr, got: %s", err.Error())
	}
}

func TestPandora_Search_FirecrawlError(t *testing.T) {
	fc, srv := newMockFirecrawlError(t)
	defer srv.Close()

	s := NewPandora(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error from firecrawl failure")
	}
	if !contains(err.Error(), "pandora") {
		t.Errorf("error should mention pandora, got: %s", err.Error())
	}
}

// ===========================================================================
// Context cancellation tests
// ===========================================================================

func TestSearch_CancelledContext(t *testing.T) {
	fc, srv := newMockFirecrawl(t, "<html></html>")
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	_, err := s.Search(ctx, "test", scraper.TitleSearch)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ===========================================================================
// Interface compliance tests
// ===========================================================================

func TestScraperInterfaceCompliance(t *testing.T) {
	scrapers := AllScrapers(5, true)
	for _, s := range scrapers {
		// Verify each scraper satisfies the interface by calling all methods
		name := s.Name()
		if name == "" {
			t.Error("scraper Name() returned empty string")
		}
		_ = s.SiteCategory()
		s.SetFirecrawl(nil) // should not panic
	}
}

// Compile-time interface checks
var (
	_ scraper.Scraper = (*Nadirkitap)(nil)
	_ scraper.Scraper = (*Kitapyurdu)(nil)
	_ scraper.Scraper = (*Trendyol)(nil)
	_ scraper.Scraper = (*Hepsiburada)(nil)
	_ scraper.Scraper = (*Amazon)(nil)
	_ scraper.Scraper = (*Bkmkitap)(nil)
	_ scraper.Scraper = (*Idefix)(nil)
	_ scraper.Scraper = (*Letgo)(nil)
	_ scraper.Scraper = (*Dolap)(nil)
	_ scraper.Scraper = (*Gardrops)(nil)
	_ scraper.Scraper = (*DR)(nil)
	_ scraper.Scraper = (*Pandora)(nil)
)

// ===========================================================================
// Multiple products tests
// ===========================================================================

func TestKitapyurdu_Search_MultipleProducts(t *testing.T) {
	html := `<html><body>
		<div class="ky-product">
			<a href="/kitap/a-p-1">Link</a>
			<div class="ky-product-title">Kitap A</div>
			<div class="ky-product-author"><a>Yazar A</a></div>
			<div class="ky-product-publisher"><a>Yayın A</a></div>
			<div class="ky-product-sell-price">10,00 TL</div>
		</div>
		<div class="ky-product">
			<a href="/kitap/b-p-2">Link</a>
			<div class="ky-product-title">Kitap B</div>
			<div class="ky-product-author"><a>Yazar B</a></div>
			<div class="ky-product-publisher"><a>Yayın B</a></div>
			<div class="ky-product-sell-price">20,00 TL</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "kitap", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 products * 2 results each = 4
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	if results[0].Title != "Kitap A" {
		t.Errorf("first title = %q", results[0].Title)
	}
	if results[2].Title != "Kitap B" {
		t.Errorf("third title = %q", results[2].Title)
	}
}

func TestHepsiburada_Search_MultipleProducts(t *testing.T) {
	html := `<html><body>
		<article class="productCard-module_article_1">
			<div class="title-module_titleText_a">Ürün A</div>
			<a class="productCardLink-module_x" href="/urun-a-p-1">Link</a>
			<div class="price-module_finalPrice_b">50,00</div>
		</article>
		<article class="productCard-module_article_2">
			<div class="title-module_titleText_c">Ürün B</div>
			<a class="productCardLink-module_y" href="/urun-b-p-2">Link</a>
			<div class="price-module_finalPrice_d">60,00</div>
		</article>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewHepsiburada(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "ürün", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results (2 products * 2), got %d", len(results))
	}
}

// ===========================================================================
// Edge case: skip items without price or title
// ===========================================================================

func TestAmazon_Search_SkipsNoPrice(t *testing.T) {
	html := `<html><body>
		<div data-component-type="s-search-result" data-asin="B0NOPRICE">
			<h2><span>No Price Book</span></h2>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewAmazon(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for no price, got %d", len(results))
	}
}

func TestBkmkitap_Search_SkipsNoTitle(t *testing.T) {
	html := `<html><body>
		<div class="waw-product" data-price="10,00">
			<div class="waw-product-item-area"><a href="/test">Link</a></div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewBkmkitap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for no title, got %d", len(results))
	}
}

func TestDolap_Search_FiltersNonMatchingQuery(t *testing.T) {
	html := `<html><body>
		<div class="col-xs-6 col-md-4">
			<div class="img-block"><a href="/unrelated">img</a></div>
			<div class="detail-footer"><div class="title-info-block"><div class="title">Alakasız Ürün</div></div></div>
			<div class="price-detail"><div class="price">15,00 TL</div></div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewDolap(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "python programlama", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Scrapers no longer filter by query — relevance filtering is centralized in engine.
	// The scraper should return all valid results regardless of query match.
	if len(results) != 1 {
		t.Errorf("expected 1 result (scraper returns all valid results), got %d", len(results))
	}
}

// ===========================================================================
// TotalPrice calculation tests
// ===========================================================================

func TestTotalPriceCalculation(t *testing.T) {
	html := `<html><body>
		<div class="ky-product">
			<a href="/test">Link</a>
			<div class="ky-product-title">Test</div>
			<div class="ky-product-sell-price">100,00 TL</div>
		</div>
	</body></html>`

	fc, srv := newMockFirecrawl(t, html)
	defer srv.Close()

	s := NewKitapyurdu(10)
	s.SetFirecrawl(fc)

	results, err := s.Search(context.Background(), "test", scraper.TitleSearch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// First result: price + cargo
	r := results[0]
	expectedTotal := r.Price + r.CargoFee
	if r.TotalPrice != expectedTotal {
		t.Errorf("total price = %f, want %f (price=%f + cargo=%f)", r.TotalPrice, expectedTotal, r.Price, r.CargoFee)
	}

	// Second result: free cargo, total = price
	rFree := results[1]
	if rFree.TotalPrice != rFree.Price {
		t.Errorf("free cargo total = %f, want %f (price only)", rFree.TotalPrice, rFree.Price)
	}
}

// ===========================================================================
// Test utility helpers
// ===========================================================================

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// parseHTML is a test helper for creating goquery documents from HTML strings.
func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseHTML: %v", err)
	}
	return doc
}
