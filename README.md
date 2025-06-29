# Daily Go - GitHub Repository Tracker

A Go application that shows repositories from your GitHub account where you've pushed commits in the last 7 days, with interactive commit viewing.

## Features

- Fetches repositories from your GitHub account
- **Includes organization repositories** where you contribute
- **Checks all branches** - not just the main branch
- Filters repositories to only show those with commits in the last 7 days
- **Interactive repository selection** - Choose which repository to explore
- **Detailed commit information** - View commit messages, authors, dates, and changed files
- **File change tracking** - See which files were modified in each commit
- **Statistics** - View additions and deletions for each commit
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

## Interactive Workflow

1. **Repository List**: The app shows all repositories with commits in the last 7 days
2. **Selection**: Enter the number of the repository you want to explore
3. **Commit Details**: View detailed information about each commit including:
   - Commit SHA (shortened)
   - Commit message
   - Author name
   - Commit date and time
   - Number of lines added/removed
   - List of files that were changed

## Output Example

```
Found 2 repositories with commits in the last 7 days:

1. username/project-a
   Description: A sample project
   Language: Go
   Last Push: 2024-01-15 14:30:25
   URL: https://github.com/username/project-a

2. username/project-b
   Language: JavaScript
   Last Push: 2024-01-14 09:15:10
   URL: https://github.com/username/project-b

Enter the number of the repository to view commits (or 0 to exit): 1

Fetching commits for username/project-a...

=== Commits for username/project-a (Last 7 days) ===

Commit 1:
  SHA: a1b2c3d4
  Message: Add new feature for user authentication
  Author: John Doe
  Date: 2024-01-15 14:30:25
  Changes: +45 -12 lines
  Files changed:
    - src/auth.go
    - src/user.go
    - tests/auth_test.go

Commit 2:
  SHA: e5f6g7h8
  Message: Fix bug in login validation
  Author: John Doe
  Date: 2024-01-14 10:20:15
  Changes: +8 -3 lines
  Files changed:
    - src/auth.go
```

## Dependencies

- `github.com/google/go-github/v62/github` - GitHub API client
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/charmbracelet/*` - UI components (already included)

## How it works

1. The application loads environment variables from a `.env` file or system environment
2. It authenticates with GitHub using your personal access token
3. **Fetches repositories from multiple sources:**
   - Your owned repositories
   - Organization repositories where you have access
4. **Checks all branches** in each repository for recent activity
5. Filters repositories to only include those updated in the last 7 days
6. For each candidate repository, it checks if there are actual commits in the last 7 days across all branches
7. Displays the filtered results and prompts for user selection
8. When a repository is selected, fetches detailed commit information from all branches including:
   - Commit metadata (SHA, message, author, date)
   - File changes (which files were modified)
   - Statistics (lines added/removed)

## Error Handling

- Validates that required environment variables are set
- Handles API rate limiting gracefully
- Provides clear error messages for authentication issues
- Continues processing even if individual repositories can't be checked
- Gracefully handles missing commit details 