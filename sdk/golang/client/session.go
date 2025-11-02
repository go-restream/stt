package asr

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Session represents an OpenAI Realtime API session
type Session struct {
	ID                           string
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
	Status                        string
	Modality                      string
	Instructions                   string
	Voice                         string
	InputAudioFormat               AudioFormat
	OutputAudioFormat              AudioFormat
	InputAudioTranscription        *TranscriptionConfig
	TurnDetection                 *TurnDetectionConfig
	Tools                         []interface{}
	ToolChoice                    string
	IsInitialized                 bool
}

// AudioFormat represents audio format configuration
type AudioFormat struct {
	Type       string `json:"type"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

// TranscriptionConfig represents audio transcription configuration
type TranscriptionConfig struct {
	Model    string `json:"model"`
	Language string `json:"language"`
}

// TurnDetectionConfig represents turn detection configuration
type TurnDetectionConfig struct {
	Type              string  `json:"type"`
	Threshold         float32 `json:"threshold"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms"`
	SilenceDurationMs int     `json:"silence_duration_ms"`
}

// SessionStatus represents the lifecycle status of a session
type SessionStatus string

const (
	SessionStatusCreated    SessionStatus = "created"
	SessionStatusUpdated    SessionStatus = "updated"
	SessionStatusActive     SessionStatus = "active"
	SessionStatusInactive  SessionStatus = "inactive"
	SessionStatusFailed    SessionStatus = "failed"
)

// SessionManager manages session lifecycle and state
type SessionManager struct {
	session      *Session
	sessionMutex sync.RWMutex
	eventHandler  EventHandler
}

// NewSessionManager creates a new session manager
func NewSessionManager(handler EventHandler) *SessionManager {
	return &SessionManager{
		session:     nil,
		eventHandler: handler,
	}
}

// CreateSession initializes a new session with default configuration
func (sm *SessionManager) CreateSession() *Session {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	now := time.Now()
	session := &Session{
		ID:        generateSessionID(),
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     string(SessionStatusCreated),
		Modality:   "audio",
		InputAudioFormat: AudioFormat{
			Type:       "pcm16",
			SampleRate: 16000,
			Channels:   1,
		},
		OutputAudioFormat: AudioFormat{
			Type:       "pcm16",
			SampleRate: 16000,
			Channels:   1,
		},
		IsInitialized: false,
	}

	sm.session = session

	log.Printf("[🆔 Session] Created new session: %s", session.ID)
	return session
}

// UpdateSession applies configuration updates to the current session
func (sm *SessionManager) UpdateSession(config SessionConfig) error {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session == nil {
		return fmt.Errorf("no active session")
	}

	// Apply configuration changes
	if config.Modality != "" {
		sm.session.Modality = config.Modality
	}
	if config.Instructions != "" {
		sm.session.Instructions = config.Instructions
	}
	if config.Voice != "" {
		sm.session.Voice = config.Voice
	}
	if config.InputSampleRate > 0 {
		sm.session.InputAudioFormat.SampleRate = config.InputSampleRate
	}
	if config.OutputSampleRate > 0 {
		sm.session.OutputAudioFormat.SampleRate = config.OutputSampleRate
	}
	if config.InputChannels > 0 {
		sm.session.InputAudioFormat.Channels = config.InputChannels
	}
	if config.OutputChannels > 0 {
		sm.session.OutputAudioFormat.Channels = config.OutputChannels
	}

	if config.TranscriptionModel != "" {
		if sm.session.InputAudioTranscription == nil {
			sm.session.InputAudioTranscription = &TranscriptionConfig{}
		}
		sm.session.InputAudioTranscription.Model = config.TranscriptionModel
	}
	if config.TranscriptionLanguage != "" {
		if sm.session.InputAudioTranscription == nil {
			sm.session.InputAudioTranscription = &TranscriptionConfig{}
		}
		sm.session.InputAudioTranscription.Language = config.TranscriptionLanguage
	}

	if config.TurnDetectionType != "" {
		if sm.session.TurnDetection == nil {
			sm.session.TurnDetection = &TurnDetectionConfig{}
		}
		sm.session.TurnDetection.Type = config.TurnDetectionType
	}
	if config.TurnDetectionThreshold > 0 {
		if sm.session.TurnDetection == nil {
			sm.session.TurnDetection = &TurnDetectionConfig{}
		}
		sm.session.TurnDetection.Threshold = config.TurnDetectionThreshold
	}
	if config.TurnDetectionPrefixPaddingMs > 0 {
		if sm.session.TurnDetection == nil {
			sm.session.TurnDetection = &TurnDetectionConfig{}
		}
		sm.session.TurnDetection.PrefixPaddingMs = config.TurnDetectionPrefixPaddingMs
	}
	if config.TurnDetectionSilenceDurationMs > 0 {
		if sm.session.TurnDetection == nil {
			sm.session.TurnDetection = &TurnDetectionConfig{}
		}
		sm.session.TurnDetection.SilenceDurationMs = config.TurnDetectionSilenceDurationMs
	}

	if len(config.Tools) > 0 {
		sm.session.Tools = config.Tools
	}
	if config.ToolChoice != "" {
		sm.session.ToolChoice = config.ToolChoice
	}

	sm.session.UpdatedAt = time.Now()
	sm.session.Status = string(SessionStatusUpdated)

	log.Printf("[⚙️ Session] Updated session %s with new configuration", sm.session.ID)
	return nil
}

// GetSession returns the current session
func (sm *SessionManager) GetSession() *Session {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()
	return sm.session
}

// SetSessionStatus updates the session status
func (sm *SessionManager) SetSessionStatus(status SessionStatus) {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session == nil {
		return
	}

	sm.session.Status = string(status)
	sm.session.UpdatedAt = time.Now()

	log.Printf("[📊 Session] Session %s status changed to: %s", sm.session.ID, status)
}

// MarkSessionInitialized marks the session as initialized
func (sm *SessionManager) MarkSessionInitialized() {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session == nil {
		return
	}

	sm.session.IsInitialized = true
	sm.session.UpdatedAt = time.Now()

	log.Printf("[✅ Session] Session %s marked as initialized", sm.session.ID)
}

// IsSessionInitialized returns true if session is initialized
func (sm *SessionManager) IsSessionInitialized() bool {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	if sm.session == nil {
		return false
	}

	return sm.session.IsInitialized
}

// HandleSessionCreated processes session.created event
func (sm *SessionManager) HandleSessionCreated(event *SessionCreatedEvent) {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session == nil {
		log.Printf("[⚠️ Session] Received session.created but no local session exists")
		return
	}

	// Update session info from server event
	sm.session.Status = string(SessionStatusActive)
	sm.session.UpdatedAt = time.Now()

	log.Printf("[✅ Session] Session %s created and activated", event.Session.ID)

	// Notify event handler
	if sm.eventHandler != nil {
		sm.eventHandler.OnSessionCreated(event)
	}
}

// HandleSessionUpdated processes session.updated event
func (sm *SessionManager) HandleSessionUpdated(event *SessionUpdatedEvent) {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session == nil {
		log.Printf("[⚠️ Session] Received session.updated but no local session exists")
		return
	}

	sm.session.Status = string(SessionStatusActive)
	sm.session.UpdatedAt = time.Now()

	log.Printf("[🔄 Session] Session %s updated", event.Session.ID)

	// Notify event handler
	if sm.eventHandler != nil {
		sm.eventHandler.OnSessionUpdated(event)
	}
}

// GetSessionInfo returns session information for debugging
func (sm *SessionManager) GetSessionInfo() map[string]interface{} {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	if sm.session == nil {
		return map[string]interface{}{
			"hasSession": false,
		}
	}

	info := map[string]interface{}{
		"hasSession":    true,
		"sessionID":      sm.session.ID,
		"status":         sm.session.Status,
		"modality":       sm.session.Modality,
		"isInitialized":  sm.session.IsInitialized,
		"createdAt":       sm.session.CreatedAt,
		"updatedAt":       sm.session.UpdatedAt,
		"inputFormat":     sm.session.InputAudioFormat,
		"outputFormat":    sm.session.OutputAudioFormat,
	}

	if sm.session.InputAudioTranscription != nil {
		info["transcription"] = sm.session.InputAudioTranscription
	}

	if sm.session.TurnDetection != nil {
		info["turnDetection"] = sm.session.TurnDetection
	}

	return info
}

// Cleanup performs cleanup of session resources
func (sm *SessionManager) Cleanup() {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	if sm.session != nil {
		log.Printf("[🧹 Session] Cleaning up session: %s", sm.session.ID)
		sm.session.Status = string(SessionStatusInactive)
	}

	sm.session = nil
	log.Printf("[✅ Session] Session manager cleanup completed")
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

// SessionConfig represents configuration options for session updates
type SessionConfig struct {
	// Audio format configuration
	Modality               string
	Instructions           string
	Voice                  string
	InputSampleRate        int
	OutputSampleRate       int
	InputChannels          int
	OutputChannels         int

	// Transcription configuration
	TranscriptionModel     string
	TranscriptionLanguage  string

	// Turn detection configuration
	TurnDetectionType               string
	TurnDetectionThreshold          float32
	TurnDetectionPrefixPaddingMs     int
	TurnDetectionSilenceDurationMs   int

	// Tools and configuration
	Tools       []interface{}
	ToolChoice  string
}