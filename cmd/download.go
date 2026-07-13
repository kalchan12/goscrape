package cmd

import (
	"fmt"
	"os"

	"github.com/kalchan12/goscrape/internal/downloader"
	"github.com/kalchan12/goscrape/internal/scraper"
	"github.com/spf13/cobra"
	"context"
)

var downloadFlags struct {
	url       string
	types     []string
	depth     int
	output    string
	workers   int
	overwrite bool
	dryRun    bool
	minSize   string
	maxSize   string
}

var downloadCmd = &cobra.Command{
	Use:   "download --url <URL> --types pdf,json",
	Short: "Download files of specified types from a URL",
	Long:  `Find and download files (PDF, JSON, CSV, etc.) linked from a webpage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if downloadFlags.url == "" {
			return cmd.Help()
		}

		cfg := scraper.Config{
			URL:           downloadFlags.url,
			Depth:         downloadFlags.depth,
			MaxPages:      50,
			Workers:       3,
			DownloadTypes: downloadFlags.types,
			DownloadAll:   false,
		}

		s := scraper.NewScraper(cfg)
		results, err := s.Run(context.Background())
		if err != nil {
			return err
		}

		var tasks []downloader.DownloadTask
		for _, r := range results {
			for _, f := range r.Files {
				tasks = append(tasks, downloader.DownloadTask{
					URL:      f.URL,
					Filename: f.Filename,
					Filetype: f.Type,
					DestDir:  downloadFlags.output,
				})
			}
		}

		if downloadFlags.dryRun {
			fmt.Printf("Found %d files:\n", len(tasks))
			for _, t := range tasks {
				fmt.Printf("  [%s] %s\n", t.Filetype, t.URL)
			}
			return nil
		}

		if len(tasks) == 0 {
			fmt.Println("No files found to download.")
			return nil
		}

		dl := downloader.NewDownloader(downloadFlags.workers, downloadFlags.overwrite, 0, 0)
		results2 := dl.Run(tasks)

		var downloaded, skipped, failed int
		for _, r := range results2 {
			if r.Error != nil {
				failed++
				fmt.Fprintf(os.Stderr, "  FAILED: %s - %s\n", r.Task.URL, r.Error)
			} else if r.Skipped {
				skipped++
			} else {
				downloaded++
			}
		}

		fmt.Printf("\nDownload complete: %d downloaded, %d skipped, %d failed\n",
			downloaded, skipped, failed)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVar(&downloadFlags.url, "url", "", "Target URL (required)")
	downloadCmd.Flags().StringArrayVar(&downloadFlags.types, "types", nil, "File types to download (required)")
	downloadCmd.Flags().IntVar(&downloadFlags.depth, "depth", 1, "Max depth to search")
	downloadCmd.Flags().StringVar(&downloadFlags.output, "output", "./downloads", "Output directory")
	downloadCmd.Flags().IntVar(&downloadFlags.workers, "workers", 5, "Download concurrency")
	downloadCmd.Flags().BoolVar(&downloadFlags.overwrite, "overwrite", false, "Re-download existing files")
	downloadCmd.Flags().BoolVar(&downloadFlags.dryRun, "dry-run", false, "List files without downloading")
	downloadCmd.Flags().StringVar(&downloadFlags.minSize, "min-size", "", "Skip files smaller than this")
	downloadCmd.Flags().StringVar(&downloadFlags.maxSize, "max-size", "", "Skip files larger than this")
}
