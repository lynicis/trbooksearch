package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersionInfo(version string) {
	appVersion = version
}

var rootCmd = &cobra.Command{
	Use:   "trbooksearch",
	Short: "Türk kitap sitelerinde fiyat karşılaştırma aracı",
	Long: `trbooksearch - Türkiye'deki kitap sitelerinde arama yaparak
ikinci el ve yeni kitap fiyatlarını karşılaştırmanızı sağlar.

Desteklenen siteler:
  İkinci El: nadirkitap.com
  Yeni:      kitapyurdu.com, trendyol.com, hepsiburada.com, amazon.com.tr

Firecrawl ile ek siteler (--firecrawl):
  İkinci El: letgo.com, dolap.com, gardrops.com

Firecrawl API anahtarını ayarlamak için:
  trbooksearch set-api-key <anahtar>`,
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Sürüm bilgilerini göster",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("trbooksearch %s\n", appVersion)
		},
	})
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
