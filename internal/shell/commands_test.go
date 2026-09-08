package shell

import "testing"

func TestParseCommand(t *testing.T) {
	command, ok := Parse("/mode vibe")
	if !ok {
		t.Fatal("expected slash command to parse")
	}
	if command.Name != "mode" {
		t.Fatalf("expected command name mode, got %q", command.Name)
	}
	if command.Arg != "vibe" {
		t.Fatalf("expected arg vibe, got %q", command.Arg)
	}
}

func TestParseNonCommand(t *testing.T) {
	if _, ok := Parse("hello world"); ok {
		t.Fatal("expected plain input to bypass command parsing")
	}
}
