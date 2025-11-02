package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-restream/stt/pkg/logger"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Session represents an OpenAI Realtime API session
type Session struct {
	ID        string    `json:"id"`
	Conn      *websocket.Conn `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
	Modality  string    `json:"modality"` // "text", "audio", "text_and_audio"

	// Session configuration
	Instructions string `json:"instructions,omitempty"`
	Voice        string `json:"voice,omitempty"`

	// Audio format configuration
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

	// Audio transcription configuration
	InputAudioTranscription struct {
		Model    string `json:"model"`
		Language string `json:"language"`
	} `json:"input_audio_transcription,omitempty"`

	// Turn detection configuration
	TurnDetection struct {
		Type              string  `json:"type"`
		Threshold         float32 `json:"threshold"`
		PrefixPaddingMs   int     `json:"prefix_padding_ms"`
		SilenceDurationMs int     `json:"silence_duration_ms"`
	} `json:"turn_detection,omitempty"`

	// Tools and tool choice
	Tools      []interface{} `json:"tools,omitempty"`
	ToolChoice string        `json:"tool_choice,omitempty"`

	// Conversation state
	ConversationItems []*ConversationItem `json:"conversation_items,omitempty"`

	// Audio buffer state
	AudioBuffer      []int16 `json:"-"`
	AudioBufferMutex sync.RWMutex `json:"-"`

	// VAD-processed audio buffer for ASR (contains only speech segments)
	VADAudioBuffer      []int16 `json:"-"`
	VADAudioBufferMutex sync.RWMutex `json:"-"`

	// Audio file saving state
	AccumulatedAudio   []int16     `json:"-"`           // Accumulated audio data for file saving
	AccumulationStartTime time.Time `json:"-"`         // Current accumulation cycle start time
	LastSaveTime      time.Time   `json:"-"`           // Last save time
	AudioSaveMutex    sync.RWMutex `json:"-"`          // Audio save operation mutex

	// Session mutex for thread-safe operations
	mutex sync.RWMutex `json:"-"`

	// VAD state
	IsSpeaking      bool `json:"-"`
	SpeechStartTime time.Time `json:"-"`

	// Recognition state
	CurrentItemID string `json:"current_item_id,omitempty"`

	// Heartbeat tracking
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// ConversationItem represents a conversation item in the session
type ConversationItem struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"` // "message", "function_call", "function_response"
	Status    string        `json:"status"` // "in_progress", "completed", "failed"
	Role      string        `json:"role,omitempty"` // "user", "assistant"
	Content   []interface{} `json:"content,omitempty"`
	Audio     *AudioContent `json:"audio,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// AudioContent represents audio content in a conversation item
type AudioContent struct {
	Data   string `json:"data"` // Base64 encoded audio
	Format string `json:"format"`
}

// SessionManager manages all active sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex

	// Configuration
	SessionTimeout time.Duration
	MaxSessions    int
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionTimeout time.Duration, maxSessions int) *SessionManager {
	return &SessionManager{
		sessions:       make(map[string]*Session),
		SessionTimeout: sessionTimeout,
		MaxSessions:    maxSessions,
	}
}

// CreateSession creates a new session for a WebSocket connection
func (sm *SessionManager) CreateSession(conn *websocket.Conn, modality string) (*Session, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if len(sm.sessions) >= sm.MaxSessions {
		return nil, fmt.Errorf("maximum number of sessions reached")
	}

	sessionID := GenerateSessionID()
	session := &Session{
		ID:        sessionID,
		Conn:      conn,
		CreatedAt: time.Now(),
		LastActive: time.Now(),
		Modality:  modality,
		AudioBuffer: make([]int16, 0),
		LastHeartbeat: time.Now(),
	}

	session.InputAudioFormat.Type = "pcm16"
	session.InputAudioFormat.SampleRate = 0
	session.InputAudioFormat.Channels = 1

	sm.sessions[sessionID] = session

	logger.WithFields(logrus.Fields{
		"component": "mg_session_ctrl",
		"action":    "session_created",
		"sessionID": sessionID,
		"modality":  modality,
	}).Info("Created new session")
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// SessionExists checks if a session exists
func (sm *SessionManager) SessionExists(sessionID string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	_, exists := sm.sessions[sessionID]
	return exists
}

// UpdateSession updates session activity and configuration
func (sm *SessionManager) UpdateSession(sessionID string, updateFunc func(*Session)) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	updateFunc(session)
	session.LastActive = time.Now()

	return nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.AudioBuffer = nil
		delete(sm.sessions, sessionID)
		logger.WithFields(logrus.Fields{
			"component": "mg_session_ctrl",
			"action":    "session_deleted",
			"sessionID": sessionID,
		}).Info("Deleted session")
	}
}

// RemoveSession removes a specific session
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return
	}

	if session.Conn != nil {
		session.Conn.Close()
		session.Conn = nil
	}

	session.AudioBuffer = nil
	session.VADAudioBuffer = nil
	delete(sm.sessions, sessionID)

	logger.WithFields(logrus.Fields{
		"component": "mg_session_ctrl",
		"action":    "session_removed",
		"sessionID": sessionID,
	}).Info("Removed session")
}

// CleanupInactiveSessions removes sessions that have timed out
func (sm *SessionManager) CleanupInactiveSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActive) > sm.SessionTimeout {
			if session.Conn != nil {
				session.Conn.Close()
			}

			session.AudioBuffer = nil
			delete(sm.sessions, sessionID)

			logger.WithFields(logrus.Fields{
				"component": "mg_session_ctrl",
				"action":    "session_cleanup",
				"sessionID": sessionID,
				"inactiveDuration": now.Sub(session.LastActive),
			}).Info("Cleaned up inactive session")
		}
	}
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return len(sm.sessions)
}

// SendEventToSession sends an event to a specific session
func (sm *SessionManager) SendEventToSession(sessionID string, event interface{}) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return sm.SendEvent(session, event)
}

// SendEvent sends an event to a session
func (sm *SessionManager) SendEvent(session *Session, event interface{}) error {
	if session.Conn == nil {
		return fmt.Errorf("session connection is nil")
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Conn == nil {
		return fmt.Errorf("session connection closed")
	}

	if err := session.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %v", err)
	}

	return session.Conn.WriteMessage(websocket.TextMessage, jsonData)
}

// AddAudioToBuffer adds audio data to the session's audio buffer
func (sm *SessionManager) AddAudioToBuffer(sessionID string, audioData []int16) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.AudioBufferMutex.Lock()
	defer session.AudioBufferMutex.Unlock()

	session.AudioBuffer = append(session.AudioBuffer, audioData...)
	session.LastActive = time.Now()

	return nil
}

// GetAudioBuffer retrieves the current audio buffer
func (sm *SessionManager) GetAudioBuffer(sessionID string) ([]int16, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.AudioBufferMutex.RLock()
	defer session.AudioBufferMutex.RUnlock()

	buffer := make([]int16, len(session.AudioBuffer))
	copy(buffer, session.AudioBuffer)

	return buffer, nil
}

// GetAudioBufferSize returns the current size of the audio buffer
func (sm *SessionManager) GetAudioBufferSize(sessionID string) (int, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}

	session.AudioBufferMutex.RLock()
	defer session.AudioBufferMutex.RUnlock()

	return len(session.AudioBuffer), nil
}

// ClearAudioBuffer clears the audio buffer
func (sm *SessionManager) ClearAudioBuffer(sessionID string) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.AudioBufferMutex.Lock()
	defer session.AudioBufferMutex.Unlock()

	session.AudioBuffer = make([]int16, 0)
	session.LastActive = time.Now()

	return nil
}

// AddVADAudioToBuffer adds VAD-processed audio data to the session's VAD audio buffer
func (sm *SessionManager) AddVADAudioToBuffer(sessionID string, audioData []int16) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.VADAudioBufferMutex.Lock()
	defer session.VADAudioBufferMutex.Unlock()

	session.VADAudioBuffer = append(session.VADAudioBuffer, audioData...)
	session.LastActive = time.Now()

	return nil
}

// GetVADAudioBuffer retrieves the current VAD audio buffer
func (sm *SessionManager) GetVADAudioBuffer(sessionID string) ([]int16, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.VADAudioBufferMutex.RLock()
	defer session.VADAudioBufferMutex.RUnlock()

	buffer := make([]int16, len(session.VADAudioBuffer))
	copy(buffer, session.VADAudioBuffer)

	return buffer, nil
}

// GetVADAudioBufferSize returns the current size of the VAD audio buffer
func (sm *SessionManager) GetVADAudioBufferSize(sessionID string) (int, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}

	session.VADAudioBufferMutex.RLock()
	defer session.VADAudioBufferMutex.RUnlock()

	return len(session.VADAudioBuffer), nil
}

// ClearVADAudioBuffer clears the VAD audio buffer
func (sm *SessionManager) ClearVADAudioBuffer(sessionID string) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.VADAudioBufferMutex.Lock()
	defer session.VADAudioBufferMutex.Unlock()

	session.VADAudioBuffer = make([]int16, 0)
	session.LastActive = time.Now()

	return nil
}

// CreateConversationItem creates a new conversation item in the session
func (sm *SessionManager) CreateConversationItem(sessionID string, itemType string, role string) (*ConversationItem, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	itemID := GenerateItemID()
	item := &ConversationItem{
		ID:        itemID,
		Type:      itemType,
		Status:    "in_progress",
		Role:      role,
		Content:   make([]interface{}, 0),
		CreatedAt: time.Now(),
	}

	session.ConversationItems = append(session.ConversationItems, item)
	session.CurrentItemID = itemID
	session.LastActive = time.Now()

	return item, nil
}

// UpdateConversationItem updates a conversation item
func (sm *SessionManager) UpdateConversationItem(sessionID string, itemID string, updateFunc func(*ConversationItem)) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	for _, item := range session.ConversationItems {
		if item.ID == itemID {
			updateFunc(item)
			session.LastActive = time.Now()
			return nil
		}
	}

	return fmt.Errorf("conversation item not found: %s", itemID)
}

// GetConversationItem retrieves a conversation item
func (sm *SessionManager) GetConversationItem(sessionID string, itemID string) (*ConversationItem, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	for _, item := range session.ConversationItems {
		if item.ID == itemID {
			return item, nil
		}
	}

	return nil, fmt.Errorf("conversation item not found: %s", itemID)
}

// MarkConversationItemCompleted marks a conversation item as completed
func (sm *SessionManager) MarkConversationItemCompleted(sessionID string, itemID string) error {
	return sm.UpdateConversationItem(sessionID, itemID, func(item *ConversationItem) {
		item.Status = "completed"
		now := time.Now()
		item.CompletedAt = &now
	})
}

// MarkConversationItemFailed marks a conversation item as failed
func (sm *SessionManager) MarkConversationItemFailed(sessionID string, itemID string, errorMsg string) error {
	return sm.UpdateConversationItem(sessionID, itemID, func(item *ConversationItem) {
		item.Status = "failed"
		now := time.Now()
		item.CompletedAt = &now
		errorContent := map[string]interface{}{
			"type": "error",
			"text": errorMsg,
		}
		item.Content = append(item.Content, errorContent)
	})
}

// UpdateHeartbeat updates the session's last heartbeat time
func (sm *SessionManager) UpdateHeartbeat(sessionID string) error {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastHeartbeat = time.Now()
	session.LastActive = time.Now()

	return nil
}

// GetSessionStats returns statistics about active sessions
func (sm *SessionManager) GetSessionStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_sessions"] = len(sm.sessions)

	modalityCount := make(map[string]int)
	for _, session := range sm.sessions {
		modalityCount[session.Modality]++
	}
	stats["sessions_by_modality"] = modalityCount

	return stats
}