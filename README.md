# lazyslackui

Lazy Slack!

# Slack TUI with Bubbletea

A text-based user interface for Slack built with Go and the Bubbletea framework. This application provides a lightweight alternative to the Slack desktop client, focusing on essential features like viewing messages, changing status, and sending preset messages.

## Features

- View recent Slack messages across multiple channels
- Quickly change your Slack status (Active, Away, Do Not Disturb)
- Send preset messages with a single action
- Keyboard-driven navigation for efficient workflow

## Requirements

- Go 1.16 or higher
- Slack API token with appropriate scopes
- Terminal with support for TUI applications

## Installation

1. Clone this repository:

```sh
git clone https://github.com/yourusername/slack-tui
cd slack-tui
```

2. Install dependencies:

```sh
go mod init slack-tui
go mod tidy
```

3. Build the application:

```sh
go build -o slack-tui .
```

## Setting up Slack API Token

1. Go to [Slack API Apps page](https://api.slack.com/apps)
2. Create a new app
3. Add the following scopes to your app:
   - `channels:history`
   - `channels:read`
   - `chat:write`
   - `groups:history`
   - `groups:read`
   - `users:read`
   - `users:write`
   - `users.profile:write`
4. Install the app to your workspace
5. Copy the OAuth Access Token

## Usage

1. Set your Slack API token as an environment variable:

```sh
export SLACK_TOKEN=xoxp-your-token-here
```

2. Run the application:

```sh
./slack-tui
```

## Keyboard Shortcuts

- `↑/↓`: Navigate through options
- `Enter`: Select the highlighted option
- `Esc`: Go back to the main menu
- `q` or `Ctrl+C`: Quit the application

## Customization

### Adding Custom Preset Messages

To add your own preset messages, modify the `presetMessages` list in the `initialModel()` function:

```go
presetMessages := []list.Item{
    QuickAction{
        name:        "Your Custom Message Name",
        description: "Your custom message text here",
    },
    // Add more preset messages here
}
```

### Modifying Colors and Styles

The application uses Lipgloss for styling. You can customize the appearance by modifying the color constants and style variables at the top of the file.

## Project Structure

- `main.go`: Contains the entire application code
  - Model definitions and initialization
  - Slack API integration
  - TUI rendering and event handling
  - Message formatting and display logic

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea): TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles): UI components for Bubbletea
- [Lipgloss](https://github.com/charmbracelet/lipgloss): Style definitions for terminal applications
- [slack-go](https://github.com/slack-go/slack): Slack API client for Go

## License

MIT
