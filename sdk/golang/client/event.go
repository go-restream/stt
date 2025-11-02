package asr

import (
	"encoding/json"
	"log"
)


// sendErrorEvent sends an error event
func (r *Recognizer) sendErrorEvent(message string, err error) {
	// Create error event JSON
	errorEvent := map[string]interface{}{
		"type":    "error",
		"code":    -1,
		"message": message,
	}

	// Include session ID if available
	if r.sessionManager != nil && r.sessionManager.GetSession() != nil {
		errorEvent["session_id"] = r.sessionManager.GetSession().ID
	}

	// Include error details if provided
	if err != nil {
		errorEvent["error_detail"] = err.Error()
	}

	eventData, marshalErr := json.Marshal(errorEvent)
	if marshalErr != nil {
		log.Printf("Failed to marshal error event: %v", marshalErr)
		return
	}

	select {
	case r.eventChan <- eventData:
	case <-r.closeChan:
	}
}