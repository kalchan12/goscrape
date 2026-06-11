package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ExamQuestion struct {
	QuestionKey    string         `json:"questionKey"`
	SourceID       int            `json:"sourceId"`
	Question       []TextBlock    `json:"question"`
	Options        []OptionBlock  `json:"options"`
	CorrectAnswers []int          `json:"correctAnswers"`
	Explanation    string         `json:"explanation"`
	SelectionMode  string         `json:"selectionMode"`
	IsValid        bool           `json:"isValid"`
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
	URL       string          `json:"url"`
	Title     string          `json:"title"`
	Questions []ExamQuestion  `json:"questions"`
	Count     int             `json:"count"`
}

var examFlags struct {
	url    string
	output string
	pretty bool
}

var examCmd = &cobra.Command{
	Use:   "exam --url <exam-url>",
	Short: "Extract exam questions from a Next.js exam page",
	Long:  `Fetch an exam page, parse the embedded RSC payload, and extract all questions with options and answers.`,
	Example: `  goscrape exam --url https://exitexamstudio.app/exam/computer-science__2016__regular
  goscrape exam --url https://exitexamstudio.app/exam/computer-science__2016__regular --pretty
  goscrape exam --url https://exitexamstudio.app/exam/computer-science__2016__regular --output exam.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if examFlags.url == "" {
			return cmd.Help()
		}

		// Fetch the page using a direct HTTP client
		body, err := fetchPage(examFlags.url)
		if err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}

		// Extract questions from RSC payload
		questions, err := extractQuestionsFromRSC(body)
		if err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}

		if len(questions) == 0 {
			return fmt.Errorf("no questions found on the page")
		}

		// Extract title
		titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
		title := ""
		if m := titleRe.FindStringSubmatch(string(body)); len(m) > 1 {
			title = m[1]
		}

		data := ExamData{
			URL:       examFlags.url,
			Title:     title,
			Questions: questions,
			Count:     len(questions),
		}

		var out []byte
		if examFlags.pretty {
			out, err = json.MarshalIndent(data, "", "  ")
		} else {
			out, err = json.Marshal(data)
		}
		if err != nil {
			return err
		}

		if examFlags.output != "" {
			if err := os.WriteFile(examFlags.output, out, 0644); err != nil {
				return err
			}
			fmt.Printf("Saved %d questions to %s\n", len(questions), examFlags.output)
		} else {
			fmt.Println(string(out))
		}

		return nil
	},
}

func fetchPage(url string) ([]byte, error) {
	return getURL(url)
}

func extractQuestionsFromRSC(body []byte) ([]ExamQuestion, error) {
	html := string(body)

	// Find the __next_f.push chunk containing the "questions" array
	chunkRe := regexp.MustCompile(`__next_f\.push\(\[1,"((?:[^"\\]|\\.)*)"\]\)`)
	matches := chunkRe.FindAllStringSubmatch(html, -1)

	for _, m := range matches {
		chunk := m[1]
		if !strings.Contains(chunk, "correctAnswer") {
			continue
		}

		// Unescape the JS string
		raw := unescapeJSString(chunk)

		// Find the questions array
		qIdx := strings.LastIndex(raw, `"questions":`)
		if qIdx == -1 {
			continue
		}

		arrStart := strings.Index(raw[qIdx:], "[")
		if arrStart == -1 {
			continue
		}
		arrStart += qIdx

		depth := 0
		arrEnd := arrStart
		for i := arrStart; i < len(raw); i++ {
			switch raw[i] {
			case '[':
				depth++
			case ']':
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
		var questions []ExamQuestion
		if err := json.Unmarshal([]byte(questionsJSON), &questions); err != nil {
			return nil, fmt.Errorf("parse questions JSON: %w", err)
		}
		return questions, nil
	}

	return nil, fmt.Errorf("no RSC chunk with questions found")
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
			case '/':
				b.WriteByte('/')
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

func getURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func init() {
	rootCmd.AddCommand(examCmd)

	examCmd.Flags().StringVar(&examFlags.url, "url", "", "Exam page URL (required)")
	examCmd.Flags().StringVar(&examFlags.output, "output", "", "Save to file")
	examCmd.Flags().BoolVar(&examFlags.pretty, "pretty", false, "Pretty-print JSON")
}
