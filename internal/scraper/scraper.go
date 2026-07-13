package scraper

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

type ScrapeResult struct {
	URL       string            `json:"url"`
	Title     string            `json:"title"`
	Status    int               `json:"status_code"`
	Text      string            `json:"text"`
	HTML      string            `json:"-"`
	Links     []string          `json:"links"`
	Files     []FileRef         `json:"files"`
	Data      map[string]string `json:"extracted_data"`
	Error     string            `json:"error,omitempty"`
}

type FileRef struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
}

type Config struct {
	URL            string
	Depth          int
	MaxPages       int
	Workers        int
	Delay          time.Duration
	Timeout        time.Duration
	Retries        int
	UserAgent      string
	RotateAgents   bool
	AllowDomains   []string
	IgnoreRobots   bool
	Selector       string
	Attr           string
	JS             bool
	DownloadTypes  []string
	DownloadAll    bool
	Verbose        bool
	RateLimitRPS   float64
	RateLimitBurst int
}

type Scraper struct {
	config       Config
	collector    *colly.Collector
	results      []ScrapeResult
	pagesVisited int
	mu           sync.Mutex
	done         chan bool
	errors       int
	startTime    time.Time
	fileChan     chan FileRef
	rateLimiter  *DomainRateLimiter
}

func NewScraper(cfg Config) *Scraper {
	s := &Scraper{
		config:   cfg,
		results:  make([]ScrapeResult, 0, cfg.MaxPages),
		done:     make(chan bool),
		fileChan: make(chan FileRef, 100),
	}

	rps := cfg.RateLimitRPS
	if rps <= 0 {
		rps = 2
	}
	burst := cfg.RateLimitBurst
	if burst <= 0 {
		burst = 3
	}
	s.rateLimiter = NewDomainRateLimiter(rps, burst)

	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(cfg.Depth),
		colly.UserAgent(cfg.UserAgent),
	)

	if cfg.IgnoreRobots {
		c.IgnoreRobotsTxt = true
	}

	if len(cfg.AllowDomains) > 0 {
		allDomains := append(cfg.AllowDomains, extractDomain(cfg.URL))
		c.AllowedDomains = allDomains
	}

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       cfg.Delay,
		Parallelism: cfg.Workers,
	})

	c.SetRequestTimeout(cfg.Timeout)

	c.OnRequest(func(r *colly.Request) {
		domain := extractDomain(r.URL.String())
		if err := s.rateLimiter.Wait(domain); err != nil {
			return
		}

		zap.L().Debug("Fetching",
			zap.String("url", r.URL.String()),
			zap.Int("depth", r.Depth),
		)
	})

	c.OnResponse(func(r *colly.Response) {
		s.mu.Lock()
		s.pagesVisited++
		visited := s.pagesVisited
		s.mu.Unlock()

		if visited > cfg.MaxPages {
			return
		}

		zap.L().Debug("Response received",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Int("bytes", len(r.Body)),
		)
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		doc := e.DOM

		title := doc.Find("title").First().Text()
		bodyText := strings.TrimSpace(doc.Find("body").Text())

		result := ScrapeResult{
			URL:    e.Request.URL.String(),
			Title:  title,
			Text:   bodyText,
			Status: e.Response.StatusCode,
			HTML:   string(e.Response.Body),
			Links:  make([]string, 0),
			Files:  make([]FileRef, 0),
		}

		if cfg.Selector != "" {
			result.Data = make(map[string]string)
			doc.Find(cfg.Selector).Each(func(i int, sel *goquery.Selection) {
				if cfg.Attr != "" {
					val, _ := sel.Attr(cfg.Attr)
					result.Data[fmt.Sprintf("match_%d", i)] = val
				} else {
					result.Data[fmt.Sprintf("match_%d", i)] = strings.TrimSpace(sel.Text())
				}
			})
		}

		doc.Find("a[href]").Each(func(i int, sel *goquery.Selection) {
			href, _ := sel.Attr("href")
			if href != "" {
				absURL := e.Request.AbsoluteURL(href)
				if absURL != "" {
					result.Links = append(result.Links, absURL)
				}
			}
		})

		doc.Find("a[href], link[href]").Each(func(i int, sel *goquery.Selection) {
			href, _ := sel.Attr("href")
			if href == "" {
				return
			}
			absURL := e.Request.AbsoluteURL(href)
			if absURL == "" {
				return
			}
			ext := strings.ToLower(filepath.Ext(absURL))
			ft := fileTypeFromExt(ext)
			if ft != "" && (cfg.DownloadAll || contains(cfg.DownloadTypes, ft)) {
				result.Files = append(result.Files, FileRef{
					URL:      absURL,
					Filename: filepath.Base(absURL),
					Type:     ft,
				})
			}
		})

		s.mu.Lock()
		s.results = append(s.results, result)
		s.mu.Unlock()
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		e.Request.Visit(link)
	})

	c.OnError(func(r *colly.Response, err error) {
		s.mu.Lock()
		s.errors++
		s.mu.Unlock()
		zap.L().Warn("Request error",
			zap.String("url", r.Request.URL.String()),
			zap.Error(err),
		)
	})

	s.collector = c
	return s
}

func (s *Scraper) Run(ctx context.Context) ([]ScrapeResult, error) {
	s.startTime = time.Now()
	zap.L().Info("Starting scrape",
		zap.String("url", s.config.URL),
		zap.Int("depth", s.config.Depth),
		zap.Int("max_pages", s.config.MaxPages),
	)

	err := s.visitWithRetry(ctx, s.config.URL, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to start scrape: %w", err)
	}

	s.collector.Wait()
	close(s.done)

	elapsed := time.Since(s.startTime)
	zap.L().Info("Scrape completed",
		zap.Int("pages", len(s.results)),
		zap.Int("errors", s.errors),
		zap.Duration("duration", elapsed),
	)

	return s.results, nil
}

func (s *Scraper) visitWithRetry(ctx context.Context, url string, depth int) error {
	var lastErr error
	maxRetries := s.config.Retries
	if maxRetries < 0 {
		maxRetries = 0
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * time.Second * 2
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := s.collector.Visit(url); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (s *Scraper) Results() []ScrapeResult {
	return s.results
}

func (s *Scraper) PageCount() int {
	return len(s.results)
}

func (s *Scraper) ErrorCount() int {
	return s.errors
}

func (s *Scraper) FileChan() <-chan FileRef {
	return s.fileChan
}

func (s *Scraper) StartTime() time.Time {
	return s.startTime
}

func (s *Scraper) Elapsed() time.Duration {
	return time.Since(s.startTime)
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func fileTypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".pdf":
		return "pdf"
	case ".json":
		return "json"
	case ".csv":
		return "csv"
	case ".xlsx", ".xls":
		return "xlsx"
	case ".docx", ".doc":
		return "docx"
	case ".zip", ".tar", ".gz", ".bz2":
		return "zip"
	case ".png":
		return "png"
	case ".jpg", ".jpeg", ".webp":
		return "jpg"
	default:
		return ""
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
