package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kalchan12/goscrape/internal/examutil"
	"github.com/spf13/cobra"
)

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

		cfg := ScrapeConfig{URL: examFlags.url}
		if err := cfg.Validate(); err != nil {
			return err
		}

		questions, title, err := examutil.ExtractQuestionsFromURL(examFlags.url)
		if err != nil {
			return err
		}

		if len(questions) == 0 {
			return fmt.Errorf("no questions found on the page")
		}

		data := examutil.ExamData{
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

func init() {
	rootCmd.AddCommand(examCmd)

	examCmd.Flags().StringVar(&examFlags.url, "url", "", "Exam page URL (required)")
	examCmd.Flags().StringVar(&examFlags.output, "output", "", "Save to file")
	examCmd.Flags().BoolVar(&examFlags.pretty, "pretty", false, "Pretty-print JSON")
}
