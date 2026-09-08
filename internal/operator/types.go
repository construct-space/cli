package operator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Mode string

const (
	ModeChat  Mode = "chat"
	ModeVibe  Mode = "vibe"
	ModeAgent Mode = "agent"
)

type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
}

type DisplayMessage struct {
	Role    string
	Content string
}

type ProjectState struct {
	ID           string
	Name         string
	Path         string
	ProjectsRoot string
	Spaces       []string
}

type VibeSessionMeta struct {
	SessionID     string `json:"session_id,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	Goal          string `json:"goal,omitempty"`
	Source        string `json:"source,omitempty"`
	Space         string `json:"space,omitempty"`
	Status        string `json:"status,omitempty"`
	CurrentPhase  string `json:"current_phase,omitempty"`
	SessionType   string `json:"session_type,omitempty"`
	AutonomyLevel string `json:"autonomy_level,omitempty"`
	ProjectName   string `json:"project_name,omitempty"`
	ProjectPath   string `json:"project_path,omitempty"`
}

type VibeStoredSession struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"project_id,omitempty"`
	ProjectName   string            `json:"project_name,omitempty"`
	ProjectPath   string            `json:"project_path,omitempty"`
	Goal          string            `json:"goal"`
	Source        string            `json:"source,omitempty"`
	SessionType   string            `json:"session_type,omitempty"`
	AutonomyLevel string            `json:"autonomy_level,omitempty"`
	Status        string            `json:"status,omitempty"`
	CurrentPhase  string            `json:"current_phase,omitempty"`
	NextStep      string            `json:"next_step,omitempty"`
	Verification  map[string]string `json:"verification,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	UpdatedAt     string            `json:"updated_at,omitempty"`
}

type VibeStoredEvent struct {
	ID        int                    `json:"id"`
	EventType string                 `json:"event_type"`
	Phase     string                 `json:"phase,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt string                 `json:"created_at,omitempty"`
}

type VibeStoredCheckpoint struct {
	ID          int    `json:"id"`
	Branch      string `json:"branch"`
	CommitHash  string `json:"commit_hash,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type VibeSessionDetails struct {
	Session     *VibeStoredSession     `json:"session,omitempty"`
	Events      []VibeStoredEvent      `json:"events,omitempty"`
	Checkpoints []VibeStoredCheckpoint `json:"checkpoints,omitempty"`
}

type EventKind string

const (
	EventAssistantText EventKind = "assistant_text"
	EventProgress      EventKind = "progress"
	EventToolCall      EventKind = "tool_call"
	EventToolResult    EventKind = "tool_result"
	EventVibeSession   EventKind = "vibe_session"
	EventOrchestration EventKind = "orchestration_complete"
	EventDone          EventKind = "done"
	EventError         EventKind = "error"
	EventInfo          EventKind = "info"
)

type Event struct {
	Kind      EventKind        `json:"kind"`
	RawType   string           `json:"raw_type"`
	Text      string           `json:"text,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
	CallID    string           `json:"call_id,omitempty"`
	Input     string           `json:"input,omitempty"`
	Content   string           `json:"content,omitempty"`
	IsError   bool             `json:"is_error,omitempty"`
	Session   *VibeSessionMeta `json:"session,omitempty"`
	Data      map[string]any   `json:"data,omitempty"`
	Timestamp string           `json:"timestamp,omitempty"`
}

type TranscriptRecord struct {
	Timestamp   string       `json:"timestamp"`
	Kind        string       `json:"kind"`
	Mode        Mode         `json:"mode,omitempty"`
	RequestType string       `json:"request_type,omitempty"`
	Payload     any          `json:"payload,omitempty"`
	Event       *Event       `json:"event,omitempty"`
	Chunk       *StreamChunk `json:"chunk,omitempty"`
	Note        string       `json:"note,omitempty"`
}

type Transcript struct {
	mu      sync.Mutex
	Records []TranscriptRecord
}

func (t *Transcript) AddRequest(mode Mode, requestType string, payload any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Records = append(t.Records, TranscriptRecord{
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Kind:        "request",
		Mode:        mode,
		RequestType: requestType,
		Payload:     payload,
	})
}

func (t *Transcript) AddChunk(mode Mode, chunk StreamChunk) {
	t.mu.Lock()
	defer t.mu.Unlock()
	copyChunk := chunk
	t.Records = append(t.Records, TranscriptRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Kind:      "chunk",
		Mode:      mode,
		Chunk:     &copyChunk,
	})
}

func (t *Transcript) AddNote(mode Mode, note string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Records = append(t.Records, TranscriptRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Kind:      "note",
		Mode:      mode,
		Note:      note,
	})
}

func SaveTranscript(path string, state *SessionState) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(TranscriptRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Kind:      "session_state",
		Mode:      state.Mode,
		Payload: map[string]any{
			"mode":         state.Mode,
			"agent_id":     state.AgentID,
			"model":        state.Model,
			"status":       state.Status,
			"project":      state.Project,
			"vibe_session": state.VibeSession,
		},
	}); err != nil {
		return err
	}
	state.Transcript.mu.Lock()
	defer state.Transcript.mu.Unlock()
	for _, record := range state.Transcript.Records {
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

type SessionState struct {
	Mode         Mode
	AgentID      string
	Model        string
	Status       string
	Project      ProjectState
	VibeSession  VibeSessionMeta
	Messages     []DisplayMessage
	Conversation []ConversationMessage
	Transcript   Transcript

	PendingAssistant string
}

func NewSessionState() *SessionState {
	return &SessionState{
		Mode:    ModeChat,
		AgentID: "general",
		Model:   "",
		Status:  "Ready",
		Messages: []DisplayMessage{
			{Role: "system", Content: "Connected. Use /mode vibe to reproduce a Vibe run."},
		},
	}
}

func (s *SessionState) AddUserMessage(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.Messages = append(s.Messages, DisplayMessage{Role: "user", Content: trimmed})
	s.Conversation = append(s.Conversation, ConversationMessage{Role: "user", Content: trimmed})
}

func (s *SessionState) AddSystemMessage(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.Messages = append(s.Messages, DisplayMessage{Role: "system", Content: trimmed})
}

func (s *SessionState) AddErrorMessage(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.Messages = append(s.Messages, DisplayMessage{Role: "error", Content: trimmed})
}

func (s *SessionState) ResetConversation() {
	s.Messages = nil
	s.Conversation = nil
	s.PendingAssistant = ""
	s.VibeSession = VibeSessionMeta{}
	s.Transcript = Transcript{}
	s.Status = "Ready"
}

func (s *SessionState) StartAssistantStream() {
	s.PendingAssistant = ""
	s.Messages = append(s.Messages, DisplayMessage{Role: "assistant", Content: ""})
}

func (s *SessionState) SetStreamingAssistant(text string) {
	s.PendingAssistant += text
	if len(s.Messages) == 0 || s.Messages[len(s.Messages)-1].Role != "assistant" {
		s.Messages = append(s.Messages, DisplayMessage{Role: "assistant", Content: s.PendingAssistant})
		return
	}
	s.Messages[len(s.Messages)-1].Content = s.PendingAssistant + "~"
}

func (s *SessionState) FinalizeAssistant(finalContent string) {
	content := strings.TrimSpace(finalContent)
	if content == "" {
		content = strings.TrimSpace(s.PendingAssistant)
	}
	if len(s.Messages) > 0 && s.Messages[len(s.Messages)-1].Role == "assistant" {
		s.Messages[len(s.Messages)-1].Content = content
	} else if content != "" {
		s.Messages = append(s.Messages, DisplayMessage{Role: "assistant", Content: content})
	}
	if content != "" {
		s.Conversation = append(s.Conversation, ConversationMessage{Role: "assistant", Content: content})
	}
	s.PendingAssistant = ""
}

func (s *SessionState) ToolMessage(prefix, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.Messages = append(s.Messages, DisplayMessage{Role: "tool", Content: prefix + trimmed})
}

func (s *SessionState) SyncProjectFromVibe() {
	if strings.TrimSpace(s.VibeSession.ProjectName) != "" {
		s.Project.Name = strings.TrimSpace(s.VibeSession.ProjectName)
	}
	if strings.TrimSpace(s.VibeSession.ProjectPath) != "" {
		s.Project.Path = strings.TrimSpace(s.VibeSession.ProjectPath)
	}
	if strings.TrimSpace(s.VibeSession.ProjectID) != "" {
		s.Project.ID = strings.TrimSpace(s.VibeSession.ProjectID)
	}
}
