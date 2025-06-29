package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"daily-go/github"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get environment variables
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	username := os.Getenv("GITHUB_USERNAME")

	if accessToken == "" {
		log.Fatal("GITHUB_ACCESS_TOKEN environment variable is required")
	}
	if username == "" {
		log.Fatal("GITHUB_USERNAME environment variable is required")
	}

	// Create GitHub client
	ctx := context.Background()
	client := github.NewClient(accessToken)

	// Calculate date 7 days ago
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	// Get repositories with recent activity
	repos, err := client.GetRepositoriesWithRecentCommits(ctx, username, sevenDaysAgo)
	if err != nil {
		log.Fatalf("Error fetching repositories: %v", err)
	}

	// Display results
	if len(repos) == 0 {
		fmt.Println("No repositories found with commits in the last 7 days.")
		return
	}

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
}
