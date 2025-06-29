package main

import (
	"daily-go/github"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	githubAPIBaseURL = "https://api.github.com"
	// geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key="
)

type repoSelectionMsg struct {
	repos []string
	err   error
}

type commitsMsg struct {
	repoName string
	commits  []github.Commit
	err      error
}

func SelectedRepos(repos []string) tea.Cmd {
	return func() tea.Msg {
		return repoSelectionMsg{repos: repos}
	}
}

func FetchCommits(gh *github.GitHub, repoName string) tea.Cmd {
	return func() tea.Msg {
		commits, err := gh.GetCommitsForRepoByDay(repoName)
		return commitsMsg{
			repoName: repoName,
			commits:  commits,
			err:      err,
		}
	}
}

type Action int

const (
	ACTION__SHOW_ALL_REPOS Action = iota
	ACTION__SHOW_SELECTED_REPOS
	ACTION__SHOW_COMMITS
)

type model struct {
	choices      []string         // items on the to-do list
	cursor       int              // which to-do list item our cursor is pointing at
	selected     map[int]struct{} // which to-do items are selected
	paginator    paginator.Model
	repoSelected []string
	action       Action
	gh           *github.GitHub
	commits      []github.Commit
	repoName     string
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if m.action == ACTION__SHOW_SELECTED_REPOS {
				// Fetch commits for the first selected repo
				if len(m.repoSelected) > 0 {
					// Extract repo name from the selected repo string
					// Assuming the format is "[visibility] repoName - Created at: ..."
					repoStr := m.repoSelected[0]
					parts := strings.Split(repoStr, "  ")
					if len(parts) >= 2 {
						repoName := parts[1]
						return m, FetchCommits(m.gh, repoName)
					}
				}
			} else {
				// Original behavior for selecting repos
				m.action = ACTION__SHOW_ALL_REPOS
				selected := []string{}

				for s := range m.selected {
					selected = append(selected, m.choices[s])
					// fmt.Println(m.choices[s])
				}
				return m, SelectedRepos(selected)
			}

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.paginator.Page > 0 {
				// Vai para a última opção da página anterior
				m.paginator.PrevPage()
				m.cursor = m.paginator.PerPage - 1
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < m.paginator.PerPage-1 && (m.cursor+m.paginator.Page*m.paginator.PerPage) < len(m.choices)-1 {
				m.cursor++
			} else if m.paginator.Page < m.paginator.TotalPages-1 {
				// Vai para a primeira opção da próxima página
				m.paginator.NextPage()
				m.cursor = 0
			}

		case "left", "h": // Voltar página
			if m.paginator.Page > 0 {
				m.paginator.PrevPage()
			}

		case "right", "l": // Avançar página
			if m.paginator.Page < m.paginator.TotalPages-1 {
				m.paginator.NextPage()
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case " ":
			offset := m.paginator.Page * m.paginator.PerPage
			_, ok := m.selected[offset+m.cursor]
			if ok {
				delete(m.selected, offset+m.cursor)
			} else {
				m.selected[offset+m.cursor] = struct{}{}
			}
		}

	case repoSelectionMsg:
		m.action = ACTION__SHOW_SELECTED_REPOS
		m.repoSelected = msg.repos // []str
		// m.repoSelected = msg.name // str

	case commitsMsg:
		if msg.err != nil {
			// Handle error - you might want to show an error message
			return m, nil
		}
		m.action = ACTION__SHOW_COMMITS
		m.commits = msg.commits
		m.repoName = msg.repoName
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	var viewText strings.Builder
	viewText.WriteString("What repos should we choose?\n\n")

	start, end := m.paginator.GetSliceBounds(len(m.choices))
	pageChoices := m.choices[start:end]

	switch m.action {

	case ACTION__SHOW_ALL_REPOS:
		// Iterate over our choices
		for i, choice := range pageChoices {
			// Is the cursor pointing at this choice?
			cursor := " " // no cursor
			if m.cursor == i {
				cursor = ">" // cursor!
			}

			// Is this choice selected?
			checked := " " // not selected
			offset := m.paginator.Page * m.paginator.PerPage
			if _, ok := m.selected[offset+i]; ok {
				checked = "x" // selected!
			}

			// Render the row
			viewText.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
		}

		viewText.WriteString(m.paginator.View())

		// The footer
		viewText.WriteString("\nPress q to quit.\n")

		// Send the UI for rendering
		return viewText.String()
	case ACTION__SHOW_SELECTED_REPOS:
		var selectedText strings.Builder
		selectedText.WriteString("Selected: \n\n")
		for _, repo := range m.repoSelected {
			selectedText.WriteString(repo + "\n")
		}
		selectedText.WriteString("\nPress enter to continue or q to quit.\n")
		return selectedText.String()

	case ACTION__SHOW_COMMITS:
		var commitsText strings.Builder

		// Determine which day(s) we're showing
		now := time.Now()
		weekday := now.Weekday()
		var dayRange string

		switch weekday {
		case time.Monday:
			dayRange = "Friday, Saturday, Sunday, and Monday"
		case time.Tuesday:
			dayRange = "Monday"
		case time.Wednesday:
			dayRange = "Tuesday"
		case time.Thursday:
			dayRange = "Wednesday"
		case time.Friday:
			dayRange = "Thursday"
		case time.Saturday:
			dayRange = "Friday"
		case time.Sunday:
			dayRange = "Saturday"
		}

		commitsText.WriteString(fmt.Sprintf("Commits for %s (%s):\n\n", m.repoName, dayRange))

		if len(m.commits) == 0 {
			commitsText.WriteString("No commits found for the specified day(s).\n")
		} else {
			for i, commit := range m.commits {
				if i >= 20 { // Limit to first 20 commits for display
					commitsText.WriteString(fmt.Sprintf("... and %d more commits\n", len(m.commits)-20))
					break
				}
				message := commit.Commit.Message
				if len(message) > 60 {
					message = message[:57] + "..."
				}
				commitsText.WriteString(fmt.Sprintf("%s - %s (%s)\n",
					commit.SHA[:8],
					message,
					commit.Commit.Author.Date.Format("2006-01-02 15:04")))
			}
		}

		commitsText.WriteString("\nPress q to quit.\n")
		return commitsText.String()
	}

	return ""
}

func main() {
	godotenv.Load()

	gh := github.NewGithub(github.GitHubOptions{
		APIKey:   os.Getenv("GITHUB_ACCESS_TOKEN"),
		Username: "RafaZeero",
	})

	// repo := github.Repo{}
	// for _, r := range gh.GetRepos() {
	// 	fmt.Println(r)
	// 	if r.Name == "contas-casa" {
	// 		fmt.Println("found!!")
	// 		repo = r
	// 		break
	// 	}
	// }

	// commits, err := gh.GetCommitsForRepo("contas-casa")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// for _, c := range commits {
	// 	fmt.Println(c)
	// }

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 10
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(len(gh.GetReposChoices()))

	m := model{
		choices:   gh.GetReposChoices(),
		selected:  make(map[int]struct{}),
		paginator: p,
		gh:        gh,
	}

	t := tea.NewProgram(m)
	if _, err := t.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
