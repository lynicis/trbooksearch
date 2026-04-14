package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lynicis/trbooksearch/internal/engine"
	"github.com/lynicis/trbooksearch/internal/scraper"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	usedSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	newSectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236"))

	priceStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	freeCargoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("48"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	scrollTrackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	scrollThumbStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))

	goldStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	silverStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	bronzeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
)

// Column widths
const (
	colSite   = 18
	colTitle  = 40
	colAuthor = 20
	colSeller = 16
	colPrice  = 10
	colCargo  = 10
	colTotal  = 10
	colNote   = 20
)

// Column index constants for filter/sort
const (
	colIdxSite   = 0
	colIdxTitle  = 1
	colIdxAuthor = 2
	colIdxSeller = 3
	colIdxPrice  = 4
	colIdxCargo  = 5
	colIdxTotal  = 6
)

// View renders the TUI.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder

	if m.searching {
		m.renderSearching(&b)
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("q: çıkış"))
		b.WriteString("\n")
	} else {
		// Header
		fmt.Fprintf(&b, " %s Arama: %s\n",
			titleStyle.Render("📖"),
			titleStyle.Render(fmt.Sprintf("%q", m.query)),
		)

		// Cheapest 3 panel (always from full results, not filtered)
		cheapest3 := renderCheapest3(m.results.Results)
		if cheapest3 != "" {
			b.WriteString(cheapest3)
		}

		// Filter bar (when typing) or filter indicator (when confirmed)
		if m.filterActive {
			colLabel := "Tümü"
			colNames := []string{"Site", "Başlık", "Yazar", "Satıcı", "Fiyat", "Kargo", "Toplam"}
			if m.filterColumn >= 0 && m.filterColumn < len(colNames) {
				colLabel = colNames[m.filterColumn]
			}
			fmt.Fprintf(&b, " 🔍 [%s] %s\n", colLabel, m.filterInput.View())
		} else if m.filterInput.Value() != "" {
			// Show confirmed filter indicator
			colLabel := "Tümü"
			colNames := []string{"Site", "Başlık", "Yazar", "Satıcı", "Fiyat", "Kargo", "Toplam"}
			if m.filterColumn >= 0 && m.filterColumn < len(colNames) {
				colLabel = colNames[m.filterColumn]
			}
			b.WriteString(dimStyle.Render(fmt.Sprintf(" 🔍 [%s] %q  (/ düzenle • esc temizle)", colLabel, m.filterInput.Value())))
			b.WriteString("\n")
		}

		// Sort indicator (when not default)
		if m.sortColumn != -1 && m.sortDirection != 2 {
			colNames := []string{"Site", "Başlık", "Yazar", "Satıcı", "Fiyat", "Kargo", "Toplam"}
			arrow := "↑"
			if m.sortDirection == 1 {
				arrow = "↓"
			}
			b.WriteString(dimStyle.Render(fmt.Sprintf(" Sıralama: %s %s", colNames[m.sortColumn], arrow)))
			b.WriteString("\n")
		}

		// Scrollable viewport with scrollbar
		if m.viewportReady {
			vpContent := m.viewport.View()
			vpLines := strings.Split(vpContent, "\n")
			totalContentLines := m.viewport.TotalLineCount()
			needsScrollbar := totalContentLines > m.viewport.Height

			if needsScrollbar {
				b.WriteString(renderWithScrollbar(vpLines, m.viewport.Height, m.viewport.ScrollPercent()))
			} else {
				b.WriteString(vpContent)
			}
		}

		// Footer with help
		b.WriteString("\n")
		totalContentLines := m.viewport.TotalLineCount()
		scrollPct := m.viewport.ScrollPercent() * 100
		footer := fmt.Sprintf(" ↑↓ kaydır • / filtre • 1-7 sütun • s sırala • q çıkış • %%%d", int(scrollPct))
		if totalContentLines <= m.viewport.Height {
			footer = " / filtre • 1-7 sütun • s sırala • q çıkış"
		}
		b.WriteString(helpStyle.Render(footer))
		b.WriteString("\n")
	}

	return b.String()
}

// renderSearching shows the spinner and per-site status.
func (m Model) renderSearching(b *strings.Builder) {
	fmt.Fprintf(b, "%s Aranıyor: %s  %s\n\n",
		m.spinner.View(),
		titleStyle.Render(fmt.Sprintf("%q", m.query)),
		dimStyle.Render(fmt.Sprintf("(%.1fs)", m.elapsed.Seconds())),
	)

	for _, sc := range m.engine.Scrapers() {
		status, ok := m.statuses[sc.Name()]
		if !ok {
			status = engine.SiteStatus{Site: sc.Name(), Status: "pending"}
		}

		var icon, detail string
		switch status.Status {
		case "done":
			icon = priceStyle.Render("✓")
			detail = dimStyle.Render(fmt.Sprintf("(%d sonuç)", status.Count))
		case "error":
			icon = errorStyle.Render("✗")
			errMsg := "hata"
			if status.Err != nil {
				errMsg = status.Err.Error()
				if len(errMsg) > 40 {
					errMsg = errMsg[:40] + "…"
				}
			}
			detail = errorStyle.Render(fmt.Sprintf("(%s)", errMsg))
		case "searching":
			icon = warningStyle.Render("⏳")
			detail = ""
		default:
			icon = dimStyle.Render("⏳")
			detail = ""
		}

		fmt.Fprintf(b, "  %s %s %s\n", icon, sc.Name(), detail)
	}
}

// renderCheapest3 renders the top 3 cheapest unique books panel.
// Deduplicates by URL+Site so loyalty/cargo variants of the same listing
// don't occupy multiple slots. Shows the cheapest variant of each unique book.
func renderCheapest3(results []scraper.BookResult) string {
	if len(results) == 0 {
		return ""
	}

	// Collect unique books (by URL+Site), keeping only the cheapest variant
	seen := make(map[string]bool)
	var unique []scraper.BookResult
	for _, r := range results {
		key := r.Site + "|" + r.URL
		if r.URL == "" {
			key = r.Site + "|" + r.Title
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, r)
	}

	count := 3
	if len(unique) < count {
		count = len(unique)
	}
	if count == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" En Ucuz 3 Sonuç"))
	b.WriteString("\n")

	rankStyles := []lipgloss.Style{goldStyle, silverStyle, bronzeStyle}
	rankLabels := []string{"1.", "2.", "3."}

	for i := 0; i < count; i++ {
		r := unique[i]
		style := rankStyles[i]
		label := style.Render(rankLabels[i])

		titleText := style.Render(truncate(r.Title, 35))
		if r.URL != "" {
			titleText = hyperlink(r.URL, titleText)
		}

		card := fmt.Sprintf("  %s %s - %s | %s",
			label,
			titleText,
			dimStyle.Render(r.Site),
			priceStyle.Render(fmt.Sprintf("%.2f₺", r.TotalPrice)),
		)
		if r.Seller != "" {
			card += " | " + dimStyle.Render(truncate(r.Seller, 15))
		}
		b.WriteString(card + "\n")
	}

	return b.String()
}

// renderTable renders a formatted table of book results.
func renderTable(b *strings.Builder, results []scraper.BookResult) {
	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %*s %*s %*s %-*s",
		colSite, "Site",
		colTitle, "Başlık",
		colAuthor, "Yazar",
		colSeller, "Satıcı",
		colPrice, "Fiyat",
		colCargo, "Kargo",
		colTotal, "Toplam",
		colNote, "Not",
	)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	for _, r := range results {
		site := truncate(r.Site, colSite)
		title := truncate(r.Title, colTitle)
		author := truncate(r.Author, colAuthor)
		seller := truncate(r.Seller, colSeller)
		price := fmt.Sprintf("%.2f₺", r.Price)
		note := truncate(r.LoyaltyNote, colNote)

		// Pad title to column width first, then wrap with hyperlink
		// so the invisible escape codes don't break alignment
		paddedTitle := padRight(title, colTitle)
		if r.URL != "" {
			paddedTitle = hyperlink(r.URL, paddedTitle)
		}

		var cargo string
		if r.CargoUnknown {
			cargo = "?"
		} else if r.CargoFee == 0 {
			cargo = "Ücretsiz"
		} else {
			cargo = fmt.Sprintf("%.2f₺", r.CargoFee)
		}

		total := fmt.Sprintf("%.2f₺", r.TotalPrice)

		if r.FreeCargo || r.CargoFee == 0 {
			cargo = freeCargoStyle.Render(cargo)
		}

		// Build line manually — use paddedTitle (with hyperlink) instead of %-*s for title
		line := fmt.Sprintf("  %-*s %s %-*s %-*s %*s %*s %*s %-*s",
			colSite, site,
			paddedTitle,
			colAuthor, author,
			colSeller, seller,
			colPrice, price,
			colCargo, cargo,
			colTotal, total,
			colNote, note,
		)
		b.WriteString(line)
		b.WriteString("\n")
	}
}

// hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
// Supported by iTerm2, Ghostty, Windows Terminal, GNOME Terminal, etc.
// Terminals that don't support it will simply display the text without the link.
func hyperlink(url, text string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

// padRight pads a string with spaces to the given width (by visible rune count).
func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// truncate shortens a string to maxLen, adding "…" if truncated.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// countSites counts the number of distinct sites in results.
func countSites(results []scraper.BookResult) int {
	sites := make(map[string]bool)
	for _, r := range results {
		sites[r.Site] = true
	}
	return len(sites)
}

// renderWithScrollbar appends a scrollbar track to the right side of each
// viewport line. The thumb position and size reflect the current scroll state.
func renderWithScrollbar(lines []string, viewportHeight int, scrollPercent float64) string {
	// Calculate thumb size — proportional to how much of the content is visible
	thumbSize := viewportHeight
	if viewportHeight > 0 && len(lines) > 0 {
		thumbSize = max(1, (viewportHeight*viewportHeight)/max(len(lines), viewportHeight))
	}

	// Calculate thumb start position
	trackSpace := viewportHeight - thumbSize
	thumbStart := int(float64(trackSpace) * scrollPercent)

	var b strings.Builder
	for i, line := range lines {
		if i >= viewportHeight {
			break
		}
		b.WriteString(line)

		// Append scrollbar character
		if i >= thumbStart && i < thumbStart+thumbSize {
			b.WriteString(scrollThumbStyle.Render(" ┃"))
		} else {
			b.WriteString(scrollTrackStyle.Render(" │"))
		}

		if i < viewportHeight-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderFilteredResultsContent builds viewport content from pre-filtered/sorted results.
func renderFilteredResultsContent(results []scraper.BookResult, fullResults engine.SearchResult, grouped bool, elapsed time.Duration) string {
	var b strings.Builder

	totalResults := len(fullResults.Results)
	filteredCount := len(results)
	siteCount := countSites(results)

	if filteredCount == 0 && totalResults > 0 {
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("  Filtreye uygun sonuç bulunamadı"))
		b.WriteString("\n")
	} else if filteredCount == 0 {
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("  Sonuç bulunamadı"))
		b.WriteString("\n")
	} else if grouped {
		used, new_ := engine.GroupByCategory(results)
		if len(used) > 0 {
			b.WriteString("\n")
			b.WriteString(usedSectionStyle.Render("━━━ İKİNCİ EL KİTAPLAR ━━━"))
			b.WriteString("\n")
			renderTable(&b, used)
		}
		if len(new_) > 0 {
			b.WriteString("\n")
			b.WriteString(newSectionStyle.Render("━━━ YENİ KİTAPLAR ━━━"))
			b.WriteString("\n")
			renderTable(&b, new_)
		}
	} else {
		b.WriteString("\n")
		renderTable(&b, results)
	}

	// Errors
	if len(fullResults.Errors) > 0 {
		b.WriteString("\n")
		for _, e := range fullResults.Errors {
			fmt.Fprintf(&b, "  %s %s: %s\n",
				errorStyle.Render("⚠"),
				warningStyle.Render(e.Site),
				errorStyle.Render(e.Err.Error()),
			)
		}
	}

	// Summary
	b.WriteString("\n")
	if filteredCount < totalResults {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d sitede %d/%d sonuç gösteriliyor (%.1fs)",
			siteCount, filteredCount, totalResults, elapsed.Seconds())))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d sitede %d sonuç bulundu (%.1fs)",
			siteCount, totalResults, elapsed.Seconds())))
	}
	b.WriteString("\n")

	return b.String()
}

// applyFilter returns results matching the filter text.
// column == -1 matches across all string fields; 0-6 matches specific column.
func applyFilter(results []scraper.BookResult, text string, column int) []scraper.BookResult {
	if text == "" {
		return results
	}
	text = strings.ToLower(text)
	var filtered []scraper.BookResult
	for _, r := range results {
		if matchesFilter(r, text, column) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func matchesFilter(r scraper.BookResult, text string, column int) bool {
	match := func(s string) bool {
		return strings.Contains(strings.ToLower(s), text)
	}
	switch column {
	case colIdxSite:
		return match(r.Site)
	case colIdxTitle:
		return match(r.Title)
	case colIdxAuthor:
		return match(r.Author)
	case colIdxSeller:
		return match(r.Seller)
	case colIdxPrice:
		return match(fmt.Sprintf("%.2f", r.Price))
	case colIdxCargo:
		return match(fmt.Sprintf("%.2f", r.CargoFee))
	case colIdxTotal:
		return match(fmt.Sprintf("%.2f", r.TotalPrice))
	default: // -1 = all columns
		return match(r.Site) || match(r.Title) || match(r.Author) ||
			match(r.Seller) || match(r.LoyaltyNote)
	}
}

// applySort returns a sorted copy of results. Does not modify the input slice.
func applySort(results []scraper.BookResult, column int, direction int) []scraper.BookResult {
	if direction == 2 || column == -1 {
		return results // default engine sort
	}
	sorted := make([]scraper.BookResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		var cmp int
		switch column {
		case colIdxSite:
			cmp = strings.Compare(strings.ToLower(sorted[i].Site), strings.ToLower(sorted[j].Site))
		case colIdxTitle:
			cmp = strings.Compare(strings.ToLower(sorted[i].Title), strings.ToLower(sorted[j].Title))
		case colIdxAuthor:
			cmp = strings.Compare(strings.ToLower(sorted[i].Author), strings.ToLower(sorted[j].Author))
		case colIdxSeller:
			cmp = strings.Compare(strings.ToLower(sorted[i].Seller), strings.ToLower(sorted[j].Seller))
		case colIdxPrice:
			cmp = compareFloat(sorted[i].Price, sorted[j].Price)
		case colIdxCargo:
			cmp = compareFloat(sorted[i].CargoFee, sorted[j].CargoFee)
		case colIdxTotal:
			cmp = compareFloat(sorted[i].TotalPrice, sorted[j].TotalPrice)
		}
		if direction == 1 { // descending
			cmp = -cmp
		}
		return cmp < 0
	})
	return sorted
}

func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
