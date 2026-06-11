package cmd

import (
	"github.com/kalchan12/goscrape/internal/tui"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"tui", "i"},
	Short:   "Launch interactive TUI mode",
	Long:    `Start the terminal UI for interactive web scraping with real-time feedback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
