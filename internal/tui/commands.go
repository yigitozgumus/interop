package tui

import (
	"fmt"
	"interop/internal/settings"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// CommandItem represents a command for the list
type CommandItem struct {
	name         string
	description  string
	cmd          string
	isEnabled    bool
	isExecutable bool
	arguments    []settings.CommandArgument
	examples     []settings.CommandExample
	preExec      []string
	postExec     []string
}

func (i CommandItem) FilterValue() string { return i.name }
func (i CommandItem) Title() string       { return i.name }
func (i CommandItem) Description() string {
	if i.description != "" {
		return i.description
	}
	return "No description"
}

// KeyMap defines key bindings
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Search key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "focus left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "focus right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "execute command"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

// Model represents the state of the TUI
type Model struct {
	cfg              *settings.Settings
	list             list.Model
	searchInput      textinput.Model
	detailViewport   viewport.Model
	selectedCommand  *CommandItem
	width            int
	height           int
	focusedPanel     int // 0 = list, 1 = search, 2 = details
	searchMode       bool
	showHelp         bool
	originalCommands []list.Item
	filteredCommands []list.Item
}

// NewCommandsModel creates a new TUI model for commands
func NewCommandsModel(cfg *settings.Settings) Model {
	// Create command items
	var items []list.Item
	for name, cmd := range cfg.Commands {
		item := CommandItem{
			name:         name,
			description:  cmd.Description,
			cmd:          cmd.Cmd,
			isEnabled:    cmd.IsEnabled,
			isExecutable: cmd.IsExecutable,
			arguments:    cmd.Arguments,
			examples:     cmd.Examples,
			preExec:      cmd.PreExec,
			postExec:     cmd.PostExec,
		}
		items = append(items, item)
	}

	// Create list
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Commands"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We'll handle filtering manually
	l.SetShowHelp(false)

	// Create search input
	ti := textinput.New()
	ti.Placeholder = "Search commands..."
	ti.CharLimit = 100
	ti.Width = 50

	// Create detail viewport
	vp := viewport.New(0, 0)

	m := Model{
		cfg:              cfg,
		list:             l,
		searchInput:      ti,
		detailViewport:   vp,
		focusedPanel:     0,
		searchMode:       false,
		showHelp:         false,
		originalCommands: items,
		filteredCommands: items,
	}

	// Set initial selection
	if len(items) > 0 {
		cmdItem := items[0].(CommandItem)
		m.selectedCommand = &cmdItem
		m.updateDetailView()
	}

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case tea.KeyMsg:
		if m.searchMode {
			return m.updateSearchMode(msg)
		}
		return m.updateNormalMode(msg)
	}

	// Update components
	if !m.searchMode {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update selected command when list selection changes
		if selected := m.list.SelectedItem(); selected != nil {
			if cmdItem, ok := selected.(CommandItem); ok {
				m.selectedCommand = &cmdItem
				m.updateDetailView()
			}
		}
	}

	m.detailViewport, cmd = m.detailViewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateSearchMode handles input in search mode
func (m Model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.searchMode = false
		m.searchInput.Blur()
		m.focusedPanel = 0
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.searchMode = false
		m.searchInput.Blur()
		m.focusedPanel = 0
		m.filterCommands(m.searchInput.Value())
		return m, nil

	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterCommands(m.searchInput.Value())
		return m, cmd
	}
}

// updateNormalMode handles input in normal mode
func (m Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, keys.Search):
		m.searchMode = true
		m.searchInput.Focus()
		m.focusedPanel = 1
		return m, nil

	case key.Matches(msg, keys.Enter):
		if m.selectedCommand != nil {
			return m, m.executeCommand(*m.selectedCommand)
		}
		return m, nil

	case key.Matches(msg, keys.Left):
		if m.focusedPanel > 0 {
			m.focusedPanel--
		}
		return m, nil

	case key.Matches(msg, keys.Right):
		if m.focusedPanel < 2 {
			m.focusedPanel++
		}
		return m, nil

	case key.Matches(msg, keys.Up), key.Matches(msg, keys.Down):
		// Forward up/down keys to the list when in command list panel
		if m.focusedPanel == 0 {
			m.list, cmd = m.list.Update(msg)
			// Update selected command when list selection changes
			if selected := m.list.SelectedItem(); selected != nil {
				if cmdItem, ok := selected.(CommandItem); ok {
					m.selectedCommand = &cmdItem
					m.updateDetailView()
				}
			}
			return m, cmd
		}
		return m, nil

	default:
		// Forward other keys to the list for navigation (j, k, page up/down, etc.)
		if m.focusedPanel == 0 {
			m.list, cmd = m.list.Update(msg)
			// Update selected command when list selection changes
			if selected := m.list.SelectedItem(); selected != nil {
				if cmdItem, ok := selected.(CommandItem); ok {
					m.selectedCommand = &cmdItem
					m.updateDetailView()
				}
			}
			return m, cmd
		}
	}

	return m, nil
}

// filterCommands filters the command list based on search query
func (m *Model) filterCommands(query string) {
	if query == "" {
		m.filteredCommands = m.originalCommands
	} else {
		var filtered []list.Item
		query = strings.ToLower(query)

		for _, item := range m.originalCommands {
			cmd := item.(CommandItem)
			if strings.Contains(strings.ToLower(cmd.name), query) ||
				strings.Contains(strings.ToLower(cmd.description), query) {
				filtered = append(filtered, item)
			}
		}
		m.filteredCommands = filtered
	}

	m.list.SetItems(m.filteredCommands)
	if len(m.filteredCommands) > 0 {
		m.list.Select(0)
		cmdItem := m.filteredCommands[0].(CommandItem)
		m.selectedCommand = &cmdItem
		m.updateDetailView()
	}
}

// updateDetailView updates the content of the detail viewport
func (m *Model) updateDetailView() {
	if m.selectedCommand == nil {
		m.detailViewport.SetContent("No command selected")
		return
	}

	cmd := *m.selectedCommand
	var content strings.Builder

	// Command name and status
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Underline(true)
	content.WriteString(nameStyle.Render(cmd.name))
	content.WriteString("\n\n")

	// Status indicators
	statusColor := "46" // Green
	statusIcon := "✓"
	statusText := "Enabled"
	if !cmd.isEnabled {
		statusColor = "196" // Red
		statusIcon = "✗"
		statusText = "Disabled"
	}

	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
	status := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, statusText))

	execType := "Shell Script"
	if cmd.isExecutable {
		execType = "Executable"
	}

	typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	execTypeFormatted := typeStyle.Render(execType)

	content.WriteString(fmt.Sprintf("Status: %s  |  Type: %s\n\n", status, execTypeFormatted))

	// Description
	if cmd.description != "" {
		sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		content.WriteString(sectionStyle.Render("Description:"))
		content.WriteString("\n")
		content.WriteString(cmd.description)
		content.WriteString("\n\n")
	}

	// Arguments
	if len(cmd.arguments) > 0 {
		sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		content.WriteString(sectionStyle.Render("Arguments:"))
		content.WriteString("\n")
		for _, arg := range cmd.arguments {
			requiredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			optionalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))

			if arg.Required {
				content.WriteString(requiredStyle.Render(fmt.Sprintf("  • %s (required)", arg.Name)))
			} else {
				content.WriteString(optionalStyle.Render(fmt.Sprintf("  • %s (optional)", arg.Name)))
			}

			if arg.Description != "" {
				content.WriteString(fmt.Sprintf(": %s", arg.Description))
			}
			if arg.Default != nil && arg.Default != "" {
				defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
				content.WriteString(defaultStyle.Render(fmt.Sprintf(" [default: %v]", arg.Default)))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Examples
	if len(cmd.examples) > 0 {
		sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		content.WriteString(sectionStyle.Render("Examples:"))
		content.WriteString("\n")
		for _, example := range cmd.examples {
			if example.Description != "" {
				exampleDescStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
				content.WriteString(exampleDescStyle.Render(fmt.Sprintf("  %s:", example.Description)))
				content.WriteString("\n")
			}
			codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			content.WriteString(codeStyle.Render(fmt.Sprintf("    %s", example.Command)))
			content.WriteString("\n\n")
		}
	}

	// Pre-execution hooks
	if len(cmd.preExec) > 0 {
		sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		content.WriteString(sectionStyle.Render("Pre-execution hooks:"))
		content.WriteString("\n")
		hookStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
		for i, hook := range cmd.preExec {
			content.WriteString(fmt.Sprintf("  %d. ", i+1))
			content.WriteString(hookStyle.Render(hook))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Post-execution hooks
	if len(cmd.postExec) > 0 {
		sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		content.WriteString(sectionStyle.Render("Post-execution hooks:"))
		content.WriteString("\n")
		hookStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
		for i, hook := range cmd.postExec {
			content.WriteString(fmt.Sprintf("  %d. ", i+1))
			content.WriteString(hookStyle.Render(hook))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Command content
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	content.WriteString(sectionStyle.Render("Command:"))
	content.WriteString("\n")

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	if cmd.isExecutable {
		content.WriteString(codeStyle.Render(cmd.cmd))
	} else {
		// For shell scripts, format each line
		lines := strings.Split(cmd.cmd, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				content.WriteString(codeStyle.Render(line))
				content.WriteString("\n")
			}
		}
	}

	m.detailViewport.SetContent(content.String())
}

// executeCommand executes the selected command
func (m Model) executeCommand(cmd CommandItem) tea.Cmd {
	return tea.ExecProcess(exec.Command("bash", "-c", cmd.cmd), func(err error) tea.Msg {
		if err != nil {
			return fmt.Sprintf("Error executing command: %v", err)
		}
		return "Command executed successfully"
	})
}

// updateSizes updates the sizes of components based on terminal size
func (m *Model) updateSizes() {
	// Calculate available space for content
	availableWidth := m.width - 4   // Account for outer margins
	availableHeight := m.height - 4 // Account for help text

	// Split width for two columns (give right column a bit more space)
	leftWidth := int(float64(availableWidth) * 0.45) // 45% for left column
	rightWidth := availableWidth - leftWidth - 2     // Rest for right column (minus gap)
	contentHeight := availableHeight - 2             // Account for margins

	// List height should account for search bar (3 lines: search + border + spacing)
	listHeight := contentHeight - 6
	if listHeight < 5 {
		listHeight = 5 // Minimum height
	}

	m.list.SetSize(leftWidth-6, listHeight)
	m.detailViewport.Width = rightWidth - 4
	m.detailViewport.Height = contentHeight - 4
	m.searchInput.Width = leftWidth - 16 // Account for "Search: " label and padding
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing TUI..."
	}

	var view strings.Builder

	// Main content - two columns
	leftColumn := m.renderLeftColumn()
	rightColumn := m.renderRightColumn()

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		" ", // Add small gap between columns
		rightColumn,
	)
	view.WriteString(columns)

	// Help text
	if m.showHelp {
		view.WriteString("\n")
		view.WriteString(m.renderHelp())
	} else {
		helpText := "Press ? for help, / to search, Enter to execute, q to quit"
		view.WriteString("\n")
		view.WriteString(helpStyle.Width(m.width).Align(lipgloss.Center).Render(helpText))
	}

	return view.String()
}

// renderLeftColumn renders the command list column with search bar
func (m Model) renderLeftColumn() string {
	style := columnStyle
	if m.focusedPanel == 0 || m.focusedPanel == 1 {
		style = selectedStyle
	}

	availableWidth := m.width - 4
	leftWidth := int(float64(availableWidth) * 0.45)
	contentHeight := m.height - 4

	// Create search bar
	searchLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render("Search: ")

	var searchContent string
	if m.searchMode {
		// In search mode, show the active input
		searchContent = m.searchInput.View()
	} else {
		// Not in search mode, show current value or placeholder
		currentSearch := m.searchInput.Value()
		if currentSearch != "" {
			searchContent = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Render(currentSearch+" ") +
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Render("(press / to modify)")
		} else {
			searchContent = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Render("(press / to search)")
		}
	}

	searchBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(leftWidth - 8).
		Render(searchLabel + searchContent)

	// Combine search bar and list
	content := searchBar + "\n\n" + m.list.View()

	return style.Width(leftWidth).Height(contentHeight).Render(content)
}

// renderRightColumn renders the command details column
func (m Model) renderRightColumn() string {
	style := columnStyle
	if m.focusedPanel == 2 {
		style = selectedStyle
	}

	availableWidth := m.width - 4
	leftWidth := int(float64(availableWidth) * 0.45)
	rightWidth := availableWidth - leftWidth - 2 // Rest minus gap
	contentHeight := m.height - 4

	return style.Width(rightWidth).Height(contentHeight).Render(m.detailViewport.View())
}

// renderHelp renders the help text
func (m Model) renderHelp() string {
	help := []string{
		"Navigation:",
		"  ↑/k, ↓/j    Navigate list",
		"  ←/h, →/l    Switch panels",
		"  enter       Execute command",
		"  /           Search commands",
		"  ?           Toggle this help",
		"  q, ctrl+c   Quit",
		"",
		"Search mode:",
		"  Type to filter commands",
		"  enter       Apply filter",
		"  esc         Exit search",
	}

	return helpStyle.Render(strings.Join(help, "\n"))
}
