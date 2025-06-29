# Daily Go - GitHub Repository Activity Viewer

A TUI (Terminal User Interface) application written in Go that shows GitHub repositories that have been recently updated and generates AI-powered summaries of commits suitable for daily standups or team meetings.

## Features

- 🔍 **Smart Day Selection**: Choose from the last 7 days with formatted dates (e.g., "June, 10th")
- 📊 **Activity-Based Filtering**: Only shows repositories updated within the selected time period
- 🤖 **AI Summary Generation**: Uses Google Gemini to create professional commit summaries
- 🎨 **Beautiful TUI**: Built with Bubble Tea and Lipgloss for a modern terminal experience
- ⚙️ **Configurable**: Customize settings via environment variables
- 📱 **Pagination**: Navigate through repositories efficiently
- 🔄 **Async Operations**: Non-blocking commit fetching and AI generation

## Project Structure

```
daily-go/
├── main.go          # Main application logic and TUI interface
├── github.go        # GitHub API integration and repository management
├── llm.go           # Google Gemini AI service for summaries
├── types.go         # Shared types and data structures
├── config.go        # Configuration management
├── README.md        # Documentation
└── Makefile         # Build and development commands
```

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
   Create a `.env` file in the project root:
   ```env
   GITHUB_ACCESS_TOKEN=your_github_personal_access_token_here
   GITHUB_USERNAME=your_github_username_here
   GEMINI_API_KEY=your_gemini_api_key_here
   
   # Optional configuration
   DAYS_BACK=7      # Number of days to look back (default: 7)
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
make build
./daily-go
```

### Navigation

- **Arrow keys** or **h/j/k/l**: Navigate through options
- **Space**: Select/deselect repositories
- **Enter**: Continue to next step
- **b**: Go back to day selection
- **q**: Quit the application

### Workflow

1. **Day Selection**: Choose a time period from the last 7 days (e.g., "June, 10th")
2. **Repository Selection**: Browse repositories updated in the selected time period
3. **Repository Selection**: Use space to select repositories you want to analyze
4. **Commit Loading**: App fetches recent commits from selected repositories
5. **Commit Review**: Review the found commits grouped by repository
6. **AI Summary**: Generate an AI-powered summary suitable for meetings
7. **Exit**: Press enter to exit after viewing the summary

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GITHUB_ACCESS_TOKEN` | GitHub Personal Access Token | Required |
| `GITHUB_USERNAME` | GitHub username | Required |
| `GEMINI_API_KEY` | Google Gemini API Key | Required |
| `DAYS_BACK` | Number of days to look back for updates | 7 |
| `PER_PAGE` | Number of repositories per page | 10 |

## Example Output

**Day Selection:**
```
Select a time period to view repositories:

> June, 10th
  June, 9th
  June, 8th
  June, 7th
  June, 6th
  June, 5th
  June, 4th

Press enter to continue or q to quit.
```

**Repository List:**
```
Repositories updated in the last 3 days:

> [ ] my-repo (Go) - Updated: 2024-06-10
  [ ] another-repo (JS) - Updated: 2024-06-09

Showing 2 repositories updated in the last 3 days
Press space to select, enter to continue, b to go back, q to quit.
```

**AI Summary:**
```
🤖 AI Generated Summary:

📊 Recent Development Summary

Repository: my-repo
• Added new authentication middleware
• Fixed critical security vulnerability
• Implemented user role management

Repository: another-repo
• Updated dependency versions
• Added comprehensive test coverage
• Improved error handling

Key Achievements:
- Enhanced security with new auth system
- Improved code quality with better testing
- Updated dependencies for better stability
```

## Architecture

The application is built with a modular architecture:

- **`main.go`**: Contains the TUI interface using Bubble Tea framework
- **`github.go`**: Handles all GitHub API interactions and repository management
- **`llm.go`**: Manages Google Gemini AI integration for summary generation
- **`types.go`**: Defines shared data structures and types
- **`config.go`**: Handles configuration loading and validation

## Dependencies

- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/joho/godotenv` - Environment variable loading

## License

MIT License

