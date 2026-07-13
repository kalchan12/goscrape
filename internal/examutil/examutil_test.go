package examutil

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

func TestExtractTitle(t *testing.T) {
	html := `<html><head><title>Computer Science 2016 Regular Exam</title></head></html>`
	title := extractTitle(html)
	assert.Equal(t, "Computer Science 2016 Regular Exam", title)
}

func TestExtractTitleEmpty(t *testing.T) {
	title := extractTitle("<html></html>")
	assert.Empty(t, title)
}

func TestUnescapeJSString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`hello world`, `hello world`},
		{`hello \"world\"`, `hello "world"`},
		{`line1\nline2`, "line1\nline2"},
		{`tab\there`, "tab\there"},
		{`back\\slash`, `back\slash`},
		{`no escaping`, `no escaping`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := unescapeJSString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractQuestions(t *testing.T) {
	html := loadFixture(t, "exam_page.html")

	questions, title, err := ExtractQuestions([]byte(html))
	require.NoError(t, err)
	require.Len(t, questions, 2)

	assert.Equal(t, "Computer Science 2016 Regular Exam", title)

	q1 := questions[0]
	assert.Equal(t, "q1", q1.QuestionKey)
	assert.Equal(t, 1, q1.SourceID)
	assert.Len(t, q1.Question, 1)
	assert.Equal(t, "text", q1.Question[0].Type)
	assert.Equal(t, "What is 2+2?", q1.Question[0].Text)
	assert.Len(t, q1.Options, 3)
	assert.Equal(t, []int{1}, q1.CorrectAnswers)
	assert.Equal(t, "Basic addition", q1.Explanation)
	assert.Equal(t, "single", q1.SelectionMode)
	assert.True(t, q1.IsValid)

	q2 := questions[1]
	assert.Equal(t, "q2", q2.QuestionKey)
	assert.Equal(t, 2, q2.SourceID)
	assert.Equal(t, "What is Go?", q2.Question[0].Text)
	assert.Len(t, q2.Options, 2)
	assert.Equal(t, []int{0}, q2.CorrectAnswers)
}

func TestExtractQuestionsNoPayload(t *testing.T) {
	html := `<html><head><title>No Exam</title></head><body></body></html>`
	_, _, err := ExtractQuestions([]byte(html))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no RSC chunk")
}

func TestExtractQuestionsEmptyBody(t *testing.T) {
	_, _, err := ExtractQuestions([]byte{})
	assert.Error(t, err)
}

func TestExtractDepartmentLinks(t *testing.T) {
	html := `<html><body>
		<a href="/departments/computer-science">CS</a>
		<a href="/departments/mathematics">Math</a>
		<a href="/departments/computer-science">CS Again</a>
		<a href="/departments/physics">Physics</a>
	</body></html>`

	links := ExtractDepartmentLinks([]byte(html))
	assert.ElementsMatch(t, []string{"computer-science", "mathematics", "physics"}, links)
}

func TestExtractDepartmentLinksEmpty(t *testing.T) {
	links := ExtractDepartmentLinks([]byte("<html></html>"))
	assert.Empty(t, links)
}

func TestExtractExamLinksFromDept(t *testing.T) {
	html := `<html><body>
		<a href="/exam/computer-science__2016__regular">Regular</a>
		<a href="/exam/computer-science__2017__supplementary">Supplementary</a>
		<a href="/exam/mathematics__2016__regular">Math Regular</a>
		<a href="/about">About</a>
	</body></html>`

	links := ExtractExamLinksFromDept([]byte(html))
	require.Len(t, links, 3)

	assert.Equal(t, "computer-science", links[0].Department)
	assert.Equal(t, "2016", links[0].Year)
	assert.Equal(t, "regular", links[0].ExamType)
	assert.Contains(t, links[0].URL, "/exam/computer-science__2016__regular")
}

func TestExtractExamLinksMalformedPath(t *testing.T) {
	html := `<html><body>
		<a href="/exam/short-path">Invalid</a>
		<a href="/exam/computer-science__2016__regular">Valid</a>
	</body></html>`

	links := ExtractExamLinksFromDept([]byte(html))
	assert.Len(t, links, 1)
	assert.Equal(t, "computer-science", links[0].Department)
}

func TestFetchPage(t *testing.T) {
	body, err := FetchPage("https://example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
}
