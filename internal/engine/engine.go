package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lynicis/trbooksearch/internal/relevance"
	"github.com/lynicis/trbooksearch/internal/scraper"
)

// SearchError holds a scraper error alongside the scraper name.
type SearchError struct {
	Site string
	Err  error
}

func (e SearchError) Error() string {
	return fmt.Sprintf("%s: %s", e.Site, e.Err.Error())
}

// SiteStatus tracks the progress of individual scrapers.
type SiteStatus struct {
	Site   string
	Status string // "searching", "done", "error"
	Count  int    // number of results found
	Err    error
}

// Engine dispatches searches to multiple scrapers in parallel.
type Engine struct {
	scrapers  []scraper.Scraper
	firecrawl *scraper.FirecrawlClient
}

// NewEngine creates a new search engine with the given scrapers.
// If firecrawl is non-nil, all scrapers will use Firecrawl for HTML fetching.
func NewEngine(firecrawl *scraper.FirecrawlClient, scrapers ...scraper.Scraper) *Engine {
	return &Engine{scrapers: scrapers, firecrawl: firecrawl}
}

// Scrapers returns the list of registered scrapers.
func (e *Engine) Scrapers() []scraper.Scraper {
	return e.scrapers
}

// SearchResult holds the aggregated results from all scrapers.
type SearchResult struct {
	Results []scraper.BookResult
	Errors  []SearchError
}

// SearchOptions holds query and filter parameters for relevance scoring.
type SearchOptions struct {
	Query           string
	SearchType      scraper.SearchType
	AuthorFilter    string  // from --author flag, empty = not set
	PublisherFilter string  // from --publisher flag, empty = not set
	MinRelevance    float64 // minimum relevance threshold (0.0-1.0)
}

// ScoreAndFilterResults computes relevance scores for all results and
// filters out those below the minimum relevance threshold.
func ScoreAndFilterResults(results []scraper.BookResult, opts SearchOptions) []scraper.BookResult {
	for i := range results {
		results[i].Relevance = relevance.ComputeRelevance(
			results[i].Title,
			results[i].Author,
			results[i].Publisher,
			opts.Query,
			opts.AuthorFilter,
			opts.PublisherFilter,
		)
	}
	return FilterByRelevance(results, opts.MinRelevance)
}

// FilterByRelevance removes results below the given relevance threshold.
func FilterByRelevance(results []scraper.BookResult, threshold float64) []scraper.BookResult {
	if threshold <= 0 {
		return results
	}
	filtered := make([]scraper.BookResult, 0, len(results))
	for _, r := range results {
		if r.Relevance >= threshold {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// Search dispatches the query to all scrapers with staggered parallel launch.
// When Firecrawl is configured, scrapers use the API instead of local browsers.
func (e *Engine) Search(ctx context.Context, opts SearchOptions, statusCh chan<- SiteStatus) SearchResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []scraper.BookResult
		errors  []SearchError
	)

	// Pass Firecrawl client to all scrapers if available.
	if e.firecrawl != nil {
		for _, s := range e.scrapers {
			s.SetFirecrawl(e.firecrawl)
		}
	}

	for i, s := range e.scrapers {
		wg.Add(1)
		// Stagger launches by 1s to reduce bot detection from simultaneous browser starts
		if i > 0 && e.firecrawl == nil {
			time.Sleep(1 * time.Second)
		}
		go func(s scraper.Scraper) {
			defer wg.Done()

			if statusCh != nil {
				statusCh <- SiteStatus{Site: s.Name(), Status: "searching"}
			}

			books, err := s.Search(ctx, opts.Query, opts.SearchType)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, SearchError{Site: s.Name(), Err: err})
				if statusCh != nil {
					statusCh <- SiteStatus{Site: s.Name(), Status: "error", Err: err}
				}
				return
			}

			results = append(results, books...)
			if statusCh != nil {
				statusCh <- SiteStatus{Site: s.Name(), Status: "done", Count: len(books)}
			}
		}(s)
	}

	wg.Wait()

	if statusCh != nil {
		close(statusCh)
	}

	results = ScoreAndFilterResults(results, opts)

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalPrice < results[j].TotalPrice
	})

	return SearchResult{Results: results, Errors: errors}
}

// GroupByCategory splits results into used and new book groups.
func GroupByCategory(results []scraper.BookResult) (used, new_ []scraper.BookResult) {
	for _, r := range results {
		if r.Category == scraper.UsedBook {
			used = append(used, r)
		} else {
			new_ = append(new_, r)
		}
	}
	return
}
