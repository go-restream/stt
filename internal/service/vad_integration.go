package service

import (
	"fmt"
	"time"

	"github.com/go-restream/stt/config"
	"github.com/go-restream/stt/pkg/logger"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
	"github.com/sirupsen/logrus"
)

type VADIntegration struct {
	sessionManager      *SessionManager
	sampleBuffer        []float32
	lastProcessingTime  time.Time
	config              *config.Config
}

func NewVADIntegration(sessionManager *SessionManager, cfg *config.Config) *VADIntegration {
	return &VADIntegration{
		sessionManager:     sessionManager,
		sampleBuffer:       make([]float32, 0),
		lastProcessingTime: time.Now(),
		config:             cfg,
	}
}

func (vi *VADIntegration) ProcessAudioSamples(sessionID string, samples []int16) error {
	startTime := time.Now()
	if len(samples) == 0 {
		logger.WithFields(logrus.Fields{
			"component": "proc_vad_audio",
			"action":    "empty_samples_received",
			"sessionID": sessionID,
		}).Warn("Received empty audio samples")
		return nil
	}

	// Get session to retrieve per-session VAD detector
	session, exists := vi.sessionManager.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.VADDetector == nil {
		logger.WithFields(logrus.Fields{
			"component": "proc_vad_audio",
			"action":    "vad_detector_not_found",
			"sessionID": sessionID,
		}).Warn("VAD detector not found for session")
		return nil
	}

	var maxAmplitude int16
	var sumAmplitude int64
	for _, sample := range samples {
		if sample < 0 {
			sumAmplitude -= int64(sample)
		} else {
			sumAmplitude += int64(sample)
		}
		if sample < 0 {
			sample = -sample
		}
		if sample > maxAmplitude {
			maxAmplitude = sample
		}
	}
	avgAmplitude := float64(sumAmplitude) / float64(len(samples))

	logger.WithFields(logrus.Fields{
		"component":    "proc_vad_audio",
		"action":       "starting_processing",
		"sampleCount":  len(samples),
		"sessionID":    sessionID,
		"maxAmplitude": maxAmplitude,
		"avgAmplitude": avgAmplitude,
		"hasAudio":     maxAmplitude > 100, // Threshold for "significant" audio
	}).Debug("Starting VAD processing with audio validation")

	conversionStart := time.Now()
	floatSamples := make([]float32, len(samples))
	for i, sample := range samples {
		floatSamples[i] = float32(sample) / 32768.0
	}
	conversionTime := time.Since(conversionStart)
	logger.WithFields(logrus.Fields{
		"component":     "proc_vad_audio",
		"action":        "conversion_completed",
		"inputSamples":  len(samples),
		"outputSamples": len(floatSamples),
		"conversionTime": conversionTime,
		"sessionID":     sessionID,
	}).Debug("Converted int16 samples to float32 samples")

	chunksProcessed := 0
	speechSegmentsDetected := 0
	vadProcessingTime := time.Duration(0)

	for i := 0; i < len(floatSamples); i += 160 {
		end := i + 160
		if end > len(floatSamples) {
			end = len(floatSamples)
		}

		chunk := floatSamples[i:end]
		vi.sampleBuffer = append(vi.sampleBuffer, chunk...)

		if len(vi.sampleBuffer) >= 160 {
			chunksProcessed++
			vadStart := time.Now()
			segment := session.VADDetector.ProcessSamples(vi.sampleBuffer)
			vadProcessingTime += time.Since(vadStart)
			vi.sampleBuffer = vi.sampleBuffer[:0]

			if segment != nil && len(segment.Samples) > 0 {
				speechSegmentsDetected++
				logger.WithFields(logrus.Fields{
					"component":   "proc_vad_audio",
					"action":      "speech_segment_detected",
					"sampleCount": len(segment.Samples),
					"vadProcessingTimeMs":   vadProcessingTime.Milliseconds(),
					"sessionID":   sessionID,
				}).Info("Speech segment detected")

				if !session.IsSpeaking {
					logger.WithFields(logrus.Fields{
						"component": "proc_vad_audio",
						"action":    "transition_to_speaking",
						"sessionID": sessionID,
					}).Info("Transition to speaking state")
					vi.handleSpeechStarted(sessionID)
				}
				session.SpeechStartTime = time.Now()

				vi.processSpeechSegment(sessionID, segment)
			} else {
				silenceTimeout := 500 * time.Millisecond // Default 500ms silence timeout
				if vi.config.Vad.MinSilenceDuration > 0 {
					silenceTimeout = time.Duration(vi.config.Vad.MinSilenceDuration * 1000) * time.Millisecond
				}

				if session.IsSpeaking && time.Since(session.SpeechStartTime) > silenceTimeout {
					logger.WithFields(logrus.Fields{
						"component":       "proc_vad_audio",
						"action":          "speech_timeout_detected",
						"sessionID":       sessionID,
						"silenceDuration": time.Since(session.SpeechStartTime),
						"timeout":         silenceTimeout,
					}).Info("Speech timeout detected - stopping speech")
					vi.handleSpeechStopped(sessionID)
				}
			}
		}
	}

	totalTime := time.Since(startTime)
	logger.WithFields(logrus.Fields{
		"component":           "proc_vad_audio",
		"action":              "processing_completed",
		"chunksProcessed":     chunksProcessed,
		"speechSegments":      speechSegmentsDetected,
		"sessionID":           sessionID,
		"totalTime":           totalTime,
		"vadProcessingTimeMs":   vadProcessingTime.Milliseconds(),
		"conversionTime":      conversionTime,
	}).Debug("Completed VAD processing")

		if vi.config.Vad.ForceASRAfterSeconds > 0 {
				if bufferSize, err := vi.sessionManager.GetVADAudioBuffer(sessionID); err == nil && len(bufferSize) > 16000 { // 1 second of audio at 16kHz
			timeSinceLastProcess := time.Since(vi.lastProcessingTime)
			logger.WithFields(logrus.Fields{
				"component":           "vad",
				"action":              "checking_timer",
				"sessionID":           sessionID,
				"vadBufferSize":       len(bufferSize),
				"timeSinceLastProcess": timeSinceLastProcess.Seconds(),
				"forceAfterSeconds":   vi.config.Vad.ForceASRAfterSeconds,
			}).Debug("Checking ASR trigger timer")

			if timeSinceLastProcess.Seconds() >= float64(vi.config.Vad.ForceASRAfterSeconds) {
				logger.WithFields(logrus.Fields{
					"component":           "vad",
					"action":              "force_asr_trigger",
					"sessionID":           sessionID,
					"vadBufferSize":       len(bufferSize),
					"timeSinceLastProcess": timeSinceLastProcess.Seconds(),
					"forceAfterSeconds":   vi.config.Vad.ForceASRAfterSeconds,
				}).Warn("Force triggering ASR processing (testing mode)")

								vi.handleSpeechStopped(sessionID)

								vi.lastProcessingTime = time.Now()
			}
		}
	}

	return nil
}

func (vi *VADIntegration) handleSpeechStarted(sessionID string) {
	vi.sessionManager.UpdateSession(sessionID, func(sess *Session) {
		sess.IsSpeaking = true
		sess.SpeechStartTime = time.Now()
	})

	session, exists := vi.sessionManager.GetSession(sessionID)
	if !exists {
		return
	}

	audioStartMs := int(time.Since(session.SpeechStartTime).Milliseconds())

	speechStartedEvent := &InputAudioBufferSpeechStartedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeInputAudioBufferSpeechStarted,
			EventID:   GenerateEventID(),
			SessionID: sessionID,
		},
		AudioStartMs: audioStartMs,
	}

	if err := vi.sessionManager.SendEventToSession(sessionID, speechStartedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "ws_event_send",
			"action":      "send_speech_started_event_failed",
			"sessionID":   sessionID,
			"error":       err,
		}).Error("Failed to send speech started event")
	} else {
		logger.WithFields(logrus.Fields{
			"component":    "ws_event_send",
			"action":       "speech_started_detected",
			"sessionID":    sessionID,
			"audioStartMs": audioStartMs,
		}).Info("Speech started detected")
	}
}

func (vi *VADIntegration) handleSpeechStopped(sessionID string) {
	session, exists := vi.sessionManager.GetSession(sessionID)
	if !exists {
		return
	}

	if !session.IsSpeaking {
		logger.WithFields(logrus.Fields{
			"component": "proc_vad_audio",
			"action":    "speech_stopped_already_not_speaking",
			"sessionID": sessionID,
		}).Warn("Speech stopped called but not currently speaking")
		return
	}

	vi.sessionManager.UpdateSession(sessionID, func(sess *Session) {
		sess.IsSpeaking = false
	})

	audioEndMs := int(time.Since(session.SpeechStartTime).Milliseconds())

	speechStoppedEvent := &InputAudioBufferSpeechStoppedEvent{
		BaseEvent: BaseEvent{
			Type:      EventTypeInputAudioBufferSpeechStopped,
			EventID:   GenerateEventID(),
			SessionID: sessionID,
		},
		AudioEndMs: audioEndMs,
	}

	if err := vi.sessionManager.SendEventToSession(sessionID, speechStoppedEvent); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "ws_event_send",
			"action":      "send_speech_stopped_event_failed",
			"sessionID":   sessionID,
			"error":       err,
		}).Error("Failed to send speech stopped event")
	} else {
		logger.WithFields(logrus.Fields{
			"component":  "ws_event_send",
			"action":     "speech_stopped_detected",
			"sessionID":  sessionID,
			"audioEndMs": audioEndMs,
		}).Info("Speech stopped detected and event sent - waiting for client to commit")
	}

	// OpenAI Realtime API spec: CLIENT sends input_audio_buffer.commit after speech_stopped
	logger.WithFields(logrus.Fields{
		"component": "ws_event_send",
		"action":    "speech_stopped_completed",
		"sessionID": sessionID,
	}).Info("Speech stopped completed - waiting for client to send commit message")
}

func (vi *VADIntegration) processSpeechSegment(sessionID string, segment *sherpa.SpeechSegment) {
	startTime := time.Now()

	if segment == nil || len(segment.Samples) == 0 {
		logger.WithFields(logrus.Fields{
			"component": "proc_vad_audio",
			"action":    "empty_speech_segment",
			"sessionID": sessionID,
		}).Warn("Empty speech segment, skipping processing")
		return
	}

	segmentDuration := float64(len(segment.Samples)) / 16000.0
	logger.WithFields(logrus.Fields{
		"component":      "proc_vad_audio",
		"action":         "processing_speech_segment",
		"sampleCount":    len(segment.Samples),
		"duration":       segmentDuration,
		"sessionID":      sessionID,
	}).Info("Processing speech segment for recognition")

	// Apply denoising if enabled and available
	processedSegment := segment
	session, exists := vi.sessionManager.GetSession(sessionID)
	if exists && session.DenoiserProcessor != nil && vi.config.Denoiser.Enable {
		denoiserStart := time.Now()
		enhancedSegment := session.DenoiserProcessor.ProcessSegment(segment)
		denoiserTime := time.Since(denoiserStart)

		if enhancedSegment != nil && len(enhancedSegment.Samples) > 0 {
			processedSegment = enhancedSegment
			logger.WithFields(logrus.Fields{
				"component":        "proc_vad_audio",
				"action":           "denoising_applied",
				"originalSamples":  len(segment.Samples),
				"enhancedSamples":  len(enhancedSegment.Samples),
				"denoiserTimeMs":     denoiserTime.Milliseconds(),
				"sessionID":        sessionID,
			}).Info("Audio segment enhanced with denoiser")
		} else {
			logger.WithFields(logrus.Fields{
				"component":     "proc_vad_audio",
				"action":        "denoising_failed_or_bypassed",
				"sampleCount":   len(segment.Samples),
				"denoiserTimeMs":  denoiserTime.Milliseconds(),
				"sessionID":     sessionID,
			}).Warn("Denoiser failed or was bypassed, using original segment")
		}
	}

	conversionStart := time.Now()
	samples := make([]int16, len(processedSegment.Samples))
	for i, sample := range processedSegment.Samples {
		samples[i] = int16(sample * 32768.0)
	}
	conversionTime := time.Since(conversionStart)
	logger.WithFields(logrus.Fields{
		"component":      "proc_vad_audio",
		"action":         "conversion_to_int16_completed",
		"inputSamples":   len(processedSegment.Samples),
		"outputSamples":  len(samples),
		"conversionTime": conversionTime,
		"sessionID":      sessionID,
	}).Info("Converted float32 samples to int16 samples")

		bufferAddStart := time.Now()
	if err := vi.sessionManager.AddVADAudioToBuffer(sessionID, samples); err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "proc_vad_audio",
			"action":      "add_speech_to_vad_buffer_failed",
			"sessionID":   sessionID,
			"error":       err,
		}).Error("Failed to add speech segment to VAD buffer")
		return
	}
	bufferAddTime := time.Since(bufferAddStart)

		bufferSize, err := vi.sessionManager.GetVADAudioBufferSize(sessionID)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component":   "proc_vad_audio",
			"action":      "get_vad_buffer_size_failed",
			"sessionID":   sessionID,
			"error":       err,
		}).Error("Failed to get VAD audio buffer size")
	} else {
		logger.WithFields(logrus.Fields{
			"component":    "proc_vad_audio",
			"action":       "speech_segment_added_to_vad_buffer",
			"vadBufferSize":   bufferSize,
			"addTime":      bufferAddTime,
			"sessionID":    sessionID,
		}).Info("Audio buffer now contains samples after adding speech segment")
	}

	// Speech segments accumulated in VAD buffer, committed when client sends input_audio_buffer.commit
	logger.WithFields(logrus.Fields{
		"component":  "proc_vad_audio",
		"action":     "speech_segment_processed",
		"sessionID":  sessionID,
		"totalTime":  time.Since(startTime),
	}).Info("Speech segment processed and added to VAD buffer - waiting for speech_stopped")
}

func (vi *VADIntegration) Reset(sessionID string) {
	vi.sampleBuffer = vi.sampleBuffer[:0]

	session, exists := vi.sessionManager.GetSession(sessionID)
	if !exists || session.VADDetector == nil {
		return
	}

	session.VADDetector.Reset()

	vi.sessionManager.UpdateSession(sessionID, func(sess *Session) {
		sess.IsSpeaking = false
	})
}

func (vi *VADIntegration) IsSpeaking(sessionID string) bool {
	session, exists := vi.sessionManager.GetSession(sessionID)
	if !exists {
		return false
	}
	return session.IsSpeaking
}