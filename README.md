# Daily Go - GitHub Commit Summary Generator

A TUI (Terminal User Interface) application written in Go that fetches the latest commits from selected GitHub repositories and generates AI-powered summaries suitable for daily standups or team meetings.

## Features

- 🔍 Browse and select GitHub repositories
- 📊 Fetch latest commits from selected repositories
- 🤖 Generate AI summaries using Google Gemini
- 🎨 Beautiful TUI interface with pagination
- ⚡ Fast and efficient with async operations
- ⚙️ Configurable settings via environment variables

## Prerequisites

- Go 1.23.4 or higher
- GitHub Personal Access Token
- Google Gemini API Key

## Setup

1. **Clone the repository**
   ```bash
   git clone <your-repo-url>
   cd daily-go
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Create environment file**
   Create a `.env` file in the project root with the following variables:
   ```env
   GITHUB_ACCESS_TOKEN=your_github_personal_access_token_here
   GITHUB_USERNAME=your_github_username_here
   GEMINI_API_KEY=your_gemini_api_key_here
   
   # Optional configuration
   DAYS_BACK=7      # Number of days to look back for commits (default: 7)
   PER_PAGE=10      # Number of repositories per page (default: 10)
   ```

4. **Get API Keys**

   **GitHub Personal Access Token:**
   - Go to GitHub Settings → Developer settings → Personal access tokens
   - Generate a new token with `repo` and `read:user` permissions
   - Copy the token to your `.env` file

   **Google Gemini API Key:**
   - Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
   - Create a new API key
   - Copy the key to your `.env` file

## Usage

Run the application:
```bash
go run main.go
```

Or build and run:
```bash
go build -o daily-go .
./daily-go
```

### Navigation

- **Arrow keys** or **h/j/k/l**: Navigate through repositories
- **Space**: Select/deselect repositories
- **Enter**: Continue to next step
- **q**: Quit the application

### Workflow

1. **Repository Selection**: Browse and select repositories you want to analyze
2. **Commit Loading**: The app fetches commits from the configured time period
3. **Commit Review**: Review the found commits grouped by repository
4. **AI Summary**: Generate an AI-powered summary suitable for meetings
5. **Exit**: Press enter to exit after viewing the summary

## Configuration

The application supports the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GITHUB_ACCESS_TOKEN` | GitHub Personal Access Token | Required |
| `GITHUB_USERNAME` | GitHub username | Required |
| `GEMINI_API_KEY` | Google Gemini API Key | Required |
| `DAYS_BACK` | Number of days to look back for commits | 7 |
| `PER_PAGE` | Number of repositories per page | 10 |

## Dependencies

- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/joho/godotenv` - Environment variable loading

## License

MIT License 