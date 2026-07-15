package examutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Question struct {
	QuestionKey    string        `json:"questionKey"`
	SourceID       int           `json:"sourceId"`
	Question       []TextBlock   `json:"question"`
	Options        []OptionBlock `json:"options"`
	CorrectAnswers []int         `json:"correctAnswers"`
	Explanation    string        `json:"explanation"`
	SelectionMode  string        `json:"selectionMode"`
	IsValid        bool          `json:"isValid"`
}

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type OptionBlock struct {
	Key    string      `json:"key"`
	Blocks []TextBlock `json:"blocks"`
}

type ExamData struct {
	URL       string     `json:"url"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
	Count     int        `json:"count"`
}

type ExamLink struct {
	Department string
	Year       string
	ExamType   string
	URL        string
}

var (
	httpClient      = &http.Client{Timeout: 60 * time.Second}
	defaultStrategy = &ExitExamStudioStrategy{}
)

func FetchPage(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func ExtractQuestions(body []byte) ([]Question, string, error) {
	html := string(body)

	title := extractTitle(html)

	chunkRe := regexp.MustCompile(`__next_f\.push\(\[1,"((?:[^"\\]|\\.)*)"\]\)`)
	matches := chunkRe.FindAllStringSubmatch(html, -1)

	for _, m := range matches {
		chunk := m[1]
		if !strings.Contains(chunk, "correctAnswer") {
			continue
		}

		raw := unescapeJSString(chunk)

		qIdx := strings.LastIndex(raw, `"questions":`)
		if qIdx == -1 {
			continue
		}

		// Find the [ after "questions":
		arrStart := strings.Index(raw[qIdx:], "[")
		if arrStart == -1 {
			continue
		}
		arrStart += qIdx

		// Count brackets with string awareness to find matching ]
		depth := 0
		inStr := false
		esc := false
		var arrEnd int
		for i := arrStart; i < len(raw); i++ {
			if esc {
				esc = false
				continue
			}
			if raw[i] == '\\' && inStr {
				esc = true
				continue
			}
			if raw[i] == '"' {
				inStr = !inStr
				continue
			}
			if inStr {
				continue
			}
			if raw[i] == '[' {
				depth++
			} else if raw[i] == ']' {
				depth--
				if depth == 0 {
					arrEnd = i + 1
					goto found
				}
			}
		}
		continue

	found:
		questionsJSON := raw[arrStart:arrEnd]
		var questions []Question
		if err := json.Unmarshal([]byte(questionsJSON), &questions); err != nil {
			return nil, "", fmt.Errorf("parse questions JSON: %w", err)
		}
		return questions, title, nil
	}

	return nil, "", fmt.Errorf("no RSC chunk with questions found")
}

func ExtractQuestionsFromURL(url string) ([]Question, string, error) {
	body, err := FetchPage(url)
	if err != nil {
		return nil, "", fmt.Errorf("fetch %s: %w", url, err)
	}
	return ExtractQuestions(body)
}

func extractTitle(html string) string {
	re := regexp.MustCompile(`<title>([^<]+)</title>`)
	if m := re.FindStringSubmatch(html); len(m) > 1 {
		return m[1]
	}
	return ""
}

func unescapeJSString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				b.WriteByte('"')
				i++
			case '\\':
				b.WriteByte('\\')
				i++
			case 'n':
				b.WriteByte('\n')
				i++
			case 't':
				b.WriteByte('\t')
				i++
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func ExtractDepartmentsWithStrategy(body []byte, s ExamSiteStrategy) []string {
	html := string(body)
	re := s.DepartmentLinkRe()
	seen := make(map[string]bool)
	var depts []string

	matches := re.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		slug := m[1]
		if seen[slug] {
			continue
		}
		seen[slug] = true
		depts = append(depts, slug)
	}
	return depts
}

func ExtractDepartmentLinks(body []byte) []string {
	return ExtractDepartmentsWithStrategy(body, defaultStrategy)
}

func ExtractExamsFromDeptWithStrategy(body []byte, s ExamSiteStrategy) []ExamLink {
	html := string(body)
	re := s.ExamLinkRe()
	seen := make(map[string]bool)
	var links []ExamLink

	matches := re.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		path := "/exam/" + m[1]
		if seen[path] {
			continue
		}
		seen[path] = true

		dept, year, examType := s.ParseExamSlug(m[1])
		if dept == "" {
			continue
		}

		links = append(links, ExamLink{
			Department: dept,
			Year:       year,
			ExamType:   examType,
			URL:        strings.TrimSuffix(s.BaseURL(), "/") + path,
		})
	}
	return links
}

func ExtractExamLinksFromDept(body []byte) []ExamLink {
	return ExtractExamsFromDeptWithStrategy(body, defaultStrategy)
}
