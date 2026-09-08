package shell

import (
	"strings"
	"testing"
)

func TestParseArchitectQuestions(t *testing.T) {
	content := strings.Join([]string{
		"```json",
		`[{"id":"platform","label":"Which platform?","description":"Pick one","type":"single","options":[{"value":"web","label":"Web Application"}]},{"id":"features","label":"Which features?","description":"Pick as many as fit","type":"multi","options":[{"value":"auth","label":"Authentication"},{"value":"billing","label":"Billing"}]}]`,
		"```",
	}, "\n")

	questions, err := ParseArchitectQuestions(content)
	if err != nil {
		t.Fatalf("ParseArchitectQuestions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
	if questions[0].Type != "single" || questions[1].Type != "multi" {
		t.Fatalf("unexpected question types: %#v", questions)
	}
}

func TestBuildArchitectPlanTask(t *testing.T) {
	task := BuildArchitectPlanTask(
		"Build a dashboard",
		[]ArchitectQuestion{{ID: "platform", Type: "single", Options: []ArchitectOption{{Value: "web", Label: "Web"}}}},
		map[string]any{"platform": "web"},
	)
	if !strings.Contains(task, "Generate a project plan") {
		t.Fatalf("expected plan prompt text, got %q", task)
	}
	if !strings.Contains(task, `"platform": "web"`) {
		t.Fatalf("expected serialized answers in plan prompt, got %q", task)
	}
}
