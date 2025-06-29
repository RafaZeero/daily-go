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

	// Get user's repositories
	opt := &gh.RepositoryListByUserOptions{
		Type:        "owner",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		userRepos, resp, err := c.client.Repositories.ListByUser(ctx, username, opt)
		if err != nil {
			return nil, fmt.Errorf("error listing repositories: %v", err)
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

				// Only add if we haven't seen this repo before
				if _, exists := reposMap[repo.GetFullName()]; !exists {
					reposMap[repo.GetFullName()] = repoInfo
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Convert map to slice and filter by actual commits
	for _, repoInfo := range reposMap {
		// Check if there are actual commits in the last 7 days
		hasRecentCommits, err := c.hasCommitsSince(ctx, username, repoInfo.Name, since)
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

func (c *Client) hasCommitsSince(ctx context.Context, owner, repo string, since time.Time) (bool, error) {
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
