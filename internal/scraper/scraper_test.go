package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(pages map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
}

func TestNewScraper(t *testing.T) {
	cfg := Config{
		URL:      "https://example.com",
		Depth:    1,
		MaxPages: 10,
		Workers:  2,
	}
	s := NewScraper(cfg)
	assert.NotNil(t, s)
	assert.NotNil(t, s.collector)
	assert.NotNil(t, s.results)
	assert.Equal(t, 10, cap(s.results))
}

func TestScraperRunSinglePage(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><head><title>Home</title></head><body><p>Hello</p></body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    1,
		MaxPages: 10,
		Workers:  2,
		Timeout:  5 * time.Second,
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, srv.URL+"/", results[0].URL)
	assert.Equal(t, "Home", results[0].Title)
	assert.Equal(t, 200, results[0].Status)
	assert.Contains(t, results[0].Text, "Hello")
}

func TestScraperRunWithLinks(t *testing.T) {
	srv := testServer(map[string]string{
		"/":      `<html><body><a href="/page1">Page 1</a><a href="/page2">Page 2</a></body></html>`,
		"/page1": `<html><head><title>Page 1</title></head><body><p>Content 1</p></body></html>`,
		"/page2": `<html><head><title>Page 2</title></head><body><p>Content 2</p></body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    2,
		MaxPages: 10,
		Workers:  2,
		Timeout:  5 * time.Second,
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestScraperWithFileDetection(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><body>
			<a href="/doc.pdf">PDF</a>
			<a href="/data.json">JSON</a>
			<a href="/about.html">About</a>
		</body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:           srv.URL + "/",
		Depth:         1,
		MaxPages:      10,
		Workers:       1,
		Timeout:       5 * time.Second,
		DownloadTypes: []string{"pdf", "json"},
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Files, 2)
}

func TestScraperWithDownloadAll(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><body>
			<a href="/doc.pdf">PDF</a>
			<a href="/data.json">JSON</a>
			<a href="/about.html">About</a>
		</body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:         srv.URL + "/",
		Depth:       1,
		MaxPages:    10,
		Workers:     1,
		Timeout:     5 * time.Second,
		DownloadAll: true,
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Files, 2)
}

func TestScraperWithSelector(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><body>
			<div class="item">Item 1</div>
			<div class="item">Item 2</div>
		</body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    1,
		MaxPages: 10,
		Workers:  1,
		Timeout:  5 * time.Second,
		Selector: ".item",
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, results[0].Data)
	assert.Equal(t, "Item 1", results[0].Data["match_0"])
	assert.Equal(t, "Item 2", results[0].Data["match_1"])
}

func TestScraperWithSelectorAndAttr(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><body>
			<a href="/page1" class="link">Link 1</a>
			<a href="/page2" class="link">Link 2</a>
		</body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    1,
		MaxPages: 10,
		Workers:  1,
		Timeout:  5 * time.Second,
		Selector: ".link",
		Attr:     "href",
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Data["match_0"], "/page1")
	assert.Contains(t, results[0].Data["match_1"], "/page2")
}

func TestScraperMaxPagesConfig(t *testing.T) {
	// Verify the MaxPages config field is passed through correctly
	cfg := Config{
		URL:      "https://example.com",
		Depth:    1,
		MaxPages: 5,
		Workers:  1,
	}
	s := NewScraper(cfg)
	assert.Equal(t, 5, cap(s.results))
}

func TestScraperWithErrors(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><body><a href="/missing">Missing</a></body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    2,
		MaxPages: 10,
		Workers:  1,
		Timeout:  5 * time.Second,
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "example.com"},
		{"https://example.com/path", "example.com"},
		{"http://sub.example.com:8080/path", "sub.example.com"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractDomain(tt.input))
		})
	}
}

func TestFileTypeFromExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".pdf", "pdf"},
		{".json", "json"},
		{".csv", "csv"},
		{".xlsx", "xlsx"},
		{".xls", "xlsx"},
		{".docx", "docx"},
		{".doc", "docx"},
		{".zip", "zip"},
		{".tar", "zip"},
		{".gz", "zip"},
		{".bz2", "zip"},
		{".png", "png"},
		{".jpg", "jpg"},
		{".jpeg", "jpg"},
		{".webp", "jpg"},
		{".html", ""},
		{".css", ""},
		{".js", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.expected, fileTypeFromExt(tt.ext))
		})
	}
}

func TestFileTypeFromExtCaseInsensitive(t *testing.T) {
	assert.Equal(t, "pdf", fileTypeFromExt(".PDF"))
	assert.Equal(t, "json", fileTypeFromExt(".JSON"))
	assert.Equal(t, "jpg", fileTypeFromExt(".JPG"))
}

func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "b", "c"}, "a"))
	assert.True(t, contains([]string{"a", "b", "c"}, "c"))
	assert.False(t, contains([]string{"a", "b", "c"}, "d"))
	assert.False(t, contains(nil, "a"))
	assert.False(t, contains([]string{}, "a"))
}

func TestResults(t *testing.T) {
	srv := testServer(map[string]string{
		"/": `<html><head><title>Test</title></head><body></body></html>`,
	})
	defer srv.Close()

	cfg := Config{
		URL:      srv.URL + "/",
		Depth:    1,
		MaxPages: 10,
		Workers:  1,
		Timeout:  5 * time.Second,
	}

	s := NewScraper(cfg)
	results, err := s.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, len(results), s.PageCount())
	assert.Equal(t, 0, s.ErrorCount())
	assert.NotZero(t, s.StartTime())
	assert.NotZero(t, s.Elapsed())
}
