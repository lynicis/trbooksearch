package sites

import "trbooksearch/internal/scraper"

// AllScrapers returns all available scrapers with the given result limit.
func AllScrapers(limit int) []scraper.Scraper {
	return []scraper.Scraper{
		NewNadirkitap(limit),
		NewKitapyurdu(limit),
		NewTrendyol(limit),
		NewHepsiburada(limit),
		NewAmazon(limit),
	}
}
