package shell

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/construct-space/cli/internal/operator"
)

func SetMode(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	switch operator.Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case operator.ModeChat:
		state.Mode = operator.ModeChat
	case operator.ModeVibe:
		state.Mode = operator.ModeVibe
	case operator.ModeAgent:
		state.Mode = operator.ModeAgent
	default:
		return fmt.Errorf("unknown mode %q", raw)
	}
	return nil
}

func SetProjectPath(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	path := strings.TrimSpace(raw)
	if path == "" {
		return fmt.Errorf("project path is required")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("project path must be absolute")
	}
	state.Project.Path = filepath.Clean(path)
	if strings.TrimSpace(state.Project.Name) == "" {
		state.Project.Name = filepath.Base(state.Project.Path)
	}
	return nil
}

func ClearProject(state *operator.SessionState) {
	if state == nil {
		return
	}
	state.Project.ID = ""
	state.Project.Name = ""
	state.Project.Path = ""
}

func SetProjectName(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	name := strings.TrimSpace(raw)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	state.Project.Name = name
	return nil
}

func SetProjectsRoot(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	path := strings.TrimSpace(raw)
	if path == "" {
		return fmt.Errorf("projects root is required")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("projects root must be absolute")
	}
	state.Project.ProjectsRoot = filepath.Clean(path)
	return nil
}

func SetModel(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	model := strings.TrimSpace(raw)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	state.Model = model
	return nil
}

func SetAgent(state *operator.SessionState, raw string) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	agentID := strings.TrimSpace(raw)
	if agentID == "" {
		return fmt.Errorf("agent id is required")
	}
	state.AgentID = agentID
	return nil
}

func StatusSummary(state *operator.SessionState) []string {
	if state == nil {
		return nil
	}
	lines := []string{
		fmt.Sprintf("mode=%s", state.Mode),
		fmt.Sprintf("agent=%s", emptyFallback(state.AgentID, "general")),
		fmt.Sprintf("model=%s", emptyFallback(state.Model, "default")),
		fmt.Sprintf("status=%s", emptyFallback(state.Status, "ready")),
	}
	if strings.TrimSpace(state.Project.Path) != "" {
		lines = append(lines, "project="+state.Project.Path)
	}
	if strings.TrimSpace(state.Project.ProjectsRoot) != "" {
		lines = append(lines, "projects_root="+state.Project.ProjectsRoot)
	}
	if strings.TrimSpace(state.VibeSession.SessionID) != "" {
		lines = append(lines, "vibe_session="+state.VibeSession.SessionID)
	}
	return lines
}

func PrettyJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

// ExtractModelIDs pulls model ID strings from an operator ai.models response.
func ExtractModelIDs(data any) []string {
	m, _ := data.(map[string]any)
	if m == nil {
		return nil
	}
	raw, _ := m["models"].([]any)
	if raw == nil {
		return nil
	}
	var ids []string
	for _, item := range raw {
		entry, _ := item.(map[string]any)
		if id, _ := entry["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// ExtractAgentIDs pulls agent ID strings from an operator agents.list response.
func ExtractAgentIDs(data any) []string {
	m, _ := data.(map[string]any)
	if m == nil {
		return nil
	}
	raw, _ := m["agents"].([]any)
	if raw == nil {
		return nil
	}
	var ids []string
	for _, item := range raw {
		entry, _ := item.(map[string]any)
		if id, _ := entry["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func EmptyFallback(value, fallback string) string {
	return emptyFallback(value, fallback)
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
