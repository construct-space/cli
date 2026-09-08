package shell

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ArchitectOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Icon        string `json:"icon,omitempty"`
	Description string `json:"description,omitempty"`
}

type ArchitectQuestion struct {
	ID          string            `json:"id"`
	Label       string            `json:"label,omitempty"`
	Question    string            `json:"question,omitempty"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Options     []ArchitectOption `json:"options"`
}

func (q ArchitectQuestion) Prompt() string {
	if strings.TrimSpace(q.Label) != "" {
		return strings.TrimSpace(q.Label)
	}
	return strings.TrimSpace(q.Question)
}

type ArchitectInterview struct {
	Description string
	Questions   []ArchitectQuestion
	Answers     map[string]any
	Index       int
	Plan        map[string]any
}

func BuildArchitectQuestionsTask(description string) string {
	return strings.Join([]string{
		"Generate interview questions for this project description.",
		"Output ONLY a JSON array of questions, no other text.",
		`Each question: {id, label, description, type: "single"|"multi", options: [{value, label, icon?, description?}]}`,
		"",
		"Description: " + strings.TrimSpace(description),
	}, "\n")
}

func BuildArchitectPlanTask(description string, questions []ArchitectQuestion, answers map[string]any) string {
	serializedAnswers, _ := json.MarshalIndent(normalizeArchitectAnswers(questions, answers), "", "  ")
	return strings.Join([]string{
		"Generate a project plan based on these decisions.",
		"Output ONLY a JSON object with: {name, description, stack: {layer: tech}, features: [], files: [{path, description}], phases: [{name, tasks: []}]}",
		"",
		"Project: " + strings.TrimSpace(description),
		"",
		"Decisions:",
		string(serializedAnswers),
	}, "\n")
}

func ParseArchitectQuestions(content string) ([]ArchitectQuestion, error) {
	cleaned := strings.TrimSpace(extractJSON(content))
	if cleaned == "" {
		return nil, fmt.Errorf("empty architect response")
	}

	tryParse := func(candidate string) ([]ArchitectQuestion, error) {
		var questions []ArchitectQuestion
		if err := json.Unmarshal([]byte(candidate), &questions); err == nil && len(questions) > 0 {
			return normalizeArchitectQuestions(questions), nil
		}

		var wrapped struct {
			Questions []ArchitectQuestion `json:"questions"`
			Data      []ArchitectQuestion `json:"data"`
			Message   []ArchitectQuestion `json:"message"`
		}
		if err := json.Unmarshal([]byte(candidate), &wrapped); err == nil {
			switch {
			case len(wrapped.Questions) > 0:
				return normalizeArchitectQuestions(wrapped.Questions), nil
			case len(wrapped.Data) > 0:
				return normalizeArchitectQuestions(wrapped.Data), nil
			case len(wrapped.Message) > 0:
				return normalizeArchitectQuestions(wrapped.Message), nil
			}
		}
		return nil, fmt.Errorf("response did not contain architect questions")
	}

	if strings.HasPrefix(cleaned, "[") {
		return tryParse(cleaned)
	}
	if strings.HasPrefix(cleaned, "{") {
		if questions, err := tryParse(cleaned); err == nil {
			return questions, nil
		}
	}

	if candidate := findFirstJSONArray(cleaned); candidate != "" {
		return tryParse(candidate)
	}
	if candidate := findFirstJSONObject(cleaned); candidate != "" {
		return tryParse(candidate)
	}

	return nil, fmt.Errorf("failed to parse architect questions")
}

func ParseArchitectPlan(content string) (map[string]any, error) {
	cleaned := strings.TrimSpace(extractJSON(content))
	if cleaned == "" {
		return nil, fmt.Errorf("empty architect plan response")
	}
	if candidate := firstJSONCandidate(cleaned); candidate != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(candidate), &payload); err == nil && len(payload) > 0 {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("failed to parse architect plan")
}

func normalizeArchitectQuestions(input []ArchitectQuestion) []ArchitectQuestion {
	questions := make([]ArchitectQuestion, 0, len(input))
	for _, question := range input {
		question.Type = strings.ToLower(strings.TrimSpace(question.Type))
		if question.Type != "single" && question.Type != "multi" {
			continue
		}
		if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.Prompt()) == "" || len(question.Options) == 0 {
			continue
		}
		for index, option := range question.Options {
			if strings.TrimSpace(option.Value) == "" {
				question.Options[index].Value = strings.TrimSpace(option.Label)
			}
			if strings.TrimSpace(question.Options[index].Label) == "" {
				question.Options[index].Label = question.Options[index].Value
			}
		}
		questions = append(questions, question)
	}
	return questions
}

func normalizeArchitectAnswers(questions []ArchitectQuestion, answers map[string]any) map[string]any {
	result := make(map[string]any)
	if len(answers) == 0 {
		return result
	}
	validIDs := make(map[string]ArchitectQuestion, len(questions))
	for _, question := range questions {
		validIDs[question.ID] = question
	}
	for questionID, raw := range answers {
		question, ok := validIDs[questionID]
		if !ok {
			result[questionID] = raw
			continue
		}
		switch typed := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				result[question.ID] = trimmed
			}
		case []string:
			values := make([]string, 0, len(typed))
			for _, value := range typed {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					values = append(values, trimmed)
				}
			}
			if len(values) > 0 {
				result[question.ID] = values
			}
		case []any:
			values := make([]string, 0, len(typed))
			for _, value := range typed {
				if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
					values = append(values, strings.TrimSpace(text))
				}
			}
			if len(values) > 0 {
				result[question.ID] = values
			}
		}
	}
	return result
}

func extractJSON(content string) string {
	replacer := strings.NewReplacer("```json", "", "```", "")
	return strings.TrimSpace(replacer.Replace(content))
}

func firstJSONCandidate(cleaned string) string {
	if strings.HasPrefix(cleaned, "{") {
		return cleaned
	}
	if candidate := findFirstJSONObject(cleaned); candidate != "" {
		return candidate
	}
	return ""
}

func findFirstJSONArray(text string) string {
	return findFirstJSON(text, '[', ']')
}

func findFirstJSONObject(text string) string {
	return findFirstJSON(text, '{', '}')
}

func findFirstJSON(text string, open, close byte) string {
	for start := 0; start < len(text); start++ {
		if text[start] != open {
			continue
		}
		depth := 0
		inString := false
		escaped := false
		for i := start; i < len(text); i++ {
			ch := text[i]
			if inString {
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == '"' {
					inString = false
				}
				continue
			}
			if ch == '"' {
				inString = true
				continue
			}
			if ch == open {
				depth++
			}
			if ch == close {
				depth--
				if depth == 0 {
					return text[start : i+1]
				}
			}
		}
	}
	return ""
}
