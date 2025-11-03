package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	config "github.com/go-restream/stt/config"
	llm "github.com/go-restream/stt/llm"
	"github.com/go-restream/stt/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type OpenAIService struct {
	upgrader       websocket.Upgrader
	eventParser    *EventParser
	audioUtils     *AudioUtils
	sessionManager *SessionManager
	vadIntegration *VADIntegration
	config         *OpenAIConfig
	appConfig      *config.Config
	cancel         context.CancelFunc
}

type OpenAIConfig struct {
	SessionTimeout time.Duration
	MaxSessions    int
	HeartbeatInterval time.Duration
}

func DefaultOpenAIConfig() *OpenAIConfig {
	return &OpenAIConfig{
		SessionTimeout:    30 * time.Minute,
		MaxSessions:       100,
		HeartbeatInterval: 30 * time.Second,
	}
}

func NewOpenAIService(openAIConfig *OpenAIConfig, configPath string) *OpenAIService {
	if openAIConfig == nil {
		openAIConfig = DefaultOpenAIConfig()
	}

	// Load configuration first before initializing session manager
	appConfig, err := config.LoadConfig(configPath)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "load_config_failed",
			"error":     err,
		}).Error("Failed to load config")
		appConfig = &config.Config{} // Use empty config as fallback
	}

	// Initialize session manager first
	sessionManager := NewSessionManager(openAIConfig.SessionTimeout, openAIConfig.MaxSessions, appConfig)

	// Set ASR configuration from config file to ensure config file takes precedence
	llm.SetAsrBaseURL(appConfig.ASR.BaseURL)
	llm.SetAsrApiKey(appConfig.ASR.APIKey)
	llm.SetAsrModel(appConfig.ASR.Model)

	logger.WithFields(logrus.Fields{
		"component": "svc_openai_api ",
		"action":    "asr_config_set",
		"baseURL":   appConfig.ASR.BaseURL,
		"model":     appConfig.ASR.Model,
		"hasApiKey": appConfig.ASR.APIKey != "",
	}).Info("ASR configuration set from config file")

	// Initialize VAD integration
	var vadIntegration *VADIntegration
	if appConfig.Vad.Enable {
		vadIntegration = NewVADIntegration(sessionManager, appConfig)
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "vad_integration_enabled",
		}).Info("Per-session VAD integration enabled")
	} else {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "vad_integration_disabled",
		}).Info("VAD integration disabled by config")
	}

	// Create context for cleanup routine
	ctx, cancel := context.WithCancel(context.Background())

	service := &OpenAIService{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow cross-origin for development
			},
		},
		eventParser:    NewEventParser(),
		audioUtils:     NewAudioUtils(),
		sessionManager: sessionManager,
		vadIntegration: vadIntegration,
		config:         openAIConfig,
		appConfig:      appConfig,
		cancel:         cancel,
	}

	// Start audio file cleanup routine
	go service.startAudioCleanup(ctx)

	return service
}

// HandleOpenAIWebSocket handles OpenAI Realtime API WebSocket connections
func (s *OpenAIService) HandleOpenAIWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "websocket_upgrade_failed",
			"error":     err,
		}).Error("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	// Create initial session (will be updated with session.update event)
	session, err := s.sessionManager.CreateSession(conn, "audio")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "create_session_failed",
			"error":     err,
		}).Error("Failed to create session")
		return
	}
	defer s.sessionManager.DeleteSession(session.ID)

	// Send session.created event to client
	createdEvent := &SessionCreatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeSessionCreated,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		Session: struct {
			ID         string   `json:"id"`
			Object     string   `json:"object"`
			Model      string   `json:"model"`
			Modalities []string `json:"modalities"`
		}{
			ID:         session.ID,
			Object:     "realtime.session",
			Model:      "gpt-4",
			Modalities: []string{"audio"},
		},
	}

	if err := s.sessionManager.SendEvent(session, createdEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "send_session_created_failed",
			"sessionID": session.ID,
			"error":     err,
		}).Error("Failed to send session.created event")
	} else {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "session_created_sent",
			"sessionID": session.ID,
		}).Info("Sent session.created event to client")
	}

	// Send conversation.created event to client
	conversationCreatedEvent := &ConversationCreatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeConversationCreated,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		Conversation: struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		}{
			ID:     GenerateConversationID(),
			Object: "realtime.conversation",
		},
	}

	if err := s.sessionManager.SendEvent(session, conversationCreatedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "send_conversation_created_failed",
			"sessionID": session.ID,
			"error":     err,
		}).Error("Failed to send conversation.created event")
	} else {
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "conversation_created_sent",
			"sessionID": session.ID,
		}).Info("Sent conversation.created event to client")
	}

	// Start heartbeat goroutine
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	go s.heartbeatLoop(ctx, session)

	// Main message processing loop
	errChan := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					errChan <- err
					return
				}

				if err := s.handleMessage(session, messageType, message); err != nil {
					logger.WithFields(logrus.Fields{
						"component": "svc_openai_api ",
						"action":    "handle_message_error",
						"sessionID": session.ID,
						"error":     err,
					}).Error("Error handling message")
					// Send error event to client
					errorEvent := &ErrorEvent{
						BaseEvent: BaseEvent{
							Type:      EventTypeError,
							EventID:   GenerateEventID(),
							SessionID: session.ID,
						},
						Error: struct {
							Type    string `json:"type"`
							Code    string `json:"code"`
							Message string `json:"message"`
							Param   string `json:"param,omitempty"`
						}{
							Type:    "invalid_request_error",
							Code:    "message_processing_error",
							Message: err.Error(),
						},
					}
					s.sessionManager.SendEvent(session, errorEvent)
				}
			}
		}
	}()

	// Wait for error or context cancellation
	select {
	case err := <-errChan:
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			logger.WithFields(logrus.Fields{
				"component": "svc_openai_api ",
				"action":    "websocket_unexpected_close_error",
				"sessionID": session.ID,
				"error":     err,
			}).Error("WebSocket unexpected close error")

			// Clean up session resources
			s.sessionManager.RemoveSession(session.ID)
		} else {
			logger.WithFields(logrus.Fields{
				"component": "svc_openai_api ",
				"action":    "websocket_closed_normally",
				"sessionID": session.ID,
				"error":     err,
			}).Info("WebSocket connection closed normally")

			// Clean up session resources
			s.sessionManager.RemoveSession(session.ID)
		}
		return
	case <-ctx.Done():
		logger.WithFields(logrus.Fields{
			"component": "svc_openai_api ",
			"action":    "websocket_closed_by_context",
			"sessionID": session.ID,
		}).Info("WebSocket connection closed by context")

		// Clean up session resources
		s.sessionManager.RemoveSession(session.ID)
		return
	}
}

// handleMessage processes incoming WebSocket messages
func (s *OpenAIService) handleMessage(session *Session, messageType int, message []byte) error {
	s.sessionManager.UpdateHeartbeat(session.ID)

	switch messageType {
	case websocket.TextMessage:
		return s.handleTextMessage(session, message)
	case websocket.BinaryMessage:
		return fmt.Errorf("binary messages not supported in OpenAI Realtime API")
	case websocket.PingMessage:
		logger.WithFields(logrus.Fields{
			"component": "mont_hrtbeat_act",
			"action":    "received_ping",
			"sessionID": session.ID,
		}).Debug("Received Ping from client")

		session.mutex.Lock()
		defer session.mutex.Unlock()

		if session.Conn == nil {
			return fmt.Errorf("session connection closed")
		}

		if err := session.Conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return fmt.Errorf("failed to set write deadline for pong: %v", err)
		}

		return session.Conn.WriteMessage(websocket.PongMessage, nil)
	case websocket.PongMessage:
		logger.WithFields(logrus.Fields{
			"component": "mont_hrtbeat_act",
			"action":    "received_pong",
			"sessionID": session.ID,
		}).Debug("Received Pong from client")
		return nil
	default:
		return fmt.Errorf("unsupported message type: %d", messageType)
	}
}

// handleTextMessage processes JSON text messages
func (s *OpenAIService) handleTextMessage(session *Session, message []byte) error {
	event, err := s.eventParser.ParseEvent(message)
	if err != nil {
		return fmt.Errorf("failed to parse event: %v", err)
	}

	if err := s.eventParser.ValidateEvent(event); err != nil {
		return fmt.Errorf("event validation failed: %v", err)
	}

	// Process the specific event type
	switch e := event.(type) {
	case *SessionUpdateEvent:
		return s.handleSessionUpdate(session, e)
	case *InputAudioBufferAppendEvent:
		return s.handleInputAudioBufferAppend(session, e)
	case *InputAudioBufferCommitEvent:
		return s.handleInputAudioBufferCommit(session, e)
	case *InputAudioBufferCommittedEvent:
		return s.handleInputAudioBufferCommitted(session, e)
	case *InputAudioBufferClearEvent:
		return s.handleInputAudioBufferClear(session, e)
	case *InputAudioBufferSpeechStartedEvent:
		return s.handleInputAudioBufferSpeechStarted(session, e)
	case *InputAudioBufferSpeechStoppedEvent:
		return s.handleInputAudioBufferSpeechStopped(session, e)
	case *HeartbeatPingEvent:
		return s.handleHeartbeatPing(session, e)
	case *HeartbeatPongEvent:
		return s.handleHeartbeatPong(session, e)
	case *ConversationItemDeletedEvent:
		return s.handleConversationItemDeleted(session, e)
	case *InputAudioBufferClearedEvent:
		return s.handleInputAudioBufferCleared(session, e)
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}
}

// handleSessionUpdate processes session.update events
func (s *OpenAIService) handleSessionUpdate(session *Session, event *SessionUpdateEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "mg_session_ctrl",
		"action":    "session_update_received",
		"sessionID": session.ID,
		"inputSampleRate": event.Session.InputAudioFormat.SampleRate,
		"outputSampleRate": event.Session.OutputAudioFormat.SampleRate,
	}).Info("Session update received")

	// Update session configuration
	s.sessionManager.UpdateSession(session.ID, func(sess *Session) {
		sess.Modality = event.Session.Modality
		sess.Instructions = event.Session.Instructions
		sess.Voice = event.Session.Voice
		sess.InputAudioFormat = event.Session.InputAudioFormat
		sess.OutputAudioFormat = event.Session.OutputAudioFormat
		sess.Tools = event.Session.Tools
		sess.ToolChoice = event.Session.ToolChoice

		// Update sample rates if provided in the event
		if event.Session.InputAudioFormat.SampleRate > 0 {
			sess.InputAudioFormat.SampleRate = event.Session.InputAudioFormat.SampleRate
		}
		if event.Session.OutputAudioFormat.SampleRate > 0 {
			sess.OutputAudioFormat.SampleRate = event.Session.OutputAudioFormat.SampleRate
		}

		// Update audio transcription configuration
		if event.Session.InputAudioTranscription != nil {
			sess.InputAudioTranscription.Model = event.Session.InputAudioTranscription.Model
			sess.InputAudioTranscription.Language = event.Session.InputAudioTranscription.Language
		}

		// Update turn detection configuration
		if event.Session.TurnDetection != nil {
			sess.TurnDetection.Type = event.Session.TurnDetection.Type
			sess.TurnDetection.Threshold = event.Session.TurnDetection.Threshold
			sess.TurnDetection.PrefixPaddingMs = event.Session.TurnDetection.PrefixPaddingMs
			sess.TurnDetection.SilenceDurationMs = event.Session.TurnDetection.SilenceDurationMs
		}

		// Log the updated configuration
		logger.WithFields(logrus.Fields{
			"component": "mg_session_ctrl",
			"action":    "session_configuration_updated",
			"sessionID": session.ID,
			"inputSampleRate": sess.InputAudioFormat.SampleRate,
			"outputSampleRate": sess.OutputAudioFormat.SampleRate,
		}).Info("Session configuration updated successfully")
	})

	// Send session.updated response
	responseEvent := &SessionUpdatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeSessionUpdated,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		Session: struct {
			ID         string   `json:"id"`
			Object     string   `json:"object"`
			Model      string   `json:"model"`
			Modalities []string `json:"modalities"`
		}{
			ID:         session.ID,
			Object:     "realtime.session",
			Model:      "gpt-4",
			Modalities: []string{"audio"},
		},
	}

	return s.sessionManager.SendEvent(session, responseEvent)
}

// handleHeartbeatPing processes heartbeat.ping events
func (s *OpenAIService) handleHeartbeatPing(session *Session, _ *HeartbeatPingEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "mont_hrtbeat_act",
		"action":    "ping_received",
		"sessionID": session.ID,
	}).Debug("Ping received for session")

	// Send heartbeat.pong response
	pongEvent := &HeartbeatPongEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeHeartbeatPong,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		HeartbeatType: 1, // PONG type
	}

	return s.sessionManager.SendEvent(session, pongEvent)
}

// handleHeartbeatPong processes heartbeat.pong events
func (s *OpenAIService) handleHeartbeatPong(session *Session, _ *HeartbeatPongEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "mont_hrtbeat_act",
		"action":    "pong_received",
		"sessionID": session.ID,
	}).Debug("Pong received for session")
	// Update session last active time
	s.sessionManager.UpdateSession(session.ID, func(sess *Session) {
		sess.LastActive = time.Now()
	})
	return nil
}

// handleInputAudioBufferAppend processes input_audio_buffer.append events
func (s *OpenAIService) handleInputAudioBufferAppend(session *Session, event *InputAudioBufferAppendEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "proc_audio_main",
		"action":    "buffer_append_received",
		"sessionID": session.ID,
		"sampleRate": session.InputAudioFormat.SampleRate,
	}).Debug("Audio buffer append received")

	// Decode Base64 audio to PCM samples
	samples, err := s.audioUtils.ConvertBase64ToPCM16(event.Audio)
	if err != nil {
		return fmt.Errorf("failed to decode audio: %v", err)
	}

	var reSamples []int16
	if  session.InputAudioFormat.SampleRate == 48000 {
		logger.WithFields(logrus.Fields{
			"component": "proc_rsmpl_audio",
			"action":    "resample_required",
			"sessionID": session.ID,
		}).Debug("Resampling audio from 48kHz to 16kHz for VAD")

	   reSamples, err = s.audioUtils.ResampleAudio(samples, 48000, 16000)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"component":   "resample",
					"action":      "resample_failed",
					"sessionID":   session.ID,
					"error":       err,
				}).Error("Failed to resample audio for VAD")
				// Fallback to original samples if resampling fails
				reSamples = samples
			} else {
				logger.WithFields(logrus.Fields{
					"component":      "resample",
					"action":         "resample_completed",
					"sessionID":      session.ID,
					"inputSamples":   len(samples),
					"outputSamples":  len(reSamples),
				}).Debug("Resampled audio from 48kHz to 16kHz")
			}
	}

	// Accumulate audio data based on buffer_size configuration
	if s.appConfig.Audio.Enable {
		if err := s.accumulateAudioForSaving(session, samples); err != nil {
			logger.WithFields(logrus.Fields{
				"component": "proc_audio_main",
				"action":    "accumulate_audio_failed",
				"sessionID": session.ID,
				"error":     err,
			}).Error("Failed to accumulate audio for saving")
		}
	}

	// Note: Removed direct addition to AudioBuffer
	// VAD-processed audio will be added to VADAudioBuffer for ASR processing
	// This prevents duplicate audio data and ensures only speech segments are processed

	// Process VAD if enabled
	if s.vadIntegration != nil {
		if  session.InputAudioFormat.SampleRate == 48000 {
			if err := s.vadIntegration.ProcessAudioSamples(session.ID, reSamples); err != nil {
				logger.WithFields(logrus.Fields{
					"component":   "vad",
					"action":      "processing_error",
					"sessionID":   session.ID,
					"error":       err,
				}).Error("VAD processing error")
			}
		}
		if session.InputAudioFormat.SampleRate == 16000 {
			if err := s.vadIntegration.ProcessAudioSamples(session.ID, samples); err != nil {
				logger.WithFields(logrus.Fields{
					"component":   "vad",
					"action":      "processing_error",
					"sessionID":   session.ID,
					"error":       err,
				}).Error("VAD processing error")
			}
		}

	}

	return nil
}

// handleInputAudioBufferCommit processes input_audio_buffer.commit events
func (s *OpenAIService) handleInputAudioBufferCommit(session *Session, _ *InputAudioBufferCommitEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "proc_audio_main",
		"action":    "buffer_commit_received",
		"sessionID": session.ID,
	}).Info("Audio buffer commit received from client")

	// Get current VAD buffer size for debugging
	bufferSize, err := s.sessionManager.GetVADAudioBufferSize(session.ID)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "proc_audio_main",
			"action":      "get_vad_buffer_size_failed",
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to get VAD buffer size")
	} else {
		logger.WithFields(logrus.Fields{
			"component":  "proc_audio_main",
			"action":     "vad_buffer_size_checked",
			"sessionID":  session.ID,
			"vadBufferSize": bufferSize,
		}).Info("VAD buffer contains samples before processing")
	}

	// Send input_audio_buffer.committed confirmation first
	committedEvent := &InputAudioBufferCommittedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeInputAudioBufferCommitted,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
	}

	if err := s.sessionManager.SendEvent(session, committedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "proc_audio_main",
			"action":      "send_committed_event_failed",
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to send committed event")
	} else {
		logger.WithFields(logrus.Fields{
			"component": "proc_audio_main",
			"action":    "committed_event_sent",
			"sessionID": session.ID,
		}).Info("Sent committed confirmation to client")
	}

	// Process the accumulated audio for recognition
	return s.processAudioForRecognition(session)
}

// handleInputAudioBufferCommitted processes input_audio_buffer.committed events
func (s *OpenAIService) handleInputAudioBufferCommitted(session *Session, _ *InputAudioBufferCommittedEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "proc_audio_main",
		"action":    "buffer_committed_received",
		"sessionID": session.ID,
	}).Info("Audio buffer committed confirmation received")

	// This is a confirmation event from client, no action needed
	return nil
}

// handleInputAudioBufferClear processes input_audio_buffer.clear events
func (s *OpenAIService) handleInputAudioBufferClear(session *Session, _ *InputAudioBufferClearEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "proc_audio_main",
		"action":    "buffer_clear_received",
		"sessionID": session.ID,
	}).Info("Audio buffer clear received")

	// Clear the audio buffer
	return s.sessionManager.ClearAudioBuffer(session.ID)
}

// handleInputAudioBufferSpeechStarted processes speech started events
func (s *OpenAIService) handleInputAudioBufferSpeechStarted(session *Session, event *InputAudioBufferSpeechStartedEvent) error {
	logger.WithFields(logrus.Fields{
		"component":     "vad",
		"action":        "speech_started",
		"sessionID":     session.ID,
		"audioStartMs":  event.AudioStartMs,
	}).Info("Speech started")

	s.sessionManager.UpdateSession(session.ID, func(sess *Session) {
		sess.IsSpeaking = true
		sess.SpeechStartTime = time.Now()
	})

	return nil
}

// handleInputAudioBufferSpeechStopped processes speech stopped events
func (s *OpenAIService) handleInputAudioBufferSpeechStopped(session *Session, event *InputAudioBufferSpeechStoppedEvent) error {
	logger.WithFields(logrus.Fields{
		"component":    "vad",
		"action":       "speech_stopped",
		"sessionID":    session.ID,
		"audioEndMs":   event.AudioEndMs,
	}).Info("Speech stopped")

	s.sessionManager.UpdateSession(session.ID, func(sess *Session) {
		sess.IsSpeaking = false
	})

	// Auto-commit audio buffer on speech stop
	return s.processAudioForRecognition(session)
}

// processAudioForRecognition processes accumulated audio for speech recognition
func (s *OpenAIService) processAudioForRecognition(session *Session) error {
	startTime := time.Now()

	// Get current VAD audio buffer (contains only speech segments)
	buffer, err := s.sessionManager.GetVADAudioBuffer(session.ID)
	if err != nil {
		return fmt.Errorf("failed to get VAD audio buffer: %v", err)
	}

	if len(buffer) == 0 {
		logger.WithFields(logrus.Fields{
			"component": "proc_audio_main",
			"action":    "no_vad_audio_data",
			"sessionID": session.ID,
		}).Info("No VAD audio data to process")
		return nil
	}

	bufferDuration := float64(len(buffer)) / 16000.0 // Calculate duration in seconds
	logger.WithFields(logrus.Fields{
		"component":     "proc_audio_main",
		"action":        "processing_vad_audio_for_recognition",
		"sampleCount":   len(buffer),
		"duration":      bufferDuration,
		"sessionID":     session.ID,
	}).Info("Processing VAD-filtered samples for recognition")

	// Create conversation item for this recognition
	item, err := s.sessionManager.CreateConversationItem(session.ID, "message", "user")
	if err != nil {
		return fmt.Errorf("failed to create conversation item: %v", err)
	}

	// Send conversation.item.created event
	itemCreatedEvent := &ConversationItemCreatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeConversationItemCreated,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		Item: struct {
			ID        string        `json:"id"`
			Type      string        `json:"type"`
			Status    string        `json:"status"`
			Audio     *struct {
				Data   string `json:"data"`
				Format string `json:"format"`
			} `json:"audio,omitempty"`
			Content   []interface{} `json:"content,omitempty"`
		}{
			ID:     item.ID,
			Type:   item.Type,
			Status: item.Status,
			Audio: &struct {
				Data   string `json:"data"`
				Format string `json:"format"`
			}{
				Data:   s.audioUtils.ConvertPCM16ToBase64(buffer),
				Format: "pcm16",
			},
		},
	}

	if err := s.sessionManager.SendEvent(session, itemCreatedEvent); err != nil {
		return fmt.Errorf("failed to send conversation.item.created event: %v", err)
	}

	// Process recognition asynchronously
	go s.processRecognition(session, item.ID, buffer)

	// Clear the VAD audio buffer after processing
	if err := s.sessionManager.ClearVADAudioBuffer(session.ID); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "proc_audio_main",
			"action":      "clear_vad_buffer_failed",
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to clear VAD audio buffer")
	}

	processingTimeMs := time.Since(startTime).Milliseconds()
	logger.WithFields(logrus.Fields{
		"component":      "proc_audio_main",
		"action":         "processing_completed",
		"sessionID":      session.ID,
		"processingTimeMs": processingTimeMs,
	}).Debug("Completed audio processing for recognition")

	return nil
}

// processRecognition processes audio recognition asynchronously
func (s *OpenAIService) processRecognition(session *Session, itemID string, audioData []int16) {
	startTime := time.Now()
	conversationItemCreationTime := startTime // Record when conversation item was created
	logger.WithFields(logrus.Fields{
		"component":   "audio_recogniz",
		"action":      "starting_processing",
		"itemID":      itemID,
		"sessionID":   session.ID,
		"sampleCount": len(audioData),
	}).Debug("Starting recognition processing")

	// Convert audio data to WAV format for recognition
	wavData, err := s.convertToWAV(audioData)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "audio_recogniz",
			"action":      "audio_conversion_failed",
			"itemID":      itemID,
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to convert audio to WAV")
		s.sendRecognitionFailed(session, itemID, "audio_conversion_error", err.Error(), conversationItemCreationTime)
		return
	}

	conversionTimeMs := time.Since(startTime).Milliseconds()
	logger.WithFields(logrus.Fields{
		"component":     "audio_recogniz",
		"action":        "audio_conversion_completed",
		"sessionID":     session.ID,
		"wavDataSize":   len(wavData),
		"conversionTimeMs": conversionTimeMs,
	}).Info("Audio conversion completed")

	// Call speech recognition API
	recognitionStartTime := time.Now()
	text, err := s.callRecognitionAPI(wavData)
	if err != nil {
		recognitionTimeMs := time.Since(recognitionStartTime).Milliseconds()
		logger.WithFields(logrus.Fields{
			"component":      "audio_recogniz",
			"action":         "recognition_failed",
			"itemID":         itemID,
			"sessionID":      session.ID,
			"recognitionTimeMs": recognitionTimeMs,
			"error":          err,
		}).Error("Recognition failed")
		s.sendRecognitionFailed(session, itemID, "recognition_error", err.Error(), conversationItemCreationTime)
		return
	}

	recognitionTimeMs := time.Since(recognitionStartTime).Milliseconds()
	totalTimeMs := time.Since(startTime).Milliseconds()
	logger.WithFields(logrus.Fields{
		"component":       "audio_recogniz",
		"action":          "recognition_successful",
		"itemID":          itemID,
		"sessionID":       session.ID,
		"text":            text,
		"recognitionTimeMs": recognitionTimeMs,
		"totalTimeMs":     totalTimeMs,
	}).Info("Recognition successful")

	// Send transcription completed event
	s.sendRecognitionCompleted(session, itemID, text, conversationItemCreationTime)
}

// convertToWAV converts PCM audio data to WAV format
func (s *OpenAIService) convertToWAV(audioData []int16) ([]byte, error) {
	logger.WithFields(logrus.Fields{
		"component":   "audio_conversion",
		"action":      "converting_pcm_to_wav",
		"sampleCount": len(audioData),
	}).Info("Converting PCM samples to WAV format")

	// Use the audio utilities to convert PCM to WAV
	wavData, err := s.audioUtils.ConvertPCM16ToWAV(audioData, 16000) // Default to 16kHz for ASR
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "audio_conversion",
			"action":      "pcm_to_wav_conversion_failed",
			"sampleCount": len(audioData),
			"error":       err,
		}).Error("Failed to convert PCM to WAV")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"component":     "audio_conversion",
		"action":        "pcm_to_wav_conversion_successful",
		"inputSamples":  len(audioData),
		"outputDataSize": len(wavData),
	}).Info("Successfully converted PCM samples to WAV data")
	return wavData, nil
}

// callRecognitionAPI calls the speech recognition API
func (s *OpenAIService) callRecognitionAPI(wavData []byte) (string, error) {
	logger.WithFields(logrus.Fields{
		"component":   "asr_api_core",
		"action":      "calling_recognition_api",
		"dataSize":    len(wavData),
	}).Info("Calling speech recognition API")

	// Use the existing LLM package for speech recognition
	text, err := llm.CallOpenaiAPI(wavData)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "api_asr_core",
			"action":      "api_call_failed",
			"dataSize":    len(wavData),
			"error":       err,
		}).Error("Speech recognition API call failed")
		return "", err
	}

	logger.WithFields(logrus.Fields{
		"component":   "api_asr_core",
		"action":      "api_call_successful",
		"dataSize":    len(wavData),
		"recognizedText": text,
	}).Info("Speech recognition API call successful")
	return text, nil
}

// sendRecognitionCompleted sends transcription completed event
func (s *OpenAIService) sendRecognitionCompleted(session *Session, itemID string, text string, conversationItemCreationTime time.Time) {
	logger.WithFields(logrus.Fields{
		"component":   "ws_event_send ",
		"action":      "sending_transcription_completed",
		"itemID":      itemID,
		"sessionID":   session.ID,
		"text":        text,
	}).Info("Sending transcription completed event")

	completedEvent := &ConversationItemInputAudioTranscriptionCompletedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeConversationItemInputAudioTranscriptionCompleted,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		Item: struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			Status    string `json:"status"`
			Content   []struct {
				Type      string `json:"type"`
				Transcript string `json:"transcript"`
			} `json:"content"`
		}{
			ID:     itemID,
			Type:   "message",
			Status: "completed",
			Content: []struct {
				Type      string `json:"type"`
				Transcript string `json:"transcript"`
			}{
				{
					Type:      "transcript",
					Transcript: text,
				},
			},
		},
	}

	if err := s.sessionManager.SendEvent(session, completedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "error",
			"action":      "send_transcription_completed_failed",
			"itemID":      itemID,
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to send transcription completed event")
	} else {
		logger.WithFields(logrus.Fields{
			"component":   "",
			"action":      "transcription_completed_sent",
			"itemID":      itemID,
			"sessionID":   session.ID,
		}).Info("Successfully sent transcription completed event")
	}

	// Mark conversation item as completed
	if err := s.sessionManager.MarkConversationItemCompleted(session.ID, itemID); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "error",
			"action":      "mark_item_completed_failed",
			"itemID":      itemID,
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to mark conversation item as completed")
	} else {
		// Calculate conversation item processing time in milliseconds
		conversationItemProcessingTimeMs := time.Since(conversationItemCreationTime).Milliseconds()

		logger.WithFields(logrus.Fields{
			"component":                      "mg_session",
			"action":                         "item_marked_completed",
			"itemID":                         itemID,
			"sessionID":                      session.ID,
			"conversationItemProcessingTimeMs": conversationItemProcessingTimeMs,
		}).Info("Conversation item processing completed")

		// Additional detailed logging for performance monitoring
		logger.WithFields(logrus.Fields{
			"component":                      "mg_performance",
			"action":                         "conversation_item_processed",
			"itemID":                         itemID,
			"sessionID":                      session.ID,
			"conversationItemProcessingTimeMs": conversationItemProcessingTimeMs,
			"textLength":                     len(text),
		}).Info("ASR Conversation item performance metrics")
	}
}

// sendRecognitionFailed sends transcription failed event
func (s *OpenAIService) sendRecognitionFailed(session *Session, itemID string, errorCode string, errorMessage string, conversationItemCreationTime time.Time) {
	logger.WithFields(logrus.Fields{
		"component":    "ws_event_send ",
		"action":       "sending_transcription_failed",
		"itemID":       itemID,
		"sessionID":    session.ID,
		"errorCode":    errorCode,
		"errorMessage": errorMessage,
	}).Info("Sending transcription failed event")

	failedEvent := &ConversationItemInputAudioTranscriptionFailedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeConversationItemInputAudioTranscriptionFailed,
			EventID:   GenerateEventID(),
			SessionID: session.ID,
		},
		ItemID: itemID,
		Error: struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
			Param   string `json:"param,omitempty"`
		}{
			Type:    "api_error",
			Code:    errorCode,
			Message: errorMessage,
		},
	}

	if err := s.sessionManager.SendEvent(session, failedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "error",
			"action":      "send_transcription_failed_failed",
			"itemID":      itemID,
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to send transcription failed event")
	} else {
		logger.WithFields(logrus.Fields{
			"component":   "ws_event_send ",
			"action":      "transcription_failed_sent",
			"itemID":      itemID,
			"sessionID":   session.ID,
		}).Info("Successfully sent transcription failed event")
	}

	// Mark conversation item as failed
	if err := s.sessionManager.MarkConversationItemFailed(session.ID, itemID, errorMessage); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "error",
			"action":      "mark_item_failed_failed",
			"itemID":      itemID,
			"sessionID":   session.ID,
			"error":       err,
		}).Error("Failed to mark conversation item as failed")
	} else {
		// Calculate conversation item processing time in milliseconds (failed case)
		conversationItemProcessingTimeMs := time.Since(conversationItemCreationTime).Milliseconds()

		logger.WithFields(logrus.Fields{
			"component":                      "mg_session",
			"action":                         "item_marked_failed",
			"itemID":                         itemID,
			"sessionID":                      session.ID,
			"conversationItemProcessingTimeMs": conversationItemProcessingTimeMs,
			"errorCode":                      errorCode,
		}).Info("Conversation item processing failed")

		// Additional detailed logging for performance monitoring (failed case)
		logger.WithFields(logrus.Fields{
			"component":                      "mg_performance",
			"action":                         "conversation_item_failed",
			"itemID":                         itemID,
			"sessionID":                      session.ID,
			"conversationItemProcessingTimeMs": conversationItemProcessingTimeMs,
			"errorCode":                      errorCode,
			"errorMessageLength":             len(errorMessage),
		}).Info("ASR Conversation item failure metrics")
	}
}

// handleConversationItemDeleted processes conversation.item.deleted events
func (s *OpenAIService) handleConversationItemDeleted(session *Session, event *ConversationItemDeletedEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "mg_conv_ctrl",
		"action":    "item_deleted",
		"sessionID": session.ID,
		"itemID":    event.ItemID,
		"eventID":   event.EventID,
	}).Info("Conversation item deleted event received")

	// Event logging only - no action needed
	return nil
}

// handleInputAudioBufferCleared processes input_audio_buffer.cleared events
func (s *OpenAIService) handleInputAudioBufferCleared(session *Session, event *InputAudioBufferClearedEvent) error {
	logger.WithFields(logrus.Fields{
		"component": "proc_audio_main",
		"action":    "buffer_cleared",
		"sessionID": session.ID,
		"eventID":   event.EventID,
	}).Info("Input audio buffer cleared event received")

	// Event logging only - no action needed
	return nil
}

// heartbeatLoop sends periodic heartbeat messages
func (s *OpenAIService) heartbeatLoop(ctx context.Context, session *Session) {
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			session.mutex.Lock()
			if session.Conn == nil {
				session.mutex.Unlock()
				logger.WithFields(logrus.Fields{
					"component":   "mont_hrtbeat_act",
					"action":      "send_ping_failed",
					"sessionID":   session.ID,
					"error":       "connection closed",
				}).Error("Failed to send ping to session: connection closed")
				return
			}

			if err := session.Conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
				session.mutex.Unlock()
				logger.WithFields(logrus.Fields{
					"component":   "mont_hrtbeat_act",
					"action":      "send_ping_failed",
					"sessionID":   session.ID,
					"error":       err,
				}).Error("Failed to set write deadline for ping")
				return
			}

			if err := session.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				session.mutex.Unlock()
				logger.WithFields(logrus.Fields{
					"component":   "mont_hrtbeat_act",
					"action":      "send_ping_failed",
					"sessionID":   session.ID,
					"error":       err,
				}).Error("Failed to send ping to session")
				return
			}
			session.mutex.Unlock()

			logger.WithFields(logrus.Fields{
				"component": "mont_hrtbeat_act",
				"action":    "ping_sent",
				"sessionID": session.ID,
			}).Debug("Sent ping to session")
		}
	}
}

// GetSessionStats returns session statistics
func (s *OpenAIService) GetSessionStats() map[string]interface{} {
	return s.sessionManager.GetSessionStats()
}

// Cleanup performs cleanup operations
func (s *OpenAIService) Cleanup() {
	// Cancel cleanup context to stop the audio cleanup routine
	if s.cancel != nil {
		s.cancel()
	}

	s.sessionManager.CleanupInactiveSessions()
}

// startAudioCleanup starts a routine to clean up old audio files
func (s *OpenAIService) startAudioCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get keep_files from config, default to 10
			keepFiles := s.appConfig.Audio.KeepFiles
			if keepFiles <= 0 {
				keepFiles = 10
			}

			if err := s.audioUtils.CleanOldAudioFiles(keepFiles); err != nil {
				logger.WithFields(logrus.Fields{
					"component": "cln_audio_proc",
					"action":    "cleanup_failed",
					"error":     err,
					"keepFiles": keepFiles,
				}).Error("Failed to clean up old audio files")
			}
		case <-ctx.Done():
			// Context cancelled, exit gracefully
			logger.WithFields(logrus.Fields{
				"component": "cln_audio_proc",
				"action":    "cleanup_stopped",
			}).Info("Audio cleanup routine stopped")
			return
		}
	}
}

// accumulateAudioForSaving accumulates audio data based on buffer_size config and saves at time intervals
func (s *OpenAIService) accumulateAudioForSaving(session *Session, samples []int16) error {
	// Get configured buffer_size in seconds
	bufferSize := s.appConfig.Audio.BufferSize
	if bufferSize <= 0 {
		bufferSize = 10 // default 10 seconds
	}

	// Get sample rate for time calculations
	sampleRate := session.InputAudioFormat.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000 // fallback to 16kHz
	}

	session.AudioSaveMutex.Lock()
	defer session.AudioSaveMutex.Unlock()

	now := time.Now()

	// Initialize accumulation cycle on first run
	if session.AccumulationStartTime.IsZero() {
		session.AccumulationStartTime = now
		session.AccumulatedAudio = make([]int16, 0)
		logger.WithFields(logrus.Fields{
			"component":       "ws_audio_core ",
			"action":          "accumulation_started",
			"sessionID":       session.ID,
			"bufferSize":      bufferSize,
			"sampleRate":      sampleRate,
		}).Info("Started audio accumulation cycle")
	}

	// Append new samples to accumulation buffer
	session.AccumulatedAudio = append(session.AccumulatedAudio, samples...)

	// Calculate elapsed time since accumulation started (seconds)
	elapsedTime := now.Sub(session.AccumulationStartTime).Seconds()

	// Calculate accumulated audio duration based on sample count
	accumulatedDuration := float64(len(session.AccumulatedAudio)) / float64(sampleRate)

	// Check save condition: time or duration reaches buffer_size
	shouldSave := elapsedTime >= float64(bufferSize) || accumulatedDuration >= float64(bufferSize)

	if shouldSave {
		// Generate filename
		filename := fmt.Sprintf("segment_%s_%d.wav", session.ID[:8], session.AccumulationStartTime.UnixNano()/1000000)

		// Save accumulated audio file
		if err := s.audioUtils.SaveAudioToFile(session.AccumulatedAudio, sampleRate, filename); err != nil {
			return fmt.Errorf("failed to save accumulated audio: %v", err)
		}

		logger.WithFields(logrus.Fields{
			"component":          "ws_audio_core ",
			"action":             "saved_accumulated_segment",
			"sessionID":          session.ID,
			"filename":           filename,
			"duration":           accumulatedDuration,
			"samples":            len(session.AccumulatedAudio),
			"elapsedTime":        elapsedTime,
			"bufferSize":         bufferSize,
		}).Info("Saved accumulated audio segment")

		// Reset accumulation state
		session.AccumulatedAudio = make([]int16, 0)
		session.AccumulationStartTime = now
		session.LastSaveTime = now
	}

	return nil
}