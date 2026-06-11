# GoScrape

A fast terminal web scraper built in Go with CLI and TUI modes. Extracts content, downloads files, discovers site structure, and batch-downloads exam questions from Next.js sites.

## Install

```bash
git clone https://github.com/kalchan12/goscrape
cd goscrape
./goscrape
```

## Commands

| Command | Description |
|---------|-------------|
| `goscrape run --url <URL>` | Scrape a website |
| `goscrape extract --url <URL>` | Extract meta, OG, JSON-LD |
| `goscrape download --url <URL> --types pdf` | Download files by type |
| `goscrape tree --url <URL>` | Discover site directory structure |
| `goscrape exam --url <URL> --pretty` | Extract exam questions from RSC payload |
| `goscrape download-exams` | Batch-download all exams from all departments |
| `goscrape history list` | View crawl history |
| `goscrape interactive` | Launch TUI mode |

## Batch Download Exams

```bash
goscrape download-exams -o ./exams -w 10
```

Downloads all exams organized as `{department}/{year}/{type}.json`.

## Examples

```bash
goscrape run --url https://example.com
goscrape extract --url https://example.com --format json
goscrape tree --url https://exitexamstudio.app/ --depth 2
goscrape exam --url "https://exitexamstudio.app/exam/computer-science__2016__regular" --output exam.json --pretty
goscrape download-exams -o ./all-exams
```

## Build

Requires Go 1.26.

```bash
go build -o goscrape .
```
