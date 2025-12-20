package a2a

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageRole indicates the originator of a message.
type MessageRole string

const (
	// MessageRoleUser is the role for messages from the user/client.
	MessageRoleUser MessageRole = "user"
	// MessageRoleAgent is the role for messages from the agent/server.
	MessageRoleAgent MessageRole = "agent"
)

// TaskState represents the lifecycle state of a task.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateFailed        TaskState = "failed"
	TaskStateRejected      TaskState = "rejected"
	TaskStateAuthRequired  TaskState = "auth-required"
)

// IsTerminal returns true if the state is a terminal state.
func (s TaskState) IsTerminal() bool {
	switch s {
	case TaskStateCompleted, TaskStateCanceled, TaskStateFailed, TaskStateRejected:
		return true
	default:
		return false
	}
}

// Message represents a single exchange between a user and an agent.
type Message struct {
	Kind             string         `json:"kind"`
	MessageID        string         `json:"messageId"`
	Role             MessageRole    `json:"role"`
	Parts            []Part         `json:"parts"`
	ContextID        *string        `json:"contextId,omitempty"`
	TaskID           *string        `json:"taskId,omitempty"`
	ReferenceTaskIDs []string       `json:"referenceTaskIds,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	Extensions       []string       `json:"extensions,omitempty"`
}

// NewMessage creates a new message with the given role and parts.
func NewMessage(role MessageRole, parts ...Part) Message {
	return Message{
		Kind:      "message",
		MessageID: uuid.New().String(),
		Role:      role,
		Parts:     parts,
	}
}

// NewMessageWithContext creates a new message with context and optional task ID.
func NewMessageWithContext(role MessageRole, contextID string, taskID *string, parts ...Part) Message {
	m := NewMessage(role, parts...)
	m.ContextID = &contextID
	m.TaskID = taskID
	return m
}

// TextContent returns the concatenated text from all TextParts in the message.
func (m Message) TextContent() string {
	var text string
	for _, p := range m.Parts {
		if tp, ok := p.(TextPart); ok {
			text += tp.Text
		}
	}
	return text
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
// This is needed because Parts is a []Part interface which can't be
// directly unmarshaled.
func (m *Message) UnmarshalJSON(data []byte) error {
	// Unmarshal into a temporary struct with Parts as raw JSON
	type messageAlias Message
	var tmp struct {
		messageAlias
		Parts []json.RawMessage `json:"parts"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	*m = Message(tmp.messageAlias)
	m.Parts = make([]Part, 0, len(tmp.Parts))

	for _, raw := range tmp.Parts {
		part, err := UnmarshalPart(raw)
		if err != nil {
			return err
		}
		m.Parts = append(m.Parts, part)
	}

	return nil
}

// Part represents a segment of a message (text, file, or data).
type Part interface {
	partMarker()
	GetKind() string
}

// TextPart represents a text segment within a message.
type TextPart struct {
	Kind     string         `json:"kind"`
	Text     string         `json:"text"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (TextPart) partMarker()       {}
func (p TextPart) GetKind() string { return p.Kind }

// NewTextPart creates a new TextPart with the given text.
func NewTextPart(text string) TextPart {
	return TextPart{Kind: "text", Text: text}
}

// FilePart represents a file included in a message.
type FilePart struct {
	Kind     string         `json:"kind"`
	File     FileContent    `json:"file"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (FilePart) partMarker()       {}
func (p FilePart) GetKind() string { return p.Kind }

// FileContent represents file content, either inline bytes or a URI reference.
type FileContent struct {
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Bytes    string `json:"bytes,omitempty"` // Base64 encoded
	URI      string `json:"uri,omitempty"`
}

// NewFilePartWithBytes creates a FilePart with inline base64-encoded content.
func NewFilePartWithBytes(name, mimeType, bytes string) FilePart {
	return FilePart{
		Kind: "file",
		File: FileContent{Name: name, MimeType: mimeType, Bytes: bytes},
	}
}

// NewFilePartWithURI creates a FilePart with a URI reference.
func NewFilePartWithURI(name, mimeType, uri string) FilePart {
	return FilePart{
		Kind: "file",
		File: FileContent{Name: name, MimeType: mimeType, URI: uri},
	}
}

// DataPart represents arbitrary structured data (JSON) within a message.
type DataPart struct {
	Kind     string         `json:"kind"`
	Data     any            `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (DataPart) partMarker()       {}
func (p DataPart) GetKind() string { return p.Kind }

// NewDataPart creates a new DataPart with the given data.
func NewDataPart(data any) DataPart {
	return DataPart{Kind: "data", Data: data}
}

// TaskStatus represents the current status of a task.
type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   *Message  `json:"message,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
}

// NewTaskStatus creates a new TaskStatus with the given state.
func NewTaskStatus(state TaskState) TaskStatus {
	return TaskStatus{
		State:     state,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// NewTaskStatusWithMessage creates a new TaskStatus with a message.
func NewTaskStatusWithMessage(state TaskState, msg *Message) TaskStatus {
	return TaskStatus{
		State:     state,
		Message:   msg,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Task represents a unit of work being processed by the agent.
type Task struct {
	Kind      string         `json:"kind"`
	ID        string         `json:"id"`
	ContextID string         `json:"contextId"`
	Status    TaskStatus     `json:"status"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	History   []Message      `json:"history,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewTask creates a new task with the given ID and context ID.
func NewTask(id, contextID string) *Task {
	return &Task{
		Kind:      "task",
		ID:        id,
		ContextID: contextID,
		Status:    NewTaskStatus(TaskStateSubmitted),
	}
}

// Artifact represents an output generated by a task.
type Artifact struct {
	ArtifactID  string         `json:"artifactId"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parts       []Part         `json:"parts"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Extensions  []string       `json:"extensions,omitempty"`
}

// NewArtifact creates a new artifact with the given parts.
func NewArtifact(name, description string, parts ...Part) Artifact {
	return Artifact{
		ArtifactID:  uuid.New().String(),
		Name:        name,
		Description: description,
		Parts:       parts,
	}
}

// TaskStatusUpdateEvent represents a streaming task status update.
type TaskStatusUpdateEvent struct {
	Kind      string     `json:"kind"`
	TaskID    string     `json:"taskId"`
	ContextID string     `json:"contextId"`
	Status    TaskStatus `json:"status"`
	Final     bool       `json:"final"`
}

// NewTaskStatusUpdateEvent creates a new task status update event.
func NewTaskStatusUpdateEvent(taskID, contextID string, status TaskStatus, final bool) TaskStatusUpdateEvent {
	return TaskStatusUpdateEvent{
		Kind:      "status-update",
		TaskID:    taskID,
		ContextID: contextID,
		Status:    status,
		Final:     final,
	}
}

// TaskArtifactUpdateEvent represents a streaming artifact update.
type TaskArtifactUpdateEvent struct {
	Kind      string   `json:"kind"`
	TaskID    string   `json:"taskId"`
	ContextID string   `json:"contextId"`
	Artifact  Artifact `json:"artifact"`
}

// NewTaskArtifactUpdateEvent creates a new task artifact update event.
func NewTaskArtifactUpdateEvent(taskID, contextID string, artifact Artifact) TaskArtifactUpdateEvent {
	return TaskArtifactUpdateEvent{
		Kind:      "artifact-update",
		TaskID:    taskID,
		ContextID: contextID,
		Artifact:  artifact,
	}
}

// MarshalPart marshals a Part to JSON with the correct type.
func MarshalPart(p Part) ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalPart unmarshals a Part from JSON.
func UnmarshalPart(data []byte) (Part, error) {
	var raw struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	switch raw.Kind {
	case "text":
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "file":
		var p FilePart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "data":
		var p DataPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		// Unknown part type, return as DataPart
		var p DataPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	}
}
