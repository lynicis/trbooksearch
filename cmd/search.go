package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"trbooksearch/internal/config"
	"trbooksearch/internal/engine"
	"trbooksearch/internal/scraper"
	"trbooksearch/internal/scraper/sites"
	"trbooksearch/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flagISBN      bool
	flagFlat      bool
	flagLimit     int
	flagSites     string
	flagExclude   string
	flagFirecrawl bool
)

var searchCmd = &cobra.Command{
	Use:   "search [kitap adı veya ISBN]",
	Short: "Kitap ara ve fiyatları karşılaştır",
	Long: `Tüm desteklenen kitap sitelerinde arama yaparak sonuçları
en ucuzdan en pahalıya sıralar. Kargo ücreti dahil toplam fiyat gösterilir.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().BoolVar(&flagISBN, "isbn", false, "ISBN ile ara")
	searchCmd.Flags().BoolVar(&flagFlat, "flat", false, "Tek liste halinde göster (gruplamadan)")
	searchCmd.Flags().IntVar(&flagLimit, "limit", 10, "Site başına maksimum sonuç sayısı")
	searchCmd.Flags().StringVar(&flagSites, "sites", "", "Sadece belirtilen sitelerde ara (virgülle ayır)")
	searchCmd.Flags().StringVar(&flagExclude, "exclude", "", "Belirtilen siteleri hariç tut (virgülle ayır)")
	searchCmd.Flags().BoolVar(&flagFirecrawl, "firecrawl", false, "Firecrawl API ile tüm siteleri tara (API anahtarı gerektirir)")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	searchType := scraper.TitleSearch
	if flagISBN {
		searchType = scraper.ISBNSearch
	}

	// Load config for Firecrawl
	var firecrawlClient *scraper.FirecrawlClient
	if flagFirecrawl {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("yapılandırma hatası: %w", err)
		}
		if cfg.Firecrawl.APIKey == "" {
			return fmt.Errorf("Firecrawl API anahtarı bulunamadı.\n\nYapılandırma dosyası oluşturun: %s\n\nİçerik:\nfirecrawl:\n  api_key: \"fc-...\"", config.ConfigPath())
		}
		firecrawlClient = scraper.NewFirecrawlClient(cfg.Firecrawl.APIKey, cfg.Firecrawl.APIURL)
	}

	// Build scraper list
	allScrapers := sites.AllScrapers(flagLimit, flagFirecrawl)

	// Filter by --sites
	var includeSites map[string]bool
	if flagSites != "" {
		includeSites = make(map[string]bool)
		for _, s := range strings.Split(flagSites, ",") {
			includeSites[strings.TrimSpace(strings.ToLower(s))] = true
		}
	}

	// Filter by --exclude
	var excludeSites map[string]bool
	if flagExclude != "" {
		excludeSites = make(map[string]bool)
		for _, s := range strings.Split(flagExclude, ",") {
			excludeSites[strings.TrimSpace(strings.ToLower(s))] = true
		}
	}

	var filtered []scraper.Scraper
	for _, s := range allScrapers {
		name := strings.ToLower(s.Name())
		if includeSites != nil && !includeSites[name] {
			continue
		}
		if excludeSites != nil && excludeSites[name] {
			continue
		}
		filtered = append(filtered, s)
	}

	if len(filtered) == 0 {
		return fmt.Errorf("hiçbir site seçilmedi, filtreleri kontrol edin")
	}

	eng := engine.NewEngine(firecrawlClient, filtered...)

	// Create TUI model and run
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	model := tui.NewModel(eng, query, searchType, !flagFlat, ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI hatası: %v\n", err)
		return err
	}

	// Check if user quit early
	if m, ok := finalModel.(tui.Model); ok && m.Quitting {
		return nil
	}

	return nil
}
