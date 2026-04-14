package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lynicis/trbooksearch/internal/config"
	"github.com/spf13/cobra"
)

var setAPIKeyCmd = &cobra.Command{
	Use:   "set-api-key [anahtar]",
	Short: "Firecrawl API anahtarını ayarla",
	Long: `Firecrawl API anahtarını yapılandırma dosyasına kaydeder.

API anahtarını argüman olarak verebilir veya interaktif olarak girebilirsiniz.

Örnekler:
  trbooksearch set-api-key fc-abc123
  trbooksearch set-api-key`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSetAPIKey,
}

func init() {
	rootCmd.AddCommand(setAPIKeyCmd)
}

func runSetAPIKey(cmd *cobra.Command, args []string) error {
	var apiKey string

	if len(args) > 0 {
		apiKey = args[0]
	} else {
		// Interactive prompt
		fmt.Print("Firecrawl API anahtarı: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("girdi okunamadı: %w", err)
		}
		apiKey = strings.TrimSpace(input)
	}

	if apiKey == "" {
		return fmt.Errorf("API anahtarı boş olamaz")
	}

	// Load existing config to preserve other settings
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("yapılandırma okunamadı: %w", err)
	}

	// Update API key
	cfg.Firecrawl.APIKey = apiKey

	// Ensure default API URL is set
	if cfg.Firecrawl.APIURL == "" {
		cfg.Firecrawl.APIURL = "https://api.firecrawl.dev"
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("yapılandırma dosyası yazılamadı: %w", err)
	}

	fmt.Printf("API anahtarı kaydedildi: %s\n", config.ConfigPath())
	return nil
}
