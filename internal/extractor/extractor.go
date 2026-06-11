package extractor

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type PageData struct {
	URL        string            `json:"url"`
	Title      string            `json:"title"`
	Meta       map[string]string `json:"meta"`
	OpenGraph  map[string]string `json:"open_graph"`
	JSONLD     []json.RawMessage `json:"json_ld"`
	Headings   []string          `json:"headings"`
	Links      []Link            `json:"links"`
	TextBlocks []string          `json:"text_blocks"`
}

type Link struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

func Extract(htmlContent, pageURL string) (*PageData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	data := &PageData{
		URL:       pageURL,
		Meta:      make(map[string]string),
		OpenGraph: make(map[string]string),
	}

	data.Title = doc.Find("title").First().Text()

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		prop, _ := s.Attr("property")
		content, _ := s.Attr("content")
		if content == "" {
			return
		}
		if strings.HasPrefix(prop, "og:") {
			data.OpenGraph[prop] = content
		} else if name != "" {
			data.Meta[name] = content
		}
	})

	doc.Find("script[type=\"application/ld+json\"]").Each(func(i int, s *goquery.Selection) {
		raw := json.RawMessage(strings.TrimSpace(s.Text()))
		if len(raw) > 0 {
			data.JSONLD = append(data.JSONLD, raw)
		}
	})

	doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			data.Headings = append(data.Headings, text)
		}
	})

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		if href != "" {
			data.Links = append(data.Links, Link{Href: href, Text: text})
		}
	})

	doc.Find("p, li, td, th, blockquote, figcaption").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if len(text) > 30 {
			data.TextBlocks = append(data.TextBlocks, text)
		}
	})

	return data, nil
}


