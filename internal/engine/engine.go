package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
	"trbooksearch/internal/scraper"
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
	scrapers []scraper.Scraper
}

// NewEngine creates a new search engine with the given scrapers.
func NewEngine(scrapers ...scraper.Scraper) *Engine {
	return &Engine{scrapers: scrapers}
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

// Search dispatches the query to all scrapers with staggered parallel launch.
// Each scraper launches its own isolated browser instance.
func (e *Engine) Search(ctx context.Context, query string, searchType scraper.SearchType, statusCh chan<- SiteStatus) SearchResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []scraper.BookResult
		errors  []SearchError
	)

	for i, s := range e.scrapers {
		wg.Add(1)
		// Stagger launches by 1s to reduce bot detection from simultaneous browser starts
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		go func(s scraper.Scraper) {
			defer wg.Done()

			if statusCh != nil {
				statusCh <- SiteStatus{Site: s.Name(), Status: "searching"}
			}

			books, err := s.Search(ctx, query, searchType)

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
