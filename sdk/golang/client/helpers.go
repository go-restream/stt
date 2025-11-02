package asr

import (
	"fmt"
	"time"
)

// Helper provides utility methods for common operations
type Helper struct {
	recognizer *Recognizer
}

// NewHelper creates a new helper instance
func NewHelper(recognizer *Recognizer) *Helper {
	return &Helper{
		recognizer: recognizer,
	}
}

// SimpleConfig provides a simple configuration interface
type SimpleConfig struct {
	URL         string
	Language    string
	SampleRate  int
	Channels    int
}

// NewSimpleConfig creates a simple configuration
func NewSimpleConfig(url, language string) *SimpleConfig {
	return &SimpleConfig{
		URL:        url,
		Language:   language,
		SampleRate: 16000,
		Channels:   1,
	}
}

// ToConfig converts SimpleConfig to Config
func (sc *SimpleConfig) ToConfig() *Config {
	config := DefaultConfig()
	config.URL = sc.URL
	config.TranscriptionLanguage = sc.Language
	config.InputSampleRate = sc.SampleRate
	config.InputChannels = sc.Channels
	return config
}

// CreateRecognizer creates a recognizer with simple configuration
func CreateRecognizer(url, language string) (*Recognizer, error) {
	simpleConfig := NewSimpleConfig(url, language)
	config := simpleConfig.ToConfig()
	return NewRecognizer(config), nil
}

// CreateRecognizerWithCallbacks creates a recognizer with simple configuration and callbacks
func CreateRecognizerWithCallbacks(url, language string, callback RecognitionCallback) (*Recognizer, error) {
	simpleConfig := NewSimpleConfig(url, language)
	config := simpleConfig.ToConfig()
	return NewRecognizerWithLegacyCallbacks(config, callback), nil
}

// CreateRecognizerWithEventHandler creates a recognizer with simple configuration and event handler
func CreateRecognizerWithEventHandler(url, language string, handler EventHandler) (*Recognizer, error) {
	simpleConfig := NewSimpleConfig(url, language)
	config := simpleConfig.ToConfig()
	return NewRecognizerWithCallbacks(config, handler), nil
}

// QuickStart provides a quick way to start recognition with minimal setup
func QuickStart(url, language string, callback RecognitionCallback) (*Recognizer, error) {
	recognizer, err := CreateRecognizerWithCallbacks(url, language, callback)
	if err != nil {
		return nil, err
	}

	if err := recognizer.Start(); err != nil {
		return nil, err
	}

	return recognizer, nil
}

// QuickStartWithEvents provides a quick way to start recognition with event handling
func QuickStartWithEvents(url, language string, handler EventHandler) (*Recognizer, error) {
	recognizer, err := CreateRecognizerWithEventHandler(url, language, handler)
	if err != nil {
		return nil, err
	}

	if err := recognizer.Start(); err != nil {
		return nil, err
	}

	return recognizer, nil
}

// ProcessAudioFile processes a single audio file and returns transcription
func (h *Helper) ProcessAudioFile(audioData []byte, timeout time.Duration) (string, error) {
	if !h.recognizer.IsRunning() {
		return "", ErrRecognizerNotRunning
	}

	// Channel to receive result
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Create callback to capture result
	simpleCallback := &SimpleRecognitionCallback{
		ResultChan: resultChan,
		ErrorChan:  errorChan,
	}

	// Replace the legacy handler
	h.recognizer.eventDispatcher.RegisterLegacyHandler(simpleCallback)

	// Send audio data
	if err := h.recognizer.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to send audio: %w", err)
	}

	// Commit the audio
	if err := h.recognizer.CommitAudio(); err != nil {
		return "", fmt.Errorf("failed to commit audio: %w", err)
	}

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("transcription timeout after %v", timeout)
	}
}

// StreamAudio continuously processes audio data from a channel
func (h *Helper) StreamAudio(audioChan <-chan []byte, errorChan chan<- error) {
	for audioData := range audioChan {
		if !h.recognizer.IsRunning() {
			errorChan <- ErrRecognizerNotRunning
			return
		}

		if err := h.recognizer.Write(audioData); err != nil {
			errorChan <- fmt.Errorf("failed to send audio chunk: %w", err)
			continue
		}
	}
}

// BatchProcess processes multiple audio files in sequence
func (h *Helper) BatchProcess(audioFiles [][]byte, timeout time.Duration) ([]string, error) {
	if !h.recognizer.IsRunning() {
		return nil, ErrRecognizerNotRunning
	}

	results := make([]string, 0, len(audioFiles))
	for i, audioData := range audioFiles {
		result, err := h.ProcessAudioFile(audioData, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to process audio file %d: %w", i+1, err)
		}
		results[i] = result
	}

	return results, nil
}

// SessionInfo provides detailed session information
type SessionInfo struct {
	SessionID     string    `json:"session_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsInitialized bool      `json:"is_initialized"`
}

// GetSessionInfo returns current session information
func (h *Helper) GetSessionInfo() *SessionInfo {
	session := h.recognizer.sessionManager.GetSession()
	if session == nil {
		return nil
	}

	return &SessionInfo{
		SessionID:     session.ID,
		Status:        session.Status,
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
		IsInitialized: session.IsInitialized,
	}
}

// WaitUntilReady waits until the recognizer is ready to receive audio
func (h *Helper) WaitUntilReady(timeout time.Duration) error {
	session := h.recognizer.sessionManager.GetSession()
	if session == nil {
		return ErrSessionNotFound
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if session.IsInitialized {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("session not ready within timeout %v", timeout)
}

// SimpleRecognitionCallback is a simple callback implementation
type SimpleRecognitionCallback struct {
	ResultChan chan<- string
	ErrorChan  chan<- error
}

func (s *SimpleRecognitionCallback) OnRecognitionStart(sessionID string) {
	// Ignored in simple callback
}

func (s *SimpleRecognitionCallback) OnRecognitionResult(sessionID, text string) {
	select {
	case s.ResultChan <- text:
	default:
		// Non-blocking send
	}
}

func (s *SimpleRecognitionCallback) OnRecognitionEnd(sessionID string) {
	// Ignored in simple callback
}

func (s *SimpleRecognitionCallback) OnRecognitionError(sessionID string, err error) {
	select {
	case s.ErrorChan <- err:
	default:
		// Non-blocking send
	}
}

// StreamingCallback handles streaming recognition results
type StreamingCallback struct {
	PartialChan chan<- string
	FinalChan   chan<- string
	ErrorChan    chan<- error
}

func NewStreamingCallback(partialChan, finalChan chan<- string, errorChan chan<- error) *StreamingCallback {
	return &StreamingCallback{
		PartialChan: partialChan,
		FinalChan:   finalChan,
		ErrorChan:    errorChan,
	}
}

func (s *StreamingCallback) OnRecognitionStart(sessionID string) {
	// Send notification about session start
}

func (s *StreamingCallback) OnRecognitionResult(sessionID, text string) {
	select {
	case s.PartialChan <- text:
	default:
		// Non-blocking send
	}
}

func (s *StreamingCallback) OnRecognitionEnd(sessionID string) {
	// Could send empty final text to indicate end
	select {
	case s.FinalChan <- "":
	default:
		// Non-blocking send
	}
}

func (s *StreamingCallback) OnRecognitionError(sessionID string, err error) {
	select {
	case s.ErrorChan <- err:
	default:
		// Non-blocking send
	}
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidateURL validates a WebSocket URL
func ValidateURL(url string) *ValidationError {
	if url == "" {
		return &ValidationError{Field: "url", Message: "URL is required"}
	}

	// Basic URL validation - could be extended
	if len(url) < 10 || url[:5] != "ws://" && url[:4] != "wss://" {
		return &ValidationError{Field: "url", Message: "Invalid WebSocket URL format", Value: url}
	}

	return nil
}

// ValidateAudioConfig validates audio configuration parameters
func ValidateAudioConfig(sampleRate, channels int) []*ValidationError {
	var errors []*ValidationError

	if sampleRate <= 0 {
		errors = append(errors, &ValidationError{Field: "sample_rate", Message: "must be positive", Value: sampleRate})
	}

	if sampleRate != 16000 && sampleRate != 48000 {
		errors = append(errors, &ValidationError{Field: "sample_rate", Message: "must be 16000 or 48000", Value: sampleRate})
	}

	if channels <= 0 {
		errors = append(errors, &ValidationError{Field: "channels", Message: "must be positive", Value: channels})
	}

	if channels != 1 && channels != 2 {
		errors = append(errors, &ValidationError{Field: "channels", Message: "must be 1 or 2", Value: channels})
	}

	return errors
}

// DebugInfo provides detailed debugging information
type DebugInfo struct {
	RecognizerStatus map[string]interface{} `json:"recognizer_status"`
	ConnectionInfo  map[string]interface{} `json:"connection_info"`
	SessionInfo    map[string]interface{} `json:"session_info"`
	EventStats     map[string]interface{} `json:"event_stats"`
	AudioInfo      map[string]interface{} `json:"audio_info"`
}

// GetDebugInfo returns comprehensive debugging information
func (h *Helper) GetDebugInfo() *DebugInfo {
	return &DebugInfo{
		RecognizerStatus: map[string]interface{}{
			"is_running": h.recognizer.IsRunning(),
			"config":     h.recognizer.config,
		},
		ConnectionInfo: map[string]interface{}{
			"status": h.recognizer.GetConnectionStatus(),
		},
		SessionInfo: h.recognizer.sessionManager.GetSessionInfo(),
		EventStats:  h.recognizer.eventDispatcher.GetStats(),
		AudioInfo: map[string]interface{}{
			"buffer_size":     h.recognizer.audioBuffer.Size(),
			"buffer_duration": h.recognizer.audioBuffer.GetDuration(),
			"sample_rate":      h.recognizer.config.InputSampleRate,
			"channels":        h.recognizer.config.InputChannels,
		},
	}
}

// Cleanup performs full cleanup of recognizer and helper
func (h *Helper) Cleanup() error {
	if h.recognizer == nil {
		return nil
	}

	if err := h.recognizer.Stop(); err != nil {
		return fmt.Errorf("failed to stop recognizer: %w", err)
	}

	return nil
}