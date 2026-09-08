package operator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveTranscriptWritesSessionStateHeader(t *testing.T) {
	state := NewSessionState()
	state.Mode = ModeVibe
	state.Model = "claude-sonnet-4-6"
	state.Project.Path = "/tmp/demo"
	state.VibeSession.SessionID = "vibe_123"
	state.Transcript.AddNote(state.Mode, "hello")

	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := SaveTranscript(path, state); err != nil {
		t.Fatalf("SaveTranscript: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 transcript records, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"kind":"session_state"`) {
		t.Fatalf("expected session_state header, got %q", lines[0])
	}
	if !strings.Contains(lines[0], `"session_id":"vibe_123"`) {
		t.Fatalf("expected vibe session metadata, got %q", lines[0])
	}
}
