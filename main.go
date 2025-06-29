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
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	githubAPIBaseURL = "https://api.github.com"
	geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key="
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

type Commit struct {
	SHA      string    `json:"sha"`
	Message  string    `json:"commit"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	RepoName string    `json:"repo_name"`
	HTMLURL  string    `json:"html_url"`
}

type CommitMessage struct {
	Message string `json:"message"`
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type CommitDetails struct {
	Message CommitMessage `json:"message"`
	Author  CommitAuthor  `json:"author"`
}

type CommitResponse struct {
	SHA     string        `json:"sha"`
	Commit  CommitDetails `json:"commit"`
	HTMLURL string        `json:"html_url"`
	Author  *struct {
		Login string `json:"login"`
	} `json:"author"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiPartResponse struct {
	Text string `json:"text"`
}

func (r Repo) String() string {
	return fmt.Sprintf("%s (%s) - Updated: %s", r.Name, r.Language, r.UpdatedAt.Format("2006-01-02"))
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

func (gh *GitHub) GetLatestCommits(repoNames []string, daysBack int) ([]Commit, error) {
	var allCommits []Commit
	since := time.Now().AddDate(0, 0, -daysBack)

	for _, repoName := range repoNames {
		// Extract repo name from the choice string
		name := strings.Split(repoName, " (")[0]

		url := fmt.Sprintf("%s/repos/%s/%s/commits?since=%s",
			githubAPIBaseURL, gh.user.username, name, since.Format(time.RFC3339))

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gh.user.apiKey))
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Add("Accept", "application/vnd.github+json")

		client := http.Client{Timeout: 15 * time.Second}

		res, err := client.Do(req)
		if err != nil {
			continue
		}

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			continue
		}

		var commits []CommitResponse
		if err := json.Unmarshal(body, &commits); err != nil {
			continue
		}

		for _, c := range commits {
			date, _ := time.Parse(time.RFC3339, c.Commit.Author.Date)
			author := c.Commit.Author.Name
			if c.Author != nil {
				author = c.Author.Login
			}

			commit := Commit{
				SHA:      c.SHA[:8],
				Message:  c.Commit.Message.Message,
				Author:   author,
				Date:     date,
				RepoName: name,
				HTMLURL:  c.HTMLURL,
			}
			allCommits = append(allCommits, commit)
		}
	}

	return allCommits, nil
}

type LLMService struct {
	apiKey string
}

func NewLLMService(apiKey string) *LLMService {
	return &LLMService{apiKey: apiKey}
}

func (llm *LLMService) GenerateSummary(commits []Commit) (string, error) {
	if len(commits) == 0 {
		return "No commits found in the specified time period.", nil
	}

	// Create a structured summary of commits
	var commitDetails strings.Builder
	commitDetails.WriteString("Recent commits summary:\n\n")

	// Group by repository
	repoCommits := make(map[string][]Commit)
	for _, commit := range commits {
		repoCommits[commit.RepoName] = append(repoCommits[commit.RepoName], commit)
	}

	for repoName, repoCommits := range repoCommits {
		commitDetails.WriteString(fmt.Sprintf("Repository: %s\n", repoName))
		for _, commit := range repoCommits {
			commitDetails.WriteString(fmt.Sprintf("- %s: %s (by %s on %s)\n",
				commit.SHA, commit.Message, commit.Author, commit.Date.Format("2006-01-02 15:04")))
		}
		commitDetails.WriteString("\n")
	}

	// Create prompt for LLM
	prompt := fmt.Sprintf(`Please provide a concise summary of the following recent commits for a daily standup or meeting. 
Focus on the most important changes, new features, bug fixes, and any breaking changes. 
Group by repository and highlight key achievements:

%s

Please format the response as a professional summary suitable for a team meeting.`, commitDetails.String())

	// Call Gemini API
	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	url := geminiAPIURL + llm.apiKey
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "Unable to generate summary at this time.", nil
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
	ACTION__SHOW_ALL_REPOS Action = iota
	ACTION__SHOW_SELECTED_REPOS
	ACTION__LOADING_COMMITS
	ACTION__SHOW_COMMITS
	ACTION__GENERATING_SUMMARY
	ACTION__SHOW_SUMMARY
)

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

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.paginator.Page > 0 {
				m.paginator.PrevPage()
				m.cursor = m.paginator.PerPage - 1
			}

		case "down", "j":
			if m.cursor < m.paginator.PerPage-1 && (m.cursor+m.paginator.Page*m.paginator.PerPage) < len(m.choices)-1 {
				m.cursor++
			} else if m.paginator.Page < m.paginator.TotalPages-1 {
				m.paginator.NextPage()
				m.cursor = 0
			}

		case "left", "h":
			if m.paginator.Page > 0 {
				m.paginator.PrevPage()
			}

		case "right", "l":
			if m.paginator.Page < m.paginator.TotalPages-1 {
				m.paginator.NextPage()
			}

		case " ":
			offset := m.paginator.Page * m.paginator.PerPage
			_, ok := m.selected[offset+m.cursor]
			if ok {
				delete(m.selected, offset+m.cursor)
			} else {
				m.selected[offset+m.cursor] = struct{}{}
			}

		case "enter":
			if m.action == ACTION__SHOW_ALL_REPOS {
				selected := []string{}
				for s := range m.selected {
					selected = append(selected, m.choices[s])
				}
				if len(selected) > 0 {
					m.repoSelected = selected
					m.action = ACTION__LOADING_COMMITS
					return m, LoadCommits(m.gh, selected, m.daysBack)
				}
			} else if m.action == ACTION__SHOW_SELECTED_REPOS {
				m.action = ACTION__LOADING_COMMITS
				return m, LoadCommits(m.gh, m.repoSelected, m.daysBack)
			} else if m.action == ACTION__SHOW_COMMITS {
				m.action = ACTION__GENERATING_SUMMARY
				return m, GenerateSummary(m.llm, m.commits)
			} else if m.action == ACTION__SHOW_SUMMARY {
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
	case ACTION__SHOW_ALL_REPOS:
		viewText.WriteString("Select repositories to analyze:\n\n")

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
		viewText.WriteString("\nPress space to select, enter to continue, q to quit.\n")

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
	p.SetTotalPages(len(gh.GetReposChoices()))

	m := model{
		choices:   gh.GetReposChoices(),
		selected:  make(map[int]struct{}),
		paginator: p,
		spinner:   s,
		gh:        gh,
		llm:       llm,
		daysBack:  config.DaysBack,
	}

	t := tea.NewProgram(m)
	if _, err := t.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
