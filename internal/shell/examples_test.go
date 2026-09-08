package shell

import (
	"strings"
	"testing"
)

func TestExamplesIncludesRequestedFrameworks(t *testing.T) {
	examples := Examples()
	required := map[string]bool{
		"flutter": false,
		"go":      false,
		"rails":   false,
		"nuxt":    false,
		"next":    false,
	}
	for _, example := range examples {
		if _, ok := required[example.ID]; ok {
			required[example.ID] = true
		}
	}
	for id, ok := range required {
		if !ok {
			t.Fatalf("expected example %q to exist", id)
		}
	}
}

func TestFindExample(t *testing.T) {
	example, ok := FindExample("Nuxt")
	if !ok {
		t.Fatal("expected case-insensitive example lookup to succeed")
	}
	if example.Framework != "Nuxt" {
		t.Fatalf("expected Nuxt example, got %#v", example)
	}
}

func TestFormatExamplesList(t *testing.T) {
	output := FormatExamplesList()
	if !strings.Contains(output, "flutter") || !strings.Contains(output, "next") {
		t.Fatalf("expected output to list examples, got %q", output)
	}
}
