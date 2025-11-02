package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Event types for OpenAI Realtime API
const (
	EventTypeSessionCreated             = "session.created"
	EventTypeSessionUpdate              = "session.update"
	EventTypeSessionUpdated             = "session.updated"
	EventTypeConversationCreated        = "conversation.created"
	EventTypeInputAudioBufferAppend     = "input_audio_buffer.append"
	EventTypeInputAudioBufferCommit     = "input_audio_buffer.commit"
	EventTypeInputAudioBufferCommitted  = "input_audio_buffer.committed"
	EventTypeInputAudioBufferClear      = "input_audio_buffer.clear"
	EventTypeInputAudioBufferSpeechStarted = "input_audio_buffer.speech_started"
	EventTypeInputAudioBufferSpeechStopped = "input_audio_buffer.speech_stopped"
	EventTypeHeartbeatPing              = "heartbeat.ping"
	EventTypeHeartbeatPong              = "heartbeat.pong"
	EventTypeConversationItemCreated    = "conversation.item.created"
	EventTypeConversationItemInputAudioTranscriptionCompleted = "conversation.item.input_audio_transcription.completed"
	EventTypeConversationItemInputAudioTranscriptionFailed = "conversation.item.input_audio_transcription.failed"
	EventTypeConversationItemDeleted    = "conversation.item.deleted"
	EventTypeInputAudioBufferCleared    = "input_audio_buffer.cleared"
	EventTypeError                      = "error"
)

// BaseEvent represents the common structure for all OpenAI events
type BaseEvent struct {
	Type      string `json:"type"`
	EventID   string `json:"event_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// SessionCreatedEvent represents session.created event
type SessionCreatedEvent struct {
	BaseEvent
	Session struct {
		ID         string   `json:"id"`
		Object     string   `json:"object"`
		Model      string   `json:"model"`
		Modalities []string `json:"modalities"`
	} `json:"session"`
}

// SessionUpdateEvent represents session.update event
type SessionUpdateEvent struct {
	BaseEvent
	Session struct {
		ID        string `json:"id"`
		Modality  string `json:"modality"`
		Instructions string `json:"instructions,omitempty"`
		Voice     string `json:"voice,omitempty"`
		InputAudioFormat struct {
			Type           string `json:"type"`
			SampleRate     int    `json:"sample_rate"`
			Channels       int    `json:"channels"`
		} `json:"input_audio_format,omitempty"`
		OutputAudioFormat struct {
			Type       string `json:"type"`
			SampleRate int    `json:"sample_rate"`
			Voice      string `json:"voice,omitempty"`
		} `json:"output_audio_format,omitempty"`
		InputAudioTranscription *struct {
			Model    string `json:"model"`
			Language string `json:"language"`
		} `json:"input_audio_transcription,omitempty"`
		TurnDetection *struct {
			Type              string  `json:"type"`
			Threshold         float32 `json:"threshold"`
			PrefixPaddingMs   int     `json:"prefix_padding_ms"`
			SilenceDurationMs int     `json:"silence_duration_ms"`
		} `json:"turn_detection,omitempty"`
		Tools []interface{} `json:"tools,omitempty"`
		ToolChoice string `json:"tool_choice,omitempty"`
	} `json:"session"`
}

// SessionUpdatedEvent represents session.updated event
type SessionUpdatedEvent struct {
	BaseEvent
	Session struct {
		ID         string   `json:"id"`
		Object     string   `json:"object"`
		Model      string   `json:"model"`
		Modalities []string `json:"modalities"`
	} `json:"session"`
}

// ConversationCreatedEvent represents conversation.created event
type ConversationCreatedEvent struct {
	BaseEvent
	Conversation struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	} `json:"conversation"`
}

// InputAudioBufferAppendEvent represents input_audio_buffer.append event
type InputAudioBufferAppendEvent struct {
	BaseEvent
	Audio string `json:"audio"` // Base64 encoded audio data
}

// InputAudioBufferCommitEvent represents input_audio_buffer.commit event
type InputAudioBufferCommitEvent struct {
	BaseEvent
}

// InputAudioBufferCommittedEvent represents input_audio_buffer.committed event
type InputAudioBufferCommittedEvent struct {
	BaseEvent
}

// InputAudioBufferClearEvent represents input_audio_buffer.clear event
type InputAudioBufferClearEvent struct {
	BaseEvent
}

// InputAudioBufferSpeechStartedEvent represents input_audio_buffer.speech_started event
type InputAudioBufferSpeechStartedEvent struct {
	BaseEvent
	AudioStartMs int `json:"audio_start_ms"`
}

// InputAudioBufferSpeechStoppedEvent represents input_audio_buffer.speech_stopped event
type InputAudioBufferSpeechStoppedEvent struct {
	BaseEvent
	AudioEndMs int `json:"audio_end_ms"`
}

// ConversationItemCreatedEvent represents conversation.item.created event
type ConversationItemCreatedEvent struct {
	BaseEvent
	Item struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		Audio     *struct {
			Data string `json:"data"` // Base64 encoded audio
			Format string `json:"format"`
		} `json:"audio,omitempty"`
		Content   []interface{} `json:"content,omitempty"`
	} `json:"item"`
}

// ConversationItemInputAudioTranscriptionCompletedEvent represents transcription completed event
type ConversationItemInputAudioTranscriptionCompletedEvent struct {
	BaseEvent
	Item struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		Content   []struct {
			Type      string `json:"type"`
			Transcript string `json:"transcript"`
		} `json:"content"`
	} `json:"item"`
}

// ConversationItemInputAudioTranscriptionFailedEvent represents transcription failed event
type ConversationItemInputAudioTranscriptionFailedEvent struct {
	BaseEvent
	ItemID string `json:"item_id"`
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// ConversationItemDeletedEvent represents conversation.item.deleted event
type ConversationItemDeletedEvent struct {
	BaseEvent
	ItemID string `json:"item_id"`
}

// InputAudioBufferClearedEvent represents input_audio_buffer.cleared event
type InputAudioBufferClearedEvent struct {
	BaseEvent
}

// HeartbeatPingEvent represents heartbeat.ping event
type HeartbeatPingEvent struct {
	BaseEvent
	HeartbeatType int `json:"heartbeat_type"`
}

// HeartbeatPongEvent represents heartbeat.pong event
type HeartbeatPongEvent struct {
	BaseEvent
	HeartbeatType int `json:"heartbeat_type"`
}

// ErrorEvent represents error event
type ErrorEvent struct {
	BaseEvent
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// EventParser handles parsing and validation of OpenAI events
type EventParser struct{}

// NewEventParser creates a new event parser
func NewEventParser() *EventParser {
	return &EventParser{}
}

// ParseEvent parses a JSON message into the appropriate event type
func (p *EventParser) ParseEvent(data []byte) (interface{}, error) {
	var baseEvent BaseEvent
	if err := json.Unmarshal(data, &baseEvent); err != nil {
		return nil, fmt.Errorf("failed to parse base event: %v", err)
	}

	if baseEvent.Type == "" {
		return nil, fmt.Errorf("event type is required")
	}

	switch baseEvent.Type {
	case EventTypeSessionCreated:
		var event SessionCreatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse session.created event: %v", err)
		}
		return &event, nil

	case EventTypeSessionUpdate:
		var event SessionUpdateEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse session.update event: %v", err)
		}
		return &event, nil

	case EventTypeConversationCreated:
		var event ConversationCreatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse conversation.created event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferAppend:
		var event InputAudioBufferAppendEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.append event: %v", err)
		}
		// Validate Base64 audio data
		if _, err := base64.StdEncoding.DecodeString(event.Audio); err != nil {
			return nil, fmt.Errorf("invalid Base64 audio data: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferCommit:
		var event InputAudioBufferCommitEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.commit event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferCommitted:
		var event InputAudioBufferCommittedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.committed event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferClear:
		var event InputAudioBufferClearEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.clear event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferSpeechStarted:
		var event InputAudioBufferSpeechStartedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.speech_started event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferSpeechStopped:
		var event InputAudioBufferSpeechStoppedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.speech_stopped event: %v", err)
		}
		return &event, nil

	case EventTypeHeartbeatPing:
		var event HeartbeatPingEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse heartbeat.ping event: %v", err)
		}
		return &event, nil

	case EventTypeHeartbeatPong:
		var event HeartbeatPongEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse heartbeat.pong event: %v", err)
		}
		return &event, nil

	case EventTypeConversationItemCreated:
		var event ConversationItemCreatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse conversation.item.created event: %v", err)
		}
		return &event, nil

	case EventTypeConversationItemInputAudioTranscriptionCompleted:
		var event ConversationItemInputAudioTranscriptionCompletedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse conversation.item.input_audio_transcription.completed event: %v", err)
		}
		return &event, nil

	case EventTypeConversationItemInputAudioTranscriptionFailed:
		var event ConversationItemInputAudioTranscriptionFailedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse conversation.item.input_audio_transcription.failed event: %v", err)
		}
		return &event, nil

	case EventTypeConversationItemDeleted:
		var event ConversationItemDeletedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse conversation.item.deleted event: %v", err)
		}
		return &event, nil

	case EventTypeInputAudioBufferCleared:
		var event InputAudioBufferClearedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse input_audio_buffer.cleared event: %v", err)
		}
		return &event, nil

	case EventTypeError:
		var event ErrorEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse error event: %v", err)
		}
		return &event, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", baseEvent.Type)
	}
}

// ValidateEvent validates an event against OpenAI Realtime API specifications
func (p *EventParser) ValidateEvent(event interface{}) error {
	switch e := event.(type) {
	case *SessionCreatedEvent:
		return p.validateSessionCreatedEvent(e)

	case *SessionUpdateEvent:
		return p.validateSessionUpdateEvent(e)
	case *ConversationCreatedEvent:
		return p.validateConversationCreatedEvent(e)
	case *InputAudioBufferAppendEvent:
		return p.validateInputAudioBufferAppendEvent(e)
	case *InputAudioBufferCommitEvent:
		return p.validateInputAudioBufferCommitEvent(e)
	case *InputAudioBufferCommittedEvent:
		return p.validateInputAudioBufferCommittedEvent(e)
	case *InputAudioBufferClearEvent:
		return p.validateInputAudioBufferClearEvent(e)
	case *InputAudioBufferSpeechStartedEvent:
		return p.validateInputAudioBufferSpeechStartedEvent(e)
	case *InputAudioBufferSpeechStoppedEvent:
		return p.validateInputAudioBufferSpeechStoppedEvent(e)
	case *ConversationItemCreatedEvent:
		return p.validateConversationItemCreatedEvent(e)
	case *ConversationItemInputAudioTranscriptionCompletedEvent:
		return p.validateConversationItemInputAudioTranscriptionCompletedEvent(e)
	case *ConversationItemInputAudioTranscriptionFailedEvent:
		return p.validateConversationItemInputAudioTranscriptionFailedEvent(e)
	case *ConversationItemDeletedEvent:
		return p.validateConversationItemDeletedEvent(e)
	case *InputAudioBufferClearedEvent:
		return p.validateInputAudioBufferClearedEvent(e)
	case *ErrorEvent:
		return p.validateErrorEvent(e)
	case *HeartbeatPingEvent:
		return p.validateHeartbeatPingEvent(e)
	case *HeartbeatPongEvent:
		return p.validateHeartbeatPongEvent(e)
	default:
		return fmt.Errorf("unknown event type for validation")
	}
}

func (p *EventParser) validateSessionCreatedEvent(event *SessionCreatedEvent) error {
	if event.Session.ID == "" {
		return fmt.Errorf("session ID is required")
	}
	if event.Session.Object == "" {
		return fmt.Errorf("session object is required")
	}
	if event.Session.Model == "" {
		return fmt.Errorf("session model is required")
	}
	if len(event.Session.Modalities) == 0 {
		return fmt.Errorf("session modalities are required")
	}
	return nil
}

func (p *EventParser) validateSessionUpdateEvent(event *SessionUpdateEvent) error {
	// Session ID can be empty for initial session creation
	// The server will assign a session ID if not provided
	if event.Session.Modality == "" {
		return fmt.Errorf("session modality is required")
	}
	if event.Session.Modality != "text" && event.Session.Modality != "audio" && event.Session.Modality != "text_and_audio" {
		return fmt.Errorf("invalid session modality: %s", event.Session.Modality)
	}
	return nil
}

func (p *EventParser) validateConversationCreatedEvent(event *ConversationCreatedEvent) error {
	if event.Conversation.ID == "" {
		return fmt.Errorf("conversation ID is required")
	}
	if event.Conversation.Object == "" {
		return fmt.Errorf("conversation object is required")
	}
	return nil
}

func (p *EventParser) validateInputAudioBufferAppendEvent(event *InputAudioBufferAppendEvent) error {
	if event.Audio == "" {
		return fmt.Errorf("audio data is required")
	}
	// Verify Base64 encoding
	if _, err := base64.StdEncoding.DecodeString(event.Audio); err != nil {
		return fmt.Errorf("invalid Base64 audio data: %v", err)
	}
	return nil
}

func (p *EventParser) validateInputAudioBufferCommitEvent(_ *InputAudioBufferCommitEvent) error {
	// No specific validation needed for commit events
	return nil
}

func (p *EventParser) validateInputAudioBufferCommittedEvent(_ *InputAudioBufferCommittedEvent) error {
	// No specific validation needed for committed events
	return nil
}

func (p *EventParser) validateInputAudioBufferClearEvent(_ *InputAudioBufferClearEvent) error {
	// No specific validation needed for clear events
	return nil
}

func (p *EventParser) validateInputAudioBufferSpeechStartedEvent(event *InputAudioBufferSpeechStartedEvent) error {
	if event.AudioStartMs < 0 {
		return fmt.Errorf("audio_start_ms must be non-negative")
	}
	return nil
}

func (p *EventParser) validateInputAudioBufferSpeechStoppedEvent(event *InputAudioBufferSpeechStoppedEvent) error {
	if event.AudioEndMs < 0 {
		return fmt.Errorf("audio_end_ms must be non-negative")
	}
	return nil
}

func (p *EventParser) validateConversationItemCreatedEvent(event *ConversationItemCreatedEvent) error {
	if event.Item.ID == "" {
		return fmt.Errorf("item ID is required")
	}
	if event.Item.Type == "" {
		return fmt.Errorf("item type is required")
	}
	if event.Item.Status == "" {
		return fmt.Errorf("item status is required")
	}
	return nil
}

func (p *EventParser) validateConversationItemInputAudioTranscriptionCompletedEvent(event *ConversationItemInputAudioTranscriptionCompletedEvent) error {
	if event.Item.ID == "" {
		return fmt.Errorf("item ID is required")
	}
	if len(event.Item.Content) == 0 {
		return fmt.Errorf("content is required")
	}
	for _, content := range event.Item.Content {
		if content.Type != "transcript" {
			return fmt.Errorf("invalid content type: %s", content.Type)
		}
	}
	return nil
}

func (p *EventParser) validateConversationItemInputAudioTranscriptionFailedEvent(event *ConversationItemInputAudioTranscriptionFailedEvent) error {
	if event.ItemID == "" {
		return fmt.Errorf("item ID is required")
	}
	if event.Error.Type == "" {
		return fmt.Errorf("error type is required")
	}
	if event.Error.Code == "" {
		return fmt.Errorf("error code is required")
	}
	if event.Error.Message == "" {
		return fmt.Errorf("error message is required")
	}
	return nil
}

func (p *EventParser) validateConversationItemDeletedEvent(event *ConversationItemDeletedEvent) error {
	if event.ItemID == "" {
		return fmt.Errorf("item ID is required")
	}
	return nil
}

func (p *EventParser) validateInputAudioBufferClearedEvent(_ *InputAudioBufferClearedEvent) error {
	// No specific validation needed for cleared events
	return nil
}

func (p *EventParser) validateErrorEvent(event *ErrorEvent) error {
	if event.Error.Type == "" {
		return fmt.Errorf("error type is required")
	}
	if event.Error.Code == "" {
		return fmt.Errorf("error code is required")
	}
	if event.Error.Message == "" {
		return fmt.Errorf("error message is required")
	}
	return nil
}

func (p *EventParser) validateHeartbeatPingEvent(_ *HeartbeatPingEvent) error {
	// Heartbeat events don't require strict validation
	// They can be sent without session ID in some cases
	return nil
}

func (p *EventParser) validateHeartbeatPongEvent(_ *HeartbeatPongEvent) error {
	// Heartbeat events don't require strict validation
	// They can be sent without session ID in some cases
	return nil
}

// GenerateEventID generates a unique event ID
func GenerateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// GenerateSessionID generates a unique session ID
func GenerateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

// GenerateItemID generates a unique conversation item ID
func GenerateItemID() string {
	return fmt.Sprintf("item_%d", time.Now().UnixNano())
}

// GenerateConversationID generates a unique conversation ID
func GenerateConversationID() string {
	return fmt.Sprintf("conv_%d", time.Now().UnixNano())
}

// IsValidEventType checks if an event type is valid
func IsValidEventType(eventType string) bool {
	validTypes := []string{
		EventTypeSessionCreated,
		EventTypeSessionUpdate,
		EventTypeSessionUpdated,
		EventTypeConversationCreated,
		EventTypeInputAudioBufferAppend,
		EventTypeInputAudioBufferCommit,
		EventTypeInputAudioBufferCommitted,
		EventTypeInputAudioBufferClear,
		EventTypeInputAudioBufferSpeechStarted,
		EventTypeInputAudioBufferSpeechStopped,
		EventTypeConversationItemCreated,
		EventTypeConversationItemInputAudioTranscriptionCompleted,
		EventTypeConversationItemInputAudioTranscriptionFailed,
		EventTypeConversationItemDeleted,
		EventTypeInputAudioBufferCleared,
		EventTypeError,
		EventTypeHeartbeatPing,
		EventTypeHeartbeatPong,
	}

	for _, validType := range validTypes {
		if eventType == validType {
			return true
		}
	}
	return false
}

// DecodeBase64Audio decodes Base64 audio data to PCM bytes
func DecodeBase64Audio(base64Audio string) ([]byte, error) {
	base64Audio = strings.TrimPrefix(base64Audio, "data:audio/wav;base64,")
	data, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64 audio: %v", err)
	}
	return data, nil
}

// EncodeAudioToBase64 encodes PCM audio data to Base64
func EncodeAudioToBase64(audioData []byte) string {
	return base64.StdEncoding.EncodeToString(audioData)
}