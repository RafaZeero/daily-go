package main

import (
	"context"
	"daily-go/github"
	"log"
	"os"
	"time"

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

	// Run interactive repository selector
	if err := client.InteractiveRepositorySelector(ctx, username, sevenDaysAgo); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
