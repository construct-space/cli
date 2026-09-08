package shell

import (
	"fmt"
	"strings"
)

type Example struct {
	ID          string
	Name        string
	Framework   string
	Language    string
	Mode        string
	ProjectName string
	Prompt      string
	Outcome     string
}

var builtInExamples = []Example{
	{
		ID:          "flutter",
		Name:        "Habit Garden",
		Framework:   "Flutter",
		Language:    "Dart",
		Mode:        "vibe",
		ProjectName: "habit-garden",
		Prompt:      "Build a Flutter mobile app called Habit Garden for tracking daily habits. Use local persistence, onboarding, streak tracking, a progress dashboard, polished empty states, and production-ready UI. Create docs, scaffold the app, and keep the structure ready for future syncing.",
		Outcome:     "Cross-platform Flutter app with onboarding, habit tracking, streaks, and a documented project layout.",
	},
	{
		ID:          "go",
		Name:        "Ops Pulse",
		Framework:   "Go",
		Language:    "Go",
		Mode:        "vibe",
		ProjectName: "ops-pulse",
		Prompt:      "Build a Go web service called Ops Pulse with a JSON API for service status, incident timelines, and health checks. Use idiomatic Go project structure, structured logging, configuration via environment variables, tests for the core handlers, and docs explaining local development.",
		Outcome:     "Idiomatic Go API service with health endpoints, incident resources, tests, and deployment-ready docs.",
	},
	{
		ID:          "rails",
		Name:        "Studio Booker",
		Framework:   "Rails",
		Language:    "Ruby",
		Mode:        "vibe",
		ProjectName: "studio-booker",
		Prompt:      "Build a Ruby on Rails booking app called Studio Booker for reserving recording rooms. Include rooms, bookings, calendar views, admin management, validation for overlapping reservations, and clean server-rendered UI with modern styling. Generate docs and initialize the app end to end.",
		Outcome:     "Rails scheduling app with room booking flows, admin CRUD, validations, and end-to-end setup docs.",
	},
	{
		ID:          "nuxt",
		Name:        "Field Notes",
		Framework:   "Nuxt",
		Language:    "TypeScript",
		Mode:        "vibe",
		ProjectName: "field-notes",
		Prompt:      "Build a Nuxt 3 app called Field Notes for publishing outdoor travel journals. Include a bold editorial homepage, article listing, article detail pages, tag filters, author cards, and a distinctive visual system. Use TypeScript, pure CSS or the project's chosen styling approach, and create the full Construct project layout with docs.",
		Outcome:     "Nuxt editorial app with branded pages, content browsing, and a complete Construct project structure.",
	},
	{
		ID:          "next",
		Name:        "Signal Desk",
		Framework:   "Next.js",
		Language:    "TypeScript",
		Mode:        "vibe",
		ProjectName: "signal-desk",
		Prompt:      "Build a Next.js app called Signal Desk for monitoring product launches. Include a dashboard, launch cards, filters, a detail view, a compact metrics panel, and a strong visual identity. Use modern app-router patterns, real UI polish, and generate docs plus runnable scaffolding in the standard Construct layout.",
		Outcome:     "Next.js dashboard app with app-router structure, polished monitoring UI, and complete docs.",
	},
}

func Examples() []Example {
	out := make([]Example, len(builtInExamples))
	copy(out, builtInExamples)
	return out
}

func FindExample(id string) (Example, bool) {
	needle := strings.TrimSpace(strings.ToLower(id))
	for _, example := range builtInExamples {
		if strings.ToLower(example.ID) == needle {
			return example, true
		}
	}
	return Example{}, false
}

func FormatExamplesList() string {
	lines := []string{"Built-in examples:"}
	for _, example := range builtInExamples {
		lines = append(lines, fmt.Sprintf("- %s: %s (%s, %s)", example.ID, example.Name, example.Framework, example.Language))
	}
	lines = append(lines, "Use /example show <id> to inspect one or /example run <id> to launch it.")
	return strings.Join(lines, "\n")
}

func FormatExample(example Example) string {
	return strings.Join([]string{
		fmt.Sprintf("%s (%s / %s)", example.Name, example.Framework, example.Language),
		"Mode: " + example.Mode,
		"Project: " + example.ProjectName,
		"Prompt:",
		example.Prompt,
		"Expected outcome:",
		example.Outcome,
	}, "\n")
}
