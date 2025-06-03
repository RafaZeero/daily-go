package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

type Repo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Private   bool      `json:"private"`
	HTMLURL   string    `json:"html_url"`
	URL       string    `json:"url"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r Repo) String() string {
	return fmt.Sprintf("%s  - Created at: %s", r.Name, r.CreatedAt.String())
}

type user struct {
	username string
	apiKey   string
}

type GitHub struct {
	user  user
	repos []Repo
}

type GitHubOptions struct {
	APIKey   string
	Username string
}

func NewGithub(opts GitHubOptions) *GitHub {
	if opts.APIKey == "" {
		log.Fatal("authorization token should not be empty")
	}

	if opts.Username == "" {
		log.Fatal("user should not be empty")
	}

	gh := &GitHub{
		user: user{
			username: opts.Username,
			apiKey:   opts.APIKey,
		},
		repos: make([]Repo, 0),
	}

	gh.LoadReposFromUser()

	return gh
}

// @TODO: Validate errors
func (gh *GitHub) LoadReposFromUser() {
	url := fmt.Sprintf("%s/users/%s/repos", githubAPIBaseURL, gh.user.username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("failed to create request")
		return
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gh.user.apiKey))
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Add("Accept", "application/vnd.github+json")

	client := http.Client{Timeout: 15 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to do request")
		return
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("failed to read data")
		return
	}

	var Repos []Repo
	if err := json.Unmarshal(body, &Repos); err != nil {
		log.Fatal("failed to unmarshal data")
		return
	}

	gh.repos = append(gh.repos, Repos...)
}

func (gh *GitHub) GetRepos() []Repo {
	return gh.repos
}

func (gh *GitHub) GetReposChoices() []string {
	choices := []string{}
	for _, r := range gh.GetRepos() {
		choices = append(choices, fmt.Sprint(r))
	}
	return choices
}

type repoSelectionMsg struct {
	repos []string
	err   error
}

func SelectedRepos(repos []string) tea.Cmd {
	return func() tea.Msg {
		return repoSelectionMsg{repos: repos}
	}
}

type Action int

const (
	ACTION__SHOW_ALL_REPOS Action = iota
	ACTION__SHOW_SELECTED_REPOS
)

type model struct {
	choices      []string         // items on the to-do list
	cursor       int              // which to-do list item our cursor is pointing at
	selected     map[int]struct{} // which to-do items are selected
	paginator    paginator.Model
	repoSelected []string
	action       Action
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:
		m.action = ACTION__SHOW_ALL_REPOS
		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

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

		case "enter":
			selected := []string{}

			for s := range m.selected {
				selected = append(selected, m.choices[s])
				// fmt.Println(m.choices[s])
			}
			return m, SelectedRepos(selected)

		}

	case repoSelectionMsg:
		m.action = ACTION__SHOW_SELECTED_REPOS
		m.repoSelected = msg.repos // []str
		// m.repoSelected = msg.name // str
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
	}

	return ""
}

func main() {
	godotenv.Load()

	gh := NewGithub(GitHubOptions{
		APIKey:   os.Getenv("GITHUB_ACCESS_TOKEN"),
		Username: "RafaZeero",
	})

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
	}

	t := tea.NewProgram(m)
	if _, err := t.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
