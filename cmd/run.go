package cmd

import (
	"context"
	"time"

	"github.com/kalchan12/goscrape/internal/output"
	"github.com/kalchan12/goscrape/internal/scraper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runFlags struct {
	url          string
	depth        int
	maxPages     int
	workers      int
	delay        string
	timeout      string
	retries      int
	userAgent    string
	rotateAgents bool
	allowDomains []string
	ignoreRobots bool
	js           bool
	selector     string
	attr         string
	format       string
	output       string
	download     []string
	downloadAll  bool
	flatten      bool
	noProgress   bool
	stdin        bool
}

var runCmd = &cobra.Command{
	Use:   "run --url <URL>",
	Short: "Crawl a website and extract content",
	Long: `Start a web scraping session. Crawls pages from the seed URL, extracts content, and optionally downloads files.

Supports depth-limited crawling, concurrent workers, custom CSS selectors, headless JS rendering, and file download.`,
	Example: `  goscrape run --url https://example.com
  goscrape run --url https://example.com --depth 3 --workers 5
  goscrape run --url https://example.com --js
  goscrape run --url https://example.com --selector "article" --format json
  goscrape run --url https://example.com --delay 2s --retries 3
  goscrape run --url https://example.com --download pdf,json --output ./downloads`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runFlags.url == "" && !runFlags.stdin {
			return cmd.Help()
		}

		delay, _ := time.ParseDuration(runFlags.delay)
		timeout, _ := time.ParseDuration(runFlags.timeout)

		if runFlags.userAgent == "" {
			runFlags.userAgent = viper.GetString("user_agent")
		}

		cfg := scraper.Config{
			URL:          runFlags.url,
			Depth:        runFlags.depth,
			MaxPages:     runFlags.maxPages,
			Workers:      runFlags.workers,
			Delay:        delay,
			Timeout:      timeout,
			Retries:      runFlags.retries,
			UserAgent:    runFlags.userAgent,
			RotateAgents: runFlags.rotateAgents,
			UserAgents:   viper.GetStringSlice("user_agents"),
			AllowDomains: runFlags.allowDomains,
			IgnoreRobots: runFlags.ignoreRobots,
			Selector:     runFlags.selector,
			Attr:         runFlags.attr,
			JS:           runFlags.js,
			DownloadTypes: runFlags.download,
			DownloadAll:  runFlags.downloadAll,
		}

		s := scraper.NewScraper(cfg)
		results, err := s.Run(context.Background())
		if err != nil {
			return err
		}

		wrCfg := output.WriteConfig{
			OutputDir: runFlags.output,
			Format:    runFlags.format,
			SeedURL:   runFlags.url,
			Depth:     runFlags.depth,
			Duration:  time.Since(s.StartTime()),
		}

		_, err = output.WriteResults(results, wrCfg)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runFlags.url, "url", "", "Target URL to scrape")
	runCmd.Flags().IntVar(&runFlags.depth, "depth", 1, "Max crawl depth")
	runCmd.Flags().IntVar(&runFlags.maxPages, "max-pages", 50, "Max pages to visit")
	runCmd.Flags().IntVar(&runFlags.workers, "workers", 3, "Concurrent workers")
	runCmd.Flags().StringVar(&runFlags.delay, "delay", "1s", "Delay between requests")
	runCmd.Flags().StringVar(&runFlags.timeout, "timeout", "10s", "Request timeout")
	runCmd.Flags().IntVar(&runFlags.retries, "retries", 2, "Retry count")
	runCmd.Flags().StringVar(&runFlags.userAgent, "user-agent", "", "Custom User-Agent")
	runCmd.Flags().BoolVar(&runFlags.rotateAgents, "rotate-agents", false, "Rotate User-Agents")
	runCmd.Flags().StringArrayVar(&runFlags.allowDomains, "allow-domain", nil, "Additional domains")
	runCmd.Flags().BoolVar(&runFlags.ignoreRobots, "ignore-robots", false, "Bypass robots.txt")
	runCmd.Flags().BoolVar(&runFlags.js, "js", false, "Use headless Chrome")
	runCmd.Flags().StringVar(&runFlags.selector, "selector", "", "CSS selector to extract")
	runCmd.Flags().StringVar(&runFlags.attr, "attr", "", "HTML attribute to extract")
	runCmd.Flags().StringVar(&runFlags.format, "format", "json", "Output format: json | csv | text")
	runCmd.Flags().StringVar(&runFlags.output, "output", "", "Output directory")
	runCmd.Flags().StringArrayVar(&runFlags.download, "download", nil, "File types to download")
	runCmd.Flags().BoolVar(&runFlags.downloadAll, "download-all", false, "Download all file types")
	runCmd.Flags().BoolVar(&runFlags.flatten, "flatten", false, "Flatten download dirs")
	runCmd.Flags().BoolVar(&runFlags.noProgress, "no-progress", false, "Disable progress bar")
	runCmd.Flags().BoolVar(&runFlags.stdin, "stdin", false, "Read URLs from stdin")
}
