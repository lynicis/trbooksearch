package tui

import (
	"context"
	"time"

	"trbooksearch/internal/engine"
	"trbooksearch/internal/scraper"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// refreshViewport re-applies filter/sort and updates the viewport content.
func (m *Model) refreshViewport() {
	filtered := applyFilter(m.results.Results, m.filterInput.Value(), m.filterColumn)
	sorted := applySort(filtered, m.sortColumn, m.sortDirection)
	content := renderFilteredResultsContent(sorted, m.results, m.grouped, m.elapsed)
	m.viewport.SetContent(content)
}

// Messages

type searchStartMsg struct{}

type siteStatusMsg engine.SiteStatus

type searchDoneMsg engine.SearchResult

type tickMsg time.Time

// Init returns the initial commands: start spinner, start search, start tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return searchStartMsg{} },
		tickCmd(),
	)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		// When filter is active, delegate to textinput
		if m.filterActive {
			switch msg.String() {
			case "ctrl+c":
				m.Quitting = true
				return m, tea.Quit
			case "escape":
				m.filterActive = false
				m.filterInput.Reset()
				m.filterInput.Blur()
				m.refreshViewport()
				// Restore viewport height
				vpHeight := m.height - m.reservedLines()
				if vpHeight < 1 {
					vpHeight = 1
				}
				m.viewport.Height = vpHeight
				return m, nil
			case "enter":
				m.filterActive = false
				m.filterInput.Blur()
				// Keep filter text active, just exit input mode
				// Restore viewport height (filter bar hidden but filter still applied)
				vpHeight := m.height - m.reservedLines()
				if vpHeight < 1 {
					vpHeight = 1
				}
				m.viewport.Height = vpHeight
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.refreshViewport()
				return m, cmd
			}
		}

		// Normal mode keys (filter NOT active)
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "escape":
			// Clear confirmed filter if one exists
			if !m.searching && m.filterInput.Value() != "" {
				m.filterInput.Reset()
				m.filterColumn = -1
				m.refreshViewport()
				vpHeight := m.height - m.reservedLines()
				if vpHeight < 1 {
					vpHeight = 1
				}
				m.viewport.Height = vpHeight
				return m, nil
			}
		case "/":
			if !m.searching {
				m.filterActive = true
				m.filterColumn = -1
				m.filterInput.Placeholder = "filtre..."
				cmd := m.filterInput.Focus()
				m.filterInput.SetValue("")
				vpHeight := m.height - m.reservedLines()
				if vpHeight < 1 {
					vpHeight = 1
				}
				m.viewport.Height = vpHeight
				return m, cmd
			}
		case "1", "2", "3", "4", "5", "6", "7":
			if !m.searching {
				col := int(msg.String()[0] - '1') // "1"->0, "7"->6
				m.filterActive = true
				m.filterColumn = col
				colNames := []string{"Site", "Başlık", "Yazar", "Satıcı", "Fiyat", "Kargo", "Toplam"}
				m.filterInput.Placeholder = colNames[col] + " filtre..."
				cmd := m.filterInput.Focus()
				m.filterInput.SetValue("")
				vpHeight := m.height - m.reservedLines()
				if vpHeight < 1 {
					vpHeight = 1
				}
				m.viewport.Height = vpHeight
				return m, cmd
			}
		case "s":
			if !m.searching {
				sortCols := []int{-1, colIdxSite, colIdxTitle, colIdxAuthor, colIdxSeller, colIdxTotal}
				current := -1
				for i, c := range sortCols {
					if c == m.sortColumn {
						current = i
						break
					}
				}
				next := (current + 1) % len(sortCols)
				m.sortColumn = sortCols[next]
				if m.sortColumn == -1 {
					m.sortDirection = 2
				} else {
					m.sortDirection = 0
				}
				m.refreshViewport()
				return m, nil
			}
		case "S":
			if !m.searching && m.sortColumn != -1 {
				m.sortDirection = (m.sortDirection + 1) % 3
				if m.sortDirection == 2 {
					m.sortColumn = -1
				}
				m.refreshViewport()
				return m, nil
			}
		case "r":
			if !m.searching {
				m.sortColumn = -1
				m.sortDirection = 2
				m.refreshViewport()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		vpHeight := m.height - m.reservedLines()
		if vpHeight < 1 {
			vpHeight = 1
		}

		if !m.viewportReady {
			m.viewport = viewport.New(m.width, vpHeight)
			m.viewport.SetContent("")
			m.viewportReady = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}

		// If results are already loaded, re-render into the viewport
		if !m.searching && len(m.results.Results) > 0 {
			m.refreshViewport()
		}

		return m, nil

	case searchStartMsg:
		m.startTime = time.Now()
		m.searching = true

		statusCh := make(chan engine.SiteStatus, len(m.engine.Scrapers())*2)
		m.statusCh = statusCh

		return m, tea.Batch(
			listenForStatus(statusCh),
			runSearch(m.engine, m.ctx, m.query, m.searchType, statusCh),
		)

	case siteStatusMsg:
		status := engine.SiteStatus(msg)
		m.statuses[status.Site] = status
		if m.statusCh != nil {
			return m, listenForStatus(m.statusCh)
		}
		return m, nil

	case searchDoneMsg:
		m.results = engine.SearchResult(msg)
		m.searching = false
		m.statusCh = nil
		m.elapsed = time.Since(m.startTime)
		// Recalculate viewport height now that results are loaded
		// (cheapest 3 panel takes space that wasn't accounted for during initial setup)
		vpHeight := m.height - m.reservedLines()
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.viewport.Height = vpHeight
		m.refreshViewport()
		m.viewport.GotoTop()
		return m, nil

	case tickMsg:
		if m.searching {
			m.elapsed = time.Since(m.startTime)
			return m, tickCmd()
		}
		return m, nil
	}

	// While searching, update spinner
	if m.searching {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// After search, delegate to viewport for scrolling
	if m.viewportReady && !m.searching {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// listenForStatus reads a single status update from the channel.
func listenForStatus(statusCh chan engine.SiteStatus) tea.Cmd {
	return func() tea.Msg {
		status, ok := <-statusCh
		if !ok {
			return nil
		}
		return siteStatusMsg(status)
	}
}

// runSearch launches the engine search. When done, sends searchDoneMsg.
func runSearch(eng *engine.Engine, ctx context.Context, query string, searchType scraper.SearchType, statusCh chan engine.SiteStatus) tea.Cmd {
	return func() tea.Msg {
		result := eng.Search(ctx, query, searchType, statusCh)
		return searchDoneMsg(result)
	}
}

// tickCmd returns a command that sends a tickMsg after a short delay.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
