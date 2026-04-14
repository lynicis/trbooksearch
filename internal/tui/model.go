package tui

import (
	"context"
	"time"

	"trbooksearch/internal/engine"
	"trbooksearch/internal/scraper"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Model is the Bubble Tea model for the search TUI.
type Model struct {
	engine     *engine.Engine
	query      string
	searchType scraper.SearchType
	grouped    bool // true = show used/new sections, false = flat list
	ctx        context.Context

	// State
	searching bool
	results   engine.SearchResult
	statuses  map[string]engine.SiteStatus // per-site status
	statusCh  chan engine.SiteStatus       // channel for receiving status updates
	startTime time.Time
	elapsed   time.Duration
	spinner   spinner.Model
	Quitting  bool

	// Filter state
	filterActive bool
	filterInput  textinput.Model
	filterColumn int // -1 = all columns, 0-6 = specific column

	// Sort state
	sortColumn    int // -1 = default (price asc), 0-6 = column index
	sortDirection int // 0 = asc, 1 = desc, 2 = none (default engine sort)

	// Viewport for scrollable results
	viewport      viewport.Model
	viewportReady bool
	width         int
	height        int
}

// NewModel creates a new TUI model ready for tea.NewProgram.
func NewModel(eng *engine.Engine, query string, searchType scraper.SearchType, grouped bool, ctx context.Context) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize status map with all scrapers as pending
	statuses := make(map[string]engine.SiteStatus)
	for _, sc := range eng.Scrapers() {
		statuses[sc.Name()] = engine.SiteStatus{
			Site:   sc.Name(),
			Status: "pending",
		}
	}

	ti := textinput.New()
	ti.Placeholder = "filtre..."
	ti.CharLimit = 100

	return Model{
		engine:        eng,
		query:         query,
		searchType:    searchType,
		grouped:       grouped,
		ctx:           ctx,
		searching:     true,
		statuses:      statuses,
		spinner:       s,
		width:         80,
		height:        24,
		filterColumn:  -1,
		filterInput:   ti,
		sortColumn:    -1,
		sortDirection: 2,
	}
}

// reservedLines returns the number of lines reserved for header/footer
// outside the viewport. Accounts for cheapest 3 panel, filter bar, and sort indicator.
func (m Model) reservedLines() int {
	n := 3 // header line + footer + padding
	// Cheapest 3 panel
	count := len(m.results.Results)
	if count > 3 {
		count = 3
	}
	if count > 0 {
		n += 1 + count // title line + card lines
	}
	// Filter bar or confirmed filter indicator
	if m.filterActive || m.filterInput.Value() != "" {
		n++
	}
	// Sort indicator
	if m.sortColumn != -1 && m.sortDirection != 2 {
		n++
	}
	return n
}
