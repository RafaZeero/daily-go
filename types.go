package main

import "time"

// Commit-related types
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

// Gemini API types
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
