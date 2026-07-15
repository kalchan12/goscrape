# GoScrape

A fast terminal web scraper built in Go with CLI and TUI modes. Extracts content, downloads files, discovers site structure, and batch-downloads exam questions from Next.js sites.

## Install

```bash
git clone https://github.com/kalchan12/goscrape
cd goscrape
go build -o goscrape .
./goscrape
```

Requires Go 1.26.

## Commands

| Command | Description |
|---------|-------------|
| `goscrape run --url <URL>` | Scrape a website with depth, workers, selectors, JS rendering |
| `goscrape extract --url <URL>` | Extract meta, OG, JSON-LD structured data |
| `goscrape download --url <URL> --types pdf` | Download files by type with concurrency |
| `goscrape tree --url <URL>` | Discover site directory structure |
| `goscrape exam --url <URL>` | Extract exam questions from RSC payload |
| `goscrape download-exams` | Batch-download all exams from all departments |
| `goscrape history [list|show|delete|clear|stats|export|cleanup]` | View and manage crawl history |
| `goscrape interactive` | Launch TUI mode |
| `goscrape config init` | Generate default config file |
| `goscrape config show` | Show current configuration |
| `goscrape version` | Print version information |
| `goscrape completion bash|zsh|fish|powershell` | Generate shell completion |

## Examples

```bash
# Basic scrape
goscrape run --url https://example.com

# Scrape with depth and workers
goscrape run --url https://example.com --depth 3 --workers 5

# Extract structured data
goscrape extract --url https://example.com --format json

# Download PDFs
goscrape download --url https://example.com --types pdf --output ./files

# Discover site structure
goscrape tree --url https://example.com --depth 3

# Extract exam questions
goscrape exam --url "https://exitexamstudio.app/exam/computer-science__2016__regular" --pretty

# Batch download all exams
goscrape download-exams -o ./all-exams

# View history
goscrape history list
goscrape history stats
goscrape history export json
goscrape history cleanup 30

# Interactive mode
goscrape interactive
```

## Test

```bash
go test ./... -v
go test ./... -race
```
