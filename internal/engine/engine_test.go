package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

// ---------------------------------------------------------------------------
// Mock scraper
// ---------------------------------------------------------------------------

type mockScraper struct {
	name     string
	category scraper.Category
	results  []scraper.BookResult
	err      error
	delay    time.Duration
	fc       *scraper.FirecrawlClient
}

func (m *mockScraper) Name() string                                 { return m.name }
func (m *mockScraper) SiteCategory() scraper.Category               { return m.category }
func (m *mockScraper) SetFirecrawl(client *scraper.FirecrawlClient) { m.fc = client }
func (m *mockScraper) Search(_ context.Context, _ string, _ scraper.SearchType) ([]scraper.BookResult, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.results, m.err
}

// ---------------------------------------------------------------------------
// 1. TestNewEngine
// ---------------------------------------------------------------------------

func TestNewEngine(t *testing.T) {
	s1 := &mockScraper{name: "site-a"}
	s2 := &mockScraper{name: "site-b"}

	e := NewEngine(nil, s1, s2)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if len(e.scrapers) != 2 {
		t.Fatalf("expected 2 scrapers, got %d", len(e.scrapers))
	}
	if e.scrapers[0].Name() != "site-a" {
		t.Errorf("expected first scraper name %q, got %q", "site-a", e.scrapers[0].Name())
	}
	if e.scrapers[1].Name() != "site-b" {
		t.Errorf("expected second scraper name %q, got %q", "site-b", e.scrapers[1].Name())
	}
	if e.firecrawl != nil {
		t.Error("expected nil firecrawl")
	}
}

func TestNewEngine_WithFirecrawl(t *testing.T) {
	fc := scraper.NewFirecrawlClient("key", "http://localhost")
	e := NewEngine(fc)
	if e.firecrawl == nil {
		t.Fatal("expected non-nil firecrawl")
	}
	if len(e.scrapers) != 0 {
		t.Fatalf("expected 0 scrapers, got %d", len(e.scrapers))
	}
}

// ---------------------------------------------------------------------------
// 2. TestEngine_Scrapers
// ---------------------------------------------------------------------------

func TestEngine_Scrapers(t *testing.T) {
	s1 := &mockScraper{name: "alpha"}
	s2 := &mockScraper{name: "beta"}
	s3 := &mockScraper{name: "gamma"}

	e := NewEngine(nil, s1, s2, s3)
	got := e.Scrapers()

	if len(got) != 3 {
		t.Fatalf("expected 3 scrapers, got %d", len(got))
	}

	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range expected {
		if got[i].Name() != name {
			t.Errorf("scraper[%d]: expected %q, got %q", i, name, got[i].Name())
		}
	}
}

func TestEngine_Scrapers_Empty(t *testing.T) {
	e := NewEngine(nil)
	got := e.Scrapers()
	if len(got) != 0 {
		t.Fatalf("expected 0 scrapers, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// 3. TestSearchError_Error
// ---------------------------------------------------------------------------

func TestSearchError_Error(t *testing.T) {
	tests := []struct {
		site string
		err  error
		want string
	}{
		{"kitapyurdu.com", errors.New("timeout"), "kitapyurdu.com: timeout"},
		{"bkmkitap.com", errors.New("404 not found"), "bkmkitap.com: 404 not found"},
		{"", errors.New("empty site"), ": empty site"},
	}

	for _, tt := range tests {
		se := SearchError{Site: tt.site, Err: tt.err}
		if got := se.Error(); got != tt.want {
			t.Errorf("SearchError{%q, %q}.Error() = %q, want %q", tt.site, tt.err, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 4. TestEngine_Search_SingleScraper_Success
// ---------------------------------------------------------------------------

func TestEngine_Search_SingleScraper_Success(t *testing.T) {
	books := []scraper.BookResult{
		{Title: "Go Programming", TotalPrice: 50.0, Site: "site-a", Category: scraper.NewBook},
		{Title: "Advanced Go", TotalPrice: 75.0, Site: "site-a", Category: scraper.NewBook},
	}
	s := &mockScraper{name: "site-a", results: books}
	e := NewEngine(nil, s)

	result := e.Search(context.Background(), "go", scraper.TitleSearch, nil)

	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].Title != "Go Programming" {
		t.Errorf("expected first result title %q, got %q", "Go Programming", result.Results[0].Title)
	}
}

// ---------------------------------------------------------------------------
// 5. TestEngine_Search_MultipleScraper_Success
// ---------------------------------------------------------------------------

func TestEngine_Search_MultipleScraper_Success(t *testing.T) {
	fc := scraper.NewFirecrawlClient("key", "http://localhost")

	s1 := &mockScraper{
		name: "site-a",
		results: []scraper.BookResult{
			{Title: "Book C", TotalPrice: 100.0, Site: "site-a"},
			{Title: "Book A", TotalPrice: 30.0, Site: "site-a"},
		},
	}
	s2 := &mockScraper{
		name: "site-b",
		results: []scraper.BookResult{
			{Title: "Book B", TotalPrice: 50.0, Site: "site-b"},
		},
	}

	// Use firecrawl so there's no 1-second sleep between scrapers
	e := NewEngine(fc, s1, s2)
	result := e.Search(context.Background(), "test", scraper.TitleSearch, nil)

	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}

	// Results must be sorted by TotalPrice ascending
	if result.Results[0].Title != "Book A" {
		t.Errorf("expected first result %q, got %q", "Book A", result.Results[0].Title)
	}
	if result.Results[1].Title != "Book B" {
		t.Errorf("expected second result %q, got %q", "Book B", result.Results[1].Title)
	}
	if result.Results[2].Title != "Book C" {
		t.Errorf("expected third result %q, got %q", "Book C", result.Results[2].Title)
	}
}

// ---------------------------------------------------------------------------
// 6. TestEngine_Search_ScraperError
// ---------------------------------------------------------------------------

func TestEngine_Search_ScraperError(t *testing.T) {
	scrErr := errors.New("network timeout")
	s := &mockScraper{name: "failing-site", err: scrErr}
	e := NewEngine(nil, s)

	result := e.Search(context.Background(), "query", scraper.TitleSearch, nil)

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Site != "failing-site" {
		t.Errorf("expected error site %q, got %q", "failing-site", result.Errors[0].Site)
	}
	if result.Errors[0].Err != scrErr {
		t.Errorf("expected error %v, got %v", scrErr, result.Errors[0].Err)
	}
}

// ---------------------------------------------------------------------------
// 7. TestEngine_Search_MixedResults
// ---------------------------------------------------------------------------

func TestEngine_Search_MixedResults(t *testing.T) {
	fc := scraper.NewFirecrawlClient("key", "http://localhost")

	s1 := &mockScraper{
		name: "good-site",
		results: []scraper.BookResult{
			{Title: "Found Book", TotalPrice: 40.0, Site: "good-site"},
		},
	}
	s2 := &mockScraper{
		name: "bad-site",
		err:  errors.New("500 internal error"),
	}
	s3 := &mockScraper{
		name: "another-good",
		results: []scraper.BookResult{
			{Title: "Another Book", TotalPrice: 20.0, Site: "another-good"},
		},
	}

	e := NewEngine(fc, s1, s2, s3)
	result := e.Search(context.Background(), "mixed", scraper.TitleSearch, nil)

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Site != "bad-site" {
		t.Errorf("expected error from %q, got %q", "bad-site", result.Errors[0].Site)
	}

	// Results sorted by price
	if result.Results[0].TotalPrice != 20.0 {
		t.Errorf("expected cheapest first (20.0), got %.2f", result.Results[0].TotalPrice)
	}
	if result.Results[1].TotalPrice != 40.0 {
		t.Errorf("expected second result price 40.0, got %.2f", result.Results[1].TotalPrice)
	}
}

// ---------------------------------------------------------------------------
// 8. TestEngine_Search_SortsByTotalPrice
// ---------------------------------------------------------------------------

func TestEngine_Search_SortsByTotalPrice(t *testing.T) {
	books := []scraper.BookResult{
		{Title: "Expensive", TotalPrice: 200.0},
		{Title: "Cheap", TotalPrice: 10.0},
		{Title: "Mid", TotalPrice: 80.0},
		{Title: "Free", TotalPrice: 0.0},
		{Title: "Mid2", TotalPrice: 80.0},
	}
	s := &mockScraper{name: "site", results: books}
	e := NewEngine(nil, s)

	result := e.Search(context.Background(), "sort-test", scraper.TitleSearch, nil)

	if len(result.Results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(result.Results))
	}

	for i := 1; i < len(result.Results); i++ {
		if result.Results[i].TotalPrice < result.Results[i-1].TotalPrice {
			t.Errorf("results not sorted: index %d (%.2f) < index %d (%.2f)",
				i, result.Results[i].TotalPrice, i-1, result.Results[i-1].TotalPrice)
		}
	}

	if result.Results[0].Title != "Free" {
		t.Errorf("expected cheapest first %q, got %q", "Free", result.Results[0].Title)
	}
	if result.Results[4].Title != "Expensive" {
		t.Errorf("expected most expensive last %q, got %q", "Expensive", result.Results[4].Title)
	}
}

// ---------------------------------------------------------------------------
// 9. TestEngine_Search_StatusChannel
// ---------------------------------------------------------------------------

func TestEngine_Search_StatusChannel(t *testing.T) {
	fc := scraper.NewFirecrawlClient("key", "http://localhost")

	s1 := &mockScraper{
		name: "ok-site",
		results: []scraper.BookResult{
			{Title: "B1", TotalPrice: 10.0},
		},
	}
	s2 := &mockScraper{
		name: "err-site",
		err:  errors.New("fail"),
	}

	e := NewEngine(fc, s1, s2)
	ch := make(chan SiteStatus, 10)

	var statuses []SiteStatus
	done := make(chan struct{})
	go func() {
		for st := range ch {
			statuses = append(statuses, st)
		}
		close(done)
	}()

	e.Search(context.Background(), "status-test", scraper.TitleSearch, ch)
	<-done

	// We expect 4 status messages: searching+done for ok-site, searching+error for err-site
	if len(statuses) != 4 {
		t.Fatalf("expected 4 status updates, got %d: %+v", len(statuses), statuses)
	}

	// Build a map for easier checking
	type key struct {
		site   string
		status string
	}
	seen := make(map[key]SiteStatus)
	for _, st := range statuses {
		seen[key{st.Site, st.Status}] = st
	}

	// ok-site: searching then done
	if _, ok := seen[key{"ok-site", "searching"}]; !ok {
		t.Error("missing 'searching' status for ok-site")
	}
	if st, ok := seen[key{"ok-site", "done"}]; !ok {
		t.Error("missing 'done' status for ok-site")
	} else if st.Count != 1 {
		t.Errorf("expected count 1 for ok-site done, got %d", st.Count)
	}

	// err-site: searching then error
	if _, ok := seen[key{"err-site", "searching"}]; !ok {
		t.Error("missing 'searching' status for err-site")
	}
	if st, ok := seen[key{"err-site", "error"}]; !ok {
		t.Error("missing 'error' status for err-site")
	} else if st.Err == nil {
		t.Error("expected non-nil error in error status for err-site")
	}
}

// ---------------------------------------------------------------------------
// 10. TestEngine_Search_NilStatusChannel
// ---------------------------------------------------------------------------

func TestEngine_Search_NilStatusChannel(t *testing.T) {
	s := &mockScraper{
		name: "site",
		results: []scraper.BookResult{
			{Title: "Book", TotalPrice: 25.0},
		},
	}
	e := NewEngine(nil, s)

	// Should not panic with nil channel
	result := e.Search(context.Background(), "nil-ch", scraper.TitleSearch, nil)

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
}

// ---------------------------------------------------------------------------
// 11. TestEngine_Search_NoScrapers
// ---------------------------------------------------------------------------

func TestEngine_Search_NoScrapers(t *testing.T) {
	e := NewEngine(nil)

	result := e.Search(context.Background(), "empty", scraper.TitleSearch, nil)

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestEngine_Search_NoScrapers_WithChannel(t *testing.T) {
	e := NewEngine(nil)
	ch := make(chan SiteStatus, 10)

	var statuses []SiteStatus
	done := make(chan struct{})
	go func() {
		for st := range ch {
			statuses = append(statuses, st)
		}
		close(done)
	}()

	result := e.Search(context.Background(), "empty", scraper.TitleSearch, ch)
	<-done

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(statuses))
	}
}

// ---------------------------------------------------------------------------
// 12. TestEngine_Search_FirecrawlPassedToScrapers
// ---------------------------------------------------------------------------

func TestEngine_Search_FirecrawlPassedToScrapers(t *testing.T) {
	fc := scraper.NewFirecrawlClient("test-key", "http://fc.local")
	s1 := &mockScraper{name: "site-a"}
	s2 := &mockScraper{name: "site-b"}
	s3 := &mockScraper{name: "site-c"}

	e := NewEngine(fc, s1, s2, s3)
	e.Search(context.Background(), "fc-test", scraper.TitleSearch, nil)

	for _, s := range []*mockScraper{s1, s2, s3} {
		if s.fc == nil {
			t.Errorf("expected SetFirecrawl called on %q, but fc is nil", s.name)
		}
		if s.fc != fc {
			t.Errorf("expected scraper %q to receive the engine's firecrawl client", s.name)
		}
	}
}

func TestEngine_Search_FirecrawlNotPassedWhenNil(t *testing.T) {
	s1 := &mockScraper{name: "site-a"}
	s2 := &mockScraper{name: "site-b"}

	e := NewEngine(nil, s1, s2)
	e.Search(context.Background(), "no-fc", scraper.TitleSearch, nil)

	for _, s := range []*mockScraper{s1, s2} {
		if s.fc != nil {
			t.Errorf("expected SetFirecrawl NOT called on %q when firecrawl is nil", s.name)
		}
	}
}

// ---------------------------------------------------------------------------
// 13. TestGroupByCategory
// ---------------------------------------------------------------------------

func TestGroupByCategory(t *testing.T) {
	results := []scraper.BookResult{
		{Title: "Used 1", Category: scraper.UsedBook},
		{Title: "New 1", Category: scraper.NewBook},
		{Title: "Used 2", Category: scraper.UsedBook},
		{Title: "New 2", Category: scraper.NewBook},
		{Title: "New 3", Category: scraper.NewBook},
	}

	used, new_ := GroupByCategory(results)

	if len(used) != 2 {
		t.Fatalf("expected 2 used books, got %d", len(used))
	}
	if len(new_) != 3 {
		t.Fatalf("expected 3 new books, got %d", len(new_))
	}

	for _, b := range used {
		if b.Category != scraper.UsedBook {
			t.Errorf("expected UsedBook category for %q, got %d", b.Title, b.Category)
		}
	}
	for _, b := range new_ {
		if b.Category != scraper.NewBook {
			t.Errorf("expected NewBook category for %q, got %d", b.Title, b.Category)
		}
	}

	// Verify order is preserved
	if used[0].Title != "Used 1" || used[1].Title != "Used 2" {
		t.Errorf("used books order not preserved: got %q, %q", used[0].Title, used[1].Title)
	}
	if new_[0].Title != "New 1" || new_[1].Title != "New 2" || new_[2].Title != "New 3" {
		t.Errorf("new books order not preserved: got %q, %q, %q", new_[0].Title, new_[1].Title, new_[2].Title)
	}
}

// ---------------------------------------------------------------------------
// 14. TestGroupByCategory_Empty
// ---------------------------------------------------------------------------

func TestGroupByCategory_Empty(t *testing.T) {
	used, new_ := GroupByCategory(nil)
	if used != nil {
		t.Errorf("expected nil used slice, got %v", used)
	}
	if new_ != nil {
		t.Errorf("expected nil new slice, got %v", new_)
	}

	used2, new2 := GroupByCategory([]scraper.BookResult{})
	if used2 != nil {
		t.Errorf("expected nil used slice for empty input, got %v", used2)
	}
	if new2 != nil {
		t.Errorf("expected nil new slice for empty input, got %v", new2)
	}
}

// ---------------------------------------------------------------------------
// 15. TestGroupByCategory_OnlyUsed
// ---------------------------------------------------------------------------

func TestGroupByCategory_OnlyUsed(t *testing.T) {
	results := []scraper.BookResult{
		{Title: "Used A", Category: scraper.UsedBook},
		{Title: "Used B", Category: scraper.UsedBook},
		{Title: "Used C", Category: scraper.UsedBook},
	}

	used, new_ := GroupByCategory(results)

	if len(used) != 3 {
		t.Fatalf("expected 3 used books, got %d", len(used))
	}
	if new_ != nil {
		t.Errorf("expected nil new slice, got %v", new_)
	}

	expected := []string{"Used A", "Used B", "Used C"}
	for i, name := range expected {
		if used[i].Title != name {
			t.Errorf("used[%d]: expected %q, got %q", i, name, used[i].Title)
		}
	}
}

// ---------------------------------------------------------------------------
// 16. TestGroupByCategory_OnlyNew
// ---------------------------------------------------------------------------

func TestGroupByCategory_OnlyNew(t *testing.T) {
	results := []scraper.BookResult{
		{Title: "New A", Category: scraper.NewBook},
		{Title: "New B", Category: scraper.NewBook},
	}

	used, new_ := GroupByCategory(results)

	if used != nil {
		t.Errorf("expected nil used slice, got %v", used)
	}
	if len(new_) != 2 {
		t.Fatalf("expected 2 new books, got %d", len(new_))
	}

	if new_[0].Title != "New A" || new_[1].Title != "New B" {
		t.Errorf("expected [New A, New B], got [%s, %s]", new_[0].Title, new_[1].Title)
	}
}
