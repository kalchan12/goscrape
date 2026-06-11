package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kalchan12/goscrape/internal/scraper"
)

type ResultSet struct {
	URL        string                 `json:"seed_url"`
	Timestamp  string                 `json:"timestamp"`
	TotalPages int                    `json:"total_pages"`
	TotalFiles int                    `json:"total_files"`
	Duration   string                 `json:"duration"`
	Results    []scraper.ScrapeResult `json:"results"`
}

func FormatJSON(results []scraper.ScrapeResult, seedURL string, duration time.Duration) ([]byte, error) {
	rs := ResultSet{
		URL:        seedURL,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		TotalPages: len(results),
		Duration:   duration.Round(time.Millisecond).String(),
		Results:    results,
	}

	totalFiles := 0
	for _, r := range results {
		totalFiles += len(r.Files)
	}
	rs.TotalFiles = totalFiles

	return json.MarshalIndent(rs, "", "  ")
}

func FormatCSV(results []scraper.ScrapeResult, seedURL string) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	writer.Write([]string{"url", "title", "status_code", "links", "files"})
	for _, r := range results {
		writer.Write([]string{
			r.URL,
			r.Title,
			fmt.Sprintf("%d", r.Status),
			strings.Join(r.Links, "; "),
			formatFileRefs(r.Files),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func FormatText(results []scraper.ScrapeResult, seedURL string, duration time.Duration) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("GoScrape Results\n"))
	sb.WriteString(fmt.Sprintf("Seed URL: %s\n", seedURL))
	sb.WriteString(fmt.Sprintf("Pages: %d\n", len(results)))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", duration.Round(time.Millisecond)))
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, r.URL))
		sb.WriteString(fmt.Sprintf("    Title: %s\n", r.Title))
		sb.WriteString(fmt.Sprintf("    Status: %d\n", r.Status))
		if len(r.Links) > 0 {
			sb.WriteString(fmt.Sprintf("    Links: %d\n", len(r.Links)))
		}
		if len(r.Files) > 0 {
			sb.WriteString(fmt.Sprintf("    Files: %d\n", len(r.Files)))
			for _, f := range r.Files {
				sb.WriteString(fmt.Sprintf("      - %s (%s)\n", f.Filename, f.Type))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatFileRefs(refs []scraper.FileRef) string {
	parts := make([]string, len(refs))
	for i, f := range refs {
		parts[i] = f.Filename
	}
	return strings.Join(parts, "; ")
}
