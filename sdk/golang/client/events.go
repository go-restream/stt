package asr

// Event types for OpenAI Realtime API
const (
	EventTypeSessionCreated                          = "session.created"
	EventTypeSessionUpdate                           = "session.update"
	EventTypeSessionUpdated                          = "session.updated"
	EventTypeConversationCreated                     = "conversation.created"
	EventTypeInputAudioBufferAppend                  = "input_audio_buffer.append"
	EventTypeInputAudioBufferCommit                  = "input_audio_buffer.commit"
	EventTypeInputAudioBufferCommitted               = "input_audio_buffer.committed"
	EventTypeInputAudioBufferClear                   = "input_audio_buffer.clear"
	EventTypeInputAudioBufferSpeechStarted           = "input_audio_buffer.speech_started"
	EventTypeInputAudioBufferSpeechStopped           = "input_audio_buffer.speech_stopped"
	EventTypeHeartbeatPing                          = "heartbeat.ping"
	EventTypeHeartbeatPong                          = "heartbeat.pong"
	EventTypeConversationItemCreated                   = "conversation.item.created"
	EventTypeConversationItemInputAudioTranscriptionCompleted = "conversation.item.input_audio_transcription.completed"
	EventTypeConversationItemInputAudioTranscriptionFailed = "conversation.item.input_audio_transcription.failed"
	EventTypeConversationItemDeleted                  = "conversation.item.deleted"
	EventTypeInputAudioBufferCleared                  = "input_audio_buffer.cleared"
	EventTypeError                                  = "error"
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

// Event represents any OpenAI event type
type Event interface {
	GetType() string
	GetEventID() string
	GetSessionID() string
}

// Implementation of Event interface for all event types
func (e *BaseEvent) GetType() string                     { return e.Type }
func (e *BaseEvent) GetEventID() string                  { return e.EventID }
func (e *BaseEvent) GetSessionID() string                { return e.SessionID }

func (e *SessionCreatedEvent) GetType() string               { return e.Type }
func (e *SessionCreatedEvent) GetEventID() string            { return e.EventID }
func (e *SessionCreatedEvent) GetSessionID() string         { return e.SessionID }

func (e *SessionUpdateEvent) GetType() string                { return e.Type }
func (e *SessionUpdateEvent) GetEventID() string             { return e.EventID }
func (e *SessionUpdateEvent) GetSessionID() string          { return e.SessionID }

func (e *SessionUpdatedEvent) GetType() string               { return e.Type }
func (e *SessionUpdatedEvent) GetEventID() string            { return e.EventID }
func (e *SessionUpdatedEvent) GetSessionID() string         { return e.SessionID }

func (e *ConversationCreatedEvent) GetType() string            { return e.Type }
func (e *ConversationCreatedEvent) GetEventID() string         { return e.EventID }
func (e *ConversationCreatedEvent) GetSessionID() string      { return e.SessionID }

func (e *InputAudioBufferAppendEvent) GetType() string       { return e.Type }
func (e *InputAudioBufferAppendEvent) GetEventID() string    { return e.EventID }
func (e *InputAudioBufferAppendEvent) GetSessionID() string  { return e.SessionID }

func (e *InputAudioBufferCommitEvent) GetType() string        { return e.Type }
func (e *InputAudioBufferCommitEvent) GetEventID() string     { return e.EventID }
func (e *InputAudioBufferCommitEvent) GetSessionID() string   { return e.SessionID }

func (e *InputAudioBufferCommittedEvent) GetType() string     { return e.Type }
func (e *InputAudioBufferCommittedEvent) GetEventID() string  { return e.EventID }
func (e *InputAudioBufferCommittedEvent) GetSessionID() string{ return e.SessionID }

func (e *InputAudioBufferClearEvent) GetType() string         { return e.Type }
func (e *InputAudioBufferClearEvent) GetEventID() string      { return e.EventID }
func (e *InputAudioBufferClearEvent) GetSessionID() string    { return e.SessionID }

func (e *InputAudioBufferSpeechStartedEvent) GetType() string   { return e.Type }
func (e *InputAudioBufferSpeechStartedEvent) GetEventID() string{ return e.EventID }
func (e *InputAudioBufferSpeechStartedEvent) GetSessionID() string{ return e.SessionID }

func (e *InputAudioBufferSpeechStoppedEvent) GetType() string   { return e.Type }
func (e *InputAudioBufferSpeechStoppedEvent) GetEventID() string{ return e.EventID }
func (e *InputAudioBufferSpeechStoppedEvent) GetSessionID() string{ return e.SessionID }

func (e *ConversationItemCreatedEvent) GetType() string       { return e.Type }
func (e *ConversationItemCreatedEvent) GetEventID() string    { return e.EventID }
func (e *ConversationItemCreatedEvent) GetSessionID() string  { return e.SessionID }

func (e *ConversationItemInputAudioTranscriptionCompletedEvent) GetType() string { return e.Type }
func (e *ConversationItemInputAudioTranscriptionCompletedEvent) GetEventID() string { return e.EventID }
func (e *ConversationItemInputAudioTranscriptionCompletedEvent) GetSessionID() string { return e.SessionID }

func (e *ConversationItemInputAudioTranscriptionFailedEvent) GetType() string { return e.Type }
func (e *ConversationItemInputAudioTranscriptionFailedEvent) GetEventID() string { return e.EventID }
func (e *ConversationItemInputAudioTranscriptionFailedEvent) GetSessionID() string { return e.SessionID }

func (e *ConversationItemDeletedEvent) GetType() string        { return e.Type }
func (e *ConversationItemDeletedEvent) GetEventID() string     { return e.EventID }
func (e *ConversationItemDeletedEvent) GetSessionID() string   { return e.SessionID }

func (e *InputAudioBufferClearedEvent) GetType() string        { return e.Type }
func (e *InputAudioBufferClearedEvent) GetEventID() string     { return e.EventID }
func (e *InputAudioBufferClearedEvent) GetSessionID() string   { return e.SessionID }

func (e *HeartbeatPingEvent) GetType() string               { return e.Type }
func (e *HeartbeatPingEvent) GetEventID() string            { return e.EventID }
func (e *HeartbeatPingEvent) GetSessionID() string          { return e.SessionID }

func (e *HeartbeatPongEvent) GetType() string               { return e.Type }
func (e *HeartbeatPongEvent) GetEventID() string            { return e.EventID }
func (e *HeartbeatPongEvent) GetSessionID() string          { return e.SessionID }

func (e *ErrorEvent) GetType() string                          { return e.Type }
func (e *ErrorEvent) GetEventID() string                       { return e.EventID }
func (e *ErrorEvent) GetSessionID() string                     { return e.SessionID }