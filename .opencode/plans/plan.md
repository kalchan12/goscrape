# GoScrape Implementation Plan

## Current Codebase State (as of 2026-07-13)

### Project Structure
```
goscrape/
├── main.go                          # Entry point (calls cmd.Execute())
├── go.mod / go.sum                  # Go 1.26 (NOTE: doesn't exist - latest is 1.23/1.24)
├── README.md                        # Basic docs
├── .gitignore
├── cmd/                             # Cobra commands (8 commands)
│   ├── root.go                      # Root command, config, logging, banner
│   ├── run.go                       # Main scrape command
│   ├── extract.go                   # Structured data extraction
│   ├── download.go                  # File downloading
│   ├── tree.go                      # Site structure discovery
│   ├── exam.go                      # Single exam extraction
│   ├── download_exams.go            # Batch exam downloader
│   ├── history.go                   # SQLite history management
│   └── interactive.go               # TUI mode
├── internal/
│   ├── scraper/                     # Core scraping engine
│   │   ├── scraper.go               # Colly-based crawler (292 lines)
│   │   ├── headless.go              # Chromedp rendering
│   │   └── ratelimiter.go           # Domain rate limiter (UNUSED)
│   ├── extractor/                   # HTML data extraction
│   │   └── extractor.go             # Meta, OG, JSON-LD, headings
│   ├── downloader/                  # File downloader
│   │   └── downloader.go            # Concurrent downloader
│   ├── storage/                     # SQLite history
│   │   └── db.go                    # GORM-based storage
│   ├── tree/                        # Site tree discovery
│   │   └── tree.go                  # Path tree builder
│   ├── examutil/                    # Exam-specific logic
│   │   └── examutil.go              # RSC payload parsing (237 lines)
│   ├── python/                      # Python bridge (TO REMOVE)
│   │   └── bridge.go                # Executes Python scripts
│   ├── output/                      # Result formatting
│   │   ├── writer.go                # File output
│   │   └── formatter.go             # JSON/CSV/Text formatting
│   └── tui/                         # Terminal UI
│       └── tui.go                   # Bubble Tea interactive mode
└── scripts/
    └── extract_questions.py         # Duplicate of examutil (TO REMOVE)
```

### Dependencies (go.mod)
- **Web**: `colly/v2`, `goquery`, `chromedp`
- **CLI/TUI**: `cobra`, `bubbletea`, `bubbles`, `lipgloss`
- **Data**: `sqlite` (glebarez), `gorm`, `gjson`
- **Utils**: `zap` (logging), `viper` (config), `rate` (rate limiting), `progressbar`
- **Indirect**: 60+ transitive dependencies

---

## Identified Issues

### Critical (Must Fix)
| ID | Issue | Location | Severity |
|----|-------|----------|----------|
| C1 | Go 1.26 doesn't exist | `go.mod:3` | Build fails |
| C2 | Zero tests | Entire codebase | No regression protection |
| C3 | Python bridge broken | `internal/python/bridge.go:60-75` | Undefined vars `outBytes`/`errBytes` |
| C4 | Duplicate exam extraction | `internal/examutil/examutil.go` + `scripts/extract_questions.py` | Maintenance burden |
| C5 | DomainRateLimiter unused | `internal/scraper/ratelimiter.go` | Rate limiting not enforced per-domain |

### High Priority
| ID | Issue | Location |
|----|-------|----------|
| H1 | Duplicate link extraction | `internal/scraper/scraper.go:185-188` (lines 150-158 also extract links) |
| H2 | Retries config unused | `scraper.Config.Retries` defined but never used |
| H3 | No context.Context support | Scraper, downloader, tree lack cancellation |
| H4 | No interfaces for core components | Hard to test/mock |
| H5 | User-Agent rotation configured but unused | `viper.SetDefault("rotate_agents", true)` |

### Medium Priority
| ID | Issue | Location |
|----|-------|----------|
| M1 | Downloader: no retry, resume, progress | `internal/downloader/downloader.go` |
| M2 | Storage: limited queries, no stats/export | `internal/storage/db.go` |
| M3 | Exam downloader: hardcoded to exitexamstudio.app | `cmd/download_exams.go`, `internal/examutil/examutil.go` |
| M4 | Config validation missing | `cmd/root.go` uses raw viper.Get* |
| M5 | robots.txt bypass default | `IgnoreRobots` default false but should be opt-in |

---

## Implementation Phases

### Phase 1: Critical Cleanup (Day 1) ✅ LOW RISK

| Task | Files | Action |
|------|-------|--------|
| 1.1 Remove Python bridge | `internal/python/bridge.go` | DELETE entire package |
| 1.2 Remove python_path config | `cmd/root.go:71` | Remove viper default |
| 1.3 Remove --python flag | `cmd/run.go:8,34,93-102,131` | Remove import, flag, bridge logic |
| 1.4 Remove duplicate Python script | `scripts/extract_questions.py` | DELETE file |
| 1.5 Fix duplicate link extraction | `internal/scraper/scraper.go:185-188` | Remove lines 185-188 (2nd OnHTML for a[href]) |
| 1.6 Note Go version | `go.mod:3` | Keep 1.26 per user request (add comment) |

**Verification**: `go build -o goscrape .` succeeds, all commands work

---

### Phase 2: Testing Foundation (Day 2-3) 🧪

#### 2.1 Test Infrastructure
- Add `github.com/stretchr/testify` to `go.mod`
- Create `internal/testutil/` with:
  - `server.go` - `httptest.Server` helpers for HTML fixtures
  - `fixtures/` - Sample HTML files (exam page, generic page, file listing)

#### 2.2 Unit Tests (Priority Order)
| Package | Test File | Coverage Target |
|---------|-----------|-----------------|
| `examutil` | `internal/examutil/examutil_test.go` | RSC parsing, question extraction, dept/exam link parsing |
| `extractor` | `internal/extractor/extractor_test.go` | Meta, OG, JSON-LD, headings, links, text blocks |
| `tree` | `internal/tree/tree_test.go` | Tree building, path collection, rendering |
| `storage` | `internal/storage/db_test.go` | CRUD, List, GetByID, Delete, Clear with temp DB |
| `downloader` | `internal/downloader/downloader_test.go` | Mock HTTP, retry, skip existing, size limits, MD5 |
| `scraper` | `internal/scraper/scraper_test.go` | Integration with test server: crawl, links, files, depth |

#### 2.3 Test Commands
```bash
go test ./... -v
go test ./... -cover
go test ./internal/examutil/... -v -run TestExtractQuestions
```

---

### Phase 3: Architecture & Core Fixes (Day 4-5) 🏗️

#### 3.1 Core Interfaces (New Files)
```go
// internal/scraper/interfaces.go
type Scraper interface {
    Run(ctx context.Context) ([]ScrapeResult, error)
    Results() []ScrapeResult
    PageCount() int
    ErrorCount() int
}

// internal/downloader/interfaces.go
type Downloader interface {
    Run(ctx context.Context, tasks []DownloadTask) []DownloadResult
}

// internal/extractor/interfaces.go
type Extractor interface {
    Extract(htmlContent, pageURL string) (*PageData, error)
}
```

#### 3.2 DomainRateLimiter Integration
- `internal/scraper/scraper.go`: Add `rateLimiter *DomainRateLimiter` field
- `NewScraper()`: Initialize with config rate limit params
- `collector.OnRequest()`: Call `rateLimiter.Wait(domain)` before visit
- Add config fields: `RateLimitRPS`, `RateLimitBurst`

#### 3.3 Context.Context Plumbing
| Function | Add Parameter |
|----------|---------------|
| `Scraper.Run()` | `ctx context.Context` |
| `Downloader.Run()` | `ctx context.Context` |
| `Downloader.downloadFile()` | `ctx context.Context` |
| `tree.SiteTree.Crawl()` | `ctx context.Context` |
| `examutil.FetchPage()` | `ctx context.Context` |
| `examutil.ExtractQuestionsFromURL()` | `ctx context.Context` |

#### 3.4 Retry with Exponential Backoff
```go
// internal/scraper/scraper.go
func (s *Scraper) visitWithRetry(ctx context.Context, url string, depth int) error {
    var lastErr error
    for attempt := 0; attempt <= s.config.Retries; attempt++ {
        if attempt > 0 {
            delay := time.Duration(attempt) * time.Second * 2 // 2s, 4s, 8s...
            select {
            case <-ctx.Done(): return ctx.Err()
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
```

#### 3.5 Config Validation
```go
// cmd/config.go (new)
type Config struct {
    URL          string
    Depth        int
    MaxPages     int
    Workers      int
    Delay        time.Duration
    Timeout      time.Duration
    Retries      int
    UserAgent    string
    RotateAgents bool
    // ... validation tags
}

func (c *Config) Validate() error {
    if c.URL == "" { return errors.New("url required") }
    if c.Depth < 0 { return errors.New("depth >= 0") }
    if c.Workers < 1 { return errors.New("workers >= 1") }
    // ...
}
```

---

### Phase 4: Feature Improvements (Day 6-7) ⭐

#### 4.1 Generic Exam Downloader Strategy
```go
// internal/examutil/strategy.go (new)
type ExamSiteStrategy interface {
    BaseURL() string
    DepartmentsPath() string
    DepartmentLinkPattern() *regexp.Regexp
    ExamLinkPattern() *regexp.Regexp
    ParseExamSlug(path string) (dept, year, examType string)
    ExtractQuestions(html string) ([]Question, string, error)
}

// internal/examutil/exitexamstudio.go (new)
type ExitExamStudioStrategy struct{}

func (s *ExitExamStudioStrategy) BaseURL() string { return "https://exitexamstudio.app" }
// ... implement all methods using existing logic from examutil.go
```

**Refactor**: `cmd/download_exams.go` accepts `--strategy` flag (default: `exitexamstudio`)

#### 4.2 Downloader Enhancements
| Feature | Implementation |
|---------|----------------|
| Retry with backoff | Add `MaxRetries`, `RetryDelay` to `Downloader` |
| Resume (Range header) | Check file size, send `Range: bytes=X-` |
| Progress callback | Add `ProgressFunc func(completed, total int)` to `Run()` |
| Per-domain rate limit | Reuse `DomainRateLimiter` in downloader |

#### 4.3 Storage Enhancements
```go
// internal/storage/db.go additions
func (d *DB) ListByURL(url string, limit int) ([]CrawlRecord, error)
func (d *DB) ListByDateRange(start, end time.Time, limit int) ([]CrawlRecord, error)
func (d *DB) ListByStatus(status string, limit int) ([]CrawlRecord, error)
func (d *DB) Stats() (totalCrawls, totalPages, totalFiles int, successRate float64)
func (d *DB) ExportJSON(w io.Writer) error
func (d *DB) ExportCSV(w io.Writer) error
func (d *DB) CleanupOlderThan(retention time.Duration) (int64, error)
```

#### 4.4 Expanded Config File (~/.goscrape.yaml)
```yaml
# Core
default_output: "~/goscrape-output"
default_delay: "1s"
default_workers: 3
default_format: "json"

# Rate limiting (NEW)
rate_limit:
  requests_per_second: 2
  burst: 5

# Politeness (NEW)
respect_robots: true
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
rotate_agents: true

# Exam defaults (NEW)
exam:
  base_url: "https://exitexamstudio.app"
  strategy: "exitexamstudio"
  output_dir: "~/exams"
  workers: 10

# Downloader (NEW)
downloader:
  max_retries: 3
  retry_delay: "2s"
  resume: true
  min_size: 0
  max_size: 0
```

---

### Phase 5: Polish & Documentation (Day 8) 📝

| Task | Command |
|------|---------|
| Run linter | `golangci-lint run ./...` |
| Run tests | `go test ./... -race -cover` |
| Build | `go build -o goscrape .` |
| Manual smoke test | Test all 8 commands |
| Update README | Document all commands, config, examples |
| Improve --help | Add examples to each command |

---

## File Change Summary

### Deleted Files
| File | Reason |
|------|--------|
| `internal/python/bridge.go` | Unused, broken, duplicates Go logic |
| `scripts/extract_questions.py` | Duplicate of examutil |

### Modified Files
| File | Changes |
|------|---------|
| `go.mod` | Add testify; note Go version |
| `cmd/root.go` | Remove python_path; add config struct |
| `cmd/run.go` | Remove python import/flag/logic; use config struct |
| `internal/scraper/scraper.go` | Remove duplicate OnHTML; add rate limiter; add context; add retry |
| `internal/scraper/ratelimiter.go` | Integrate into scraper |
| `internal/downloader/downloader.go` | Add retry, resume, progress, rate limit |
| `internal/storage/db.go` | Add queries, stats, export, cleanup |
| `internal/examutil/examutil.go` | Extract strategy interface; move site logic to new file |
| `cmd/download_exams.go` | Use generic strategy; add --strategy flag |
| `cmd/extract.go` | Use config struct |
| `cmd/download.go` | Use config struct |
| `cmd/tree.go` | Add context support |
| `cmd/exam.go` | Use config struct |
| `cmd/history.go` | Use new storage queries |
| `internal/tree/tree.go` | Add context support |
| `internal/tui/tui.go` | Add context support |

### New Files
| File | Purpose |
|------|---------|
| `cmd/config.go` | Config struct with validation |
| `internal/scraper/interfaces.go` | Scraper interface |
| `internal/downloader/interfaces.go` | Downloader interface |
| `internal/extractor/interfaces.go` | Extractor interface |
| `internal/examutil/strategy.go` | ExamSiteStrategy interface |
| `internal/examutil/exitexamstudio.go` | Default strategy implementation |
| `internal/testutil/server.go` | Test server helpers |
| `internal/testutil/fixtures/*.html` | Test HTML fixtures |
| `internal/examutil/examutil_test.go` | Unit tests |
| `internal/extractor/extractor_test.go` | Unit tests |
| `internal/tree/tree_test.go` | Unit tests |
| `internal/storage/db_test.go` | Unit tests |
| `internal/downloader/downloader_test.go` | Unit tests |
| `internal/scraper/scraper_test.go` | Integration tests |

---

## Testing Strategy

### Test Pyramid
```
        E2E (manual)
           ↑
    Integration (scraper_test.go with httptest.Server)
           ↑
    Unit (all *_test.go with mocks/fixtures)
```

### Test Data
- `testdata/exam_page.html` - Real Next.js RSC payload from exitexamstudio.app
- `testdata/generic_page.html` - Standard HTML with meta, OG, JSON-LD, files
- `testdata/file_listing.html` - Page with various file links (pdf, json, csv, etc.)

### CI Checks (Future)
```yaml
# .github/workflows/test.yml (not in scope now)
- go vet ./...
- go test ./... -race -coverprofile=coverage.out
- golangci-lint run ./...
- go build -o goscrape .
```

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking changes during refactor | Medium | High | Phase 1 cleanup first; tests in Phase 2 before refactor |
| Rate limiter breaks existing behavior | Low | Medium | Default to current colly limits; make configurable |
| Exam downloader strategy over-engineering | Medium | Low | Keep simple: single default strategy + interface |
| Context plumbing misses cancellation points | Medium | Medium | Audit all long-running loops; add tests with cancelled context |
| Go version 1.26 issue | High | Build fails | User wants to keep; add comment; test on 1.23/1.24 |

---

## Success Criteria

- [ ] `go build` succeeds
- [ ] All 8 commands work: `run`, `extract`, `download`, `tree`, `exam`, `download-exams`, `history`, `interactive`
- [ ] `go test ./...` passes with >70% coverage on core packages
- [ ] `golangci-lint run` passes with zero issues
- [ ] Exam downloader works with generic strategy
- [ ] Rate limiting enforced per-domain
- [ ] Retry with backoff on failures
- [ ] Context cancellation works (Ctrl+C stops cleanly)
- [ ] Config file supports all new options
- [ ] README updated with all features

---

## Next Steps

1. **Approve plan** - Confirm phases, priorities, any adjustments
2. **Start Phase 1** - Safe deletions, quick wins
3. **Iterate** - Each phase builds on previous; verify before proceeding

---

*Generated: 2026-07-13 | Codebase: goscrape v1.0.0*