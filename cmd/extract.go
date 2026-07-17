package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kalchan12/goscrape/internal/extractor"
	"github.com/kalchan12/goscrape/internal/scraper"
	"github.com/spf13/cobra"
	"context"
)

var extractFlags struct {
	url      string
	selector string
	format   string
	depth    int
}

var extractCmd = &cobra.Command{
	Use:   "extract --url <URL>",
	Short: "Extract structured data (JSON-LD, OG, meta) from a page",
	Long: `Crawl a URL and extract JSON-LD structured data, Open Graph tags, meta tags, headings, and links.

Outputs formatted results to stdout or a file.`,
	Example: `  goscrape extract --url https://example.com
  goscrape extract --url https://example.com --format json
  goscrape extract --url https://example.com --selector "article" --depth 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if extractFlags.url == "" {
			return cmd.Help()
		}

		cfg := ScrapeConfig{
			URL:      extractFlags.url,
			Depth:    extractFlags.depth,
			MaxPages: 10,
			Workers:  2,
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		s := scraper.NewScraper(scraper.Config{
			URL:      cfg.URL,
			Depth:    cfg.Depth,
			MaxPages: cfg.MaxPages,
			Workers:  cfg.Workers,
		})
		results, err := s.Run(context.Background())
		if err != nil {
			return err
		}

		var allData []*extractor.PageData
		for _, r := range results {
			data, err := extractor.Extract(r.HTML, r.URL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "extract error for %s: %v\n", r.URL, err)
				continue
			}
			allData = append(allData, data)
		}

		switch extractFlags.format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(allData)
		case "csv":
			fmt.Println("url,title,description,og_title")
			for _, d := range allData {
				fmt.Printf("%q,%q,%q,%q\n",
					d.URL, d.Title, d.Meta["description"], d.OpenGraph["og:title"])
			}
		default:
			for _, d := range allData {
				fmt.Printf("=== %s ===\n", d.URL)
				fmt.Printf("Title: %s\n", d.Title)
				fmt.Printf("Description: %s\n", d.Meta["description"])
				fmt.Printf("OG Title: %s\n", d.OpenGraph["og:title"])
				fmt.Printf("Headings: %d | Links: %d | TextBlocks: %d\n",
					len(d.Headings), len(d.Links), len(d.TextBlocks))
				if len(d.JSONLD) > 0 {
					fmt.Printf("JSON-LD items: %d\n", len(d.JSONLD))
				}
				fmt.Println()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringVar(&extractFlags.url, "url", "", "Target URL (required)")
	extractCmd.Flags().StringVar(&extractFlags.selector, "selector", "", "CSS selector for targeted extraction")
	extractCmd.Flags().StringVar(&extractFlags.format, "format", "text", "Output format: json | csv | text")
	extractCmd.Flags().IntVar(&extractFlags.depth, "depth", 1, "Max crawl depth")
}
