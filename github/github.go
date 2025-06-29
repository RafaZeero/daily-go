package github

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	githubAPIBaseURL = "https://api.github.com"
	// geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key="
)

type Repo struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	FullName   string    `json:"full_name"`
	Private    bool      `json:"private"`
	CommitsURL string    `json:"commits_url"`
	HTMLURL    string    `json:"html_url"`
	URL        string    `json:"url"`
	Language   string    `json:"language"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (r Repo) String() string {
	visibility := "public"
	if r.Private {
		visibility = "private"
	}
	return fmt.Sprintf("[%s]  %s  - Created at: %s", visibility, r.Name, r.CreatedAt.String())
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
	allRepos := []Repo{}
	page := 1
	perPage := 100 // GitHub API max per page

	for {
		url := fmt.Sprintf("%s/user/repos?page=%d&per_page=%d", githubAPIBaseURL, page, perPage)

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

		var repos []Repo
		if err := json.Unmarshal(body, &repos); err != nil {
			log.Fatal("failed to unmarshal data")
			return
		}

		// If no repos returned, we've reached the end
		if len(repos) == 0 {
			break
		}

		// Filter repos to only include those owned by the user
		for _, repo := range repos {
			owner := strings.Split(repo.FullName, "/")[0]
			if owner == gh.user.username {
				allRepos = append(allRepos, repo)
			}
		}

		// If we got fewer repos than per_page, we've reached the end
		if len(repos) < perPage {
			break
		}

		page++
	}

	gh.repos = allRepos
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
