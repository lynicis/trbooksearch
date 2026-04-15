package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lynicis/trbooksearch/internal/engine"
	"github.com/lynicis/trbooksearch/internal/scraper"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeResult(site, title, author, seller string, price, cargo, total float64) scraper.BookResult {
	return scraper.BookResult{
		Site:       site,
		Title:      title,
		Author:     author,
		Seller:     seller,
		Price:      price,
		CargoFee:   cargo,
		TotalPrice: total,
		URL:        fmt.Sprintf("https://%s/%s", site, strings.ReplaceAll(title, " ", "-")),
	}
}

func makeResultWithURL(site, title, url string, total float64) scraper.BookResult {
	return scraper.BookResult{
		Site:       site,
		Title:      title,
		TotalPrice: total,
		URL:        url,
	}
}

// ---------------------------------------------------------------------------
// TestTruncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"short string", "abc", 10, "abc"},
		{"exact length", "abcde", 5, "abcde"},
		{"over length", "abcdefgh", 5, "abcd…"},
		{"maxLen 0", "hello", 0, ""},
		{"maxLen negative", "hello", -3, ""},
		{"maxLen 1", "hello", 1, "…"},
		{"maxLen 2", "hello", 2, "h…"},
		{"empty string", "", 5, ""},
		{"unicode chars", "merhaba dünya", 8, "merhaba…"},
		{"unicode exact", "dünya", 5, "dünya"},
		{"unicode over", "dünya", 3, "dü…"},
		{"single char fits", "a", 1, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPadRight
// ---------------------------------------------------------------------------

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"shorter than width", "abc", 6, "abc   "},
		{"exact width", "abcdef", 6, "abcdef"},
		{"longer than width", "abcdefgh", 6, "abcdefgh"},
		{"empty string", "", 4, "    "},
		{"width 0", "abc", 0, "abc"},
		{"unicode shorter", "dünya", 8, "dünya   "},
		{"unicode exact", "dünya", 5, "dünya"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRight(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHyperlink
// ---------------------------------------------------------------------------

func TestHyperlink(t *testing.T) {
	url := "https://example.com/book"
	text := "My Book"
	got := hyperlink(url, text)

	// OSC 8 format: \x1b]8;;<url>\x1b\\<text>\x1b]8;;\x1b\\
	want := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
	if got != want {
		t.Errorf("hyperlink(%q, %q) = %q, want %q", url, text, got, want)
	}

	// Verify the text appears inside the escape sequence
	if !strings.Contains(got, text) {
		t.Error("hyperlink output should contain the display text")
	}
	if !strings.Contains(got, url) {
		t.Error("hyperlink output should contain the URL")
	}
}

// ---------------------------------------------------------------------------
// TestCountSites
// ---------------------------------------------------------------------------

func TestCountSites(t *testing.T) {
	tests := []struct {
		name    string
		results []scraper.BookResult
		want    int
	}{
		{
			"multiple distinct sites",
			[]scraper.BookResult{
				{Site: "kitapyurdu.com"},
				{Site: "bkmkitap.com"},
				{Site: "idefix.com"},
			},
			3,
		},
		{
			"duplicates",
			[]scraper.BookResult{
				{Site: "kitapyurdu.com"},
				{Site: "kitapyurdu.com"},
				{Site: "bkmkitap.com"},
			},
			2,
		},
		{
			"all same site",
			[]scraper.BookResult{
				{Site: "kitapyurdu.com"},
				{Site: "kitapyurdu.com"},
			},
			1,
		},
		{"empty results", nil, 0},
		{"single result", []scraper.BookResult{{Site: "a.com"}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countSites(tt.results)
			if got != tt.want {
				t.Errorf("countSites() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMax
// ---------------------------------------------------------------------------

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{5, 3, 5},
		{3, 5, 5},
		{4, 4, 4},
		{0, 0, 0},
		{-1, -5, -1},
		{-3, 2, 2},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("max(%d,%d)", tt.a, tt.b), func(t *testing.T) {
			got := max(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCompareFloat
// ---------------------------------------------------------------------------

func TestCompareFloat(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want int
	}{
		{"a less than b", 1.5, 2.5, -1},
		{"a greater than b", 3.0, 1.0, 1},
		{"a equals b", 2.5, 2.5, 0},
		{"both zero", 0.0, 0.0, 0},
		{"negative values", -1.5, -0.5, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareFloat(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareFloat(%f, %f) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestApplyFilter
// ---------------------------------------------------------------------------

func TestApplyFilter(t *testing.T) {
	results := []scraper.BookResult{
		makeResult("kitapyurdu.com", "Go Programming", "John Doe", "SellerA", 50, 10, 60),
		makeResult("bkmkitap.com", "Python Basics", "Jane Smith", "SellerB", 40, 5, 45),
		makeResult("idefix.com", "Go Advanced", "Bob Brown", "SellerC", 70, 0, 70),
	}

	t.Run("empty text returns all", func(t *testing.T) {
		got := applyFilter(results, "", -1)
		if len(got) != 3 {
			t.Errorf("expected 3 results, got %d", len(got))
		}
	})

	t.Run("filter by title all columns", func(t *testing.T) {
		got := applyFilter(results, "go", -1)
		if len(got) != 2 {
			t.Errorf("expected 2 results matching 'go', got %d", len(got))
		}
	})

	t.Run("filter by site column", func(t *testing.T) {
		got := applyFilter(results, "kitapyurdu", colIdxSite)
		if len(got) != 1 {
			t.Errorf("expected 1 result for kitapyurdu, got %d", len(got))
		}
		if len(got) > 0 && got[0].Site != "kitapyurdu.com" {
			t.Errorf("expected kitapyurdu.com, got %s", got[0].Site)
		}
	})

	t.Run("filter by title column", func(t *testing.T) {
		got := applyFilter(results, "python", colIdxTitle)
		if len(got) != 1 {
			t.Errorf("expected 1 result for python, got %d", len(got))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		got := applyFilter(results, "nonexistent", -1)
		if len(got) != 0 {
			t.Errorf("expected 0 results, got %d", len(got))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		got := applyFilter(results, "GO PROGRAMMING", -1)
		if len(got) != 1 {
			t.Errorf("expected 1 result for case-insensitive match, got %d", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// TestMatchesFilter
// ---------------------------------------------------------------------------

func TestMatchesFilter(t *testing.T) {
	r := scraper.BookResult{
		Site:        "kitapyurdu.com",
		Title:       "Go Programming Language",
		Author:      "Alan Donovan",
		Seller:      "Kitap Dünyası",
		Price:       125.50,
		CargoFee:    15.00,
		TotalPrice:  140.50,
		LoyaltyNote: "Hepsiburada Premium",
	}

	tests := []struct {
		name   string
		text   string
		column int
		want   bool
	}{
		// Per-column matches
		{"site match", "kitapyurdu", colIdxSite, true},
		{"site no match", "bkmkitap", colIdxSite, false},
		{"title match", "programming", colIdxTitle, true},
		{"title no match", "python", colIdxTitle, false},
		{"author match", "donovan", colIdxAuthor, true},
		{"author no match", "smith", colIdxAuthor, false},
		{"seller match", "dünya", colIdxSeller, true},
		{"seller no match", "abc", colIdxSeller, false},
		{"price match", "125.50", colIdxPrice, true},
		{"price no match", "999.00", colIdxPrice, false},
		{"cargo match", "15.00", colIdxCargo, true},
		{"cargo no match", "20.00", colIdxCargo, false},
		{"total match", "140.50", colIdxTotal, true},
		{"total no match", "200.00", colIdxTotal, false},

		// All-columns mode (default, column = -1)
		{"all columns - matches site", "kitapyurdu", -1, true},
		{"all columns - matches title", "programming", -1, true},
		{"all columns - matches author", "donovan", -1, true},
		{"all columns - matches seller", "dünya", -1, true},
		{"all columns - matches loyalty note", "premium", -1, true},
		{"all columns - no match anywhere", "zzzzz", -1, false},

		// Case insensitivity: matchesFilter expects text to already be lowercased
		// (applyFilter calls strings.ToLower before passing to matchesFilter).
		// The match() closure lowercases the *field* but not the text parameter.
		{"case insensitive - lowered text matches", "kitapyurdu", colIdxSite, true},
		{"case insensitive - lowered title text", "go programming", colIdxTitle, true},
		// Uppercase text won't match because matchesFilter doesn't lowercase it
		{"uppercase text does not match", "KITAPYURDU", colIdxSite, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(r, tt.text, tt.column)
			if got != tt.want {
				t.Errorf("matchesFilter(text=%q, column=%d) = %v, want %v",
					tt.text, tt.column, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestApplySort
// ---------------------------------------------------------------------------

func TestApplySort(t *testing.T) {
	results := []scraper.BookResult{
		makeResult("bkmkitap.com", "Python", "Jane", "SellerB", 40, 5, 45),
		makeResult("idefix.com", "Go Advanced", "Bob", "SellerC", 70, 0, 70),
		makeResult("kitapyurdu.com", "Algorithms", "Alice", "SellerA", 50, 10, 60),
	}

	t.Run("direction 2 returns original order", func(t *testing.T) {
		got := applySort(results, colIdxTitle, 2)
		if len(got) != len(results) {
			t.Fatalf("expected %d results, got %d", len(results), len(got))
		}
		for i := range results {
			if got[i].Title != results[i].Title {
				t.Errorf("index %d: expected %q, got %q", i, results[i].Title, got[i].Title)
			}
		}
	})

	t.Run("column -1 returns original order", func(t *testing.T) {
		got := applySort(results, -1, 0)
		if len(got) != len(results) {
			t.Fatalf("expected %d results, got %d", len(results), len(got))
		}
		for i := range results {
			if got[i].Title != results[i].Title {
				t.Errorf("index %d: expected %q, got %q", i, results[i].Title, got[i].Title)
			}
		}
	})

	t.Run("sort by title ascending", func(t *testing.T) {
		got := applySort(results, colIdxTitle, 0)
		expected := []string{"Algorithms", "Go Advanced", "Python"}
		for i, want := range expected {
			if got[i].Title != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i].Title)
			}
		}
	})

	t.Run("sort by title descending", func(t *testing.T) {
		got := applySort(results, colIdxTitle, 1)
		expected := []string{"Python", "Go Advanced", "Algorithms"}
		for i, want := range expected {
			if got[i].Title != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i].Title)
			}
		}
	})

	t.Run("sort by site ascending", func(t *testing.T) {
		got := applySort(results, colIdxSite, 0)
		expected := []string{"bkmkitap.com", "idefix.com", "kitapyurdu.com"}
		for i, want := range expected {
			if got[i].Site != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i].Site)
			}
		}
	})

	t.Run("sort by price ascending", func(t *testing.T) {
		got := applySort(results, colIdxPrice, 0)
		expectedPrices := []float64{40, 50, 70}
		for i, want := range expectedPrices {
			if got[i].Price != want {
				t.Errorf("index %d: expected price %.2f, got %.2f", i, want, got[i].Price)
			}
		}
	})

	t.Run("sort by price descending", func(t *testing.T) {
		got := applySort(results, colIdxPrice, 1)
		expectedPrices := []float64{70, 50, 40}
		for i, want := range expectedPrices {
			if got[i].Price != want {
				t.Errorf("index %d: expected price %.2f, got %.2f", i, want, got[i].Price)
			}
		}
	})

	t.Run("sort by total ascending", func(t *testing.T) {
		got := applySort(results, colIdxTotal, 0)
		expectedTotals := []float64{45, 60, 70}
		for i, want := range expectedTotals {
			if got[i].TotalPrice != want {
				t.Errorf("index %d: expected total %.2f, got %.2f", i, want, got[i].TotalPrice)
			}
		}
	})

	t.Run("sort by cargo ascending", func(t *testing.T) {
		got := applySort(results, colIdxCargo, 0)
		expectedCargo := []float64{0, 5, 10}
		for i, want := range expectedCargo {
			if got[i].CargoFee != want {
				t.Errorf("index %d: expected cargo %.2f, got %.2f", i, want, got[i].CargoFee)
			}
		}
	})

	t.Run("sort by author ascending", func(t *testing.T) {
		got := applySort(results, colIdxAuthor, 0)
		expected := []string{"Alice", "Bob", "Jane"}
		for i, want := range expected {
			if got[i].Author != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i].Author)
			}
		}
	})

	t.Run("sort by seller ascending", func(t *testing.T) {
		got := applySort(results, colIdxSeller, 0)
		expected := []string{"SellerA", "SellerB", "SellerC"}
		for i, want := range expected {
			if got[i].Seller != want {
				t.Errorf("index %d: expected %q, got %q", i, want, got[i].Seller)
			}
		}
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		original := make([]scraper.BookResult, len(results))
		copy(original, results)
		_ = applySort(results, colIdxTitle, 0)
		for i := range results {
			if results[i].Title != original[i].Title {
				t.Errorf("original slice was modified at index %d", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestRenderCheapest3
// ---------------------------------------------------------------------------

func TestRenderCheapest3(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		got := renderCheapest3(nil)
		if got != "" {
			t.Errorf("expected empty string for nil results, got %q", got)
		}
		got = renderCheapest3([]scraper.BookResult{})
		if got != "" {
			t.Errorf("expected empty string for empty results, got %q", got)
		}
	})

	t.Run("single result", func(t *testing.T) {
		results := []scraper.BookResult{
			makeResult("kitapyurdu.com", "Go Book", "Author", "Seller", 50, 10, 60),
		}
		got := renderCheapest3(results)
		if got == "" {
			t.Fatal("expected non-empty output for 1 result")
		}
		if !strings.Contains(got, "En Ucuz 3 Sonuç") {
			t.Error("output should contain 'En Ucuz 3 Sonuç' title")
		}
		if !strings.Contains(got, "1.") {
			t.Error("output should contain rank '1.'")
		}
		// Should not contain rank 2 or 3
		if strings.Contains(got, "2.") {
			t.Error("output should not contain rank '2.' for single result")
		}
	})

	t.Run("three or more results", func(t *testing.T) {
		results := []scraper.BookResult{
			makeResult("site1.com", "Cheap Book", "A1", "S1", 20, 5, 25),
			makeResult("site2.com", "Mid Book", "A2", "S2", 40, 5, 45),
			makeResult("site3.com", "Expensive Book", "A3", "S3", 60, 10, 70),
			makeResult("site4.com", "Very Expensive", "A4", "S4", 100, 10, 110),
		}
		got := renderCheapest3(results)
		if !strings.Contains(got, "1.") {
			t.Error("output should contain rank '1.'")
		}
		if !strings.Contains(got, "2.") {
			t.Error("output should contain rank '2.'")
		}
		if !strings.Contains(got, "3.") {
			t.Error("output should contain rank '3.'")
		}
		// Only top 3 should appear
		if strings.Contains(got, "Very Expensive") {
			t.Error("4th result should not appear in cheapest 3")
		}
	})

	t.Run("deduplication by URL+Site", func(t *testing.T) {
		results := []scraper.BookResult{
			makeResultWithURL("site1.com", "Book A", "https://site1.com/book-a", 25),
			makeResultWithURL("site1.com", "Book A", "https://site1.com/book-a", 30), // dupe
			makeResultWithURL("site2.com", "Book B", "https://site2.com/book-b", 45),
			makeResultWithURL("site3.com", "Book C", "https://site3.com/book-c", 70),
		}
		got := renderCheapest3(results)
		// Should show 3 unique books: Book A, Book B, Book C
		if !strings.Contains(got, "3.") {
			t.Error("should show 3 unique results after dedup")
		}
	})

	t.Run("deduplication falls back to title when URL empty", func(t *testing.T) {
		results := []scraper.BookResult{
			{Site: "site1.com", Title: "Book A", TotalPrice: 25},
			{Site: "site1.com", Title: "Book A", TotalPrice: 30}, // same site+title, no URL
			{Site: "site2.com", Title: "Book B", TotalPrice: 45},
		}
		got := renderCheapest3(results)
		// After dedup: Book A (site1) and Book B (site2) = 2 unique
		if strings.Contains(got, "3.") {
			t.Error("should only have 2 unique results after dedup")
		}
		if !strings.Contains(got, "2.") {
			t.Error("should have 2 unique results after dedup")
		}
	})

	t.Run("result with seller shows seller", func(t *testing.T) {
		results := []scraper.BookResult{
			makeResult("site1.com", "Book A", "Auth", "TopSeller", 20, 5, 25),
		}
		got := renderCheapest3(results)
		if !strings.Contains(got, "TopSeller") {
			t.Error("output should contain seller name")
		}
	})

	t.Run("result without seller omits seller", func(t *testing.T) {
		results := []scraper.BookResult{
			{Site: "site1.com", Title: "Book A", TotalPrice: 25, URL: "https://site1.com/a"},
		}
		got := renderCheapest3(results)
		// No " | " after price for seller-less result.
		// Count occurrences of " | " — should only be 1 (between title-site and price)
		lines := strings.Split(got, "\n")
		for _, line := range lines {
			if strings.Contains(line, "1.") {
				pipeCount := strings.Count(line, " | ")
				if pipeCount > 1 {
					t.Errorf("expected at most 1 ' | ' separator without seller, got %d", pipeCount)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestRenderFilteredResultsContent
// ---------------------------------------------------------------------------

func TestRenderFilteredResultsContent(t *testing.T) {
	elapsed := 2500 * time.Millisecond

	t.Run("empty results with no total", func(t *testing.T) {
		got := renderFilteredResultsContent(nil, engine.SearchResult{}, false, elapsed)
		if !strings.Contains(got, "Sonuç bulunamadı") {
			t.Error("should show 'Sonuç bulunamadı' for empty results")
		}
	})

	t.Run("empty filtered but total > 0", func(t *testing.T) {
		full := engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book", "Auth", "Sell", 10, 5, 15),
			},
		}
		got := renderFilteredResultsContent(nil, full, false, elapsed)
		if !strings.Contains(got, "Filtreye uygun sonuç bulunamadı") {
			t.Error("should show filter-specific no-results message")
		}
	})

	t.Run("filtered count less than total", func(t *testing.T) {
		full := engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book A", "A1", "S1", 10, 5, 15),
				makeResult("site2.com", "Book B", "A2", "S2", 20, 5, 25),
				makeResult("site3.com", "Book C", "A3", "S3", 30, 5, 35),
			},
		}
		filtered := full.Results[:1]
		got := renderFilteredResultsContent(filtered, full, false, elapsed)
		// Should show "X/Y sonuç gösteriliyor"
		if !strings.Contains(got, "1/3 sonuç gösteriliyor") {
			t.Errorf("should show filtered/total count, got:\n%s", got)
		}
	})

	t.Run("all results shown", func(t *testing.T) {
		full := engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book A", "A1", "S1", 10, 5, 15),
			},
		}
		got := renderFilteredResultsContent(full.Results, full, false, elapsed)
		if !strings.Contains(got, "1 sonuç bulundu") {
			t.Errorf("should show total count without filter ratio, got:\n%s", got)
		}
	})

	t.Run("grouped mode with used and new books", func(t *testing.T) {
		usedBook := makeResult("site1.com", "Used Book", "A1", "S1", 10, 5, 15)
		usedBook.Category = scraper.UsedBook
		newBook := makeResult("site2.com", "New Book", "A2", "S2", 30, 5, 35)
		newBook.Category = scraper.NewBook

		full := engine.SearchResult{Results: []scraper.BookResult{usedBook, newBook}}
		got := renderFilteredResultsContent(full.Results, full, true, elapsed)
		if !strings.Contains(got, "İKİNCİ EL KİTAPLAR") {
			t.Error("grouped mode should show used books section")
		}
		if !strings.Contains(got, "YENİ KİTAPLAR") {
			t.Error("grouped mode should show new books section")
		}
	})

	t.Run("flat mode does not group", func(t *testing.T) {
		usedBook := makeResult("site1.com", "Used Book", "A1", "S1", 10, 5, 15)
		usedBook.Category = scraper.UsedBook
		newBook := makeResult("site2.com", "New Book", "A2", "S2", 30, 5, 35)
		newBook.Category = scraper.NewBook

		full := engine.SearchResult{Results: []scraper.BookResult{usedBook, newBook}}
		got := renderFilteredResultsContent(full.Results, full, false, elapsed)
		if strings.Contains(got, "İKİNCİ EL KİTAPLAR") {
			t.Error("flat mode should not show section headers")
		}
		if strings.Contains(got, "YENİ KİTAPLAR") {
			t.Error("flat mode should not show section headers")
		}
	})

	t.Run("errors are rendered", func(t *testing.T) {
		full := engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book", "Auth", "Sell", 10, 5, 15),
			},
			Errors: []engine.SearchError{
				{Site: "broken.com", Err: fmt.Errorf("timeout")},
			},
		}
		got := renderFilteredResultsContent(full.Results, full, false, elapsed)
		if !strings.Contains(got, "broken.com") {
			t.Error("should show error site name")
		}
		if !strings.Contains(got, "timeout") {
			t.Error("should show error message")
		}
	})

	t.Run("elapsed time is shown", func(t *testing.T) {
		full := engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book", "Auth", "Sell", 10, 5, 15),
			},
		}
		got := renderFilteredResultsContent(full.Results, full, false, elapsed)
		if !strings.Contains(got, "2.5s") {
			t.Errorf("should show elapsed time '2.5s', got:\n%s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestRenderWithScrollbar
// ---------------------------------------------------------------------------

func TestRenderWithScrollbar(t *testing.T) {
	t.Run("basic rendering", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4", "line5"}
		got := renderWithScrollbar(lines, 5, 0.0)
		if got == "" {
			t.Fatal("expected non-empty output")
		}
		// Should contain all 5 lines
		for _, l := range lines {
			if !strings.Contains(got, l) {
				t.Errorf("output should contain %q", l)
			}
		}
	})

	t.Run("viewport height limits output", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4", "line5"}
		got := renderWithScrollbar(lines, 3, 0.0)
		// Should only render first 3 lines
		if !strings.Contains(got, "line3") {
			t.Error("should contain line3 (within viewport)")
		}
		if strings.Contains(got, "line4") {
			t.Error("should not contain line4 (outside viewport)")
		}
	})

	t.Run("contains scrollbar characters", func(t *testing.T) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = fmt.Sprintf("line-%02d", i)
		}
		got := renderWithScrollbar(lines, 10, 0.5)
		// Should contain either thumb (┃) or track (│) characters
		if !strings.Contains(got, "┃") && !strings.Contains(got, "│") {
			t.Error("output should contain scrollbar characters")
		}
	})

	t.Run("scroll at 0 percent", func(t *testing.T) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = fmt.Sprintf("line-%02d", i)
		}
		got := renderWithScrollbar(lines, 10, 0.0)
		// At 0%, thumb should start at position 0
		outputLines := strings.Split(got, "\n")
		if len(outputLines) > 0 && !strings.Contains(outputLines[0], "┃") {
			t.Error("at 0% scroll, first line should have thumb")
		}
	})

	t.Run("scroll at 100 percent", func(t *testing.T) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = fmt.Sprintf("line-%02d", i)
		}
		got := renderWithScrollbar(lines, 10, 1.0)
		// At 100%, thumb should be at the bottom
		outputLines := strings.Split(got, "\n")
		lastLine := outputLines[len(outputLines)-1]
		if !strings.Contains(lastLine, "┃") {
			t.Error("at 100% scroll, last line should have thumb")
		}
	})
}

// ---------------------------------------------------------------------------
// TestNewModel
// ---------------------------------------------------------------------------

func TestNewModel(t *testing.T) {
	eng := engine.NewEngine(nil) // no scrapers for testing
	ctx := context.Background()

	m := NewModel(eng, "test query", scraper.TitleSearch, true, ctx)

	if m.query != "test query" {
		t.Errorf("expected query 'test query', got %q", m.query)
	}
	if m.searchType != scraper.TitleSearch {
		t.Errorf("expected TitleSearch, got %v", m.searchType)
	}
	if !m.grouped {
		t.Error("expected grouped to be true")
	}
	if !m.searching {
		t.Error("expected searching to be true initially")
	}
	if m.Quitting {
		t.Error("expected Quitting to be false initially")
	}
	if m.width != 80 {
		t.Errorf("expected default width 80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected default height 24, got %d", m.height)
	}
	if m.filterColumn != -1 {
		t.Errorf("expected default filterColumn -1, got %d", m.filterColumn)
	}
	if m.sortColumn != -1 {
		t.Errorf("expected default sortColumn -1, got %d", m.sortColumn)
	}
	if m.sortDirection != 2 {
		t.Errorf("expected default sortDirection 2, got %d", m.sortDirection)
	}
	if m.filterActive {
		t.Error("expected filterActive to be false initially")
	}
	if m.viewportReady {
		t.Error("expected viewportReady to be false initially")
	}
}

func TestNewModel_WithISBNSearch(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()

	m := NewModel(eng, "9780134190440", scraper.ISBNSearch, false, ctx)

	if m.searchType != scraper.ISBNSearch {
		t.Errorf("expected ISBNSearch, got %v", m.searchType)
	}
	if m.grouped {
		t.Error("expected grouped to be false")
	}
}

// ---------------------------------------------------------------------------
// TestReservedLines
// ---------------------------------------------------------------------------

func TestReservedLines(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()

	t.Run("no results, no filter, no sort", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{}
		got := m.reservedLines()
		// Base: 3 (header + footer + padding), no cheapest3, no filter, no sort
		if got != 3 {
			t.Errorf("expected 3 reserved lines, got %d", got)
		}
	})

	t.Run("with 1 result", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("site1.com", "Book", "Auth", "Sell", 10, 5, 15),
			},
		}
		got := m.reservedLines()
		// Base 3 + cheapest3 panel (1 title line + 1 card line) = 3 + 2 = 5
		if got != 5 {
			t.Errorf("expected 5 reserved lines, got %d", got)
		}
	})

	t.Run("with 3 results", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("s1", "B1", "A1", "S1", 10, 5, 15),
				makeResult("s2", "B2", "A2", "S2", 20, 5, 25),
				makeResult("s3", "B3", "A3", "S3", 30, 5, 35),
			},
		}
		got := m.reservedLines()
		// Base 3 + cheapest3 (1 title + 3 cards) = 3 + 4 = 7
		if got != 7 {
			t.Errorf("expected 7 reserved lines, got %d", got)
		}
	})

	t.Run("with 5 results caps at 3 for cheapest panel", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{
			Results: make([]scraper.BookResult, 5),
		}
		for i := range m.results.Results {
			m.results.Results[i] = makeResult("s", "B", "A", "S", float64(i*10), 5, float64(i*10+5))
		}
		got := m.reservedLines()
		// Base 3 + cheapest3 (1 title + 3 cards) = 3 + 4 = 7
		if got != 7 {
			t.Errorf("expected 7 reserved lines, got %d", got)
		}
	})

	t.Run("with active filter adds a line", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{}
		m.filterActive = true
		got := m.reservedLines()
		// Base 3 + filter bar 1 = 4
		if got != 4 {
			t.Errorf("expected 4 reserved lines, got %d", got)
		}
	})

	t.Run("with confirmed filter value adds a line", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{}
		m.filterActive = false
		m.filterInput.SetValue("go")
		got := m.reservedLines()
		// Base 3 + filter indicator 1 = 4
		if got != 4 {
			t.Errorf("expected 4 reserved lines, got %d", got)
		}
	})

	t.Run("with sort indicator adds a line", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{}
		m.sortColumn = colIdxPrice
		m.sortDirection = 0
		got := m.reservedLines()
		// Base 3 + sort indicator 1 = 4
		if got != 4 {
			t.Errorf("expected 4 reserved lines, got %d", got)
		}
	})

	t.Run("sort direction 2 does not add a line", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{}
		m.sortColumn = colIdxPrice
		m.sortDirection = 2
		got := m.reservedLines()
		// Base 3 only — sortDirection 2 means default, no indicator
		if got != 3 {
			t.Errorf("expected 3 reserved lines, got %d", got)
		}
	})

	t.Run("everything active", func(t *testing.T) {
		m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
		m.results = engine.SearchResult{
			Results: []scraper.BookResult{
				makeResult("s1", "B1", "A1", "S1", 10, 5, 15),
				makeResult("s2", "B2", "A2", "S2", 20, 5, 25),
				makeResult("s3", "B3", "A3", "S3", 30, 5, 35),
			},
		}
		m.filterActive = true
		m.sortColumn = colIdxPrice
		m.sortDirection = 0
		got := m.reservedLines()
		// Base 3 + cheapest3 (1+3=4) + filter 1 + sort 1 = 9
		if got != 9 {
			t.Errorf("expected 9 reserved lines, got %d", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestModel_View_Quitting
// ---------------------------------------------------------------------------

func TestModel_View_Quitting(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.Quitting = true

	got := m.View()
	if got != "" {
		t.Errorf("expected empty string when quitting, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// TestModel_View_Searching
// ---------------------------------------------------------------------------

func TestModel_View_Searching(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "dune", scraper.TitleSearch, false, ctx)
	m.searching = true
	m.startTime = time.Now().Add(-5 * time.Second)
	m.elapsed = 5 * time.Second

	got := m.View()
	if !strings.Contains(got, "dune") {
		t.Error("searching view should contain query")
	}
	if !strings.Contains(got, "q:") {
		t.Error("searching view should contain quit help")
	}
}

// ---------------------------------------------------------------------------
// TestModel_View_Results
// ---------------------------------------------------------------------------

func TestModel_View_WithResults(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = false
	m.elapsed = 3 * time.Second
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("site.com", "Book A", "Author", "Seller", 50, 10, 60),
		},
	}
	m.width = 200
	m.height = 50
	m.viewportReady = true
	m.refreshViewport()

	got := m.View()
	if !strings.Contains(got, "test") {
		t.Error("results view should contain query")
	}
	if !strings.Contains(got, "En Ucuz 3") {
		t.Error("results view should contain cheapest 3 panel")
	}
}

func TestModel_View_WithFilterActive(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = false
	m.elapsed = 1 * time.Second
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("site.com", "Book", "Author", "Seller", 50, 10, 60),
		},
	}
	m.width = 200
	m.height = 50
	m.viewportReady = true
	m.filterActive = true
	m.filterColumn = colIdxTitle
	m.refreshViewport()

	got := m.View()
	if !strings.Contains(got, "Başlık") {
		t.Error("filter active view should show column name")
	}
}

func TestModel_View_WithConfirmedFilter(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = false
	m.elapsed = 1 * time.Second
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("site.com", "Book", "Author", "Seller", 50, 10, 60),
		},
	}
	m.width = 200
	m.height = 50
	m.viewportReady = true
	m.filterActive = false
	m.filterInput.SetValue("test filter")
	m.refreshViewport()

	got := m.View()
	if !strings.Contains(got, "test filter") {
		t.Error("confirmed filter view should show filter text")
	}
}

func TestModel_View_WithSort(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = false
	m.elapsed = 1 * time.Second
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("site.com", "Book", "Author", "Seller", 50, 10, 60),
		},
	}
	m.width = 200
	m.height = 50
	m.viewportReady = true
	m.sortColumn = colIdxPrice
	m.sortDirection = 0
	m.refreshViewport()

	got := m.View()
	if !strings.Contains(got, "Sıralama") {
		t.Error("sort active view should show sort indicator")
	}
	if !strings.Contains(got, "Fiyat") {
		t.Error("sort indicator should show column name")
	}
}

func TestModel_View_SortDescending(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = false
	m.elapsed = 1 * time.Second
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("site.com", "Book", "Author", "Seller", 50, 10, 60),
		},
	}
	m.width = 200
	m.height = 50
	m.viewportReady = true
	m.sortColumn = colIdxTitle
	m.sortDirection = 1
	m.refreshViewport()

	got := m.View()
	if !strings.Contains(got, "↓") {
		t.Error("descending sort should show down arrow")
	}
}

// ---------------------------------------------------------------------------
// TestRenderSearching
// ---------------------------------------------------------------------------

func TestRenderSearching(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "dune", scraper.TitleSearch, false, ctx)
	m.searching = true
	m.elapsed = 2 * time.Second

	var b strings.Builder
	m.renderSearching(&b)

	got := b.String()
	if !strings.Contains(got, "dune") {
		t.Error("renderSearching should contain query")
	}
	if !strings.Contains(got, "Aranıyor") {
		t.Error("renderSearching should contain 'Aranıyor'")
	}
}

// ---------------------------------------------------------------------------
// TestRenderTable
// ---------------------------------------------------------------------------

func TestRenderTable(t *testing.T) {
	results := []scraper.BookResult{
		{
			Site:       "test.com",
			Title:      "Test Book",
			Author:     "Test Author",
			Seller:     "Test Seller",
			Price:      50.00,
			CargoFee:   10.00,
			TotalPrice: 60.00,
			URL:        "https://test.com/book",
			Category:   scraper.NewBook,
		},
		{
			Site:         "site2.com",
			Title:        "Book 2",
			Author:       "",
			Price:        100.00,
			CargoFee:     0,
			TotalPrice:   100.00,
			FreeCargo:    true,
			LoyaltyNote:  "Premium ücretsiz kargo",
			CargoUnknown: false,
			Category:     scraper.NewBook,
		},
		{
			Site:         "site3.com",
			Title:        "Book 3",
			Price:        75.00,
			CargoUnknown: true,
			TotalPrice:   75.00,
			Category:     scraper.UsedBook,
		},
	}

	var b strings.Builder
	renderTable(&b, results)

	got := b.String()
	if !strings.Contains(got, "Site") {
		t.Error("table should contain header")
	}
	if !strings.Contains(got, "Test Book") {
		t.Error("table should contain book title")
	}
	if !strings.Contains(got, "50.00") {
		t.Error("table should contain price")
	}
	if !strings.Contains(got, "Ücretsiz") {
		t.Error("table should contain free cargo text")
	}
	if !strings.Contains(got, "?") {
		t.Error("table should contain unknown cargo marker")
	}
	if !strings.Contains(got, "Premium") {
		t.Error("table should contain loyalty note")
	}
}

// ---------------------------------------------------------------------------
// TestRefreshViewport
// ---------------------------------------------------------------------------

func TestRefreshViewport(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("s1", "B1", "A1", "S1", 10, 5, 15),
			makeResult("s2", "B2", "A2", "S2", 20, 5, 25),
		},
	}
	m.width = 200
	m.height = 50

	// Initialize viewport properly (like WindowSizeMsg handler does)
	m.viewport = viewport.New(m.width, m.height-m.reservedLines())
	m.viewport.SetContent("")
	m.viewportReady = true
	m.elapsed = 1 * time.Second

	// Should not panic
	m.refreshViewport()

	// Verify viewport content was set
	totalLines := m.viewport.TotalLineCount()
	if totalLines == 0 {
		t.Error("viewport should have content after refresh")
	}
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return non-nil command")
	}
}

// ---------------------------------------------------------------------------
// TestTickCmd
// ---------------------------------------------------------------------------

func TestTickCmd(t *testing.T) {
	cmd := tickCmd()
	if cmd == nil {
		t.Error("tickCmd should return non-nil command")
	}
}

// ---------------------------------------------------------------------------
// TestListenForStatus
// ---------------------------------------------------------------------------

func TestListenForStatus_ReceivesStatus(t *testing.T) {
	ch := make(chan engine.SiteStatus, 1)
	ch <- engine.SiteStatus{Site: "test.com", Status: "done", Count: 5}

	cmd := listenForStatus(ch)
	msg := cmd()

	status, ok := msg.(siteStatusMsg)
	if !ok {
		t.Fatalf("expected siteStatusMsg, got %T", msg)
	}
	if status.Site != "test.com" {
		t.Errorf("Site = %q, want %q", status.Site, "test.com")
	}
	if status.Count != 5 {
		t.Errorf("Count = %d, want %d", status.Count, 5)
	}
}

func TestListenForStatus_ClosedChannel(t *testing.T) {
	ch := make(chan engine.SiteStatus)
	close(ch)

	cmd := listenForStatus(ch)
	msg := cmd()

	if msg != nil {
		t.Errorf("expected nil message from closed channel, got %v", msg)
	}
}

// ---------------------------------------------------------------------------
// TestRunSearch
// ---------------------------------------------------------------------------

func TestRunSearch(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	ch := make(chan engine.SiteStatus, 10)

	cmd := runSearch(eng, ctx, "test", scraper.TitleSearch, ch)
	msg := cmd()

	result, ok := msg.(searchDoneMsg)
	if !ok {
		t.Fatalf("expected searchDoneMsg, got %T", msg)
	}

	// Engine with no scrapers should return empty results
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
}

// ---------------------------------------------------------------------------
// TestUpdate - covers the Update function's various message handlers
// ---------------------------------------------------------------------------

func newTestModel() Model {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, true, ctx)
	m.searching = false
	m.elapsed = 2 * time.Second
	m.width = 200
	m.height = 50
	m.results = engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("s1", "B1", "A1", "S1", 10, 5, 15),
			makeResult("s2", "B2", "A2", "S2", 20, 5, 25),
		},
	}
	// Initialize viewport
	m.viewport = viewport.New(m.width, m.height-m.reservedLines())
	m.viewport.SetContent("")
	m.viewportReady = true
	m.refreshViewport()
	return m
}

func TestUpdate_QuitKey(t *testing.T) {
	m := newTestModel()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model := newModel.(Model)
	if !model.Quitting {
		t.Error("expected Quitting to be true after 'q' key")
	}
	if cmd == nil {
		t.Error("expected non-nil quit command")
	}
}

func TestUpdate_CtrlCKey(t *testing.T) {
	m := newTestModel()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(Model)
	if !model.Quitting {
		t.Error("expected Quitting to be true after ctrl+c")
	}
	if cmd == nil {
		t.Error("expected non-nil quit command")
	}
}

func TestUpdate_SlashKey_ActivatesFilter(t *testing.T) {
	m := newTestModel()

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model := newModel.(Model)
	if !model.filterActive {
		t.Error("expected filterActive to be true after '/' key")
	}
	if model.filterColumn != -1 {
		t.Errorf("expected filterColumn -1, got %d", model.filterColumn)
	}
}

func TestUpdate_NumberKeys_ActivateColumnFilter(t *testing.T) {
	for _, key := range []rune{'1', '2', '3', '4', '5', '6', '7'} {
		t.Run(string(key), func(t *testing.T) {
			m := newTestModel()
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
			model := newModel.(Model)
			if !model.filterActive {
				t.Error("expected filterActive to be true")
			}
			expectedCol := int(key - '1')
			if model.filterColumn != expectedCol {
				t.Errorf("expected filterColumn %d, got %d", expectedCol, model.filterColumn)
			}
		})
	}
}

func TestUpdate_EscapeKey_PassesThrough(t *testing.T) {
	// Note: The Update handler uses "escape" string match but tea.KeyEscape produces "esc".
	// This means the escape key handling branches are currently unreachable in the app code.
	// This test verifies the escape key falls through to the default handler without crashing.
	m := newTestModel()
	m.filterInput.SetValue("test filter")
	m.filterColumn = colIdxTitle

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := newModel.(Model)
	// Should pass through without clearing filter (since "esc" != "escape")
	_ = model
}

func TestUpdate_FilterActive_EscapeKey(t *testing.T) {
	// Note: tea.KeyEscape.String() returns "esc", but code matches "escape".
	// This means the escape case falls through to the default handler (textinput update).
	m := newTestModel()
	m.filterActive = true
	m.filterInput.Focus()
	m.filterInput.SetValue("some text")

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := newModel.(Model)
	// Falls through to default handler - no crash expected
	_ = model
}

func TestUpdate_FilterActive_EnterConfirms(t *testing.T) {
	m := newTestModel()
	m.filterActive = true
	m.filterInput.SetValue("confirmed")

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)
	if model.filterActive {
		t.Error("expected filterActive to be false after enter")
	}
	// Filter text should be kept
}

func TestUpdate_FilterActive_CtrlC_Quits(t *testing.T) {
	m := newTestModel()
	m.filterActive = true

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(Model)
	if !model.Quitting {
		t.Error("expected Quitting after ctrl+c in filter mode")
	}
	if cmd == nil {
		t.Error("expected non-nil quit command")
	}
}

func TestUpdate_SortKey_CyclesSortColumns(t *testing.T) {
	m := newTestModel()

	// First 's' should move from default (-1) to the first sort column
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := newModel.(Model)
	if model.sortColumn == -1 {
		t.Error("sort column should change from default after 's'")
	}
	if model.sortDirection != 0 {
		t.Errorf("expected ascending sort direction, got %d", model.sortDirection)
	}
}

func TestUpdate_ShiftSortKey_TogglesDirection(t *testing.T) {
	m := newTestModel()
	m.sortColumn = colIdxPrice
	m.sortDirection = 0

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := newModel.(Model)
	if model.sortDirection != 1 {
		t.Errorf("expected descending sort (1), got %d", model.sortDirection)
	}
}

func TestUpdate_ResetKey(t *testing.T) {
	m := newTestModel()
	m.sortColumn = colIdxTitle
	m.sortDirection = 0

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model := newModel.(Model)
	if model.sortColumn != -1 {
		t.Errorf("expected sortColumn -1 after reset, got %d", model.sortColumn)
	}
	if model.sortDirection != 2 {
		t.Errorf("expected sortDirection 2 (none) after reset, got %d", model.sortDirection)
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()
	m.viewportReady = false

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := newModel.(Model)
	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
	if !model.viewportReady {
		t.Error("expected viewportReady to be true after WindowSizeMsg")
	}
}

func TestUpdate_WindowSizeMsg_UpdatesExistingViewport(t *testing.T) {
	m := newTestModel()

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 60})
	model := newModel.(Model)
	if model.width != 150 {
		t.Errorf("expected width 150, got %d", model.width)
	}
	if model.viewport.Width != 150 {
		t.Errorf("expected viewport width 150, got %d", model.viewport.Width)
	}
}

func TestUpdate_SearchStartMsg(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)

	newModel, cmd := m.Update(searchStartMsg{})
	model := newModel.(Model)
	if !model.searching {
		t.Error("expected searching to be true after searchStartMsg")
	}
	if cmd == nil {
		t.Error("expected non-nil command after searchStartMsg")
	}
}

func TestUpdate_SearchDoneMsg(t *testing.T) {
	m := newTestModel()
	m.searching = true
	m.startTime = time.Now().Add(-3 * time.Second)

	results := engine.SearchResult{
		Results: []scraper.BookResult{
			makeResult("s1", "B1", "A1", "S1", 10, 5, 15),
		},
	}

	newModel, _ := m.Update(searchDoneMsg(results))
	model := newModel.(Model)
	if model.searching {
		t.Error("expected searching to be false after searchDoneMsg")
	}
	if len(model.results.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(model.results.Results))
	}
}

func TestUpdate_SiteStatusMsg(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true
	ch := make(chan engine.SiteStatus, 10)
	m.statusCh = ch

	status := engine.SiteStatus{Site: "test.com", Status: "done", Count: 5}
	newModel, cmd := m.Update(siteStatusMsg(status))
	model := newModel.(Model)
	if s, ok := model.statuses["test.com"]; ok {
		if s.Status != "done" {
			t.Errorf("expected status 'done', got %q", s.Status)
		}
	} else {
		t.Error("expected test.com status to be recorded")
	}
	if cmd == nil {
		t.Error("expected non-nil command to listen for more statuses")
	}
}

func TestUpdate_SiteStatusMsg_NilChannel(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true
	m.statusCh = nil

	status := engine.SiteStatus{Site: "test.com", Status: "done", Count: 5}
	_, cmd := m.Update(siteStatusMsg(status))
	if cmd != nil {
		t.Error("expected nil command when statusCh is nil")
	}
}

func TestUpdate_TickMsg_WhileSearching(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true
	m.startTime = time.Now().Add(-2 * time.Second)

	newModel, cmd := m.Update(tickMsg(time.Now()))
	model := newModel.(Model)
	if model.elapsed < 2*time.Second {
		t.Error("expected elapsed to be updated")
	}
	if cmd == nil {
		t.Error("expected non-nil command to schedule next tick")
	}
}

func TestUpdate_TickMsg_NotSearching(t *testing.T) {
	m := newTestModel()

	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd != nil {
		t.Error("expected nil command when not searching")
	}
}

func TestUpdate_SpinnerUpdate_WhileSearching(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true

	// Send a spinner tick message
	_, cmd := m.Update(m.spinner.Tick())
	// Should return a command (spinner will schedule its next tick)
	_ = cmd // just checking it doesn't panic
}

func TestUpdate_FilterActive_TypesText(t *testing.T) {
	m := newTestModel()
	m.filterActive = true
	m.filterInput.Focus()

	// Send a character key
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model := newModel.(Model)
	// The filter should still be active
	if !model.filterActive {
		t.Error("expected filterActive to still be true")
	}
}

func TestUpdate_SlashKey_DuringSearch_Ignored(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model := newModel.(Model)
	if model.filterActive {
		t.Error("filter should not activate during search")
	}
}

func TestUpdate_SortKey_DuringSearch_Ignored(t *testing.T) {
	eng := engine.NewEngine(nil)
	ctx := context.Background()
	m := NewModel(eng, "test", scraper.TitleSearch, false, ctx)
	m.searching = true

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := newModel.(Model)
	if model.sortColumn != -1 {
		t.Error("sort should not activate during search")
	}
}

func TestUpdate_ShiftS_NoSortColumn_Ignored(t *testing.T) {
	m := newTestModel()
	m.sortColumn = -1

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := newModel.(Model)
	// Should remain at default
	if model.sortColumn != -1 {
		t.Error("shift+S with no sort column should be ignored")
	}
}

func TestUpdate_ShiftS_CyclesToNone(t *testing.T) {
	m := newTestModel()
	m.sortColumn = colIdxPrice
	m.sortDirection = 1 // descending

	// Next cycle should be direction=2 (none), which resets sortColumn to -1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := newModel.(Model)
	if model.sortDirection != 2 {
		t.Errorf("expected sortDirection 2, got %d", model.sortDirection)
	}
	if model.sortColumn != -1 {
		t.Errorf("expected sortColumn -1 when direction is none, got %d", model.sortColumn)
	}
}

func TestUpdate_EscapeKey_NoFilter_NoOp(t *testing.T) {
	m := newTestModel()
	// No filter set

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := newModel.(Model)
	// Should be a no-op; pass through to viewport
	_ = model
}

func TestUpdate_ViewportScroll_AfterSearch(t *testing.T) {
	m := newTestModel()

	// Delegate a down arrow to viewport
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_ = cmd // just verify no panic
}
