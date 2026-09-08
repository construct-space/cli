package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileChangeMsg struct {
	Path string
	Time time.Time
}

type ViteOutputMsg struct {
	Line string
}

type ViteErrorMsg struct {
	Err error
}

type DevModel struct {
	spinner     spinner.Model
	spaceID     string
	status      string
	lastChange  string
	lastBuild   time.Time
	viteOutput  []string
	errors      []string
	width       int
	height      int
	quitting    bool
}

func NewDevModel(spaceID string) DevModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Purple)

	return DevModel{
		spinner: s,
		spaceID: spaceID,
		status:  "Starting...",
	}
}

func (m DevModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m DevModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "c":
			m.viteOutput = nil
			m.errors = nil
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case FileChangeMsg:
		m.lastChange = msg.Path
		m.status = "Rebuilding..."
		return m, nil

	case ViteOutputMsg:
		m.viteOutput = append(m.viteOutput, msg.Line)
		if len(m.viteOutput) > 20 {
			m.viteOutput = m.viteOutput[len(m.viteOutput)-20:]
		}
		if strings.Contains(msg.Line, "built in") {
			m.status = "Watching"
			m.lastBuild = time.Now()
		}
		return m, nil

	case ViteErrorMsg:
		m.errors = append(m.errors, msg.Err.Error())
		m.status = "Error"
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m DevModel) View() string {
	if m.quitting {
		return DimStyle.Render("Shutting down...") + "\n"
	}

	var sb strings.Builder

	// Header
	header := TitleStyle.Render(fmt.Sprintf("  construct dev — %s", m.spaceID))
	sb.WriteString(header + "\n")

	// Status line
	statusIcon := m.spinner.View()
	if m.status == "Watching" {
		statusIcon = SuccessStyle.Render("●")
	} else if m.status == "Error" {
		statusIcon = ErrorStyle.Render("●")
	}
	sb.WriteString(fmt.Sprintf("  %s %s\n", statusIcon, m.status))

	// Last change
	if m.lastChange != "" {
		sb.WriteString(fmt.Sprintf("  %s %s\n", DimStyle.Render("Changed:"), m.lastChange))
	}

	// Last build time
	if !m.lastBuild.IsZero() {
		sb.WriteString(fmt.Sprintf("  %s %s\n", DimStyle.Render("Built:"), m.lastBuild.Format("15:04:05")))
	}

	sb.WriteString("\n")

	// Vite output
	if len(m.viteOutput) > 0 {
		sb.WriteString(DimStyle.Render("  Vite output:") + "\n")
		for _, line := range m.viteOutput {
			sb.WriteString(fmt.Sprintf("  %s\n", DimStyle.Render(line)))
		}
		sb.WriteString("\n")
	}

	// Errors
	if len(m.errors) > 0 {
		sb.WriteString(ErrorStyle.Render("  Errors:") + "\n")
		for _, e := range m.errors {
			sb.WriteString(fmt.Sprintf("  %s\n", ErrorStyle.Render(e)))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(DimStyle.Render("  q quit • c clear") + "\n")

	return sb.String()
}
