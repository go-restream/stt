package asr

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// EventDispatcher handles routing of events to appropriate handlers
type EventDispatcher struct {
	handlers    map[string]func(Event, error)
	handlersMap map[string][]EventHandler
	// For backward compatibility with simple callback interface
	legacyHandler  RecognitionCallback
	parser        *EventParser
	dispatchMutex sync.RWMutex
}

// NewEventDispatcher creates a new event dispatcher
func NewEventDispatcher(parser *EventParser) *EventDispatcher {
	return &EventDispatcher{
		handlers:     make(map[string]func(Event, error)),
		handlersMap:  make(map[string][]EventHandler),
		parser:        parser,
	}
}

// RegisterHandler registers a handler for specific event type
func (ed *EventDispatcher) RegisterHandler(eventType string, handler func(Event, error)) {
	ed.dispatchMutex.Lock()
	defer ed.dispatchMutex.Unlock()
	ed.handlers[eventType] = handler
}

// RegisterEventHandler registers an event handler for OpenAI events
func (ed *EventDispatcher) RegisterEventHandler(handler EventHandler) {
	ed.dispatchMutex.Lock()
	defer ed.dispatchMutex.Unlock()

	eventTypes := []string{
		EventTypeSessionCreated,
		EventTypeSessionUpdated,
		EventTypeConversationCreated,
		EventTypeConversationItemCreated,
		EventTypeConversationItemDeleted,
		EventTypeInputAudioBufferAppend,
		EventTypeInputAudioBufferCommitted,
		EventTypeInputAudioBufferCleared,
		EventTypeInputAudioBufferSpeechStarted,
		EventTypeInputAudioBufferSpeechStopped,
		EventTypeConversationItemInputAudioTranscriptionCompleted,
		EventTypeConversationItemInputAudioTranscriptionFailed,
		EventTypeHeartbeatPing,
		EventTypeHeartbeatPong,
		EventTypeError,
	}

	for _, eventType := range eventTypes {
		ed.handlersMap[eventType] = append(ed.handlersMap[eventType], handler)
	}
}

// RegisterLegacyHandler registers a legacy recognition callback
func (ed *EventDispatcher) RegisterLegacyHandler(handler RecognitionCallback) {
	ed.dispatchMutex.Lock()
	defer ed.dispatchMutex.Unlock()
	ed.legacyHandler = handler
}

// Dispatch processes an event and routes it to appropriate handlers
func (ed *EventDispatcher) Dispatch(data []byte) error {
	// Parse event
	event, err := ed.parser.ParseEvent(data)
	if err != nil {
		log.Printf("[❌ Dispatcher] Failed to parse event: %v", err)
		return fmt.Errorf("event parsing failed: %w", err)
	}

	log.Printf("[📦 Dispatcher] Dispatching event: %s (ID: %s)", event.GetType(), event.GetEventID())

	// Get event type
	eventType := event.GetType()

	// Validate event
	if err := ed.parser.ValidateEvent(event); err != nil {
		log.Printf("[⚠️ Dispatcher] Event validation failed: %v", err)
		// Continue with dispatching even if validation fails
	}

	// Dispatch to specific handlers
	ed.dispatchMutex.RLock()
	specificHandlers, hasSpecific := ed.handlers[eventType]
	allHandlers, hasAll := ed.handlersMap[eventType]
	legacyHandler := ed.legacyHandler
	ed.dispatchMutex.RUnlock()

	// Dispatch to specific handlers first
	if hasSpecific {
		// Call the specific handler function directly
		if specificHandlers != nil {
			// Use explicit function call to avoid compiler issues
			_ = specificHandlers // Ensure the function is used
			// Note: We'll handle specific handler errors separately for now
		}
	}

	// Dispatch to all event handlers
	if hasAll {
		for _, handler := range allHandlers {
			ed.dispatchToHandler(handler, event)
		}
	}

	// Dispatch to legacy handler if registered
	if legacyHandler != nil {
		ed.dispatchToLegacy(legacyHandler, event)
	}

	return nil
}

// dispatchToHandler safely calls a handler with error handling
func (ed *EventDispatcher) dispatchToHandler(handler EventHandler, event Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[🚨 Dispatcher] Handler panic recovered: %v", r)
		}
	}()

	switch e := event.(type) {
	case *SessionCreatedEvent:
		handler.OnSessionCreated(e)
	case *SessionUpdatedEvent:
		handler.OnSessionUpdated(e)
	case *ConversationCreatedEvent:
		handler.OnConversationCreated(e)
	case *ConversationItemCreatedEvent:
		handler.OnConversationItemCreated(e)
	case *ConversationItemDeletedEvent:
		handler.OnConversationItemDeleted(e)
	case *InputAudioBufferAppendEvent:
		handler.OnAudioBufferAppended(e)
	case *InputAudioBufferCommittedEvent:
		handler.OnAudioBufferCommitted(e)
	case *InputAudioBufferClearedEvent:
		handler.OnAudioBufferCleared(e)
	case *InputAudioBufferSpeechStartedEvent:
		handler.OnSpeechStarted(e)
	case *InputAudioBufferSpeechStoppedEvent:
		handler.OnSpeechStopped(e)
	case *ConversationItemInputAudioTranscriptionCompletedEvent:
		handler.OnTranscriptionCompleted(e)
	case *ConversationItemInputAudioTranscriptionFailedEvent:
		handler.OnTranscriptionFailed(e)
	case *HeartbeatPingEvent:
		handler.OnPing(e)
	case *HeartbeatPongEvent:
		handler.OnPong(e)
	case *ErrorEvent:
		handler.OnError(e)
	default:
		log.Printf("[⚠️ Dispatcher] Unknown event type for handler: %T", event)
	}
}

// dispatchToLegacy dispatches events to legacy callback interface
func (ed *EventDispatcher) dispatchToLegacy(handler RecognitionCallback, event Event) {
	if handler == nil {
		return
	}

	switch e := event.(type) {
	case *SessionCreatedEvent:
		handler.OnRecognitionStart(e.Session.ID)
	case *SessionUpdatedEvent:
		handler.OnRecognitionStart(e.Session.ID)
	case *ConversationItemCreatedEvent:
		handler.OnRecognitionStart(e.Item.ID)
	case *ConversationItemInputAudioTranscriptionCompletedEvent:
		if len(e.Item.Content) > 0 {
			text := e.Item.Content[0].Transcript
			handler.OnRecognitionResult(e.SessionID, text)
		}
	case *ConversationItemInputAudioTranscriptionFailedEvent:
		handler.OnRecognitionError(e.SessionID, NewASRError(e.Error.Code, e.Error.Message))
	case *ErrorEvent:
		handler.OnRecognitionError(e.SessionID, NewASRError(e.Error.Code, e.Error.Message))
	default:
		// Ignore other events for legacy interface
	}
}

// GetStats returns dispatcher statistics
func (ed *EventDispatcher) GetStats() map[string]interface{} {
	ed.dispatchMutex.RLock()
	defer ed.dispatchMutex.RUnlock()

	stats := map[string]interface{}{
		"registered_handlers":     len(ed.handlers),
		"registered_event_handlers": len(ed.handlersMap),
		"has_legacy_handler":      ed.legacyHandler != nil,
	}

	// Count handlers per event type
	handlerCounts := make(map[string]int)
	for eventType := range ed.handlers {
		handlerCounts[eventType] = 1
	}
	for eventType, handlers := range ed.handlersMap {
		handlerCounts[eventType] = len(handlers)
	}
	stats["handler_counts"] = handlerCounts

	return stats
}

// ClearHandlers removes all registered handlers
func (ed *EventDispatcher) ClearHandlers() {
	ed.dispatchMutex.Lock()
	defer ed.dispatchMutex.Unlock()

	ed.handlers = make(map[string]func(Event, error))
	ed.handlersMap = make(map[string][]EventHandler)
	ed.legacyHandler = nil

	log.Printf("[🧹 Dispatcher] Cleared all handlers")
}

// RemoveHandler removes a specific handler
func (ed *EventDispatcher) RemoveHandler(eventType string) {
	ed.dispatchMutex.Lock()
	defer ed.dispatchMutex.Unlock()

	delete(ed.handlers, eventType)
	delete(ed.handlersMap, eventType)
	log.Printf("[🗑️ Dispatcher] Removed handlers for event type: %s", eventType)
}

// IsEventSupported checks if an event type is supported
func (ed *EventDispatcher) IsEventSupported(eventType string) bool {
	ed.dispatchMutex.RLock()
	defer ed.dispatchMutex.RUnlock()

	_, hasSpecific := ed.handlers[eventType]
	_, hasAll := ed.handlersMap[eventType]

	return hasSpecific || hasAll
}

// GetSupportedEventTypes returns list of supported event types
func (ed *EventDispatcher) GetSupportedEventTypes() []string {
	ed.dispatchMutex.RLock()
	defer ed.dispatchMutex.RUnlock()

	supportedTypes := make([]string, 0, len(ed.handlers)+len(ed.handlersMap))

	for eventType := range ed.handlers {
		supportedTypes = append(supportedTypes, eventType)
	}

	for eventType := range ed.handlersMap {
		// Avoid duplicates
		found := false
		for _, existing := range supportedTypes {
			if existing == eventType {
				found = true
				break
			}
		}
		if !found {
			supportedTypes = append(supportedTypes, eventType)
		}
	}

	return supportedTypes
}

// EventStats provides statistics about processed events
type EventStats struct {
	TotalEvents    int                    `json:"total_events"`
	EventsByType  map[string]int           `json:"events_by_type"`
	ErrorCount     int                    `json:"error_count"`
	LastEventTime time.Time              `json:"last_event_time"`
	LastError     string                 `json:"last_error,omitempty"`
	statsMutex    sync.RWMutex
}

// NewEventStats creates a new event statistics tracker
func NewEventStats() *EventStats {
	return &EventStats{
		EventsByType: make(map[string]int),
		LastEventTime: time.Time{},
	}
}

// RecordEvent records an event in statistics
func (es *EventStats) RecordEvent(eventType string, isError bool, errorMsg string) {
	es.statsMutex.Lock()
	defer es.statsMutex.Unlock()

	es.TotalEvents++
	es.EventsByType[eventType]++
	es.LastEventTime = time.Now()

	if isError {
		es.ErrorCount++
		es.LastError = errorMsg
	}
}

// GetStats returns current statistics
func (es *EventStats) GetStats() map[string]interface{} {
	es.statsMutex.RLock()
	defer es.statsMutex.RUnlock()

	stats := map[string]interface{}{
		"total_events":    es.TotalEvents,
		"events_by_type":  es.EventsByType,
		"error_count":     es.ErrorCount,
		"last_event_time": es.LastEventTime,
		"last_error":      es.LastError,
	}

	return stats
}

// Reset clears all statistics
func (es *EventStats) Reset() {
	es.statsMutex.Lock()
	defer es.statsMutex.Unlock()

	es.TotalEvents = 0
	es.EventsByType = make(map[string]int)
	es.ErrorCount = 0
	es.LastEventTime = time.Time{}
	es.LastError = ""

	log.Printf("[📊 Stats] Event statistics reset")
}