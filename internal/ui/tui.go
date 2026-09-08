package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/construct-space/cli/internal/auth"
	"github.com/construct-space/cli/internal/operator"
	"github.com/construct-space/cli/internal/shell"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUIApp struct {
	app    *tview.Application
	client *operator.Client
	state  *operator.SessionState

	transcript   *tview.TextView
	status       *tview.TextView
	sidebar      *tview.TextView
	questionText *tview.TextView
	options      *tview.List
	input        *tview.InputField

	busy          bool
	clientID      string
	currentStream string
	cancelStream  context.CancelFunc // cancels the active stream
	interview     *shell.ArchitectInterview
	multiSelected map[string]bool
	watchPath     string
	modelPicker   []string // non-nil when model picker is active
	modePicker    bool     // true when mode picker is active
	agentPicker   []string // non-nil when agent picker is active
	runningProcs  []runningProc // dev servers started by /run
	pendingInput  []string      // queued messages while busy
}

type runningProc struct {
	pid  int
	name string
	url  string
	cmd  string
}

// RunTUI connects to the operator and launches the interactive TUI.
func RunTUI(addr, clientID string) error {
	client, err := operator.NewClient(addr, clientID)
	if err != nil {
		return fmt.Errorf("connect to operator: %w", err)
	}
	defer client.Close()

	// Sync construct login token to operator if authenticated
	if creds, err := auth.LoadCredentials(); err == nil {
		operator.SyncAuthToken(client, creds.Token)
	}

	state := operator.NewSessionState()

	// Default project to current working directory
	if cwd, err := os.Getwd(); err == nil {
		state.Project.Path = cwd
		state.Project.Name = filepath.Base(cwd)
	}

	ui := newTUI(client, clientID, state)
	ui.syncMode(state.Mode)

	// Tell the operator about the project context
	if state.Project.Path != "" {
		go func() {
			client.Send("context.set_project", map[string]any{
				"name":     state.Project.Name,
				"rootPath": state.Project.Path,
			})
		}()
	}

	return ui.app.Run()
}

func newTUI(client *operator.Client, clientID string, state *operator.SessionState) *TUIApp {
	app := tview.NewApplication()
	transcript := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)
	transcript.SetBorder(true).SetTitle(" Transcript ")
	transcript.SetChangedFunc(func() {
		app.Draw()
	})

	status := tview.NewTextView().
		SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" Session ")

	questionText := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	questionText.SetBorder(true).SetTitle(" Question ")

	options := tview.NewList().
		ShowSecondaryText(true)
	options.SetBorder(true).SetTitle(" Options ")

	sidebarView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	sidebarView.SetBorder(true).SetTitle(" Details ")

	inputField := tview.NewInputField().
		SetLabel("> ").
		SetFieldWidth(0)
	inputField.SetBorder(true).SetTitle(" Input ")

	home, _ := os.UserHomeDir()
	defaultWatchPath := filepath.Join(home, "ConstructProjects")

	ui := &TUIApp{
		app:           app,
		client:        client,
		state:         state,
		transcript:    transcript,
		status:        status,
		sidebar:       sidebarView,
		questionText:  questionText,
		options:       options,
		input:         inputField,
		clientID:      clientID,
		multiSelected: map[string]bool{},
		watchPath:     defaultWatchPath,
	}

	options.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		ui.handleOptionSelect()
	})
	options.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.currentQuestion() == nil && len(ui.modelPicker) == 0 && len(ui.agentPicker) == 0 && !ui.modePicker {
			return event
		}
		if ui.currentQuestion() != nil && event.Key() == tcell.KeyRune && event.Rune() == ' ' && ui.currentQuestion().Type == "multi" {
			ui.toggleCurrentMultiOption()
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			if len(ui.modelPicker) > 0 {
				ui.modelPicker = nil
				ui.refresh()
			}
			if len(ui.agentPicker) > 0 {
				ui.agentPicker = nil
				ui.refresh()
			}
			if ui.modePicker {
				ui.modePicker = false
				ui.refresh()
			}
			ui.app.SetFocus(ui.input)
			return nil
		}
		return event
	})

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		value := strings.TrimSpace(inputField.GetText())
		if value == "" {
			return
		}
		inputField.SetText("")
		ui.handleInput(value)
	})

	side := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(questionText, 8, 0, false).
		AddItem(options, 0, 1, false).
		AddItem(sidebarView, 7, 0, false)

	body := tview.NewFlex().
		AddItem(transcript, 0, 3, false).
		AddItem(side, 0, 2, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(status, 4, 0, false).
		AddItem(body, 0, 1, false).
		AddItem(inputField, 3, 0, true)

	app.SetRoot(root, true)
	app.SetFocus(inputField)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyTAB:
			if ui.currentQuestion() == nil && len(ui.modelPicker) == 0 && len(ui.agentPicker) == 0 && !ui.modePicker {
				return event
			}
			if app.GetFocus() == inputField {
				app.SetFocus(options)
			} else {
				app.SetFocus(inputField)
			}
			return nil
		}
		return event
	})

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ui.app.QueueUpdateDraw(func() {
				ui.refresh()
			})
		}
	}()

	ui.refresh()
	ui.importProviderKeysFromAuth("", true)
	return ui
}

func (ui *TUIApp) handleInput(input string) {
	command, isCommand := shell.Parse(input)
	if isCommand {
		ui.handleCommand(command)
		return
	}

	// Intercept common run/dev requests and handle locally
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "run" || lower == "run project" || lower == "run current project" ||
		lower == "start dev" || lower == "start dev server" || lower == "dev" {
		ui.handleRunCommand()
		return
	}
	if ui.currentQuestion() != nil {
		ui.state.AddSystemMessage("Use the question panel to answer the active architect prompt.")
		ui.refresh()
		return
	}
	if ui.busy {
		ui.pendingInput = append(ui.pendingInput, input)
		ui.state.AddSystemMessage(fmt.Sprintf("queued (%d pending)", len(ui.pendingInput)))
		ui.refresh()
		return
	}

	switch ui.state.Mode {
	case operator.ModeChat:
		ui.startChat(input)
	case operator.ModeVibe:
		ui.startVibe(input)
	case operator.ModeAgent:
		ui.startAgent(input)
	default:
		ui.state.AddErrorMessage("unsupported mode")
		ui.refresh()
	}
}

func (ui *TUIApp) handleCommand(command shell.Command) {
	switch command.Name {
	case "help":
		ui.state.AddSystemMessage("Commands: /mode chat|vibe|agent, /project <abs-path>, /project clear, /project-name <name>, /projects-root <abs-path>, /model <name>, /agent <id>, /architect <prompt>, /vibe <goal>, /examples, /example show <id>, /example run <id>, /watch <abs-path>, /provider status, /provider import-auth [path], /provider set deepseek|mimo|zai|xai <key>, /provider clear deepseek|mimo|zai|xai, /resume <session-id>, /providers, /models, /agents, /transcript save <path>, /clear, /status")
		ui.refresh()
	case "mode":
		if ui.busy {
			ui.state.AddSystemMessage("Wait for the active request to finish before changing mode.")
			ui.refresh()
			return
		}
		if strings.TrimSpace(command.Arg) == "" {
			// Open mode picker
			ui.modePicker = true
			ui.refresh()
			ui.app.SetFocus(ui.options)
			return
		}
		if err := shell.SetMode(ui.state, command.Arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		ui.syncMode(ui.state.Mode)
		ui.state.AddSystemMessage("mode set to " + string(ui.state.Mode))
		ui.refresh()
	case "project":
		arg := strings.TrimSpace(command.Arg)
		if strings.EqualFold(arg, "clear") {
			shell.ClearProject(ui.state)
			ui.request("context.clear_project", nil, func(any) {
				ui.state.AddSystemMessage("project context cleared")
			})
			ui.refresh()
			return
		}
		if err := shell.SetProjectPath(ui.state, arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		ui.request("context.set_project", map[string]any{
			"name":     ui.state.Project.Name,
			"rootPath": ui.state.Project.Path,
		}, func(any) {
			ui.state.AddSystemMessage("project set to " + ui.state.Project.Path)
		})
		ui.refresh()
	case "project-name":
		if err := shell.SetProjectName(ui.state, command.Arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		if strings.TrimSpace(ui.state.Project.Path) != "" {
			ui.request("context.set_project", map[string]any{
				"name":     ui.state.Project.Name,
				"rootPath": ui.state.Project.Path,
			}, nil)
		}
		ui.state.AddSystemMessage("project name set to " + ui.state.Project.Name)
		ui.refresh()
	case "projects-root":
		if err := shell.SetProjectsRoot(ui.state, command.Arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		ui.state.AddSystemMessage("projects root set to " + ui.state.Project.ProjectsRoot)
		ui.refresh()
	case "model":
		if strings.TrimSpace(command.Arg) == "" {
			// No arg — open model picker in options panel
			ui.request("ai.models", nil, func(data any) {
				models := shell.ExtractModelIDs(data)
				if len(models) == 0 {
					ui.state.AddSystemMessage(shell.PrettyJSON(data))
					return
				}
				ui.modelPicker = models
				ui.refresh()
				ui.app.SetFocus(ui.options)
			})
			return
		}
		if err := shell.SetModel(ui.state, command.Arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		ui.state.AddSystemMessage("model set to " + ui.state.Model)
		ui.refresh()
	case "agent":
		if strings.TrimSpace(command.Arg) == "" {
			// Open agent picker
			ui.request("agents.list", nil, func(data any) {
				agents := shell.ExtractAgentIDs(data)
				if len(agents) == 0 {
					ui.state.AddSystemMessage(shell.PrettyJSON(data))
					return
				}
				ui.agentPicker = agents
				ui.refresh()
				ui.app.SetFocus(ui.options)
			})
			return
		}
		if err := shell.SetAgent(ui.state, command.Arg); err != nil {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
			return
		}
		ui.state.Mode = operator.ModeAgent
		ui.syncMode(ui.state.Mode)
		ui.state.AddSystemMessage("agent set to " + ui.state.AgentID)
		ui.refresh()
	case "architect":
		if ui.busy {
			ui.state.AddSystemMessage("A request is already running.")
			ui.refresh()
			return
		}
		ui.startArchitect(command.Arg)
	case "vibe":
		if ui.busy {
			ui.state.AddSystemMessage("A request is already running.")
			ui.refresh()
			return
		}
		ui.startVibe(command.Arg)
	case "examples":
		ui.state.AddSystemMessage(shell.FormatExamplesList())
		ui.refresh()
	case "example":
		ui.handleExampleCommand(command.Arg)
	case "watch", "monitor":
		ui.handleWatchCommand(command.Arg)
	case "provider":
		ui.handleProviderCommand(command.Arg)
	case "resume":
		ui.resumeVibe(command.Arg)
	case "providers":
		ui.request("providers.list", nil, func(data any) {
			ui.state.AddSystemMessage(shell.PrettyJSON(data))
		})
	case "models":
		ui.request("ai.models", nil, func(data any) {
			ui.state.AddSystemMessage(shell.PrettyJSON(data))
		})
	case "agents":
		ui.request("agents.list", nil, func(data any) {
			ui.state.AddSystemMessage(shell.PrettyJSON(data))
		})
	case "status":
		ui.state.AddSystemMessage(strings.Join(shell.StatusSummary(ui.state), "\n"))
		ui.refresh()
	case "transcript":
		if !strings.HasPrefix(strings.TrimSpace(command.Arg), "save ") {
			ui.state.AddErrorMessage("usage: /transcript save <path>")
			ui.refresh()
			return
		}
		path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(command.Arg), "save "))
		if path == "" {
			ui.state.AddErrorMessage("usage: /transcript save <path>")
			ui.refresh()
			return
		}
		if err := operator.SaveTranscript(path, ui.state); err != nil {
			ui.state.AddErrorMessage(err.Error())
		} else {
			ui.state.AddSystemMessage("saved transcript to " + path)
		}
		ui.refresh()
	case "run":
		ui.handleRunCommand()
	case "kill":
		if len(ui.runningProcs) == 0 {
			ui.state.AddSystemMessage("no running processes")
			ui.refresh()
			return
		}
		killed := 0
		for _, p := range ui.runningProcs {
			if err := exec.Command("kill", "-9", fmt.Sprint(p.pid)).Run(); err == nil {
				killed++
			}
		}
		ui.state.AddSystemMessage(fmt.Sprintf("killed %d process(es)", killed))
		ui.runningProcs = nil
		ui.refresh()
	case "stop":
		if ui.cancelStream != nil {
			ui.cancelStream()
		} else {
			ui.state.AddSystemMessage("nothing to stop")
			ui.refresh()
		}
	case "clear":
		ui.interview = nil
		ui.multiSelected = map[string]bool{}
		ui.state.ResetConversation()
		ui.state.AddSystemMessage("conversation cleared")
		ui.refresh()
	default:
		ui.state.AddErrorMessage("unknown command: /" + command.Name)
		ui.refresh()
	}
}

func (ui *TUIApp) startChat(message string) {
	ui.state.AddUserMessage(message)
	ui.startStream(func() (string, <-chan operator.Event, error) {
		return operator.StartChatStream(ui.client, ui.state, operator.ChatParams{Message: message})
	}, nil)
}

func (ui *TUIApp) startAgent(task string) {
	ui.state.AddUserMessage(task)
	ui.startStream(func() (string, <-chan operator.Event, error) {
		return operator.StartAgentStream(ui.client, ui.state, operator.AgentParams{Task: task})
	}, nil)
}

func (ui *TUIApp) startVibe(goal string) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		ui.state.AddErrorMessage("usage: /vibe <goal>")
		ui.refresh()
		return
	}
	ui.state.Mode = operator.ModeVibe
	ui.syncMode(ui.state.Mode)
	ui.state.AddUserMessage(goal)
	ui.startStream(func() (string, <-chan operator.Event, error) {
		return operator.StartVibeStream(ui.client, ui.state, operator.VibeParams{
			Goal:   goal,
			Source: "vibe",
		})
	}, nil)
}

func (ui *TUIApp) startArchitect(description string) {
	description = strings.TrimSpace(description)
	if description == "" {
		ui.state.AddErrorMessage("usage: /architect <prompt>")
		ui.refresh()
		return
	}
	ui.state.Mode = operator.ModeAgent
	ui.state.AgentID = "architect"
	ui.syncMode(ui.state.Mode)
	ui.interview = &shell.ArchitectInterview{
		Description: description,
		Answers:     map[string]any{},
	}
	task := shell.BuildArchitectQuestionsTask(description)
	ui.state.Messages = append(ui.state.Messages, operator.DisplayMessage{
		Role:    "user",
		Content: "/architect " + description,
	})
	ui.startStream(func() (string, <-chan operator.Event, error) {
		return operator.StartAgentStream(ui.client, ui.state, operator.AgentParams{Task: task})
	}, func(content string) {
		questions, err := shell.ParseArchitectQuestions(content)
		if err != nil {
			ui.state.AddErrorMessage("failed to parse architect questions: " + err.Error())
			return
		}
		ui.dropLatestAssistant(content)
		ui.interview.Questions = questions
		ui.interview.Index = 0
		ui.multiSelected = map[string]bool{}
		ui.state.AddSystemMessage(fmt.Sprintf("architect interview started with %d questions", len(questions)))
		ui.app.SetFocus(ui.options)
	})
}

func (ui *TUIApp) handleExampleCommand(arg string) {
	parts := strings.Fields(strings.TrimSpace(arg))
	if len(parts) == 0 {
		ui.state.AddErrorMessage("usage: /example show <id> or /example run <id>")
		ui.refresh()
		return
	}

	action := "run"
	id := parts[0]
	if len(parts) > 1 {
		action = strings.ToLower(parts[0])
		id = parts[1]
	}

	example, ok := shell.FindExample(id)
	if !ok {
		ui.state.AddErrorMessage("unknown example: " + id)
		ui.refresh()
		return
	}

	switch action {
	case "show":
		ui.state.AddSystemMessage(shell.FormatExample(example))
		ui.refresh()
	case "run":
		if ui.busy {
			ui.state.AddSystemMessage("A request is already running.")
			ui.refresh()
			return
		}
		ui.state.Project.Name = example.ProjectName
		ui.state.AddSystemMessage("running example: " + example.ID)
		switch example.Mode {
		case "vibe":
			ui.startVibe(example.Prompt)
		case "agent":
			ui.state.Mode = operator.ModeAgent
			ui.syncMode(ui.state.Mode)
			ui.startAgent(example.Prompt)
		default:
			ui.state.AddErrorMessage("unsupported example mode: " + example.Mode)
			ui.refresh()
		}
	default:
		ui.state.AddErrorMessage("usage: /example show <id> or /example run <id>")
		ui.refresh()
	}
}

func (ui *TUIApp) handleWatchCommand(arg string) {
	path := strings.TrimSpace(arg)
	if path == "" {
		ui.state.AddErrorMessage("usage: /watch <abs-path>")
		ui.refresh()
		return
	}
	if !filepath.IsAbs(path) {
		ui.state.AddErrorMessage("watch path must be absolute")
		ui.refresh()
		return
	}
	ui.watchPath = filepath.Clean(path)
	ui.state.AddSystemMessage("watching " + ui.watchPath)
	ui.refresh()
}

func (ui *TUIApp) handleProviderCommand(arg string) {
	parts := strings.Fields(strings.TrimSpace(arg))
	if len(parts) == 0 {
		ui.state.AddErrorMessage("usage: /provider status | /provider import-auth [path] | /provider set deepseek|mimo|zai|xai <key> | /provider clear deepseek|mimo|zai|xai")
		ui.refresh()
		return
	}

	switch parts[0] {
	case "status":
		ui.request("settings.provider_status", nil, func(data any) {
			ui.state.AddSystemMessage(shell.PrettyJSON(data))
		})
	case "import-auth":
		var path string
		if len(parts) > 1 {
			path = strings.Join(parts[1:], " ")
		}
		ui.importProviderKeysFromAuth(path, false)
	case "set":
		if len(parts) < 3 {
			ui.state.AddErrorMessage("usage: /provider set deepseek|mimo|zai|xai <key>")
			ui.refresh()
			return
		}
		ui.setProviderKey(parts[1], strings.Join(parts[2:], " "))
	case "clear":
		if len(parts) != 2 {
			ui.state.AddErrorMessage("usage: /provider clear deepseek|mimo|zai|xai")
			ui.refresh()
			return
		}
		ui.setProviderKey(parts[1], "")
	default:
		ui.state.AddErrorMessage("usage: /provider status | /provider import-auth [path] | /provider set deepseek|mimo|zai|xai <key> | /provider clear deepseek|mimo|zai|xai")
		ui.refresh()
	}
}

func (ui *TUIApp) setProviderKey(providerID, key string) {
	providerID = strings.TrimSpace(strings.ToLower(providerID))
	if providerID != "deepseek" && providerID != "mimo" && providerID != "zai" && providerID != "xai" {
		ui.state.AddErrorMessage("supported provider ids are deepseek, mimo, zai, and xai")
		ui.refresh()
		return
	}
	ui.request("settings.set", map[string]any{
		"key":   "provider_key:" + providerID,
		"value": strings.TrimSpace(key),
	}, func(any) {
		if strings.TrimSpace(key) == "" {
			ui.state.AddSystemMessage("cleared provider key for " + providerID)
		} else {
			ui.state.AddSystemMessage("saved provider key for " + providerID)
		}
	})
}

func (ui *TUIApp) importProviderKeysFromAuth(path string, silent bool) {
	imported, err := operator.LoadProviderKeysFromCodexAuth(path)
	if err != nil {
		if !silent {
			ui.state.AddErrorMessage(err.Error())
			ui.refresh()
		}
		return
	}
	if len(imported.Keys) == 0 {
		if !silent {
			ui.state.AddSystemMessage("no provider keys found in " + imported.Path)
			ui.refresh()
		}
		return
	}

	ids := make([]string, 0, len(imported.Keys))
	for providerID, key := range imported.Keys {
		ids = append(ids, providerID)
		go func(id, value string) {
			_, _ = ui.client.Send("settings.set", map[string]any{
				"key":   "provider_key:" + id,
				"value": strings.TrimSpace(value),
			})
		}(providerID, key)
	}
	sort.Strings(ids)
	ui.state.AddSystemMessage("imported provider keys from " + imported.Path + ": " + strings.Join(ids, ", "))
	ui.refresh()
}

func (ui *TUIApp) startArchitectPlan() {
	if ui.interview == nil {
		return
	}
	ui.state.AddSystemMessage("architect is generating the project plan")
	task := shell.BuildArchitectPlanTask(ui.interview.Description, ui.interview.Questions, ui.interview.Answers)
	ui.startStream(func() (string, <-chan operator.Event, error) {
		return operator.StartAgentStream(ui.client, ui.state, operator.AgentParams{Task: task})
	}, func(content string) {
		plan, err := shell.ParseArchitectPlan(content)
		if err != nil {
			ui.state.AddErrorMessage("failed to parse architect plan: " + err.Error())
			return
		}
		ui.dropLatestAssistant(content)
		ui.interview.Plan = plan
		ui.state.Messages = append(ui.state.Messages, operator.DisplayMessage{
			Role:    "assistant",
			Content: shell.PrettyJSON(plan),
		})
		ui.state.Conversation = append(ui.state.Conversation, operator.ConversationMessage{
			Role:    "assistant",
			Content: shell.PrettyJSON(plan),
		})
	})
}

func (ui *TUIApp) startStream(starter func() (string, <-chan operator.Event, error), done func(content string)) {
	if ui.busy {
		ui.state.AddSystemMessage("A request is already running.")
		ui.refresh()
		return
	}
	reqID, events, err := starter()
	if err != nil {
		ui.state.AddErrorMessage(err.Error())
		ui.refresh()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	ui.busy = true
	ui.currentStream = reqID
	ui.cancelStream = cancel
	ui.refresh()

	go func() {
		var assistant strings.Builder
		var finalContent string
		cancelled := false
		for event := range events {
			select {
			case <-ctx.Done():
				cancelled = true
				// drain remaining events
				for range events {
				}
				break
			default:
			}
			if cancelled {
				break
			}
			evt := event
			if evt.Kind == operator.EventAssistantText {
				assistant.WriteString(evt.Text)
			}
			if evt.Kind == operator.EventDone && strings.TrimSpace(evt.Content) != "" {
				finalContent = strings.TrimSpace(evt.Content)
			}
			ui.app.QueueUpdateDraw(func() {
				operator.ApplyStreamChunk(ui.state, evt)
				ui.refresh()
			})
		}

		ui.app.QueueUpdateDraw(func() {
			ui.busy = false
			ui.currentStream = ""
			ui.cancelStream = nil
			if cancelled {
				ui.state.Status = "Stopped"
				ui.state.AddSystemMessage("request stopped")
			} else {
				if strings.TrimSpace(finalContent) == "" {
					finalContent = strings.TrimSpace(assistant.String())
				}
				if done != nil {
					done(finalContent)
				}
			}
			ui.refresh()

			// Process queued input
			if len(ui.pendingInput) > 0 {
				next := ui.pendingInput[0]
				ui.pendingInput = ui.pendingInput[1:]
				ui.handleInput(next)
			}
		})
	}()
}

func (ui *TUIApp) request(reqType string, payload any, onSuccess func(any)) {
	go func() {
		resp, err := ui.client.Send(reqType, payload)
		ui.app.QueueUpdateDraw(func() {
			if err != nil {
				ui.state.AddErrorMessage(err.Error())
				ui.refresh()
				return
			}
			if !resp.Success {
				ui.state.AddErrorMessage(resp.Error)
				ui.refresh()
				return
			}
			if onSuccess != nil {
				onSuccess(resp.Data)
			}
			ui.refresh()
		})
	}()
}

func (ui *TUIApp) resumeVibe(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		ui.state.AddErrorMessage("usage: /resume <session-id>")
		ui.refresh()
		return
	}
	ui.request("vibe.session.get", map[string]any{"session_id": sessionID}, func(data any) {
		payload, err := decodeSessionDetails(data)
		if err != nil {
			ui.state.AddErrorMessage(err.Error())
			return
		}
		operator.RestoreVibeSession(ui.state, payload)
		ui.interview = nil
		ui.multiSelected = map[string]bool{}
		ui.syncMode(operator.ModeVibe)
	})
}

func (ui *TUIApp) syncMode(mode operator.Mode) {
	go func() {
		_, _ = ui.client.Send("context.set_mode", map[string]any{"mode": string(mode)})
	}()
}

func (ui *TUIApp) currentQuestion() *shell.ArchitectQuestion {
	if ui.interview == nil || ui.interview.Index < 0 || ui.interview.Index >= len(ui.interview.Questions) {
		return nil
	}
	return &ui.interview.Questions[ui.interview.Index]
}

func (ui *TUIApp) toggleCurrentMultiOption() {
	question := ui.currentQuestion()
	if question == nil || question.Type != "multi" {
		return
	}
	index := ui.options.GetCurrentItem()
	if index < 0 || index >= len(question.Options) {
		return
	}
	value := question.Options[index].Value
	ui.multiSelected[value] = !ui.multiSelected[value]
	if !ui.multiSelected[value] {
		delete(ui.multiSelected, value)
	}
	ui.refresh()
}

func (ui *TUIApp) handleOptionSelect() {
	// Mode picker
	if ui.modePicker {
		modes := []operator.Mode{operator.ModeChat, operator.ModeVibe, operator.ModeAgent}
		index := ui.options.GetCurrentItem()
		if index >= 0 && index < len(modes) {
			ui.state.Mode = modes[index]
			ui.syncMode(ui.state.Mode)
			ui.state.AddSystemMessage("mode set to " + string(ui.state.Mode))
		}
		ui.modePicker = false
		ui.app.SetFocus(ui.input)
		ui.refresh()
		return
	}

	// Agent picker
	if len(ui.agentPicker) > 0 {
		index := ui.options.GetCurrentItem()
		if index >= 0 && index < len(ui.agentPicker) {
			ui.state.AgentID = ui.agentPicker[index]
			ui.state.Mode = operator.ModeAgent
			ui.syncMode(ui.state.Mode)
			ui.state.AddSystemMessage("agent set to " + ui.state.AgentID)
		}
		ui.agentPicker = nil
		ui.app.SetFocus(ui.input)
		ui.refresh()
		return
	}

	// Model picker
	if len(ui.modelPicker) > 0 {
		index := ui.options.GetCurrentItem()
		if index >= 0 && index < len(ui.modelPicker) {
			ui.state.Model = ui.modelPicker[index]
			ui.state.AddSystemMessage("model set to " + ui.state.Model)
		}
		ui.modelPicker = nil
		ui.app.SetFocus(ui.input)
		ui.refresh()
		return
	}

	question := ui.currentQuestion()
	if question == nil {
		return
	}
	index := ui.options.GetCurrentItem()
	if index < 0 || index >= len(question.Options) {
		return
	}

	if question.Type == "single" {
		option := question.Options[index]
		ui.interview.Answers[question.ID] = option.Value
		ui.state.AddSystemMessage(fmt.Sprintf("architect answer: %s -> %s", question.Prompt(), option.Label))
		ui.advanceInterview()
		return
	}

	values := make([]string, 0, len(ui.multiSelected))
	labels := make([]string, 0, len(ui.multiSelected))
	for _, option := range question.Options {
		if ui.multiSelected[option.Value] {
			values = append(values, option.Value)
			labels = append(labels, option.Label)
		}
	}
	if len(values) == 0 {
		ui.state.AddErrorMessage("select at least one option before submitting")
		ui.refresh()
		return
	}
	ui.interview.Answers[question.ID] = values
	ui.state.AddSystemMessage(fmt.Sprintf("architect answer: %s -> %s", question.Prompt(), strings.Join(labels, ", ")))
	ui.advanceInterview()
}

func (ui *TUIApp) advanceInterview() {
	if ui.interview == nil {
		return
	}
	ui.multiSelected = map[string]bool{}
	ui.interview.Index++
	if ui.interview.Index >= len(ui.interview.Questions) {
		ui.startArchitectPlan()
		ui.app.SetFocus(ui.input)
	} else {
		ui.app.SetFocus(ui.options)
	}
	ui.refresh()
}

func (ui *TUIApp) refresh() {
	ui.renderStatus()
	ui.renderTranscript()
	ui.renderSidebar()
}

func (ui *TUIApp) renderStatus() {
	lines := []string{
		fmt.Sprintf("[yellow]Mode[-]: %s   [yellow]Agent[-]: %s   [yellow]Model[-]: %s", ui.state.Mode, valueOr(ui.state.AgentID, "general"), valueOr(ui.state.Model, "default")),
		fmt.Sprintf("[yellow]Project[-]: %s", valueOr(ui.state.Project.Path, "(none)")),
		fmt.Sprintf("[yellow]Vibe Session[-]: %s   [yellow]Status[-]: %s", valueOr(ui.state.VibeSession.SessionID, "(none)"), valueOr(ui.state.Status, "Ready")),
		fmt.Sprintf("[yellow]Busy[-]: %t   [yellow]Queue[-]: %d", ui.busy, len(ui.pendingInput)),
	}
	ui.status.SetText(strings.Join(lines, "\n"))
}

func (ui *TUIApp) renderTranscript() {
	var lines []string
	for _, message := range ui.state.Messages {
		lines = append(lines, formatMessage(message))
	}
	ui.transcript.SetText(strings.Join(lines, "\n\n"))
	// Only auto-scroll if user hasn't scrolled up
	row, _ := ui.transcript.GetScrollOffset()
	_, _, _, height := ui.transcript.GetInnerRect()
	totalLines := strings.Count(ui.transcript.GetText(false), "\n") + 1
	if row+height >= totalLines-3 {
		ui.transcript.ScrollToEnd()
	}
}

func (ui *TUIApp) renderSidebar() {
	question := ui.currentQuestion()
	currentIndex := ui.options.GetCurrentItem()
	ui.options.Clear()

	// Mode picker
	if ui.modePicker {
		ui.questionText.SetText("[yellow]Select Mode[-]\n\nUse arrows to navigate, Enter to select, Esc to cancel")
		modes := []operator.Mode{operator.ModeChat, operator.ModeVibe, operator.ModeAgent}
		descriptions := map[operator.Mode]string{
			operator.ModeChat:  "Direct conversation with LLM",
			operator.ModeVibe:  "Autonomous project generation",
			operator.ModeAgent: "Task-based agent dispatch",
		}
		for _, m := range modes {
			label := string(m)
			if m == ui.state.Mode {
				label = string(m) + " [green](current)[-]"
			}
			ui.options.AddItem(label, descriptions[m], 0, nil)
		}
		if currentIndex >= 0 && currentIndex < ui.options.GetItemCount() {
			ui.options.SetCurrentItem(currentIndex)
		}
		ui.sidebar.SetText("[yellow]Mode Picker[-]\nEnter: select mode\nEsc: cancel")
		return
	}

	// Agent picker
	if len(ui.agentPicker) > 0 {
		ui.questionText.SetText("[yellow]Select Agent[-]\n\nUse arrows to navigate, Enter to select, Esc to cancel")
		for _, a := range ui.agentPicker {
			label := a
			if a == ui.state.AgentID {
				label = a + " [green](current)[-]"
			}
			ui.options.AddItem(label, "", 0, nil)
		}
		if currentIndex >= 0 && currentIndex < ui.options.GetItemCount() {
			ui.options.SetCurrentItem(currentIndex)
		}
		ui.sidebar.SetText("[yellow]Agent Picker[-]\nEnter: select agent\nEsc: cancel")
		return
	}

	// Model picker mode
	if len(ui.modelPicker) > 0 {
		ui.questionText.SetText("[yellow]Select Model[-]\n\nUse arrows to navigate, Enter to select, Esc to cancel")
		for _, m := range ui.modelPicker {
			label := m
			if m == ui.state.Model {
				label = m + " [green](current)[-]"
			}
			ui.options.AddItem(label, "", 0, nil)
		}
		if currentIndex >= 0 && currentIndex < ui.options.GetItemCount() {
			ui.options.SetCurrentItem(currentIndex)
		}
		ui.sidebar.SetText("[yellow]Model Picker[-]\nEnter: select model\nEsc: cancel")
		return
	}

	if question == nil {
		ui.questionText.SetText("[yellow]Commands[-]\n/mode chat|vibe|agent\n/project <abs-path>\n/project-name <name>\n/projects-root <abs-path>\n/model <name>\n/agent <id>\n/architect <prompt>\n/vibe <goal>\n/examples\n/example show <id>\n/example run <id>\n/watch <abs-path>\n/provider status\n/provider import-auth [path]\n/provider set deepseek|mimo|zai|xai <key>\n/provider clear deepseek|mimo|zai|xai\n/resume <session-id>\n/providers\n/models\n/agents\n/transcript save <path>\n/clear")
		if ui.interview != nil && len(ui.interview.Plan) > 0 {
			ui.sidebar.SetText("[yellow]Architect Plan[-]\n" + shell.PrettyJSON(ui.interview.Plan))
		} else {
			ui.sidebar.SetText(strings.Join([]string{
				"[yellow]Running[-]",
				ui.renderRunningProcesses(),
				"",
				"[yellow]Keys[-]",
				"Ctrl+C quit  /stop cancel",
				"/run start dev  /clear reset",
				"Tab focus  Esc back",
				"",
				"[yellow]State[-]",
				strings.Join(shell.StatusSummary(ui.state), "\n"),
			}, "\n"))
		}
		return
	}

	ui.questionText.SetText(strings.Join([]string{
		fmt.Sprintf("[yellow]Question %d/%d[-]", ui.interview.Index+1, len(ui.interview.Questions)),
		question.Prompt(),
		strings.TrimSpace(question.Description),
	}, "\n\n"))

	for _, option := range question.Options {
		label := option.Label
		if question.Type == "multi" {
			prefix := "[ ]"
			if ui.multiSelected[option.Value] {
				prefix = "[x]"
			}
			label = prefix + " " + label
		}
		ui.options.AddItem(label, option.Description, 0, nil)
	}
	if currentIndex >= 0 && currentIndex < ui.options.GetItemCount() {
		ui.options.SetCurrentItem(currentIndex)
	}

	help := []string{
		fmt.Sprintf("[yellow]Answer Type[-]: %s", question.Type),
		"Tab to switch between input and question list",
	}
	if question.Type == "multi" {
		help = append(help, "Space toggles selection", "Enter submits selected options")
	} else {
		help = append(help, "Enter selects the highlighted option")
	}
	ui.sidebar.SetText(strings.Join(help, "\n"))
}

func formatMessage(message operator.DisplayMessage) string {
	color := "white"
	prefix := strings.ToUpper(message.Role)
	switch message.Role {
	case "assistant":
		color = "green"
	case "user":
		color = "cyan"
	case "system":
		color = "yellow"
	case "tool":
		color = "magenta"
	case "error":
		color = "red"
	}
	// Escape brackets in content so tview doesn't interpret them as color tags
	escaped := tview.Escape(message.Content)
	return fmt.Sprintf("[%s]%s[-]\n%s", color, prefix, escaped)
}

func (ui *TUIApp) dropLatestAssistant(content string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return
	}
	for i := len(ui.state.Messages) - 1; i >= 0; i-- {
		if ui.state.Messages[i].Role != "assistant" {
			continue
		}
		if strings.TrimSpace(strings.TrimSuffix(ui.state.Messages[i].Content, "~")) == trimmed {
			ui.state.Messages = append(ui.state.Messages[:i], ui.state.Messages[i+1:]...)
			break
		}
	}
	for i := len(ui.state.Conversation) - 1; i >= 0; i-- {
		if !strings.EqualFold(ui.state.Conversation[i].Role, "assistant") {
			continue
		}
		if strings.TrimSpace(ui.state.Conversation[i].Content) == trimmed {
			ui.state.Conversation = append(ui.state.Conversation[:i], ui.state.Conversation[i+1:]...)
			break
		}
	}
}

func (ui *TUIApp) handleRunCommand() {
	projectDir := ui.state.Project.Path
	if projectDir == "" {
		ui.state.AddErrorMessage("no project set — use /project <path>")
		ui.refresh()
		return
	}

	// Detect project type and build the run info
	type runInfo struct {
		cmd  string
		url  string
		name string
	}

	detect := func(dir string) *runInfo {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
			script := "dev"
			if !strings.Contains(string(data), `"dev"`) {
				if strings.Contains(string(data), `"start"`) {
					script = "start"
				} else if strings.Contains(string(data), `"serve"`) {
					script = "serve"
				}
			}
			runner := "npm"
			if _, err := os.Stat(filepath.Join(dir, "bun.lockb")); err == nil {
				runner = "bun"
			} else if _, err := os.Stat(filepath.Join(dir, "bun.lock")); err == nil {
				runner = "bun"
			} else if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
				runner = "pnpm"
			} else if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
				runner = "yarn"
			}

			port := "3000"
			if strings.Contains(string(data), "nuxt") || strings.Contains(string(data), "vite") {
				port = "3000"
			}
			if strings.Contains(string(data), "next") {
				port = "3000"
			}
			name := "Node.js"
			if strings.Contains(string(data), "nuxt") {
				name = "Nuxt"
			} else if strings.Contains(string(data), "next") {
				name = "Next.js"
			} else if strings.Contains(string(data), "vite") {
				name = "Vite"
			}
			return &runInfo{
				cmd:  fmt.Sprintf("cd %s && %s run %s", dir, runner, script),
				url:  "http://localhost:" + port,
				name: name,
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return &runInfo{cmd: fmt.Sprintf("cd %s && go run .", dir), url: "http://localhost:8080", name: "Go"}
		}
		if _, err := os.Stat(filepath.Join(dir, "manage.py")); err == nil {
			return &runInfo{cmd: fmt.Sprintf("cd %s && python manage.py runserver", dir), url: "http://localhost:8000", name: "Django"}
		}
		if _, err := os.Stat(filepath.Join(dir, "app.py")); err == nil {
			return &runInfo{cmd: fmt.Sprintf("cd %s && python app.py", dir), url: "http://localhost:5000", name: "Flask"}
		}
		if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
			return &runInfo{cmd: fmt.Sprintf("cd %s && cargo run", dir), url: "", name: "Rust"}
		}
		if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
			return &runInfo{cmd: fmt.Sprintf("cd %s && bundle exec rails server", dir), url: "http://localhost:3000", name: "Rails"}
		}
		return nil
	}

	// Try project dir, then project dir/code (vibe projects)
	info := detect(projectDir)
	if info == nil {
		info = detect(filepath.Join(projectDir, "code"))
	}
	if info == nil {
		ui.state.AddErrorMessage("no recognized project found in " + projectDir)
		ui.refresh()
		return
	}

	// Kill any existing process on the target port
	port := strings.TrimPrefix(info.url, "http://localhost:")
	if port != "" {
		exec.Command("sh", "-c", fmt.Sprintf("lsof -ti :%s | xargs kill -9 2>/dev/null", port)).Run()
	}

	msg := fmt.Sprintf("%s project detected. Starting dev server...\n  %s", info.name, info.cmd)
	if info.url != "" {
		msg += fmt.Sprintf("\n  %s", info.url)
	}
	ui.state.AddSystemMessage(msg)
	ui.refresh()

	// Start dev server detached, capture early output for the URL
	go func() {
		cmd := exec.Command("sh", "-c", info.cmd)
		cmd.Dir = projectDir
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ui.app.QueueUpdateDraw(func() {
				ui.state.AddErrorMessage(err.Error())
				ui.refresh()
			})
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			ui.app.QueueUpdateDraw(func() {
				ui.state.AddErrorMessage(err.Error())
				ui.refresh()
			})
			return
		}

		// Read output for a few seconds to capture the startup URL
		var collected strings.Builder
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					collected.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
			close(done)
		}()

		select {
		case <-done:
			// Process exited early — show output
		case <-time.After(6 * time.Second):
			// Dev server is running — show what we got
		}

		output := collected.String()
		pid := cmd.Process.Pid
		ui.app.QueueUpdateDraw(func() {
			ui.runningProcs = append(ui.runningProcs, runningProc{
				pid:  pid,
				name: info.name,
				url:  info.url,
				cmd:  info.cmd,
			})
			if strings.TrimSpace(output) != "" {
				ui.state.AddSystemMessage("dev server running:\n" + operator.Truncate(output, 600))
			} else {
				ui.state.AddSystemMessage(fmt.Sprintf("dev server started (PID %d)", pid))
			}
			ui.refresh()
		})

		// Clean up when process exits
		go func() {
			cmd.Wait()
			ui.app.QueueUpdateDraw(func() {
				for i, p := range ui.runningProcs {
					if p.pid == pid {
						ui.runningProcs = append(ui.runningProcs[:i], ui.runningProcs[i+1:]...)
						break
					}
				}
				ui.refresh()
			})
		}()
	}()
}

func (ui *TUIApp) renderRunningProcesses() string {
	if len(ui.runningProcs) == 0 {
		return "  (none)\n  Use /run to start"
	}
	var lines []string
	for _, p := range ui.runningProcs {
		line := fmt.Sprintf("  %s  PID %d", p.name, p.pid)
		if p.url != "" {
			line += "  " + p.url
		}
		lines = append(lines, line)
	}
	lines = append(lines, "  /kill to stop all")
	return strings.Join(lines, "\n")
}

func decodeSessionDetails(data any) (operator.VibeSessionDetails, error) {
	var details operator.VibeSessionDetails
	raw, err := json.Marshal(data)
	if err != nil {
		return details, err
	}
	if err := json.Unmarshal(raw, &details); err != nil {
		return details, err
	}
	return details, nil
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (ui *TUIApp) renderWatchSnapshot() string {
	root := strings.TrimSpace(ui.watchPath)
	if root == "" {
		return "(no watch path)"
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return root + "\n(unavailable)"
	}

	lines := []string{root}
	entries, err := os.ReadDir(root)
	if err != nil {
		return root + "\n(read error)"
	}

	type child struct {
		name string
		line string
	}
	children := make([]child, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		label := entry.Name()
		if entry.IsDir() {
			label += "/"
		}
		children = append(children, child{name: entry.Name(), line: "  " + label})
	}
	sort.Slice(children, func(i, j int) bool { return children[i].name < children[j].name })
	limit := 8
	if len(children) < limit {
		limit = len(children)
	}
	for _, item := range children[:limit] {
		lines = append(lines, item.line)
	}
	if len(children) > limit {
		lines = append(lines, fmt.Sprintf("  ... %d more", len(children)-limit))
	}
	return strings.Join(lines, "\n")
}
