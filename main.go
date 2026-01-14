package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type repoSelectionMsg struct {
	repos []string
	err   error
}

func SelectedRepos(repos []string) tea.Cmd {
	return func() tea.Msg {
		return repoSelectionMsg{repos: repos}
	}
}

type commitsLoadedMsg struct {
	commits []Commit
	err     error
}

func LoadCommits(gh *GitHub, repos []string, daysBack int) tea.Cmd {
	return func() tea.Msg {
		commits, err := gh.GetLatestCommits(repos, daysBack)
		return commitsLoadedMsg{commits: commits, err: err}
	}
}

type summaryGeneratedMsg struct {
	summary string
	err     error
}

func GenerateSummary(llm *LLMService, commits []Commit) tea.Cmd {
	return func() tea.Msg {
		summary, err := llm.GenerateSummary(commits)
		return summaryGeneratedMsg{summary: summary, err: err}
	}
}

type Action int

const (
	ACTION__SELECT_DAYS Action = iota
	ACTION__SHOW_ALL_REPOS
	ACTION__SHOW_SELECTED_REPOS
	ACTION__LOADING_COMMITS
	ACTION__SHOW_COMMITS
	ACTION__GENERATING_SUMMARY
	ACTION__SHOW_SUMMARY
)

type DayOption struct {
	label    string
	date     time.Time
	daysBack int
}

func getLast7Days() []DayOption {
	var options []DayOption
	today := time.Now()

	for i := range 7 {
		date := today.AddDate(0, 0, -i)
		daysBack := i
		if i == 0 {
			daysBack = 1 // Today means last 1 day
		} else {
			daysBack = i + 1 // Last N days
		}

		// Format as "Month, Day" (e.g., "June, 10th")
		daySuffix := getDaySuffix(date.Day())
		label := fmt.Sprintf("%s, %d%s", date.Format("January"), date.Day(), daySuffix)

		options = append(options, DayOption{
			label:    label,
			date:     date,
			daysBack: daysBack,
		})
	}

	return options
}

func getDaySuffix(day int) string {
	switch day {
	case 1, 21, 31:
		return "st"
	case 2, 22:
		return "nd"
	case 3, 23:
		return "rd"
	default:
		return "th"
	}
}

type model struct {
	choices      []string
	cursor       int
	selected     map[int]struct{}
	paginator    paginator.Model
	repoSelected []string
	action       Action
	spinner      spinner.Model
	commits      []Commit
	summary      string
	gh           *GitHub
	llm          *LLMService
	daysBack     int
	dayOptions   []DayOption
	selectedDay  int
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "b":
			if m.action == ACTION__SHOW_ALL_REPOS {
				// Go back to day selection
				m.action = ACTION__SELECT_DAYS
				m.cursor = 0
				m.selected = make(map[int]struct{})
			}

		case "up", "k":
			if m.action == ACTION__SELECT_DAYS {
				if m.selectedDay > 0 {
					m.selectedDay--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				} else if m.paginator.Page > 0 {
					m.paginator.PrevPage()
					m.cursor = m.paginator.PerPage - 1
				}
			}

		case "down", "j":
			if m.action == ACTION__SELECT_DAYS {
				if m.selectedDay < len(m.dayOptions)-1 {
					m.selectedDay++
				}
			} else {
				if m.cursor < m.paginator.PerPage-1 && (m.cursor+m.paginator.Page*m.paginator.PerPage) < len(m.choices)-1 {
					m.cursor++
				} else if m.paginator.Page < m.paginator.TotalPages-1 {
					m.paginator.NextPage()
					m.cursor = 0
				}
			}

		case "left", "h":
			if m.action != ACTION__SELECT_DAYS && m.paginator.Page > 0 {
				m.paginator.PrevPage()
			}

		case "right", "l":
			if m.action != ACTION__SELECT_DAYS && m.paginator.Page < m.paginator.TotalPages-1 {
				m.paginator.NextPage()
			}

		case " ":
			if m.action == ACTION__SELECT_DAYS {
				// Day selection is single choice, so we just set the selected day
				m.selectedDay = m.selectedDay
			} else {
				offset := m.paginator.Page * m.paginator.PerPage
				_, ok := m.selected[offset+m.cursor]
				if ok {
					delete(m.selected, offset+m.cursor)
				} else {
					m.selected[offset+m.cursor] = struct{}{}
				}
			}

		case "enter":
			switch m.action {
			case ACTION__SELECT_DAYS:
				// User selected a day, now show repositories for that period
				selectedOption := m.dayOptions[m.selectedDay]
				m.daysBack = selectedOption.daysBack
				m.choices = m.gh.GetRecentlyUpdatedReposChoices(m.daysBack)

				if len(m.choices) == 0 {
					// No repositories found for this period
					m.action = ACTION__SHOW_ALL_REPOS
				} else {
					// Reset paginator for new choices
					m.paginator.SetTotalPages(len(m.choices))
					m.cursor = 0
					m.selected = make(map[int]struct{})
					m.action = ACTION__SHOW_ALL_REPOS
				}
			case ACTION__SHOW_ALL_REPOS:
				if len(m.choices) == 0 {
					// No repositories found, go back to day selection
					m.action = ACTION__SELECT_DAYS
				} else {
					selected := []string{}
					for s := range m.selected {
						selected = append(selected, m.choices[s])
					}
					if len(selected) > 0 {
						m.repoSelected = selected
						m.action = ACTION__LOADING_COMMITS
						return m, LoadCommits(m.gh, selected, m.daysBack)
					}
				}
			case ACTION__SHOW_SELECTED_REPOS:
				m.action = ACTION__LOADING_COMMITS
				return m, LoadCommits(m.gh, m.repoSelected, m.daysBack)
			case ACTION__SHOW_COMMITS:
				m.action = ACTION__GENERATING_SUMMARY
				return m, GenerateSummary(m.llm, m.commits)
			case ACTION__SHOW_SUMMARY:
				return m, tea.Quit
			}
		}

	case repoSelectionMsg:
		m.action = ACTION__SHOW_SELECTED_REPOS
		m.repoSelected = msg.repos

	case commitsLoadedMsg:
		if msg.err != nil {
			m.action = ACTION__SHOW_SELECTED_REPOS
			return m, nil
		}
		m.commits = msg.commits
		m.action = ACTION__SHOW_COMMITS

	case summaryGeneratedMsg:
		if msg.err != nil {
			m.action = ACTION__SHOW_COMMITS
			return m, nil
		}
		m.summary = msg.summary
		m.action = ACTION__SHOW_SUMMARY

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var viewText strings.Builder

	switch m.action {
	case ACTION__SELECT_DAYS:
		viewText.WriteString("Select a time period to view repositories:\n\n")

		for i, option := range m.dayOptions {
			cursor := " "
			if m.selectedDay == i {
				cursor = ">"
			}

			viewText.WriteString(fmt.Sprintf("%s %s\n", cursor, option.label))
		}

		viewText.WriteString("\nPress enter to continue or q to quit.\n")

	case ACTION__SHOW_ALL_REPOS:
		if len(m.choices) == 0 {
			viewText.WriteString("No repositories found for the selected time period.\n")
			viewText.WriteString("Press q to quit or any other key to go back to day selection.\n")
		} else {
			viewText.WriteString(fmt.Sprintf("Repositories updated in the last %d days:\n\n", m.daysBack))

			start, end := m.paginator.GetSliceBounds(len(m.choices))
			pageChoices := m.choices[start:end]

			for i, choice := range pageChoices {
				cursor := " "
				if m.cursor == i {
					cursor = ">"
				}

				checked := " "
				offset := m.paginator.Page * m.paginator.PerPage
				if _, ok := m.selected[offset+i]; ok {
					checked = "x"
				}

				viewText.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
			}

			viewText.WriteString(m.paginator.View())
			viewText.WriteString(fmt.Sprintf("\nShowing %d repositories updated in the last %d days\n", len(m.choices), m.daysBack))
			viewText.WriteString("Press space to select, enter to continue, b to go back, q to quit.\n")
		}

	case ACTION__SHOW_SELECTED_REPOS:
		viewText.WriteString("Selected repositories:\n\n")
		for _, repo := range m.repoSelected {
			viewText.WriteString(fmt.Sprintf("• %s\n", repo))
		}
		viewText.WriteString(fmt.Sprintf("\nAnalyzing commits from the last %d days...\n", m.daysBack))
		viewText.WriteString("Press enter to continue or q to quit.\n")

	case ACTION__LOADING_COMMITS:
		viewText.WriteString(fmt.Sprintf("%s Loading commits from selected repositories...\n", m.spinner.View()))
		viewText.WriteString("This may take a moment...\n")

	case ACTION__SHOW_COMMITS:
		viewText.WriteString("Recent commits found:\n\n")

		if len(m.commits) == 0 {
			viewText.WriteString("No commits found in the specified time period.\n")
		} else {
			repoCommits := make(map[string][]Commit)
			for _, commit := range m.commits {
				repoCommits[commit.RepoName] = append(repoCommits[commit.RepoName], commit)
			}

			for repoName, commits := range repoCommits {
				viewText.WriteString(fmt.Sprintf("📁 %s (%d commits):\n", repoName, len(commits)))
				for i, commit := range commits {
					if i >= 5 { // Show only first 5 commits per repo
						viewText.WriteString(fmt.Sprintf("   ... and %d more commits\n", len(commits)-5))
						break
					}
					viewText.WriteString(fmt.Sprintf("   • %s: %s\n", commit.SHA, commit.Message))
				}
				viewText.WriteString("\n")
			}
		}

		viewText.WriteString("Press enter to generate summary or q to quit.\n")

	case ACTION__GENERATING_SUMMARY:
		viewText.WriteString(fmt.Sprintf("%s Generating AI summary...\n", m.spinner.View()))
		viewText.WriteString("This may take a moment...\n")

	case ACTION__SHOW_SUMMARY:
		viewText.WriteString("🤖 AI Generated Summary:\n\n")
		viewText.WriteString(m.summary)
		viewText.WriteString("\n\nPress enter to exit.\n")
	}

	return viewText.String()
}

func main() {
	godotenv.Load()

	config := LoadConfig()
	if err := config.Validate(); err != nil {
		log.Fatal(err)
	}

	gh := NewGithub(GitHubOptions{
		APIKey:   config.GitHubToken,
		Username: config.Username,
	})

	llm := NewLLMService(config.GeminiKey)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = config.PerPage
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	m := model{
		choices:     []string{},
		selected:    make(map[int]struct{}),
		paginator:   p,
		action:      ACTION__SELECT_DAYS,
		spinner:     s,
		gh:          gh,
		llm:         llm,
		daysBack:    config.DaysBack,
		dayOptions:  getLast7Days(),
		selectedDay: 0, // Default to today
	}

	t := tea.NewProgram(m)
	if _, err := t.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
