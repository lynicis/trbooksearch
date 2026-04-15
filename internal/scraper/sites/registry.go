package sites

import "github.com/lynicis/trbooksearch/internal/scraper"

// AllScrapers returns all available scrapers with the given result limit.
// When firecrawlEnabled is true, Firecrawl-only sites (letgo, dolap, gardrops, dr, pandora) are included.
func AllScrapers(limit int, firecrawlEnabled bool) []scraper.Scraper {
	scrapers := []scraper.Scraper{
		NewNadirkitap(limit),
		NewKitapyurdu(limit),
		NewTrendyol(limit),
		NewHepsiburada(limit),
		NewAmazon(limit),
		NewBkmkitap(limit),
		NewIdefix(limit),
	}
	if firecrawlEnabled {
		scrapers = append(scrapers,
			NewLetgo(limit),
			NewDolap(limit),
			NewGardrops(limit),
			NewDR(limit),
			NewPandora(limit),
		)
	}
	return scrapers
}
