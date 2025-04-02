package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/slack-go/slack"
)

// Constants for styling
const (
	// Color definitions
	primaryColor   = lipgloss.Color("#6C8EBF")
	secondaryColor = lipgloss.Color("#DAE8FC")
	accentColor    = lipgloss.Color("#D5E8D4")
	errorColor     = lipgloss.Color("#F8CECC")

	// Layout constants
	headerHeight = 3
	footerHeight = 3
)

// Global styles
var (
	appStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	channelStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Bold(true)

	statusAwayStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	statusDNDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)
)

// SlackMessage represents a message in Slack
type SlackMessage struct {
	User    string
	Content string
	Channel string
	Time    time.Time
}

// QuickAction represents a quick action like changing status or sending a preset message
type QuickAction struct {
	name        string
	description string
}

// Implement the list.Item interface
func (q QuickAction) Title() string       { return q.name }
func (q QuickAction) Description() string { return q.description }
func (q QuickAction) FilterValue() string { return q.name + " " + q.description }

// Model represents the application state
type Model struct {
	width             int
	height            int
	slackClient       *slack.Client
	userID            string
	userName          string
	userStatus        string
	messages          []SlackMessage
	channels          []slack.Channel
	spinner           spinner.Model
	viewport          viewport.Model
	quickActions      list.Model
	presetMessages    list.Model
	statusOptions     list.Model
	textInput         textinput.Model
	isLoading         bool
	error             string
	currentPage       string
	selectedChannelID string
}

// Page constants
const (
	pageMain          = "main"
	pageMessages      = "messages"
	pageQuickActions  = "quick_actions"
	pagePresetMessage = "preset_message"
	pageSetStatus     = "set_status"
)

// Status constants
const (
	statusActive = "active"
	statusAway   = "away"
	statusDND    = "dnd"
)

// Initialize the application model
func initialModel() Model {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Initialize quick actions
	quickActions := []list.Item{
		QuickAction{
			name:        "View Messages",
			description: "View recent messages from Slack",
		},
		QuickAction{
			name:        "Set Status",
			description: "Change your Slack status",
		},
		QuickAction{
			name:        "Send Preset Message",
			description: "Send a pre-configured message",
		},
		QuickAction{
			name:        "Quit",
			description: "Exit the application",
		},
	}

	// Initialize preset messages
	presetMessages := []list.Item{
		QuickAction{
			name:        "Be Right Back",
			description: "I'll be right back, give me a few minutes.",
		},
		QuickAction{
			name:        "In a Meeting",
			description: "I'm currently in a meeting, will respond later.",
		},
		QuickAction{
			name:        "Working on Issue",
			description: "I'm working on the issue, will update you soon.",
		},
		QuickAction{
			name:        "Lunch Break",
			description: "I'm on lunch break, back in an hour.",
		},
	}

	// Initialize status options
	statusOptions := []list.Item{
		QuickAction{
			name:        "Active",
			description: "Set your status to active",
		},
		QuickAction{
			name:        "Away",
			description: "Set your status to away",
		},
		QuickAction{
			name:        "Do Not Disturb",
			description: "Set your status to do not disturb",
		},
	}

	// Initialize list delegates
	actionDelegate := list.NewDefaultDelegate()
	actionDelegate.Styles.SelectedTitle = actionDelegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("0")).
		Background(primaryColor).
		Bold(true)
	actionDelegate.Styles.SelectedDesc = actionDelegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("0")).
		Background(primaryColor)

	// Create the lists
	quickActionList := list.New(quickActions, actionDelegate, 0, 0)
	quickActionList.Title = "Quick Actions"
	quickActionList.SetShowHelp(false)

	presetMessageList := list.New(presetMessages, actionDelegate, 0, 0)
	presetMessageList.Title = "Preset Messages"
	presetMessageList.SetShowHelp(false)

	statusList := list.New(statusOptions, actionDelegate, 0, 0)
	statusList.Title = "Set Status"
	statusList.SetShowHelp(false)

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Type a channel name to filter..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	// Create the viewport
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor)

	// Initialize the model
	return Model{
		currentPage:    pageMain,
		spinner:        s,
		isLoading:      false,
		quickActions:   quickActionList,
		presetMessages: presetMessageList,
		statusOptions:  statusList,
		textInput:      ti,
		viewport:       vp,
		userStatus:     statusActive,
	}
}

// Initialize the Slack client
func (m *Model) initSlackClient() tea.Msg {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return errMsg("SLACK_TOKEN environment variable not set")
	}

	client := slack.New(token)
	rtm := client.NewRTM()
	go rtm.ManageConnection()

	// Get user info
	info := rtm.GetInfo()
	if info == nil {
		return errMsg("Failed to connect to Slack. Check your token.")
	}

	// Get channels
	channels, _, err := client.GetConversations(&slack.GetConversationsParameters{
		ExcludeArchived: true,
		Types:           []string{"public_channel", "private_channel"},
	})
	if err != nil {
		return errMsg(fmt.Sprintf("Error getting channels: %v", err))
	}

	return initMsg{
		client:   client,
		userID:   info.User.ID,
		userName: info.User.Name,
		channels: channels,
	}
}

// Get recent messages from Slack
func (m *Model) fetchMessages() tea.Msg {
	if m.slackClient == nil {
		return errMsg("Slack client not initialized")
	}

	var messages []SlackMessage

	// If no channel is selected, get messages from all channels
	if m.selectedChannelID == "" {
		// Limit to 5 most recent channels to avoid rate limits
		channelLimit := 5
		if len(m.channels) < channelLimit {
			channelLimit = len(m.channels)
		}

		for i := 0; i < channelLimit; i++ {
			channel := m.channels[i]
			history, err := m.slackClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
				ChannelID: channel.ID,
				Limit:     3, // Get last 3 messages per channel
			})
			if err != nil {
				return errMsg(fmt.Sprintf("Error fetching messages: %v", err))
			}

			for j := len(history.Messages) - 1; j >= 0; j-- {
				msg := history.Messages[j]
				userName := "Unknown User"

				// Get username if it's not a bot message
				if msg.User != "" {
					user, err := m.slackClient.GetUserInfo(msg.User)
					if err == nil {
						userName = user.Name
					}
				}

				messages = append(messages, SlackMessage{
					User:    userName,
					Content: msg.Text,
					Channel: channel.Name,
					Time:    parseSlackTimestamp(msg.Timestamp),
				})
			}
		}
	} else {
		// Get messages for a specific channel
		history, err := m.slackClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: m.selectedChannelID,
			Limit:     10, // Get last 10 messages from selected channel
		})
		if err != nil {
			return errMsg(fmt.Sprintf("Error fetching messages: %v", err))
		}

		var channelName string
		for _, ch := range m.channels {
			if ch.ID == m.selectedChannelID {
				channelName = ch.Name
				break
			}
		}

		for j := len(history.Messages) - 1; j >= 0; j-- {
			msg := history.Messages[j]
			userName := "Unknown User"

			// Get username if it's not a bot message
			if msg.User != "" {
				user, err := m.slackClient.GetUserInfo(msg.User)
				if err == nil {
					userName = user.Name
				}
			}

			messages = append(messages, SlackMessage{
				User:    userName,
				Content: msg.Text,
				Channel: channelName,
				Time:    parseSlackTimestamp(msg.Timestamp),
			})
		}
	}

	return messagesMsg{messages: messages}
}

// Parse a Slack timestamp into a time.Time
func parseSlackTimestamp(timestamp string) time.Time {
	parts := strings.Split(timestamp, ".")
	if len(parts) != 2 {
		return time.Time{}
	}

	sec, err := fmt.Sscanf(parts[0], "%d", new(int64))
	if err != nil {
		return time.Time{}
	}

	return time.Unix(int64(sec), 0)
}

// Update the user's status
func (m *Model) setStatus(status string) tea.Msg {
	if m.slackClient == nil {
		return errMsg("Slack client not initialized")
	}

	var emojiText, statusText string

	switch status {
	case statusActive:
		emojiText = ":white_check_mark:"
		statusText = "Active"
	case statusAway:
		emojiText = ":away:"
		statusText = "Away"
	case statusDND:
		emojiText = ":no_entry:"
		statusText = "Do Not Disturb"
	default:
		return errMsg("Invalid status")
	}

	err := m.slackClient.SetUserPresence(status)
	if err != nil {
		return errMsg(fmt.Sprintf("Error setting presence: %v", err))
	}

	err = m.slackClient.SetUserCustomStatus(statusText, emojiText, 0)
	if err != nil {
		return errMsg(fmt.Sprintf("Error setting status: %v", err))
	}

	return statusUpdatedMsg{status: status}
}

// Send a preset message
func (m *Model) sendPresetMessage(message string) tea.Msg {
	if m.slackClient == nil {
		return errMsg("Slack client not initialized")
	}

	if m.selectedChannelID == "" {
		return errMsg("No channel selected")
	}

	_, timestamp, err := m.slackClient.PostMessage(
		m.selectedChannelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return errMsg(fmt.Sprintf("Error sending message: %v", err))
	}

	return messageSentMsg{
		channelID: m.selectedChannelID,
		timestamp: timestamp,
		text:      message,
	}
}

// Custom messages for our application
type initMsg struct {
	client   *slack.Client
	userID   string
	userName string
	channels []slack.Channel
}

type errMsg struct {
	err string
}

func (e errMsg) Error() string { return e.err }

type messagesMsg struct {
	messages []SlackMessage
}

type statusUpdatedMsg struct {
	status string
}

type messageSentMsg struct {
	channelID string
	timestamp string
	text      string
}

// Initialize the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		func() tea.Msg {
			m.isLoading = true
			return nil
		},
		m.initSlackClient,
	)
}

// Update the application state based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentPage == pageMain {
				return m, tea.Quit
			} else {
				m.currentPage = pageMain
				return m, nil
			}
		case "esc":
			if m.currentPage != pageMain {
				m.currentPage = pageMain
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions
		m.quickActions.SetSize(msg.Width-10, msg.Height-headerHeight-footerHeight)
		m.presetMessages.SetSize(msg.Width-10, msg.Height-headerHeight-footerHeight)
		m.statusOptions.SetSize(msg.Width-10, msg.Height-headerHeight-footerHeight)

		// Update viewport dimensions
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight

		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case initMsg:
		m.slackClient = msg.client
		m.userID = msg.userID
		m.userName = msg.userName
		m.channels = msg.channels
		m.isLoading = false

		// After initialization, fetch messages
		cmds = append(cmds, m.fetchMessages)

	case errMsg:
		m.error = msg.Error()
		m.isLoading = false

	case messagesMsg:
		m.messages = msg.messages
		m.isLoading = false

		// Update viewport with messages
		m.viewport.SetContent(m.formatMessages())

	case statusUpdatedMsg:
		m.userStatus = msg.status
		m.isLoading = false
		m.currentPage = pageMain

	case messageSentMsg:
		m.isLoading = false
		m.currentPage = pageMain

		// Refresh messages after sending
		cmds = append(cmds, m.fetchMessages)
	}

	// Handle page-specific updates
	switch m.currentPage {
	case pageMain:
		var cmd tea.Cmd
		m.quickActions, cmd = m.quickActions.Update(msg)
		cmds = append(cmds, cmd)

		// Handle selection on main page
		if _, ok := msg.(tea.KeyMsg); ok {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.String() == "enter" {
					i, ok := m.quickActions.SelectedItem().(QuickAction)
					if ok {
						switch i.name {
						case "View Messages":
							m.currentPage = pageMessages
							m.isLoading = true
							cmds = append(cmds, m.fetchMessages)
						case "Set Status":
							m.currentPage = pageSetStatus
						case "Send Preset Message":
							m.currentPage = pagePresetMessage
						case "Quit":
							return m, tea.Quit
						}
					}
				}
			}
		}

	case pageMessages:
		// Handle viewport scrolling
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case pageSetStatus:
		var cmd tea.Cmd
		m.statusOptions, cmd = m.statusOptions.Update(msg)
		cmds = append(cmds, cmd)

		// Handle status selection
		if _, ok := msg.(tea.KeyMsg); ok {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.String() == "enter" {
					i, ok := m.statusOptions.SelectedItem().(QuickAction)
					if ok {
						m.isLoading = true
						switch i.name {
						case "Active":
							cmds = append(cmds, func() tea.Msg {
								return m.setStatus(statusActive)
							})
						case "Away":
							cmds = append(cmds, func() tea.Msg {
								return m.setStatus(statusAway)
							})
						case "Do Not Disturb":
							cmds = append(cmds, func() tea.Msg {
								return m.setStatus(statusDND)
							})
						}
					}
				}
			}
		}

	case pagePresetMessage:
		var cmd tea.Cmd
		m.presetMessages, cmd = m.presetMessages.Update(msg)
		cmds = append(cmds, cmd)

		// Handle preset message selection
		if _, ok := msg.(tea.KeyMsg); ok {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if msg.String() == "enter" {
					i, ok := m.presetMessages.SelectedItem().(QuickAction)
					if ok {
						m.isLoading = true
						cmds = append(cmds, func() tea.Msg {
							return m.sendPresetMessage(i.description)
						})
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// Format messages for display
func (m Model) formatMessages() string {
	var sb strings.Builder

	if len(m.messages) == 0 {
		sb.WriteString("No messages found.")
		return sb.String()
	}

	for _, msg := range m.messages {
		sb.WriteString(fmt.Sprintf(
			"%s %s in #%s\n%s\n\n",
			channelStyle.Render(msg.Time.Format("15:04")),
			titleStyle.Render(msg.User),
			channelStyle.Render(msg.Channel),
			messageStyle.Render(msg.Content),
		))
	}

	return sb.String()
}

// Render the view based on current state
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var content string

	// Header displays user info and status
	header := fmt.Sprintf(
		"%s | %s",
		titleStyle.Render(fmt.Sprintf("Slack TUI - Logged in as: %s", m.userName)),
		func() string {
			switch m.userStatus {
			case statusActive:
				return statusActiveStyle.Render("● Active")
			case statusAway:
				return statusAwayStyle.Render("● Away")
			case statusDND:
				return statusDNDStyle.Render("● Do Not Disturb")
			default:
				return infoStyle.Render("● Unknown")
			}
		}(),
	)

	// Footer with help text
	footer := helpStyle.Render("q/ctrl+c: quit • esc: back • ↑/↓: navigate • enter: select")

	// Display error if any
	if m.error != "" {
		errorBox := errorStyle.Render(fmt.Sprintf("Error: %s", m.error))
		content = lipgloss.JoinVertical(lipgloss.Center, header, errorBox, footer)
		return appStyle.Render(content)
	}

	// Display loading spinner if loading
	if m.isLoading {
		loadingText := fmt.Sprintf("%s Loading...", m.spinner.View())
		content = lipgloss.JoinVertical(lipgloss.Center, header, loadingText, footer)
		return appStyle.Render(content)
	}

	// Content based on current page
	switch m.currentPage {
	case pageMain:
		content = lipgloss.JoinVertical(lipgloss.Center, header, m.quickActions.View(), footer)
	case pageMessages:
		content = lipgloss.JoinVertical(lipgloss.Center, header, m.viewport.View(), footer)
	case pageSetStatus:
		content = lipgloss.JoinVertical(lipgloss.Center, header, m.statusOptions.View(), footer)
	case pagePresetMessage:
		content = lipgloss.JoinVertical(lipgloss.Center, header, m.presetMessages.View(), footer)
	}

	return appStyle.Render(content)
}

func main() {
	// Initialize the model
	m := initialModel()

	// Start the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
