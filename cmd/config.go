package cmd

import (
	"errors"
	"strings"
	"time"
)

type ScrapeConfig struct {
	URL          string
	Depth        int
	MaxPages     int
	Workers      int
	Delay        time.Duration
	Timeout      time.Duration
	Retries      int
	UserAgent    string
	RotateAgents bool
	AllowDomains []string
	IgnoreRobots bool
	Selector     string
	Attr         string
	JS           bool
	Download     []string
	DownloadAll  bool
	Format       string
}

func (c *ScrapeConfig) Validate() error {
	var errs []string

	if c.URL == "" {
		errs = append(errs, "URL is required")
	}

	if c.Workers < 1 {
		errs = append(errs, "workers must be >= 1")
	}

	if c.Depth < 0 {
		errs = append(errs, "depth must be >= 0")
	}

	if c.MaxPages < 1 {
		errs = append(errs, "max-pages must be >= 1")
	}

	if c.Retries < 0 {
		errs = append(errs, "retries must be >= 0")
	}

	if c.Timeout < 0 {
		errs = append(errs, "timeout must be >= 0")
	}

	if c.Delay < 0 {
		errs = append(errs, "delay must be >= 0")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed: " + strings.Join(errs, "; "))
	}

	return nil
}
