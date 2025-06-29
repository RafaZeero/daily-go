# Daily Go - GitHub Repository Tracker

A Go application that shows repositories from your GitHub account where you've pushed commits in the last 7 days.

## Features

- Fetches repositories from your GitHub account
- Filters repositories to only show those with commits in the last 7 days
- Displays repository information including name, description, language, and last push date
- Uses GitHub API with proper authentication

## Setup

### 1. Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# GitHub API Configuration
# Get your personal access token from: https://github.com/settings/tokens
# Make sure it has the 'repo' scope for private repositories
GITHUB_ACCESS_TOKEN=your_github_personal_access_token_here

# Your GitHub username
GITHUB_USERNAME=your_github_username_here
```

### 2. GitHub Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Give it a descriptive name
4. Select the following scopes:
   - `repo` (for private repositories)
   - `read:user` (for public user information)
5. Copy the generated token and add it to your `.env` file

## Usage

```bash
# Run the application
go run main.go
```

## Output Example

```
Found 3 repositories with commits in the last 7 days:

1. username/project-a
   Description: A sample project
   Language: Go
   Last Push: 2024-01-15 14:30:25
   URL: https://github.com/username/project-a

2. username/project-b
   Language: JavaScript
   Last Push: 2024-01-14 09:15:10
   URL: https://github.com/username/project-b

3. username/project-c
   Description: Another project
   Language: Python
   Last Push: 2024-01-13 16:45:30
   URL: https://github.com/username/project-c
```

## Dependencies

- `github.com/google/go-github/v62/github` - GitHub API client
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/charmbracelet/*` - UI components (already included)

## How it works

1. The application loads environment variables from a `.env` file or system environment
2. It authenticates with GitHub using your personal access token
3. Fetches all repositories owned by your account
4. Filters repositories to only include those updated in the last 7 days
5. For each candidate repository, it checks if there are actual commits in the last 7 days
6. Displays the filtered results with relevant information

## Error Handling

- Validates that required environment variables are set
- Handles API rate limiting gracefully
- Provides clear error messages for authentication issues
- Continues processing even if individual repositories can't be checked 