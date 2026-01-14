package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key="

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
