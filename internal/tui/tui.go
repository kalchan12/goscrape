package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kalchan12/goscrape/internal/scraper"
)

type state int

const (
	stateInput state = iota
	stateRunning
	stateDone
)

type model struct {
	state    state
	urlInput textinput.Model
	depth    int
	spinner  spinner.Model
	viewport viewport.Model
	results  []scraper.ScrapeResult
	err      error
	status   string
}

type scrapeStarted struct{}
type scrapeDone struct {
	results []scraper.ScrapeResult
	err     error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))
)

func InitialModel() model {
	ti := textinput.New()
	ti.Placeholder = "https://example.com"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	return model{
		state:    stateInput,
		urlInput: ti,
		depth:    1,
		spinner:  s,
		viewport: vp,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateInput:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				if m.urlInput.Value() == "" {
					return m, nil
				}
				m.state = stateRunning
				return m, tea.Batch(m.spinner.Tick, m.startScrape())
			}
		case stateRunning:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		case stateDone:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "r":
				m.state = stateInput
				m.urlInput.SetValue("")
				m.viewport.SetContent("")
				m.results = nil
				m.err = nil
				return m, textinput.Blink
			}
		}

	case scrapeStarted:
		m.status = "Scraping in progress..."
		return m, nil

	case scrapeDone:
		m.state = stateDone
		m.results = msg.results
		m.err = msg.err
		m.status = "Complete"
		var sb strings.Builder
		if msg.err != nil {
			sb.WriteString(errStyle.Render(fmt.Sprintf("Error: %s\n", msg.err)))
		} else {
			sb.WriteString(statusStyle.Render(fmt.Sprintf("Scraped %d pages\n\n", len(msg.results))))
			for _, r := range msg.results[:min(len(msg.results), 5)] {
				sb.WriteString(fmt.Sprintf("• %s\n", r.URL))
				if r.Title != "" {
					sb.WriteString(fmt.Sprintf("  Title: %s\n", r.Title))
				}
				sb.WriteString(fmt.Sprintf("  Files: %d | Links: %d\n\n", len(r.Files), len(r.Links)))
			}
			if len(msg.results) > 5 {
				sb.WriteString(fmt.Sprintf("... and %d more pages\n", len(msg.results)-5))
			}
		}
		sb.WriteString("\nPress r to restart, q to quit")
		m.viewport.SetContent(sb.String())
		return m, nil

	default:
		if m.state == stateRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) startScrape() tea.Cmd {
	return func() tea.Msg {
		cfg := scraper.Config{
			URL:      m.urlInput.Value(),
			Depth:    m.depth,
			MaxPages: 50,
			Workers:  3,
		}

		s := scraper.NewScraper(cfg)
		results, err := s.Run()
		return scrapeDone{results: results, err: err}
	}
}

func (m model) View() string {
	switch m.state {
	case stateInput:
		return fmt.Sprintf(
			"%s\n%s\n\n%s\n  Depth: %d (use --depth flag to change)\n\n  Press Enter to start, q to quit\n",
			titleStyle.Render("GoScrape Interactive"),
			m.urlInput.View(),
			"Enter a URL to scrape:",
			m.depth,
		)
	case stateRunning:
		return fmt.Sprintf(
			"%s\n\n%s %s\n\nPress q to quit\n",
			titleStyle.Render("GoScrape Interactive"),
			m.spinner.View(),
			m.status,
		)
	case stateDone:
		return fmt.Sprintf(
			"%s\n\n%s\n",
			titleStyle.Render("GoScrape Interactive"),
			m.viewport.View(),
		)
	}
	return ""
}

func Run() error {
	p := tea.NewProgram(InitialModel())
	_, err := p.Run()
	return err
}
