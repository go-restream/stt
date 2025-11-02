package asr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Recognizer represents the main client for OpenAI Realtime API
type Recognizer struct {
	// Configuration
	config *Config

	// Core components
	connManager    *ConnectionManager
	sessionManager *SessionManager
	eventDispatcher *EventDispatcher
	audioUtils     *AudioUtils
	audioBuffer    *AudioBuffer
	eventStats     *EventStats

	// State management
	ctx            context.Context
	cancel         context.CancelFunc
	isRunning      bool
	runningMutex   sync.RWMutex

	// Event handling
	eventChan      chan []byte
	errorChan      chan error
	closeChan      chan struct{}
	wg             sync.WaitGroup
}

// NewRecognizer creates a new recognizer instance
func NewRecognizer(config *Config) *Recognizer {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("[❌ Config] Invalid configuration: %v", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	connManager := NewConnectionManager(config.URL)
	sessionManager := NewSessionManager(nil) // Will be set later
	eventDispatcher := NewEventDispatcher(NewEventParser())
	audioUtils := NewAudioUtils(config.InputSampleRate, config.InputChannels)
	audioBuffer := NewAudioBuffer(1024*1000, config.InputSampleRate, config.InputChannels) // 1MB buffer
	eventStats := NewEventStats()

	// Apply connection settings
	for key, value := range config.Headers {
		connManager.SetHeader(key, value)
	}
	connManager.SetPingInterval(config.HeartbeatInterval)
	connManager.SetReconnectOptions(config.EnableReconnect, config.MaxReconnectAttempts, config.ReconnectDelay)

	return &Recognizer{
		config:         config,
		connManager:    connManager,
		sessionManager: sessionManager,
		eventDispatcher: eventDispatcher,
		audioUtils:     audioUtils,
		audioBuffer:    audioBuffer,
		eventStats:     eventStats,
		ctx:            ctx,
		cancel:         cancel,
		isRunning:      false,
		eventChan:      make(chan []byte, 1000),
		errorChan:      make(chan error, 100),
		closeChan:      make(chan struct{}),
	}
}

// NewRecognizerWithCallbacks creates a recognizer with event handlers
func NewRecognizerWithCallbacks(config *Config, handler EventHandler) *Recognizer {
	recognizer := NewRecognizer(config)
	recognizer.sessionManager = NewSessionManager(handler)
	recognizer.eventDispatcher.RegisterEventHandler(handler)
	return recognizer
}

// NewRecognizerWithEventHandler creates a recognizer with event handler (alias for NewRecognizerWithCallbacks)
func NewRecognizerWithEventHandler(config *Config, handler EventHandler) (*Recognizer, error) {
	recognizer := NewRecognizerWithCallbacks(config, handler)
	return recognizer, nil
}

// NewRecognizerWithLegacyCallbacks creates a recognizer with legacy recognition callbacks
func NewRecognizerWithLegacyCallbacks(config *Config, callback RecognitionCallback) *Recognizer {
	recognizer := NewRecognizer(config)
	adapter := &RecognitionCallbackAdapter{Callback: callback}
	recognizer.sessionManager = NewSessionManager(adapter)
	recognizer.eventDispatcher.RegisterEventHandler(adapter)
	return recognizer
}

// Start establishes connection and begins recognition session
func (r *Recognizer) Start() error {
	r.runningMutex.Lock()
	defer r.runningMutex.Unlock()

	if r.isRunning {
		return ErrRecognizerRunning
	}

	log.Printf("[🚀 Recognizer] Starting recognition session")

	// Connect to WebSocket
	if err := r.connManager.Connect(); err != nil {
		r.sendError(fmt.Errorf("connection failed: %w", err))
		return err
	}

	// Create session
	session := r.sessionManager.CreateSession()

	// Configure session if needed
	if r.config.Modality != "" || r.config.InputSampleRate > 0 {
		sessionConfig := r.config.ToSessionConfig()
		if err := r.sessionManager.UpdateSession(sessionConfig); err != nil {
			log.Printf("[⚠️ Recognizer] Failed to configure session: %v", err)
			// Continue anyway, session will use defaults
		}
	}

	// Send session.update event to configure server
	if err := r.sendSessionUpdate(session); err != nil {
		r.sendError(fmt.Errorf("session configuration failed: %w", err))
		return err
	}

	// Mark as running
	r.isRunning = true
	log.Printf("[✅ Recognizer] Recognition session started (Session ID: %s)", session.ID)

	// Start background goroutines
	r.wg.Add(3)
	go r.messageReceiver()
	go r.eventProcessor()
	go r.connectionMonitor()

	// Send heartbeat ping
	r.wg.Add(1)
	go r.heartbeatLoop()

	return nil
}

// Stop stops the recognition session and cleans up resources
func (r *Recognizer) Stop() error {
	r.runningMutex.Lock()
	defer r.runningMutex.Unlock()

	if !r.isRunning {
		return ErrRecognizerNotRunning
	}

	log.Printf("[🛑 Recognizer] Stopping recognition session")

	// Cancel context to stop all goroutines
	r.cancel()

	// Mark as not running
	r.isRunning = false

	// Disconnect connection
	if err := r.connManager.Disconnect(); err != nil {
		log.Printf("[⚠️ Recognizer] Error during disconnection: %v", err)
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[✅ Recognizer] All goroutines stopped cleanly")
	case <-time.After(10 * time.Second):
		log.Printf("[⚠️ Recognizer] Timeout waiting for goroutines to stop")
	}

	// Cleanup resources
	r.sessionManager.Cleanup()
	r.audioBuffer.Clear()
	r.eventDispatcher.ClearHandlers()

	log.Printf("[✅ Recognizer] Recognition session stopped")
	return nil
}

// Write sends audio data to the server
func (r *Recognizer) Write(audioData []byte) error {
	r.runningMutex.RLock()
	defer r.runningMutex.RUnlock()

	if !r.isRunning {
		return ErrRecognizerNotRunning
	}

	// Validate audio format
	if err := r.audioUtils.ValidateAudioFormat(r.config.InputSampleRate, r.config.InputChannels); err != nil {
		return fmt.Errorf("invalid audio format: %w", err)
	}

	// Add to buffer
	if err := r.audioBuffer.Write(audioData); err != nil {
		return fmt.Errorf("audio buffer full: %w", err)
	}

	// Convert to int16 PCM if needed
	pcmSamples, err := r.convertToPCM16(audioData)
	if err != nil {
		return fmt.Errorf("audio conversion failed: %w", err)
	}

	// Create and send input_audio_buffer.append event
	event := &InputAudioBufferAppendEvent{
		BaseEvent: BaseEvent{
			Type:    EventTypeInputAudioBufferAppend,
			EventID: generateEventID(),
		},
		Audio: PCM16ToBase64(pcmSamples),
	}

	return r.sendEvent(event)
}

// CommitAudio commits the current audio buffer for processing
func (r *Recognizer) CommitAudio() error {
	r.runningMutex.RLock()
	defer r.runningMutex.RUnlock()

	if !r.isRunning {
		return ErrRecognizerNotRunning
	}

	log.Printf("[📤 Recognizer] Committing audio buffer")

	// Send input_audio_buffer.commit event
	event := &InputAudioBufferCommitEvent{
		BaseEvent: BaseEvent{
			Type:    EventTypeInputAudioBufferCommit,
			EventID: generateEventID(),
		},
	}

	return r.sendEvent(event)
}

// ClearAudioBuffer clears the audio buffer
func (r *Recognizer) ClearAudioBuffer() error {
	r.runningMutex.RLock()
	defer r.runningMutex.RUnlock()

	if !r.isRunning {
		return ErrRecognizerNotRunning
	}

	log.Printf("[🧹 Recognizer] Clearing audio buffer")

	// Clear local buffer
	r.audioBuffer.Clear()

	// Send input_audio_buffer.clear event
	event := &InputAudioBufferClearEvent{
		BaseEvent: BaseEvent{
			Type:    EventTypeInputAudioBufferClear,
			EventID: generateEventID(),
		},
	}

	return r.sendEvent(event)
}

// IsRunning returns the current running status
func (r *Recognizer) IsRunning() bool {
	r.runningMutex.RLock()
	defer r.runningMutex.RUnlock()
	return r.isRunning
}

// GetSessionID returns the current session ID
func (r *Recognizer) GetSessionID() string {
	session := r.sessionManager.GetSession()
	if session == nil {
		return ""
	}
	return session.ID
}

// GetConnectionStatus returns the current connection status
func (r *Recognizer) GetConnectionStatus() ConnectionStatus {
	return r.connManager.GetStatus()
}

// GetStats returns current statistics
func (r *Recognizer) GetStats() map[string]interface{} {
	session := r.sessionManager.GetSession()
	sessionInfo := r.sessionManager.GetSessionInfo()
	dispatcherStats := r.eventDispatcher.GetStats()
	eventStats := r.eventStats.GetStats()
	audioBufferSize := r.audioBuffer.Size()
	audioBufferDuration := r.audioBuffer.GetDuration()

	stats := map[string]interface{}{
		"is_running":            r.IsRunning(),
		"connection_status":      r.GetConnectionStatus(),
		"session_info":          sessionInfo,
		"dispatcher_stats":       dispatcherStats,
		"event_stats":          eventStats,
		"audio_buffer_size":     audioBufferSize,
		"audio_buffer_duration": audioBufferDuration,
		"config":               r.config,
	}

	if session != nil {
		stats["session_id"] = session.ID
		stats["session_status"] = session.Status
		stats["session_created_at"] = session.CreatedAt
		stats["session_updated_at"] = session.UpdatedAt
	}

	return stats
}

// sendSessionUpdate sends a session.update event to configure the server
func (r *Recognizer) sendSessionUpdate(session *Session) error {
	// Create session.update event
	event := &SessionUpdateEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeSessionUpdate,
			EventID:   generateEventID(),
			SessionID: session.ID,
		},
		Session: struct {
			ID        string `json:"id"`
			Modality  string `json:"modality"`
			Instructions string `json:"instructions,omitempty"`
			Voice     string `json:"voice,omitempty"`
			InputAudioFormat struct {
				Type       string `json:"type"`
				SampleRate int    `json:"sample_rate"`
				Channels   int    `json:"channels"`
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
		}{
			ID:       session.ID,
			Modality: session.Modality,
			InputAudioFormat: session.InputAudioFormat,
			OutputAudioFormat: struct {
				Type       string `json:"type"`
				SampleRate int    `json:"sample_rate"`
				Voice      string `json:"voice,omitempty"`
			}{
				Type:       session.OutputAudioFormat.Type,
				SampleRate: session.OutputAudioFormat.SampleRate,
			},
		},
	}

	// Add optional fields if they exist
	if session.Instructions != "" {
		event.Session.Instructions = session.Instructions
	}
	if session.InputAudioTranscription != nil {
		event.Session.InputAudioTranscription = &struct {
			Model    string `json:"model"`
			Language string `json:"language"`
		}{
			Model:    session.InputAudioTranscription.Model,
			Language: session.InputAudioTranscription.Language,
		}
	}
	if session.TurnDetection != nil {
		event.Session.TurnDetection = &struct {
			Type              string  `json:"type"`
			Threshold         float32 `json:"threshold"`
			PrefixPaddingMs   int     `json:"prefix_padding_ms"`
			SilenceDurationMs int     `json:"silence_duration_ms"`
		}{
			Type:              session.TurnDetection.Type,
			Threshold:         session.TurnDetection.Threshold,
			PrefixPaddingMs:   session.TurnDetection.PrefixPaddingMs,
			SilenceDurationMs: session.TurnDetection.SilenceDurationMs,
		}
	}
	if len(session.Tools) > 0 {
		event.Session.Tools = session.Tools
	}
	if session.ToolChoice != "" {
		event.Session.ToolChoice = session.ToolChoice
	}

	return r.sendEvent(event)
}

// sendEvent serializes and sends an event to the server
func (r *Recognizer) sendEvent(event Event) error {
	// Set session ID if available
	session := r.sessionManager.GetSession()
	if session != nil {
		switch e := event.(type) {
		case *BaseEvent:
			e.SessionID = session.ID
		case *InputAudioBufferAppendEvent:
			e.SessionID = session.ID
		case *InputAudioBufferCommitEvent:
			e.SessionID = session.ID
		case *InputAudioBufferClearEvent:
			e.SessionID = session.ID
		}
	}

	// Serialize event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("event serialization failed: %w", err)
	}

	// Send via connection manager
	return r.connManager.SendMessage(data)
}

// convertToPCM16 converts audio data to 16-bit PCM samples
func (r *Recognizer) convertToPCM16(audioData []byte) ([]int16, error) {
	// This is a simplified conversion - in production, you might want to handle
	// different audio formats and sample rates properly
	if len(audioData)%2 != 0 {
		return nil, fmt.Errorf("invalid PCM data length")
	}

	samples := make([]int16, len(audioData)/2)
	for i := 0; i < len(samples); i++ {
		// Simple 16-bit little-endian conversion
		samples[i] = int16(audioData[i*2]) | int16(audioData[i*2+1])<<8
	}

	return samples, nil
}

// messageReceiver receives messages from the WebSocket connection
func (r *Recognizer) messageReceiver() {
	defer r.wg.Done()

	log.Printf("[📡 Receiver] Starting message receiver")

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("[📡 Receiver] Message receiver stopped")
			return
		default:
			messageType, message, err := r.connManager.ReadMessage()
			if err != nil {
				r.sendError(fmt.Errorf("receive error: %w", err))
				return
			}

			if messageType == websocket.TextMessage {
				select {
				case r.eventChan <- message:
					r.eventStats.RecordEvent("message_received", false, "")
				default:
					log.Printf("[⚠️ Receiver] Event channel full, dropping message")
					r.eventStats.RecordEvent("message_dropped", true, "event channel full")
				}
			}
		}
	}
}

// eventProcessor processes incoming events
func (r *Recognizer) eventProcessor() {
	defer r.wg.Done()

	log.Printf("[⚙️ Processor] Starting event processor")

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("[⚙️ Processor] Event processor stopped")
			return
		case message := <-r.eventChan:
			if err := r.eventDispatcher.Dispatch(message); err != nil {
				r.sendError(fmt.Errorf("event processing error: %w", err))
				r.eventStats.RecordEvent("event_processing_error", true, err.Error())
			} else {
				r.eventStats.RecordEvent("event_processed", false, "")
			}
		}
	}
}

// connectionMonitor monitors connection status
func (r *Recognizer) connectionMonitor() {
	defer r.wg.Done()

	log.Printf("[📊 Monitor] Starting connection monitor")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("[📊 Monitor] Connection monitor stopped")
			return
		case <-ticker.C:
			status := r.connManager.GetStatus()
			if status == ConnectionStatusDisconnected || status == ConnectionStatusFailed {
				r.sendError(fmt.Errorf("connection lost"))
				return
			}
		}
	}
}

// heartbeatLoop sends periodic heartbeat pings
func (r *Recognizer) heartbeatLoop() {
	defer r.wg.Done()

	log.Printf("[💓 Heartbeat] Starting heartbeat loop")

	ticker := time.NewTicker(r.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("[💓 Heartbeat] Heartbeat loop stopped")
			return
		case <-ticker.C:
			if r.connManager.IsConnected() {
				event := &HeartbeatPingEvent{
					BaseEvent: BaseEvent{
						Type:    EventTypeHeartbeatPing,
						EventID: generateEventID(),
					},
					HeartbeatType: 1,
				}

				if err := r.sendEvent(event); err != nil {
					log.Printf("[⚠️ Heartbeat] Failed to send ping: %v", err)
					r.eventStats.RecordEvent("heartbeat_error", true, err.Error())
				} else {
					r.eventStats.RecordEvent("heartbeat_sent", false, "")
				}
			}
		}
	}
}

// sendError sends an error to the error channel
func (r *Recognizer) sendError(err error) {
	select {
	case r.errorChan <- err:
	default:
		log.Printf("[⚠️ Recognizer] Error channel full, dropping error: %v", err)
	}
}

// Errors returns a channel for receiving errors
func (r *Recognizer) Errors() <-chan error {
	return r.errorChan
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%s", uuid.New().String())
}