package examutil

import "regexp"

type ExamSiteStrategy interface {
	BaseURL() string
	DepartmentLinkRe() *regexp.Regexp
	ExamLinkRe() *regexp.Regexp
	ParseExamSlug(path string) (department, year, examType string)
}
