package extractor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../internal/testutil/fixtures/" + name)
	require.NoError(t, err)
	return string(data)
}

func TestExtract(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "https://example.com", data.URL)
	assert.Equal(t, "Test Page Title", data.Title)
}

func TestExtractMeta(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	assert.Equal(t, "A test page for unit testing", data.Meta["description"])
	assert.Equal(t, "scrape, test, golang", data.Meta["keywords"])
}

func TestExtractOpenGraph(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	assert.Equal(t, "OG Test Title", data.OpenGraph["og:title"])
	assert.Equal(t, "OG test description", data.OpenGraph["og:description"])
	assert.Equal(t, "https://example.com/image.jpg", data.OpenGraph["og:image"])
}

func TestExtractJSONLD(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	assert.Len(t, data.JSONLD, 2)
}

func TestExtractHeadings(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	assert.Contains(t, data.Headings, "Main Heading")
	assert.Contains(t, data.Headings, "Sub Heading One")
	assert.Contains(t, data.Headings, "Sub Heading Two")
}

func TestExtractLinks(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	urls := make([]string, len(data.Links))
	for i, l := range data.Links {
		urls[i] = l.Href
	}
	assert.Contains(t, urls, "https://example.com/page1")
	assert.Contains(t, urls, "https://example.com/page2")
	assert.Contains(t, urls, "https://example.com/page3")
}

func TestExtractTextBlocks(t *testing.T) {
	html := loadFixture(t, "generic_page.html")

	data, err := Extract(html, "https://example.com")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(data.TextBlocks), 3)
	for _, block := range data.TextBlocks {
		assert.Greater(t, len(block), 30)
	}
}

func TestExtractEmptyHTML(t *testing.T) {
	data, err := Extract("<html></html>", "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, data.Title)
	assert.Empty(t, data.Meta)
	assert.Empty(t, data.Headings)
}

func TestExtractInvalidHTML(t *testing.T) {
	data, err := Extract("<html><head><title>Test", "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "Test", data.Title)
}
