package asr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// EventParser handles parsing and validation of OpenAI Realtime API events
type EventParser struct{}

// NewEventParser creates a new event parser
func NewEventParser() *EventParser {
	return &EventParser{}
}

// ParseEvent parses a JSON message into appropriate event type
func (p *EventParser) ParseEvent(data []byte) (Event, error) {
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

	case EventTypeSessionUpdated:
		var event SessionUpdatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse session.updated event: %v", err)
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
func (p *EventParser) ValidateEvent(event Event) error {
	switch e := event.(type) {
	case *SessionCreatedEvent:
		return p.validateSessionCreatedEvent(e)
	case *SessionUpdateEvent:
		return p.validateSessionUpdateEvent(e)
	case *SessionUpdatedEvent:
		return p.validateSessionUpdatedEvent(e)
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
	case *HeartbeatPingEvent:
		return p.validateHeartbeatPingEvent(e)
	case *HeartbeatPongEvent:
		return p.validateHeartbeatPongEvent(e)
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

func (p *EventParser) validateSessionUpdatedEvent(event *SessionUpdatedEvent) error {
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
	return nil
}

func (p *EventParser) validateHeartbeatPongEvent(_ *HeartbeatPongEvent) error {
	// Heartbeat events don't require strict validation
	return nil
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

// DecodeBase64Audio decodes Base64 audio data to bytes
func DecodeBase64Audio(base64Audio string) ([]byte, error) {
	base64Audio = strings.TrimPrefix(base64Audio, "data:audio/wav;base64,")
	data, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64 audio: %v", err)
	}
	return data, nil
}

// EncodeAudioToBase64 encodes audio data to Base64
func EncodeAudioToBase64(audioData []byte) string {
	return base64.StdEncoding.EncodeToString(audioData)
}