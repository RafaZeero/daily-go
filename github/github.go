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

type Commit struct {
	SHA    string     `json:"sha"`
	Commit CommitInfo `json:"commit"`
	// Author    User       `json:"author"`
	// Committer User       `json:"committer"`
	// URL       string     `json:"url"`
	// HTMLURL   string     `json:"html_url"`
}

type CommitInfo struct {
	Author    CommitUser `json:"author"`
	Committer CommitUser `json:"committer"`
	Message   string     `json:"message"`
}

type CommitUser struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
	URL       string `json:"url"`
}

type Branch struct {
	Name      string `json:"name"`
	Commit    Commit `json:"commit"`
	Protected bool   `json:"protected"`
}

func (r Repo) String() string {
	visibility := "public"
	if r.Private {
		visibility = "private"
	}
	return fmt.Sprintf("[%s]  %s  - Created at: %s", visibility, r.Name, r.CreatedAt.String())
}

func (c Commit) String() string {
	message := c.Commit.Message
	if len(message) > 50 {
		message = message[:47] + "..."
	}
	return fmt.Sprintf("%s - %s (%s)",
		c.SHA[:8],
		message,
		c.Commit.Author.Date.Format("2006-01-02 15:04"))
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

func (gh *GitHub) GetCommitsForRepo(repoName string) ([]Commit, error) {
	return gh.GetCommitsForRepoSince(repoName, time.Time{})
}

func (gh *GitHub) GetCommitsForRepoSince(repoName string, since time.Time) ([]Commit, error) {
	return gh.GetCommitsForRepoSinceFromBranch(repoName, since, "")
}

func (gh *GitHub) GetCommitsForRepoSinceFromBranch(repoName string, since time.Time, branchName string) ([]Commit, error) {
	// Find the repo by name
	var targetRepo *Repo
	for _, repo := range gh.repos {
		if repo.Name == repoName {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		return nil, fmt.Errorf("repository %s not found", repoName)
	}

	allCommits := []Commit{}
	page := 1
	perPage := 100 // GitHub API max per page

	for {
		url := fmt.Sprintf("%s/repos/%s/commits?page=%d&per_page=%d", githubAPIBaseURL, targetRepo.FullName, page, perPage)

		// Add since parameter if provided
		if !since.IsZero() {
			url += fmt.Sprintf("&since=%s", since.Format(time.RFC3339))
		}

		// Add branch parameter if provided
		if branchName != "" {
			url += fmt.Sprintf("&sha=%s", branchName)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gh.user.apiKey))
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Add("Accept", "application/vnd.github+json")

		client := http.Client{Timeout: 15 * time.Second}

		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to do request: %v", err)
		}

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read data: %v", err)
		}

		var commits []Commit
		if err := json.Unmarshal(body, &commits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %v", err)
		}

		// If no commits returned, we've reached the end
		if len(commits) == 0 {
			break
		}

		allCommits = append(allCommits, commits...)

		// If we got fewer commits than per_page, we've reached the end
		if len(commits) < perPage {
			break
		}

		page++
	}

	return allCommits, nil
}

func (gh *GitHub) GetCommitsForRepoByDay(repoName string) ([]Commit, error) {
	now := time.Now()
	weekday := now.Weekday()

	var since time.Time

	switch weekday {
	case time.Monday:
		// Friday, Saturday, Sunday, Monday
		since = now.AddDate(0, 0, -3) // Go back 3 days to include Friday
	case time.Tuesday:
		// Monday
		since = now.AddDate(0, 0, -1)
	case time.Wednesday:
		// Tuesday
		since = now.AddDate(0, 0, -1)
	case time.Thursday:
		// Wednesday
		since = now.AddDate(0, 0, -1)
	case time.Friday:
		// Thursday
		since = now.AddDate(0, 0, -1)
	case time.Saturday:
		// Friday
		since = now.AddDate(0, 0, -1)
	case time.Sunday:
		// Saturday
		since = now.AddDate(0, 0, -1)
	}

	// Set time to start of the day
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())

	return gh.GetCommitsFromAllBranches(repoName, since)
}

func (gh *GitHub) GetBranchesForRepo(repoName string) ([]Branch, error) {
	// Find the repo by name
	var targetRepo *Repo
	for _, repo := range gh.repos {
		if repo.Name == repoName {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		return nil, fmt.Errorf("repository %s not found", repoName)
	}

	allBranches := []Branch{}
	page := 1
	perPage := 100 // GitHub API max per page

	for {
		url := fmt.Sprintf("%s/repos/%s/branches?page=%d&per_page=%d", githubAPIBaseURL, targetRepo.FullName, page, perPage)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gh.user.apiKey))
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Add("Accept", "application/vnd.github+json")

		client := http.Client{Timeout: 15 * time.Second}

		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to do request: %v", err)
		}

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read data: %v", err)
		}

		var branches []Branch
		if err := json.Unmarshal(body, &branches); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %v", err)
		}

		// If no branches returned, we've reached the end
		if len(branches) == 0 {
			break
		}

		allBranches = append(allBranches, branches...)

		// If we got fewer branches than per_page, we've reached the end
		if len(branches) < perPage {
			break
		}

		page++
	}

	return allBranches, nil
}

func (gh *GitHub) GetCommitsFromAllBranches(repoName string, since time.Time) ([]Commit, error) {
	// First, get all branches
	branches, err := gh.GetBranchesForRepo(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %v", err)
	}

	allCommits := []Commit{}
	commitMap := make(map[string]bool) // To track unique commits by SHA

	// Get commits from each branch
	for _, branch := range branches {
		commits, err := gh.GetCommitsForRepoSinceFromBranch(repoName, since, branch.Name)
		if err != nil {
			// Log error but continue with other branches
			fmt.Printf("Warning: failed to get commits from branch %s: %v\n", branch.Name, err)
			continue
		}

		// Add unique commits
		for _, commit := range commits {
			if !commitMap[commit.SHA] {
				commitMap[commit.SHA] = true
				allCommits = append(allCommits, commit)
			}
		}
	}

	// Sort commits by date (newest first)
	// Note: GitHub API already returns commits in chronological order (newest first)
	// But we might want to re-sort since we're combining from multiple branches

	return allCommits, nil
}
