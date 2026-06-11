package cmd

import (
	"fmt"
	"net/url"

	"github.com/kalchan12/goscrape/internal/tree"
	"github.com/spf13/cobra"
)

var treeFlags struct {
	url   string
	depth int
}

var treeCmd = &cobra.Command{
	Use:   "tree --url <URL>",
	Short: "Discover and display site directory structure",
	Long:  `Crawl a website and show all discovered URL paths as a directory tree.`,
	Example: `  goscrape tree --url https://exitexamstudio.app/
  goscrape tree --url https://exitexamstudio.app/ --depth 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if treeFlags.url == "" {
			return cmd.Help()
		}

		parsed, _ := url.Parse(treeFlags.url)
		seedPath := parsed.Path
		if seedPath == "" {
			seedPath = "/"
		}

		t := tree.New(seedPath)
		if err := t.Crawl(treeFlags.url, treeFlags.depth); err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}

		paths := t.Paths()
		fmt.Printf("Discovered %d paths on %s\n\n", len(paths), treeFlags.url)
		fmt.Print(t.Render())

		if len(paths) > 0 {
			u, _ := url.Parse(treeFlags.url)
			baseURL := u.Scheme + "://" + u.Host
			fmt.Println("\nTo scrape a specific directory:")
			fmt.Printf("  goscrape run --url %s%s\n", baseURL, paths[0])
			fmt.Printf("  goscrape download --url %s%s --types pdf\n", baseURL, paths[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)

	treeCmd.Flags().StringVar(&treeFlags.url, "url", "", "Target URL (required)")
	treeCmd.Flags().IntVar(&treeFlags.depth, "depth", 2, "Max crawl depth")
}
