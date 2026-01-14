package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const githubAPIBaseURL = "https://api.github.com"

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

func (gh *GitHub) GetRecentlyUpdatedRepos(daysBack int) []Repo {
	if daysBack <= 0 {
		return gh.repos
	}

	cutoffDate := time.Now().AddDate(0, 0, -daysBack)
	var recentRepos []Repo

	for _, repo := range gh.repos {
		// Check if repository was updated within the specified time period
		if repo.UpdatedAt.After(cutoffDate) {
			recentRepos = append(recentRepos, repo)
		}
	}

	return recentRepos
}

func (gh *GitHub) GetReposChoices() []string {
	choices := []string{}
	for _, r := range gh.GetRepos() {
		choices = append(choices, fmt.Sprint(r))
	}
	return choices
}

func (gh *GitHub) GetRecentlyUpdatedReposChoices(daysBack int) []string {
	choices := []string{}
	for _, r := range gh.GetRecentlyUpdatedRepos(daysBack) {
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
