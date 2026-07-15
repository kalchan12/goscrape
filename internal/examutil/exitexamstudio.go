package examutil

import (
	"regexp"
	"strings"
)

type ExitExamStudioStrategy struct{}

func (s *ExitExamStudioStrategy) BaseURL() string {
	return "https://exitexamstudio.app"
}

func (s *ExitExamStudioStrategy) DepartmentLinkRe() *regexp.Regexp {
	return regexp.MustCompile(`/departments/([a-z0-9_-]+)`)
}

func (s *ExitExamStudioStrategy) ExamLinkRe() *regexp.Regexp {
	return regexp.MustCompile(`/exam/([^"'\\]+)`)
}

func (s *ExitExamStudioStrategy) ParseExamSlug(slug string) (department, year, examType string) {
	parts := strings.Split(slug, "__")
	if len(parts) < 3 {
		return "", "", ""
	}
	examType = parts[len(parts)-1]
	year = parts[len(parts)-2]
	department = strings.Join(parts[:len(parts)-2], "__")
	department = strings.ReplaceAll(department, "\\", "")
	year = strings.ReplaceAll(year, "\\", "")
	examType = strings.ReplaceAll(examType, "\\", "")
	return
}
