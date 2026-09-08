package operator

import (
	"strings"
	"testing"
)

func TestBuildVibeLocalData(t *testing.T) {
	state := NewSessionState()
	state.Project.ID = "project-123"
	state.Project.Name = "itunes-in-vue-with-vite"
	state.Project.Path = "/Users/flakerim/ConstructProjects/itunes-in-vue-with-vite"
	state.Project.ProjectsRoot = "/Users/flakerim/ConstructProjects"
	state.Project.Spaces = []string{"vibe", "docs"}
	state.VibeSession.SessionID = "vibe_123"
	state.VibeSession.Status = "implementing"
	state.VibeSession.CurrentPhase = "implement"

	localData := BuildVibeLocalData(state, "Build the app")
	if localData["project_name"] != "itunes-in-vue-with-vite" {
		t.Fatalf("expected project_name in local data, got %#v", localData["project_name"])
	}
	if localData["project_path"] != "/Users/flakerim/ConstructProjects/itunes-in-vue-with-vite" {
		t.Fatalf("expected project_path in local data, got %#v", localData["project_path"])
	}
	vibeSession, _ := localData["vibe_session"].(map[string]any)
	if vibeSession["session_id"] != "vibe_123" {
		t.Fatalf("expected vibe session id, got %#v", vibeSession["session_id"])
	}
}

func TestNormalizeStreamChunkToolResultAndOrchestration(t *testing.T) {
	result := NormalizeStreamChunk(StreamChunk{
		Type: "tool_result",
		Data: map[string]any{
			"tool":     "write_file",
			"call_id":  "call-1",
			"content":  "wrote file",
			"is_error": false,
		},
	})
	if result.Kind != EventToolResult {
		t.Fatalf("expected tool result, got %q", result.Kind)
	}
	if result.ToolName != "write_file" || result.CallID != "call-1" {
		t.Fatalf("unexpected tool result payload: %#v", result)
	}

	complete := NormalizeStreamChunk(StreamChunk{
		Type: "orchestration.complete",
		Data: map[string]any{"status": "completed"},
	})
	if complete.Kind != EventOrchestration {
		t.Fatalf("expected orchestration event, got %q", complete.Kind)
	}
	if !strings.Contains(complete.Content, "completed") {
		t.Fatalf("expected completion content, got %#v", complete.Content)
	}
}
