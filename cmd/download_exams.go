package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kalchan12/goscrape/internal/examutil"
	"github.com/spf13/cobra"
)

var downloadExamsFlags struct {
	output   string
	workers  int
	baseURL  string
	strategy string
}

var downloadExamsCmd = &cobra.Command{
	Use:   "download-exams",
	Short: "Download all exam questions from every department",
	Long:  `Crawl all departments on the exam site, discover every exam, and save questions organized by department/year/type.`,
	Example: `  goscrape download-exams
  goscrape download-exams --output ./exams
  goscrape download-exams --workers 5
  goscrape download-exams --strategy exitexamstudio`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var s examutil.ExamSiteStrategy
		switch downloadExamsFlags.strategy {
		case "exitexamstudio":
			s = &examutil.ExitExamStudioStrategy{}
		default:
			return fmt.Errorf("unknown strategy: %s", downloadExamsFlags.strategy)
		}

		base := strings.TrimRight(downloadExamsFlags.baseURL, "/")
		if base == "" {
			base = strings.TrimRight(s.BaseURL(), "/")
		}
		outDir := downloadExamsFlags.output

		fmt.Printf("Fetching department list from %s/departments ...\n", base)

		body, err := examutil.FetchPage(base + "/departments")
		if err != nil {
			return fmt.Errorf("fetch departments: %w", err)
		}

		deptSlugs := examutil.ExtractDepartmentsWithStrategy(body, s)
		fmt.Printf("Found %d departments\n", len(deptSlugs))

		var allExamLinks []examutil.ExamLink
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, downloadExamsFlags.workers)

		for _, slug := range deptSlugs {
			wg.Add(1)
			go func(slug string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				url := base + "/departments/" + slug
				b, err := examutil.FetchPage(url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  SKIP %s: %v\n", slug, err)
					return
				}

				links := examutil.ExtractExamsFromDeptWithStrategy(b, s)
				mu.Lock()
				allExamLinks = append(allExamLinks, links...)
				mu.Unlock()
				fmt.Printf("  %s: %d exams\n", s, len(links))
			}(slug)
		}
		wg.Wait()

		fmt.Printf("\nTotal exams found: %d\n", len(allExamLinks))
		fmt.Println("Downloading questions...")

		type result struct {
			link examutil.ExamLink
			err  error
		}
		resultChan := make(chan result, len(allExamLinks))
		var dlWg sync.WaitGroup

		for _, link := range allExamLinks {
			dlWg.Add(1)
			go func(l examutil.ExamLink) {
				defer dlWg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				questions, title, err := examutil.ExtractQuestionsFromURL(l.URL)
				if err != nil {
					resultChan <- result{link: l, err: err}
					return
				}

				// Build output path
				dir := filepath.Join(outDir, l.Department, l.Year)
				if err := os.MkdirAll(dir, 0755); err != nil {
					resultChan <- result{link: l, err: fmt.Errorf("mkdir: %w", err)}
					return
				}

				data := examutil.ExamData{
					URL:       l.URL,
					Title:     title,
					Questions: questions,
					Count:     len(questions),
				}

				out, _ := json.MarshalIndent(data, "", "  ")
				filename := l.ExamType + ".json"
				fpath := filepath.Join(dir, filename)
				if err := os.WriteFile(fpath, out, 0644); err != nil {
					resultChan <- result{link: l, err: fmt.Errorf("write: %w", err)}
					return
				}

				resultChan <- result{link: l, err: nil}
			}(link)
		}

		go func() {
			dlWg.Wait()
			close(resultChan)
		}()

		var downloaded, failed int
		for r := range resultChan {
			if r.err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "  FAILED %s/%s: %v\n",
					r.link.Department, r.link.Year, r.err)
			} else {
				downloaded++
			}
		}

		fmt.Printf("\nDone! %d exams downloaded, %d failed\n", downloaded, failed)
		fmt.Printf("Output: %s/\n", outDir)
		fmt.Println("\nStructure: {department}/{year}/{type}.json")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadExamsCmd)

	downloadExamsCmd.Flags().StringVarP(&downloadExamsFlags.output, "output", "o", "./exams", "Output directory")
	downloadExamsCmd.Flags().IntVarP(&downloadExamsFlags.workers, "workers", "w", 5, "Concurrent workers")
	downloadExamsCmd.Flags().StringVar(&downloadExamsFlags.baseURL, "base", "", "Base URL (default from strategy)")
	downloadExamsCmd.Flags().StringVar(&downloadExamsFlags.strategy, "strategy", "exitexamstudio", "Exam site strategy")
}
