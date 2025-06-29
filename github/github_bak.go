package github

import (
	"context"
	"fmt"
	"log"
	"time"

	gh "github.com/google/go-github/v62/github"
)

type RepositoryInfo struct {
	Name        string
	FullName    string
	Description string
	URL         string
	LastPush    time.Time
	Language    string
}

type CommitInfo struct {
	SHA       string
	Message   string
	Author    string
	Date      time.Time
	Files     []string
	Additions int
	Deletions int
}

type Client struct {
	client *gh.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		client: gh.NewClient(nil).WithAuthToken(accessToken),
	}
}

func (c *Client) GetRepositoriesWithRecentCommits(ctx context.Context, username string, since time.Time) ([]RepositoryInfo, error) {
	var repos []RepositoryInfo
	reposMap := make(map[string]*RepositoryInfo)

	// Get user's repositories (owned)
	userRepos, err := c.repo(ctx, username, since)
	if err != nil {
		return nil, err
	}
	for _, repo := range userRepos {
		reposMap[repo.FullName] = repo
	}

	// Get organization repositories where user contributes
	orgRepos, err := c.getOrganizationRepositories(ctx, username, since)
	if err != nil {
		log.Printf("Warning: Could not fetch organization repositories: %v", err)
	} else {
		for _, repo := range orgRepos {
			reposMap[repo.FullName] = repo
		}
	}

	// Convert map to slice and filter by actual commits
	for _, repoInfo := range reposMap {
		// Check if there are actual commits in the last 7 days across all branches
		hasRecentCommits, err := c.hasCommitsSince(ctx, repoInfo.FullName, since)
		if err != nil {
			log.Printf("Warning: Could not check commits for %s: %v", repoInfo.FullName, err)
			continue
		}

		if hasRecentCommits {
			repos = append(repos, *repoInfo)
		}
	}

	return repos, nil
}

func (c *Client) getUserRepositories(ctx context.Context, username string, since time.Time) ([]*RepositoryInfo, error) {
	var repos []*RepositoryInfo

	opt := &gh.RepositoryListByUserOptions{
		Type:        "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		userRepos, resp, err := c.client.Repositories.ListByUser(ctx, username, opt)
		if err != nil {
			return nil, fmt.Errorf("error listing user repositories: %v", err)
		}

		for _, repo := range userRepos {
			// Check if repository has been updated recently
			if repo.UpdatedAt != nil && repo.UpdatedAt.After(since) {
				repoInfo := &RepositoryInfo{
					Name:        repo.GetName(),
					FullName:    repo.GetFullName(),
					Description: repo.GetDescription(),
					URL:         repo.GetHTMLURL(),
					LastPush:    repo.GetPushedAt().Time,
					Language:    repo.GetLanguage(),
				}
				repos = append(repos, repoInfo)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return repos, nil
}

func (c *Client) getOrganizationRepositories(ctx context.Context, username string, since time.Time) ([]*RepositoryInfo, error) {
	var repos []*RepositoryInfo

	// First, get user's organizations
	orgs, _, err := c.client.Organizations.List(ctx, username, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("error listing organizations: %v", err)
	}

	// For each organization, get repositories where user has access
	for _, org := range orgs {
		opt := &gh.RepositoryListByOrgOptions{
			Type:        "all", // Include all types (public, private, internal)
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: gh.ListOptions{PerPage: 100},
		}

		for {
			orgRepos, resp, err := c.client.Repositories.ListByOrg(ctx, org.GetLogin(), opt)
			if err != nil {
				log.Printf("Warning: Could not list repositories for org %s: %v", org.GetLogin(), err)
				break
			}

			for _, repo := range orgRepos {
				// Check if repository has been updated recently
				if repo.UpdatedAt != nil && repo.UpdatedAt.After(since) {
					repoInfo := &RepositoryInfo{
						Name:        repo.GetName(),
						FullName:    repo.GetFullName(),
						Description: repo.GetDescription(),
						URL:         repo.GetHTMLURL(),
						LastPush:    repo.GetPushedAt().Time,
						Language:    repo.GetLanguage(),
					}
					repos = append(repos, repoInfo)
				}
			}

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	return repos, nil
}

func (c *Client) hasCommitsSince(ctx context.Context, fullRepoName string, since time.Time) (bool, error) {
	// Parse owner and repo from full name
	owner, repo := c.parseRepoName(fullRepoName)

	opt := &gh.CommitsListOptions{
		Since:       since,
		ListOptions: gh.ListOptions{PerPage: 1},
	}

	commits, _, err := c.client.Repositories.ListCommits(ctx, owner, repo, opt)
	if err != nil {
		return false, err
	}

	return len(commits) > 0, nil
}

func (c *Client) parseRepoName(fullName string) (owner, repo string) {
	// Simple parsing - assumes format "owner/repo"
	// This could be enhanced to handle edge cases
	for i, char := range fullName {
		if char == '/' {
			return fullName[:i], fullName[i+1:]
		}
	}
	return "", fullName
}

// InteractiveRepositorySelector allows users to select a repository and view its commits
func (c *Client) InteractiveRepositorySelector(ctx context.Context, username string, since time.Time) error {
	// Get repositories with recent commits
	repos, err := c.GetRepositoriesWithRecentCommits(ctx, username, since)
	if err != nil {
		return fmt.Errorf("error fetching repositories: %v", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found with commits in the last 7 days.")
		return nil
	}

	// Display repositories for selection
	fmt.Printf("Found %d repositories with commits in the last 7 days:\n\n", len(repos))
	for i, repo := range repos {
		fmt.Printf("%d. %s\n", i+1, repo.FullName)
		if repo.Description != "" {
			fmt.Printf("   Description: %s\n", repo.Description)
		}
		fmt.Printf("   Language: %s\n", repo.Language)
		fmt.Printf("   Last Push: %s\n", repo.LastPush.Format("2006-01-02 15:04:05"))
		fmt.Printf("   URL: %s\n", repo.URL)
		fmt.Println()
	}

	// Get user selection
	var selection int
	fmt.Print("Enter the number of the repository to view commits (or 0 to exit): ")
	fmt.Scanf("%d", &selection)

	if selection <= 0 || selection > len(repos) {
		fmt.Println("Invalid selection or exit requested.")
		return nil
	}

	selectedRepo := repos[selection-1]
	fmt.Printf("\nFetching commits for %s...\n\n", selectedRepo.FullName)

	// Get detailed commits for the selected repository
	commits, err := c.GetRepositoryCommits(ctx, selectedRepo.FullName, since)
	if err != nil {
		return fmt.Errorf("error fetching commits: %v", err)
	}

	// Display commits
	c.DisplayCommits(commits, selectedRepo.FullName)

	return nil
}

// GetRepositoryCommits fetches detailed commit information for a repository from all branches
func (c *Client) GetRepositoryCommits(ctx context.Context, fullRepoName string, since time.Time) ([]CommitInfo, error) {
	var commits []CommitInfo
	owner, repo := c.parseRepoName(fullRepoName)

	// Get all branches first
	branches, _, err := c.client.Repositories.ListBranches(ctx, owner, repo, &gh.BranchListOptions{ListOptions: gh.ListOptions{PerPage: 100}})
	if err != nil {
		return nil, fmt.Errorf("error listing branches: %v", err)
	}

	// Check commits from all branches
	for _, branch := range branches {
		opt := &gh.CommitsListOptions{
			Since:       since,
			SHA:         branch.GetName(), // Check commits from this specific branch
			ListOptions: gh.ListOptions{PerPage: 100},
		}

		for {
			ghCommits, resp, err := c.client.Repositories.ListCommits(ctx, owner, repo, opt)
			if err != nil {
				log.Printf("Warning: Could not list commits for branch %s: %v", branch.GetName(), err)
				break
			}

			for _, commit := range ghCommits {
				// Skip if we already have this commit (from another branch)
				if c.commitExists(commits, commit.GetSHA()) {
					continue
				}

				// Get detailed commit information
				detailedCommit, _, err := c.client.Repositories.GetCommit(ctx, owner, repo, commit.GetSHA(), &gh.ListOptions{})
				if err != nil {
					log.Printf("Warning: Could not get detailed commit info for %s: %v", commit.GetSHA(), err)
					continue
				}

				// Extract file names
				var files []string
				for _, file := range detailedCommit.Files {
					files = append(files, file.GetFilename())
				}

				commitInfo := CommitInfo{
					SHA:       commit.GetSHA(),
					Message:   commit.GetCommit().GetMessage(),
					Author:    commit.GetCommit().GetAuthor().GetName(),
					Date:      commit.GetCommit().GetAuthor().GetDate().Time,
					Files:     files,
					Additions: detailedCommit.GetStats().GetAdditions(),
					Deletions: detailedCommit.GetStats().GetDeletions(),
				}

				commits = append(commits, commitInfo)
			}

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	return commits, nil
}

// commitExists checks if a commit with the given SHA already exists in the list
func (c *Client) commitExists(commits []CommitInfo, sha string) bool {
	for _, commit := range commits {
		if commit.SHA == sha {
			return true
		}
	}
	return false
}

// DisplayCommits shows formatted commit information
func (c *Client) DisplayCommits(commits []CommitInfo, repoName string) {
	if len(commits) == 0 {
		fmt.Printf("No commits found for %s in the last 7 days.\n", repoName)
		return
	}

	fmt.Printf("=== Commits for %s (Last 7 days) ===\n\n", repoName)

	for i, commit := range commits {
		fmt.Printf("Commit %d:\n", i+1)
		fmt.Printf("  SHA: %s\n", commit.SHA[:8]) // Show first 8 characters
		fmt.Printf("  Message: %s\n", commit.Message)
		fmt.Printf("  Author: %s\n", commit.Author)
		fmt.Printf("  Date: %s\n", commit.Date.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Changes: +%d -%d lines\n", commit.Additions, commit.Deletions)

		if len(commit.Files) > 0 {
			fmt.Printf("  Files changed:\n")
			for _, file := range commit.Files {
				fmt.Printf("    - %s\n", file)
			}
		}
		fmt.Println()
	}
}
