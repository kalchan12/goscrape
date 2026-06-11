package cmd

import (
	"fmt"
	"strconv"

	"github.com/kalchan12/goscrape/internal/storage"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history [list|show|delete|clear]",
	Short: "View and manage crawl history",
	Long:  `List, inspect, or delete previous crawl sessions stored in SQLite.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent crawls",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.New("")
		if err != nil {
			return err
		}
		defer db.Close()

		records, err := db.List(20)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			fmt.Println("No crawl history found.")
			return nil
		}
		fmt.Printf("%-4s  %-40s  %-6s  %-4s  %-10s\n", "ID", "URL", "Status", "Hits", "Date")
		for _, r := range records {
			fmt.Printf("%-4d  %-40s  %-6s  %-4d  %s\n",
				r.ID, truncate(r.URL, 40), r.Status, r.PagesHit,
				r.CreatedAt.Format("2006-01-02"))
		}
		return nil
	},
}

var historyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show crawl details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid id: %s", args[0])
		}

		db, err := storage.New("")
		if err != nil {
			return err
		}
		defer db.Close()

		record, err := db.GetByID(uint(id))
		if err != nil {
			return fmt.Errorf("record not found: %w", err)
		}

		fmt.Printf("ID:        %d\n", record.ID)
		fmt.Printf("URL:       %s\n", record.URL)
		fmt.Printf("Depth:     %d\n", record.Depth)
		fmt.Printf("MaxPages:  %d\n", record.MaxPages)
		fmt.Printf("PagesHit:  %d\n", record.PagesHit)
		fmt.Printf("Files:     %d\n", record.Files)
		fmt.Printf("Status:    %s\n", record.Status)
		fmt.Printf("Created:   %s\n", record.CreatedAt.Format(timeFormat))
		fmt.Printf("Updated:   %s\n", record.UpdatedAt.Format(timeFormat))
		return nil
	},
}

var historyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a crawl record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid id: %s", args[0])
		}

		db, err := storage.New("")
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.Delete(uint(id)); err != nil {
			return err
		}
		fmt.Printf("Deleted crawl %d\n", id)
		return nil
	},
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all crawl history",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.New("")
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.Clear(); err != nil {
			return err
		}
		fmt.Println("Cleared all crawl history.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

const timeFormat = "2006-01-02 15:04:05"
