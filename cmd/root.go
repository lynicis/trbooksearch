package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var rootCmd = &cobra.Command{
	Use:   "trbooksearch",
	Short: "Türk kitap sitelerinde fiyat karşılaştırma aracı",
	Long: `trbooksearch - Türkiye'deki kitap sitelerinde arama yaparak
ikinci el ve yeni kitap fiyatlarını karşılaştırmanızı sağlar.

Desteklenen siteler:
  İkinci El: nadirkitap.com
  Yeni:      kitapyurdu.com, trendyol.com, hepsiburada.com, amazon.com.tr`,
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Sürüm bilgilerini göster",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("trbooksearch %s (commit: %s, built: %s)\n", appVersion, appCommit, appDate)
		},
	})
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
