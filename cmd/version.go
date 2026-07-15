package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GoScrape %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Date: %s\n", Date)
		fmt.Printf("Go: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}