package output

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kalchan12/goscrape/internal/scraper"
	"go.uber.org/zap"
)

type WriteConfig struct {
	OutputDir string
	Format    string
	SeedURL   string
	Depth     int
	Duration  time.Duration
}

type WriteResult struct {
	ResultsPath string
	CSVPath     string
	LogPath     string
	DownloadDir string
}

func WriteResults(results []scraper.ScrapeResult, cfg WriteConfig) (*WriteResult, error) {
	domain := extractDomainSimple(cfg.SeedURL)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	baseDir := filepath.Join(cfg.OutputDir, domain, timestamp)

	downloadDir := filepath.Join(baseDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	wr := &WriteResult{
		DownloadDir: downloadDir,
	}

	if cfg.Format == "json" || cfg.Format == "all" {
		data, err := FormatJSON(results, cfg.SeedURL, cfg.Duration)
		if err != nil {
			return nil, fmt.Errorf("json format error: %w", err)
		}
		path := filepath.Join(baseDir, "results.json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write json: %w", err)
		}
		wr.ResultsPath = path
		zap.L().Info("Wrote results", zap.String("path", path), zap.Int("pages", len(results)))
	}

	if cfg.Format == "csv" || cfg.Format == "all" {
		data, err := FormatCSV(results, cfg.SeedURL)
		if err != nil {
			return nil, fmt.Errorf("csv format error: %w", err)
		}
		path := filepath.Join(baseDir, "results.csv")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write csv: %w", err)
		}
		wr.CSVPath = path
	}

	fmt.Println(FormatText(results, cfg.SeedURL, cfg.Duration))

	return wr, nil
}

func extractDomainSimple(rawURL string) string {
	u, err := urlParse(rawURL)
	if err != nil {
		return "unknown"
	}
	return u.Hostname()
}

func urlParse(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
