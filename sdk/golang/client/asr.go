package asr

import (
	"fmt"
	"log"
	"time"
)

// CompatibilityWrapper provides backward compatibility with older API
type CompatibilityWrapper struct {
	recognizer  *Recognizer
	config    *Config
}

// NewCompatibilityWrapper creates a new compatibility wrapper
func NewCompatibilityWrapper(config *Config) *CompatibilityWrapper {
	wrapper := &CompatibilityWrapper{
		config: config,
	}

	return wrapper
}

// Start initializes and starts the recognizer
func (w *CompatibilityWrapper) Start() error {
	// Create legacy event handler adapter
	legacyHandler := &LegacyCallbackAdapter{}

	// Create new recognizer with adapter
	newRecognizer, err := NewRecognizerWithEventHandler(w.config, legacyHandler)

	if err != nil {
		return err
	}

	// Wrap new recognizer
	w.recognizer = newRecognizer

	// Start the new recognizer
	if err := w.recognizer.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops and cleans up the recognizer
func (w *CompatibilityWrapper) Stop() error {
	return w.recognizer.Stop()
}

// Write sends audio data to the recognizer
func (w *CompatibilityWrapper) Write(audioData []byte) error {
	// Forward to new recognizer
	return w.recognizer.Write(audioData)
}

// WriteAndCommit writes and commits audio for recognition
func (w *CompatibilityWrapper) WriteAndCommit(audioData []byte) error {
	// Write audio data
	if err := w.recognizer.Write(audioData); err != nil {
		return err
	}

	// Commit audio for recognition
	return w.recognizer.CommitAudio()
}

// IsRunning checks if the recognizer is running
func (w *CompatibilityWrapper) IsRunning() bool {
	return w.recognizer.IsRunning()
}

// GetSessionID returns the session ID
func (w *CompatibilityWrapper) GetSessionID() string {
	return w.recognizer.GetSessionID()
}

// GetStats returns recognition statistics
func (w *CompatibilityWrapper) GetStats() map[string]interface{} {
	return w.recognizer.GetStats()
}

// LegacyRecognitionHandler provides legacy event handling
type LegacyRecognitionHandler struct {}

func (h *LegacyRecognitionHandler) OnRecognitionStart(sessionID string) {
	log.Printf("🔊 CompatibilityWrapper: Recognition started (session: %s)", sessionID)
}

func (h *LegacyRecognitionHandler) OnRecognitionResult(sessionID, text string) {
	log.Printf("🎤 CompatibilityWrapper: Recognition result (session: %s, text: %s)", sessionID, text)
}

func (h *LegacyRecognitionHandler) OnRecognitionEnd(sessionID string) {
	log.Printf("✅ CompatibilityWrapper: Recognition ended (session: %s)", sessionID)
}

func (h *LegacyRecognitionHandler) OnRecognitionError(sessionID string, err error) {
	log.Printf("❌ CompatibilityWrapper: Recognition error (session: %s): %v", sessionID, err)
}

// LegacyEventAdapter provides legacy event adaptation
type LegacyEventAdapter struct {
	Callback *LegacyRecognitionHandler
}

func (a *LegacyEventAdapter) OnSessionCreated(event *SessionCreatedEvent) {
	if a.Callback != nil {
		a.Callback.OnRecognitionStart(event.Session.ID)
	}
}

func (a *LegacyEventAdapter) OnConversationItemCreated(event *ConversationItemCreatedEvent) {
	if a.Callback != nil {
		a.Callback.OnRecognitionResult(event.Item.ID, "conversation.item.created")
	}
}

func (a *LegacyEventAdapter) OnTranscriptionCompleted(event *ConversationItemInputAudioTranscriptionCompletedEvent) {
	if a.Callback != nil && len(event.Item.Content) > 0 {
		for _, content := range event.Item.Content {
			if content.Type == "transcript" {
				a.Callback.OnRecognitionResult(event.SessionID, content.Transcript)
			}
		}
	}
}

func (a *LegacyEventAdapter) OnTranscriptionFailed(event *ConversationItemInputAudioTranscriptionFailedEvent) {
	if a.Callback != nil {
		asrError := &ASRError{
			Code:    event.Error.Code,
			Message: event.Error.Message,
		}

		a.Callback.OnRecognitionError(event.SessionID, asrError)
	}
}

func (a *LegacyEventAdapter) OnError(event *ErrorEvent) {
	if a.Callback != nil {
		asrError := &ASRError{
			Code:    event.Error.Code,
			Message: event.Error.Message,
		}

		a.Callback.OnRecognitionError("global", asrError)
	}
}

// Other methods are empty to avoid compilation errors
func (a *LegacyEventAdapter) OnConnected()           {}
func (a *LegacyEventAdapter) OnDisconnected()         {}
func (a *LegacyEventAdapter) OnPing(event *HeartbeatPingEvent)   {}
func (a *LegacyEventAdapter) OnPong(event *HeartbeatPongEvent)   {}
func (a *LegacyEventAdapter) OnSessionUpdated(event *SessionUpdatedEvent)    {}
func (a *LegacyEventAdapter) OnAudioBufferAppended(event *InputAudioBufferAppendEvent) {}
func (a *LegacyEventAdapter) OnAudioBufferCommitted(event *InputAudioBufferCommittedEvent) {}
func (a *LegacyEventAdapter) OnAudioBufferCleared(event *InputAudioBufferClearedEvent) {}
func (a *LegacyEventAdapter) OnSpeechStarted(event *InputAudioBufferSpeechStartedEvent) {}
func (a *LegacyEventAdapter) OnSpeechStopped(event *InputAudioBufferSpeechStoppedEvent) {}

// Helper function
func (a *LegacyEventAdapter) GenerateSessionID() string {
	return fmt.Sprintf("legacy_session_%d", time.Now().UnixNano())
}

// LegacyRecognitionCallback provides simple legacy callback interface
type LegacyRecognitionCallback interface {
	OnRecognitionStart(sessionID string)
	OnRecognitionResult(sessionID, text string)
	OnRecognitionEnd(sessionID string)
	OnRecognitionError(sessionID string, err error)
}

// LegacyCallbackAdapter implements the adapter pattern
type LegacyCallbackAdapter struct {
	callback LegacyRecognitionCallback
}

// LegacyCallbackAdapter implements EventHandler interface
func (a *LegacyCallbackAdapter) OnSessionCreated(event *SessionCreatedEvent) {
	a.callback.OnRecognitionStart(event.Session.ID)
}

func (a *LegacyCallbackAdapter) OnSessionUpdated(event *SessionUpdatedEvent) {}

func (a *LegacyCallbackAdapter) OnConversationCreated(event *ConversationCreatedEvent) {}

func (a *LegacyCallbackAdapter) OnConversationItemCreated(event *ConversationItemCreatedEvent) {}

func (a *LegacyCallbackAdapter) OnConversationItemDeleted(event *ConversationItemDeletedEvent) {}

func (a *LegacyCallbackAdapter) OnAudioBufferAppended(event *InputAudioBufferAppendEvent) {}

func (a *LegacyCallbackAdapter) OnAudioBufferCommitted(event *InputAudioBufferCommittedEvent) {}

func (a *LegacyCallbackAdapter) OnAudioBufferCleared(event *InputAudioBufferClearedEvent) {}

func (a *LegacyCallbackAdapter) OnSpeechStarted(event *InputAudioBufferSpeechStartedEvent) {}

func (a *LegacyCallbackAdapter) OnSpeechStopped(event *InputAudioBufferSpeechStoppedEvent) {}

func (a *LegacyCallbackAdapter) OnTranscriptionCompleted(event *ConversationItemInputAudioTranscriptionCompletedEvent) {
	if len(event.Item.Content) > 0 {
		for _, content := range event.Item.Content {
			if content.Type == "transcript" {
				a.callback.OnRecognitionResult(event.SessionID, content.Transcript)
				break
			}
		}
	}
}

func (a *LegacyCallbackAdapter) OnTranscriptionFailed(event *ConversationItemInputAudioTranscriptionFailedEvent) {
	a.callback.OnRecognitionError(event.SessionID, fmt.Errorf("transcription failed: %s", event.Error.Message))
}

func (a *LegacyCallbackAdapter) OnError(event *ErrorEvent) {
	a.callback.OnRecognitionError("global", fmt.Errorf("server error: %s", event.Error.Message))
}

func (a *LegacyCallbackAdapter) OnConnected() {
	a.callback.OnRecognitionStart("connected")
}

func (a *LegacyCallbackAdapter) OnDisconnected() {
	a.callback.OnRecognitionEnd("disconnected")
}

func (a *LegacyCallbackAdapter) OnPing(event *HeartbeatPingEvent) {}

func (a *LegacyCallbackAdapter) OnPong(event *HeartbeatPongEvent) {}

// MigrationHelper assists with configuration migration
type MigrationHelper struct{}

func NewMigrationHelper() *MigrationHelper {
	return &MigrationHelper{}
}

func (m *MigrationHelper) MigrateFromOldConfig(oldConfig map[string]interface{}) *Config {
	config := DefaultConfig()

	// Convert configuration parameters
	if url, ok := oldConfig["url"].(string); ok {
		config.URL = url
	}
	if lang, ok := oldConfig["language"].(string); ok {
		config.TranscriptionLanguage = lang
	}
	if sampleRate, ok := oldConfig["sample_rate"].(int); ok {
		config.InputSampleRate = sampleRate
	}

	// Set additional configuration
	config.Timeout = 10 * time.Second
	config.EnableReconnect = true
	config.MaxReconnectAttempts = 3

	return config
}

func (m *MigrationHelper) GetMigrationGuide() string {
	return "# ASR SDK Migration Guide\n\n" +
		"This version has been upgraded to OpenAI Realtime API standard with full migration support.\n" +
		"Please refer to sdk/golang/docs/MIGRATION.md for detailed documentation"
}