package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

func NewHTTPServer(handler http.HandlerFunc) (*httptest.Server, *url.URL) {
	srv := httptest.NewServer(handler)
	u, _ := url.Parse(srv.URL)
	return srv, u
}

func NewHTMLHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}
}

func NewFileHandler(contentType string, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}
}

func NewMultiPageHandler(pages map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if body, ok := pages[path]; ok {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}
}

func ContainsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func StrPtr(s string) *string { return &s }

func HTMLWithLinks(links ...string) string {
	var sb strings.Builder
	sb.WriteString(`<html><head><title>Test Page</title></head><body>`)
	for _, l := range links {
		sb.WriteString(`<a href="` + l + `">link</a>`)
	}
	sb.WriteString(`</body></html>`)
	return sb.String()
}

func HTMLWithFiles(files map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<html><head><title>Files Page</title></head><body>`)
	for path, label := range files {
		sb.WriteString(`<a href="` + path + `">` + label + `</a> `)
	}
	sb.WriteString(`</body></html>`)
	return sb.String()
}
