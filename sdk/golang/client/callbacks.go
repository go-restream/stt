package asr

import "time"

// EventHandler defines the interface for handling OpenAI Realtime API events
type EventHandler interface {
	// Session lifecycle events
	OnSessionCreated(*SessionCreatedEvent)
	OnSessionUpdated(*SessionUpdatedEvent)

	// Conversation events
	OnConversationCreated(*ConversationCreatedEvent)
	OnConversationItemCreated(*ConversationItemCreatedEvent)
	OnConversationItemDeleted(*ConversationItemDeletedEvent)

	// Audio buffer events
	OnAudioBufferAppended(*InputAudioBufferAppendEvent)
	OnAudioBufferCommitted(*InputAudioBufferCommittedEvent)
	OnAudioBufferCleared(*InputAudioBufferClearedEvent)
	OnSpeechStarted(*InputAudioBufferSpeechStartedEvent)
	OnSpeechStopped(*InputAudioBufferSpeechStoppedEvent)

	// Transcription events
	OnTranscriptionCompleted(*ConversationItemInputAudioTranscriptionCompletedEvent)
	OnTranscriptionFailed(*ConversationItemInputAudioTranscriptionFailedEvent)

	// Connection events
	OnConnected()
	OnDisconnected()
	OnError(*ErrorEvent)

	// Heartbeat events
	OnPing(*HeartbeatPingEvent)
	OnPong(*HeartbeatPongEvent)
}

// DefaultEventHandler provides default implementations for all event handlers
type DefaultEventHandler struct{}

func (h *DefaultEventHandler) OnSessionCreated(event *SessionCreatedEvent)                    {}
func (h *DefaultEventHandler) OnSessionUpdated(event *SessionUpdatedEvent)                     {}
func (h *DefaultEventHandler) OnConversationCreated(event *ConversationCreatedEvent)               {}
func (h *DefaultEventHandler) OnConversationItemCreated(event *ConversationItemCreatedEvent)           {}
func (h *DefaultEventHandler) OnConversationItemDeleted(event *ConversationItemDeletedEvent)            {}
func (h *DefaultEventHandler) OnAudioBufferAppended(event *InputAudioBufferAppendEvent)           {}
func (h *DefaultEventHandler) OnAudioBufferCommitted(event *InputAudioBufferCommittedEvent)         {}
func (h *DefaultEventHandler) OnAudioBufferCleared(event *InputAudioBufferClearedEvent)           {}
func (h *DefaultEventHandler) OnSpeechStarted(event *InputAudioBufferSpeechStartedEvent)            {}
func (h *DefaultEventHandler) OnSpeechStopped(event *InputAudioBufferSpeechStoppedEvent)            {}
func (h *DefaultEventHandler) OnTranscriptionCompleted(event *ConversationItemInputAudioTranscriptionCompletedEvent) {}
func (h *DefaultEventHandler) OnTranscriptionFailed(event *ConversationItemInputAudioTranscriptionFailedEvent) {}
func (h *DefaultEventHandler) OnConnected()                                             {}
func (h *DefaultEventHandler) OnDisconnected()                                           {}
func (h *DefaultEventHandler) OnError(event *ErrorEvent)                                      {}
func (h *DefaultEventHandler) OnPing(event *HeartbeatPingEvent)                                 {}
func (h *DefaultEventHandler) OnPong(event *HeartbeatPongEvent)                                 {}

// Config represents configuration for the OpenAI Realtime API client
type Config struct {
	// Connection configuration
	URL                   string        `json:"url"`
	Headers               map[string]string `json:"headers,omitempty"`
	Timeout               time.Duration `json:"timeout,omitempty"`

	// Audio configuration
	InputSampleRate        int           `json:"input_sample_rate,omitempty"`
	OutputSampleRate       int           `json:"output_sample_rate,omitempty"`
	InputChannels          int           `json:"input_channels,omitempty"`
	OutputChannels         int           `json:"output_channels,omitempty"`

	// Session configuration
	Modality              string        `json:"modality,omitempty"`
	Instructions          string        `json:"instructions,omitempty"`
	Voice                 string        `json:"voice,omitempty"`

	// Transcription configuration
	TranscriptionModel     string        `json:"transcription_model,omitempty"`
	TranscriptionLanguage  string        `json:"transcription_language,omitempty"`

	// Turn detection configuration
	TurnDetectionType               string  `json:"turn_detection_type,omitempty"`
	TurnDetectionThreshold          float32 `json:"turn_detection_threshold,omitempty"`
	TurnDetectionPrefixPaddingMs     int     `json:"turn_detection_prefix_padding_ms,omitempty"`
	TurnDetectionSilenceDurationMs   int     `json:"turn_detection_silence_duration_ms,omitempty"`

	// Tools configuration
	Tools                 []interface{} `json:"tools,omitempty"`
	ToolChoice             string        `json:"tool_choice,omitempty"`

	// Reconnection configuration
	EnableReconnect       bool          `json:"enable_reconnect,omitempty"`
	MaxReconnectAttempts  int           `json:"max_reconnect_attempts,omitempty"`
	ReconnectDelay       time.Duration `json:"reconnect_delay,omitempty"`

	// Heartbeat configuration
	HeartbeatInterval     time.Duration `json:"heartbeat_interval,omitempty"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		URL:                    "ws://localhost:8088/ws",
		Timeout:                10 * time.Second,
		InputSampleRate:         16000,
		OutputSampleRate:        16000,
		InputChannels:           1,
		OutputChannels:          1,
		Modality:               "audio",
		TranscriptionModel:      "whisper-1",
		TranscriptionLanguage:   "zh-CN",
		TurnDetectionType:       "server_vad",
		TurnDetectionThreshold:  0.5,
		TurnDetectionPrefixPaddingMs: 300,
		TurnDetectionSilenceDurationMs: 1000,
		EnableReconnect:        true,
		MaxReconnectAttempts:    3,
		ReconnectDelay:         2 * time.Second,
		HeartbeatInterval:      30 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.URL == "" {
		return ErrInvalidURL
	}

	if c.InputSampleRate > 0 && (c.InputSampleRate != 16000 && c.InputSampleRate != 48000) {
		return ErrInvalidSampleRate
	}

	if c.OutputSampleRate > 0 && (c.OutputSampleRate != 16000 && c.OutputSampleRate != 48000) {
		return ErrInvalidSampleRate
	}

	if c.InputChannels > 0 && c.InputChannels != 1 && c.InputChannels != 2 {
		return ErrInvalidChannels
	}

	if c.OutputChannels > 0 && c.OutputChannels != 1 && c.OutputChannels != 2 {
		return ErrInvalidChannels
	}

	if c.Modality != "" && c.Modality != "text" && c.Modality != "audio" && c.Modality != "text_and_audio" {
		return ErrInvalidModality
	}

	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}

	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 30 * time.Second
	}

	return nil
}

// ToSessionConfig converts Config to SessionConfig
func (c *Config) ToSessionConfig() SessionConfig {
	return SessionConfig{
		Modality:                     c.Modality,
		Instructions:                 c.Instructions,
		Voice:                        c.Voice,
		InputSampleRate:              c.InputSampleRate,
		OutputSampleRate:             c.OutputSampleRate,
		InputChannels:                c.InputChannels,
		OutputChannels:               c.OutputChannels,
		TranscriptionModel:            c.TranscriptionModel,
		TranscriptionLanguage:         c.TranscriptionLanguage,
		TurnDetectionType:            c.TurnDetectionType,
		TurnDetectionThreshold:       c.TurnDetectionThreshold,
		TurnDetectionPrefixPaddingMs:   c.TurnDetectionPrefixPaddingMs,
		TurnDetectionSilenceDurationMs: c.TurnDetectionSilenceDurationMs,
		Tools:                        c.Tools,
		ToolChoice:                    c.ToolChoice,
	}
}

// RecognitionCallback provides a simplified callback interface for basic use cases
type RecognitionCallback interface {
	OnRecognitionStart(sessionID string)
	OnRecognitionResult(sessionID, text string)
	OnRecognitionEnd(sessionID string)
	OnRecognitionError(sessionID string, err error)
}

// RecognitionCallbackAdapter adapts EventHandler to RecognitionCallback
type RecognitionCallbackAdapter struct {
	Callback RecognitionCallback
}

func (a *RecognitionCallbackAdapter) OnSessionCreated(event *SessionCreatedEvent) {
	if a.Callback != nil {
		a.Callback.OnRecognitionStart(event.Session.ID)
	}
}

func (a *RecognitionCallbackAdapter) OnSessionUpdated(event *SessionUpdatedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnConversationCreated(event *ConversationCreatedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnConversationItemCreated(event *ConversationItemCreatedEvent) {
	if a.Callback != nil {
		a.Callback.OnRecognitionStart(event.Item.ID)
	}
}

func (a *RecognitionCallbackAdapter) OnConversationItemDeleted(event *ConversationItemDeletedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnAudioBufferAppended(event *InputAudioBufferAppendEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnAudioBufferCommitted(event *InputAudioBufferCommittedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnAudioBufferCleared(event *InputAudioBufferClearedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnSpeechStarted(event *InputAudioBufferSpeechStartedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnSpeechStopped(event *InputAudioBufferSpeechStoppedEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnTranscriptionCompleted(event *ConversationItemInputAudioTranscriptionCompletedEvent) {
	if a.Callback != nil && len(event.Item.Content) > 0 {
		text := event.Item.Content[0].Transcript
		a.Callback.OnRecognitionResult(event.SessionID, text)
	}
}

func (a *RecognitionCallbackAdapter) OnRecognitionEnd(sessionID string) {
	if a.Callback != nil {
		a.Callback.OnRecognitionEnd(sessionID)
	}
}

func (a *RecognitionCallbackAdapter) OnTranscriptionFailed(event *ConversationItemInputAudioTranscriptionFailedEvent) {
	if a.Callback != nil {
		a.Callback.OnRecognitionError(event.SessionID,
			NewASRError(event.Error.Code, event.Error.Message))
	}
}

func (a *RecognitionCallbackAdapter) OnConnected() {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnDisconnected() {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnError(event *ErrorEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnPing(event *HeartbeatPingEvent) {
	// Ignored in simple callback interface
}

func (a *RecognitionCallbackAdapter) OnPong(event *HeartbeatPongEvent) {
	// Ignored in simple callback interface
}