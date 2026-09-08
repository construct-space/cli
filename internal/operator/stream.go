package operator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ChatParams struct {
	Message string
}

type AgentParams struct {
	Task string
}

type VibeParams struct {
	Goal          string
	Source        string
	MaxIterations int
}

func StartChatStream(client *Client, state *SessionState, params ChatParams) (string, <-chan Event, error) {
	payload := map[string]any{
		"message": strings.TrimSpace(params.Message),
	}
	if state.Model != "" {
		payload["model"] = state.Model
	}
	if messages := conversationPayload(state.Conversation, params.Message); len(messages) > 0 {
		payload["messages"] = messages
	}
	return startStream(client, state, "ai.chat_stream", payload)
}

func StartAgentStream(client *Client, state *SessionState, params AgentParams) (string, <-chan Event, error) {
	payload := map[string]any{
		"agent_id": strings.TrimSpace(state.AgentID),
		"task":     strings.TrimSpace(params.Task),
	}
	if payload["agent_id"] == "" {
		payload["agent_id"] = "general"
	}
	if state.Model != "" {
		payload["model"] = state.Model
	}
	if messages := conversationPayload(state.Conversation, params.Task); len(messages) > 0 {
		payload["messages"] = messages
	}
	return startStream(client, state, "agents.dispatch_stream", payload)
}

func StartVibeStream(client *Client, state *SessionState, params VibeParams) (string, <-chan Event, error) {
	source := strings.TrimSpace(params.Source)
	if source == "" {
		source = "vibe"
	}
	payload := map[string]any{
		"goal":       strings.TrimSpace(params.Goal),
		"source":     source,
		"local_data": BuildVibeLocalData(state, params.Goal),
		"messages":   conversationPayload(state.Conversation, params.Goal),
	}
	if state.Model != "" {
		payload["model"] = state.Model
	}
	if state.VibeSession.SessionID != "" {
		payload["session_id"] = state.VibeSession.SessionID
	}
	if params.MaxIterations > 0 {
		payload["max_iterations"] = params.MaxIterations
	}
	return startStream(client, state, "ai.vibe_stream", payload)
}

func startStream(client *Client, state *SessionState, reqType string, payload any) (string, <-chan Event, error) {
	reqID, chunkCh, err := client.Stream(reqType, payload)
	if err != nil {
		return "", nil, err
	}
	state.Transcript.AddRequest(state.Mode, reqType, payload)

	events := make(chan Event, 256)
	go func() {
		defer close(events)
		defer client.CloseStream(reqID)
		for chunk := range chunkCh {
			state.Transcript.AddChunk(state.Mode, chunk)
			evt := NormalizeStreamChunk(chunk)
			events <- evt
			if chunk.Done || evt.Kind == EventDone || evt.Kind == EventError {
				return
			}
		}
	}()

	return reqID, events, nil
}

func BuildVibeLocalData(state *SessionState, goal string) map[string]any {
	routeContext := map[string]any{
		"isProjectScoped": strings.TrimSpace(state.Project.Path) != "",
		"spaceName":       "vibe",
	}
	if state.Project.ID != "" {
		routeContext["projectId"] = state.Project.ID
	}

	spaceContext := map[string]any{
		"activeSpace": "vibe",
	}
	if state.Project.Name != "" || state.Project.Path != "" {
		project := map[string]any{}
		if state.Project.ID != "" {
			project["id"] = state.Project.ID
		}
		if state.Project.Name != "" {
			project["name"] = state.Project.Name
		}
		if state.Project.Path != "" {
			project["localPath"] = state.Project.Path
		}
		if len(state.Project.Spaces) > 0 {
			project["spaces"] = state.Project.Spaces
		}
		spaceContext["project"] = project
	}

	localData := map[string]any{
		"route_context": routeContext,
		"space_context": spaceContext,
		"vibe": map[string]any{
			"goal":   strings.TrimSpace(goal),
			"source": "vibe",
		},
		"vibe_session": map[string]any{
			"session_id":    emptyOrNil(state.VibeSession.SessionID),
			"status":        emptyOrNil(state.VibeSession.Status),
			"current_phase": emptyOrNil(state.VibeSession.CurrentPhase),
			"spaces":        state.Project.Spaces,
		},
	}

	if state.Project.ID != "" {
		localData["project_id"] = state.Project.ID
	}
	if state.Project.Name != "" {
		localData["project_name"] = state.Project.Name
	}
	if state.Project.Path != "" {
		localData["project_path"] = state.Project.Path
	}
	if state.Project.ProjectsRoot != "" {
		localData["projects_root"] = state.Project.ProjectsRoot
	}
	if len(state.Project.Spaces) > 0 {
		localData["project_spaces"] = state.Project.Spaces
	}

	return compactMap(localData)
}

func NormalizeStreamChunk(chunk StreamChunk) Event {
	evt := Event{
		RawType:   chunk.Type,
		Data:      chunk.Data,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	switch chunk.Type {
	case "text", "stream":
		evt.Kind = EventAssistantText
		evt.Text = mapStringRaw(chunk.Data, "text")
	case "progress":
		evt.Kind = EventProgress
		evt.Text = mapString(chunk.Data, "text")
	case "tool.call", "tool_call":
		evt.Kind = EventToolCall
		evt.ToolName = firstNonEmpty(mapString(chunk.Data, "tool"), mapString(chunk.Data, "name"))
		evt.CallID = mapString(chunk.Data, "call_id")
		evt.Input = normalizePayloadString(chunk.Data["input"])
	case "tool.result", "tool_result":
		evt.Kind = EventToolResult
		evt.ToolName = firstNonEmpty(mapString(chunk.Data, "tool"), mapString(chunk.Data, "name"))
		evt.CallID = mapString(chunk.Data, "call_id")
		evt.Input = normalizePayloadString(chunk.Data["input"])
		evt.Content = firstNonEmpty(mapString(chunk.Data, "content"), mapString(chunk.Data, "result"))
		evt.IsError = mapBool(chunk.Data, "is_error")
	case "vibe.session":
		evt.Kind = EventVibeSession
		evt.Session = &VibeSessionMeta{
			SessionID:     mapString(chunk.Data, "session_id"),
			ProjectID:     mapString(chunk.Data, "project_id"),
			Goal:          mapString(chunk.Data, "goal"),
			Source:        mapString(chunk.Data, "source"),
			Space:         mapString(chunk.Data, "space"),
			Status:        mapString(chunk.Data, "status"),
			CurrentPhase:  mapString(chunk.Data, "current_phase"),
			SessionType:   mapString(chunk.Data, "session_type"),
			AutonomyLevel: mapString(chunk.Data, "autonomy_level"),
			ProjectName:   mapString(chunk.Data, "project_name"),
			ProjectPath:   mapString(chunk.Data, "project_path"),
		}
	case "orchestration.complete":
		evt.Kind = EventOrchestration
		evt.Content = firstNonEmpty(mapString(chunk.Data, "content"), mapString(chunk.Data, "status"))
	case "done":
		evt.Kind = EventDone
		evt.Content = mapString(chunk.Data, "content")
		if evt.Content == "" {
			evt.Content = normalizePayloadString(chunk.Data)
		}
		if sessionID := mapString(chunk.Data, "session_id"); sessionID != "" {
			evt.Session = &VibeSessionMeta{SessionID: sessionID}
		}
	case "error":
		evt.Kind = EventError
		evt.Text = mapString(chunk.Data, "error")
	default:
		evt.Kind = EventInfo
	}

	return evt
}

func ApplyStreamChunk(state *SessionState, evt Event) {
	switch evt.Kind {
	case EventAssistantText:
		if len(state.Messages) == 0 || state.Messages[len(state.Messages)-1].Role != "assistant" {
			state.StartAssistantStream()
		}
		state.SetStreamingAssistant(evt.Text)
	case EventProgress:
		state.Status = strings.TrimSpace(evt.Text)
		state.AddSystemMessage("[progress] " + strings.TrimSpace(evt.Text))
	case EventToolCall:
		state.Status = fmt.Sprintf("Running %s", evt.ToolName)
		state.ToolMessage("", formatToolCall(evt.ToolName, evt.Input))
	case EventToolResult:
		prefix := "  ✓ "
		if evt.IsError {
			prefix = "  ✗ "
		}
		state.ToolMessage(prefix, Truncate(evt.Content, 220))
	case EventVibeSession:
		if evt.Session != nil {
			state.VibeSession = mergeVibeSession(state.VibeSession, *evt.Session)
			state.SyncProjectFromVibe()
			if state.VibeSession.Status != "" {
				state.Status = fmt.Sprintf("Vibe %s", state.VibeSession.Status)
			}
			title := firstNonEmpty(state.VibeSession.Goal, state.VibeSession.SessionID)
			state.AddSystemMessage("[vibe] " + strings.TrimSpace(title))
		}
	case EventOrchestration:
		state.Status = "Completed orchestration"
		text := strings.TrimSpace(evt.Content)
		if text == "" {
			text = "orchestration complete"
		}
		state.AddSystemMessage("[complete] " + text)
	case EventDone:
		if evt.Session != nil && evt.Session.SessionID != "" {
			state.VibeSession.SessionID = evt.Session.SessionID
		}
		state.Status = "Done"
		state.FinalizeAssistant(evt.Content)
	case EventError:
		state.Status = "Error"
		state.PendingAssistant = ""
		state.AddErrorMessage(evt.Text)
	case EventInfo:
		// Ignore low-signal transport events by default.
	}
}

func RestoreVibeSession(state *SessionState, details VibeSessionDetails) {
	state.ResetConversation()
	state.Mode = ModeVibe
	if details.Session != nil {
		state.VibeSession = VibeSessionMeta{
			SessionID:     details.Session.ID,
			ProjectID:     details.Session.ProjectID,
			Goal:          details.Session.Goal,
			Source:        details.Session.Source,
			Status:        details.Session.Status,
			CurrentPhase:  details.Session.CurrentPhase,
			SessionType:   details.Session.SessionType,
			AutonomyLevel: details.Session.AutonomyLevel,
			ProjectName:   details.Session.ProjectName,
			ProjectPath:   details.Session.ProjectPath,
		}
		state.SyncProjectFromVibe()
		state.AddSystemMessage("[resume] loaded vibe session " + details.Session.ID)
	}
	for _, stored := range details.Events {
		ApplyStreamChunk(state, NormalizeStreamChunk(StreamChunk{
			Type: stored.EventType,
			Data: toAnyMap(stored.Data),
		}))
	}
	if details.Session != nil && details.Session.Status == "complete" && strings.TrimSpace(state.PendingAssistant) != "" {
		state.FinalizeAssistant("")
	}
}

func mergeVibeSession(base, update VibeSessionMeta) VibeSessionMeta {
	if update.SessionID != "" {
		base.SessionID = update.SessionID
	}
	if update.ProjectID != "" {
		base.ProjectID = update.ProjectID
	}
	if update.Goal != "" {
		base.Goal = update.Goal
	}
	if update.Source != "" {
		base.Source = update.Source
	}
	if update.Space != "" {
		base.Space = update.Space
	}
	if update.Status != "" {
		base.Status = update.Status
	}
	if update.CurrentPhase != "" {
		base.CurrentPhase = update.CurrentPhase
	}
	if update.SessionType != "" {
		base.SessionType = update.SessionType
	}
	if update.AutonomyLevel != "" {
		base.AutonomyLevel = update.AutonomyLevel
	}
	if update.ProjectName != "" {
		base.ProjectName = update.ProjectName
	}
	if update.ProjectPath != "" {
		base.ProjectPath = update.ProjectPath
	}
	return base
}

func conversationPayload(history []ConversationMessage, latest string) []ConversationMessage {
	if len(history) == 0 && strings.TrimSpace(latest) == "" {
		return nil
	}
	messages := append([]ConversationMessage{}, history...)
	trimmedLatest := strings.TrimSpace(latest)
	if trimmedLatest == "" {
		return messages
	}
	if len(messages) == 0 {
		return []ConversationMessage{{Role: "user", Content: trimmedLatest}}
	}
	last := messages[len(messages)-1]
	if strings.EqualFold(strings.TrimSpace(last.Role), "user") && strings.TrimSpace(last.Content) == trimmedLatest {
		return messages
	}
	return append(messages, ConversationMessage{Role: "user", Content: trimmedLatest})
}

func compactMap(input map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range input {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
			result[key] = typed
		case []string:
			if len(typed) == 0 {
				continue
			}
			result[key] = typed
		case []any:
			if len(typed) == 0 {
				continue
			}
			result[key] = typed
		case map[string]any:
			compacted := compactMap(typed)
			if len(compacted) == 0 {
				continue
			}
			result[key] = compacted
		default:
			result[key] = value
		}
	}
	return result
}

func emptyOrNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func normalizePayloadString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mapString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return strings.TrimSpace(value)
}

// mapStringRaw returns a string value without trimming (preserves whitespace tokens).
func mapStringRaw(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func mapBool(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	value, _ := data[key].(bool)
	return value
}

// formatToolCall renders a tool invocation like Bash('pwd') or WriteFile('main.go')
func formatToolCall(name, input string) string {
	// Title-case the tool name: bash → Bash, write_file → WriteFile
	display := toolDisplayName(name)

	// Extract the primary argument value from JSON input
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil || len(args) == 0 {
		if input != "" && input != "{}" {
			return display + "(" + Truncate(input, 80) + ")"
		}
		return display + "()"
	}

	// Pick the most meaningful argument to show
	primary := pickPrimaryArg(name, args)
	if primary == "" {
		return display + "()"
	}
	return display + "('" + Truncate(primary, 80) + "')"
}

func toolDisplayName(name string) string {
	// Map common tool names to clean display names
	switch name {
	case "bash":
		return "Bash"
	case "write_file":
		return "Write"
	case "edit_file":
		return "Edit"
	case "read_file":
		return "Read"
	case "list_dir":
		return "List"
	case "glob":
		return "Glob"
	case "grep":
		return "Grep"
	case "spawn_agent":
		return "Agent"
	}
	// For other tools, title-case segments
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func pickPrimaryArg(toolName string, args map[string]any) string {
	// Tool-specific primary argument
	keys := map[string][]string{
		"bash":        {"command"},
		"write_file":  {"path"},
		"edit_file":   {"path"},
		"read_file":   {"path"},
		"list_dir":    {"path"},
		"glob":        {"pattern"},
		"grep":        {"pattern"},
		"spawn_agent": {"agent_id"},
	}
	if preferred, ok := keys[toolName]; ok {
		for _, key := range preferred {
			if val, ok := args[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
	}
	// Fallback: first string value
	for _, val := range args {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func Truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}

func toAnyMap(input map[string]interface{}) map[string]any {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
