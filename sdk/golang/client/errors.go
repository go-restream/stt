package asr

import (
	"errors"
	"fmt"
)

var (
	// Legacy errors (kept for compatibility during transition)
	// ErrRecognizerRunning recognizer is already running
	ErrRecognizerRunning = errors.New("recognizer is already running")
	// ErrRecognizerNotRunning recognizer is not running
	ErrRecognizerNotRunning = errors.New("recognizer is not running")
	// ErrInvalidAudioFormat invalid audio format
	ErrInvalidAudioFormat = errors.New("invalid audio format")
	// ErrConnectionFailed connection failed
	ErrConnectionFailed = errors.New("connection failed")
	// ErrRecognitionFailed recognition failed
	ErrRecognitionFailed = errors.New("recognition failed")
	// ErrInvalidParameter invalid parameter
	ErrInvalidParameter = errors.New("invalid parameter")

	// New OpenAI Realtime API errors
	// Connection errors
	ErrConnectionTimeout     = errors.New("connection timeout")
	ErrNotConnected        = errors.New("not connected")
	ErrAlreadyConnected     = errors.New("already connected")

	// Session errors
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionNotReady     = errors.New("session not ready")
	ErrInvalidSessionState = errors.New("invalid session state")

	// Audio errors
	ErrInvalidSampleRate    = errors.New("invalid sample rate")
	ErrInvalidChannels      = errors.New("invalid audio channels")
	ErrAudioBufferFull    = errors.New("audio buffer full")
	ErrAudioEncodingFailed = errors.New("audio encoding failed")
	ErrAudioDecodingFailed = errors.New("audio decoding failed")

	// Event errors
	ErrInvalidEventType     = errors.New("invalid event type")
	ErrEventValidationFailed = errors.New("event validation failed")
	ErrEventParsingFailed   = errors.New("event parsing failed")

	// Configuration errors
	ErrInvalidURL          = errors.New("invalid URL")
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrInvalidModality     = errors.New("invalid modality")

	// Protocol errors
	ErrProtocolError       = errors.New("protocol error")
	ErrProtocolVersion     = errors.New("protocol version mismatch")
	ErrUnexpectedMessage    = errors.New("unexpected message")

	// State errors
	ErrInvalidState        = errors.New("invalid state")
)

// RecognitionError represents recognition error structure
type RecognitionError struct {
	Code    int
	Message string
	Err     error
}

func (e *RecognitionError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *RecognitionError) Unwrap() error {
	return e.Err
}

// ASRError represents a detailed error with error code and message
type ASRError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ASRError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewASRError creates a new ASR error
func NewASRError(code, message string, details ...string) *ASRError {
	err := &ASRError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// WrapError wraps an error with ASR error context
func WrapError(code, message string, err error) *ASRError {
	return &ASRError{
		Code:    code,
		Message: message,
		Details: err.Error(),
	}
}

// IsConnectionError checks if error is connection related
func IsConnectionError(err error) bool {
	return err == ErrConnectionFailed ||
		err == ErrConnectionTimeout ||
		err == ErrNotConnected ||
		err == ErrAlreadyConnected
}

// IsSessionError checks if error is session related
func IsSessionError(err error) bool {
	return err == ErrSessionNotFound ||
		err == ErrSessionNotReady ||
		err == ErrInvalidSessionState
}

// IsAudioError checks if error is audio related
func IsAudioError(err error) bool {
	return err == ErrInvalidAudioFormat ||
		err == ErrInvalidSampleRate ||
		err == ErrInvalidChannels ||
		err == ErrAudioBufferFull ||
		err == ErrAudioEncodingFailed ||
		err == ErrAudioDecodingFailed
}

// IsEventError checks if error is event related
func IsEventError(err error) bool {
	return err == ErrInvalidEventType ||
		err == ErrEventValidationFailed ||
		err == ErrEventParsingFailed
}

// IsConfigError checks if error is configuration related
func IsConfigError(err error) bool {
	return err == ErrInvalidURL ||
		err == ErrInvalidParameter ||
		err == ErrInvalidConfig ||
		err == ErrInvalidModality
}

// IsProtocolError checks if error is protocol related
func IsProtocolError(err error) bool {
	return err == ErrProtocolError ||
		err == ErrProtocolVersion ||
		err == ErrUnexpectedMessage
}

// IsStateError checks if error is state related
func IsStateError(err error) bool {
	return err == ErrRecognizerNotRunning ||
		err == ErrRecognizerRunning ||
		err == ErrInvalidState
}