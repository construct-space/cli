package shell

import (
	"testing"

	"github.com/construct-space/cli/internal/operator"
)

func TestSetMode(t *testing.T) {
	state := operator.NewSessionState()
	if err := SetMode(state, "vibe"); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if state.Mode != operator.ModeVibe {
		t.Fatalf("expected vibe mode, got %q", state.Mode)
	}
}

func TestSetProjectPathRequiresAbsolutePath(t *testing.T) {
	state := operator.NewSessionState()
	if err := SetProjectPath(state, "relative/path"); err == nil {
		t.Fatal("expected relative path to fail")
	}
}

func TestSetProjectPathSetsNameWhenMissing(t *testing.T) {
	state := operator.NewSessionState()
	if err := SetProjectPath(state, "/tmp/demo-app"); err != nil {
		t.Fatalf("SetProjectPath: %v", err)
	}
	if state.Project.Name != "demo-app" {
		t.Fatalf("expected project name demo-app, got %q", state.Project.Name)
	}
}

func TestSetModel(t *testing.T) {
	state := operator.NewSessionState()
	if err := SetModel(state, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	if state.Model != "claude-sonnet-4-6" {
		t.Fatalf("expected model to be set, got %q", state.Model)
	}
}
