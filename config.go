package main

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GitHubToken string
	GeminiKey   string
	Username    string
	DaysBack    int
	PerPage     int
}

func LoadConfig() *Config {
	config := &Config{
		GitHubToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
		GeminiKey:   os.Getenv("GEMINI_API_KEY"),
		Username:    os.Getenv("GITHUB_USERNAME"),
		DaysBack:    7,  // Default to last 7 days
		PerPage:     10, // Default items per page
	}

	// Override defaults with environment variables if provided
	if daysStr := os.Getenv("DAYS_BACK"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			config.DaysBack = days
		}
	}

	if perPageStr := os.Getenv("PER_PAGE"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
			config.PerPage = perPage
		}
	}

	return config
}

func (c *Config) Validate() error {
	if c.GitHubToken == "" {
		return fmt.Errorf("GITHUB_ACCESS_TOKEN environment variable is required")
	}

	if c.GeminiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	if c.Username == "" {
		return fmt.Errorf("GITHUB_USERNAME environment variable is required")
	}

	return nil
}
