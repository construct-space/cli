package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type SpinnerDoneMsg struct {
	Err error
}

type SpinnerModel struct {
	spinner spinner.Model
	title   string
	done    bool
	err     error
	action  func() error
}

func NewSpinner(title string, action func() error) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Purple)

	return SpinnerModel{
		spinner: s,
		title:   title,
		action:  action,
	}
}

func (m SpinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := m.action()
			return SpinnerDoneMsg{Err: err}
		},
	)
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case SpinnerDoneMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SpinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return Error(fmt.Sprintf("%s — failed: %s", m.title, m.err)) + "\n"
		}
		return Success(m.title) + "\n"
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.title)
}

// RunWithSpinner runs a task with a spinner UI.
// Falls back to plain output when no TTY is available (CI, pipes, etc.)
func RunWithSpinner(title string, action func() error) error {
	// Check if we have a TTY for the spinner
	if !isTerminal(os.Stdin) {
		fmt.Println(Info(title))
		if err := action(); err != nil {
			fmt.Println(Error(fmt.Sprintf("%s — failed: %s", title, err)))
			return err
		}
		fmt.Println(Success(title))
		return nil
	}

	model := NewSpinner(title, action)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := finalModel.(SpinnerModel); ok && m.err != nil {
		return m.err
	}
	return nil
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
