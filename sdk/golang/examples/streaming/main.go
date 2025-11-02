package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	asr "gosdk/client"
)

func main() {
	fmt.Println("🎤 ASR SDK - Streaming Recognition Example")
	fmt.Println("Press Ctrl+C to exit")

	// Create streaming callback handler
	handler := &StreamingHandler{
		partialChan: make(chan string, 100),
		finalChan:   make(chan string, 100),
		errorChan:   make(chan error, 100),
	}

	// Create recognizer
	recognizer, err := asr.CreateRecognizerWithEventHandler("ws://localhost:8088/ws", "zh-CN", handler)
	if err != nil {
		log.Fatalf("Failed to create recognizer: %v", err)
	}

	// Start recognition
	if err := recognizer.Start(); err != nil {
		log.Fatalf("Failed to start recognition: %v", err)
	}
	defer recognizer.Stop()

	// Start result display
	go handler.displayResults()

	fmt.Println("✅ Recognizer started, listening for audio input...")
	fmt.Println("Please input audio data (press Enter to send, type 'quit' to exit):")

	// Simple command line audio input
	go func() {
		for {
			var input string
			fmt.Print("Audio> ")
			fmt.Scanln(&input)

			if input == "quit" || input == "exit" {
				fmt.Println("👋 Exiting...")
				recognizer.Stop()
				return
			}

			if input == "" {
				continue
			}

			// Simulate audio data (in real applications, this would be actual audio data)
			audioData := generateTestAudio(input)
			if err := recognizer.Write(audioData); err != nil {
				log.Printf("Failed to send audio: %v", err)
			}
		}
	}()

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 Program exiting")
}

// StreamingHandler handles streaming events
type StreamingHandler struct {
	partialChan chan string
	finalChan   chan string
	errorChan   chan error
}

func (h *StreamingHandler) OnSessionCreated(event *asr.SessionCreatedEvent) {
	fmt.Printf("📝 Streaming session created: %s\n", event.Session.ID)
}

func (h *StreamingHandler) OnConversationItemCreated(event *asr.ConversationItemCreatedEvent) {
	fmt.Printf("🎤 Streaming conversation item created: %s\n", event.Item.ID)
}

func (h *StreamingHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
	if len(event.Item.Content) > 0 {
		for _, content := range event.Item.Content {
			if content.Type == "transcript" {
				// Send to final result channel
				select {
				case h.finalChan <- content.Transcript:
				default:
					fmt.Println("⚠️ Result channel full, dropping final result")
				}
			}
		}
	}
}

func (h *StreamingHandler) OnTranscriptionFailed(event *asr.ConversationItemInputAudioTranscriptionFailedEvent) {
		select {
		case h.errorChan <- fmt.Errorf("Transcription failed: %s", event.Error.Message):
		default:
			fmt.Println("⚠️ Error channel full, dropping error message")
		}
}

func (h *StreamingHandler) OnError(event *asr.ErrorEvent) {
	select {
	case h.errorChan <- fmt.Errorf("Connection error: %s", event.Error.Message):
		default:
			fmt.Println("⚠️ Error channel full, dropping error message")
		}
}

func (h *StreamingHandler) OnConnected() {
	fmt.Println("🔗 Streaming connection established")
}

func (h *StreamingHandler) OnDisconnected() {
	fmt.Println("🔌 Streaming connection disconnected")
}

func (h *StreamingHandler) OnPing(event *asr.HeartbeatPingEvent) {
	fmt.Println("💓 Received heartbeat ping")
}

func (h *StreamingHandler) OnPong(event *asr.HeartbeatPongEvent) {
	fmt.Println("💓 Received heartbeat pong")
}

// Empty implementations for other callback methods
func (h *StreamingHandler) OnSessionUpdated(event *asr.SessionUpdatedEvent) {}
func (h *StreamingHandler) OnConversationCreated(event *asr.ConversationCreatedEvent) {}
func (h *StreamingHandler) OnConversationItemDeleted(event *asr.ConversationItemDeletedEvent) {}
func (h *StreamingHandler) OnAudioBufferAppended(event *asr.InputAudioBufferAppendEvent) {}
func (h *StreamingHandler) OnAudioBufferCommitted(event *asr.InputAudioBufferCommittedEvent) {}
func (h *StreamingHandler) OnAudioBufferCleared(event *asr.InputAudioBufferClearedEvent) {}
func (h *StreamingHandler) OnSpeechStarted(event *asr.InputAudioBufferSpeechStartedEvent) {}
func (h *StreamingHandler) OnSpeechStopped(event *asr.InputAudioBufferSpeechStoppedEvent) {}

// displayResults displays streaming recognition results
func (h *StreamingHandler) displayResults() {
	for {
		select {
		case partial := <-h.partialChan:
			fmt.Printf("🔊 Real-time result: %s\n", partial)

		case final := <-h.finalChan:
			fmt.Printf("✅ Final result: %s\n", final)

		case err := <-h.errorChan:
			fmt.Printf("❌ Error: %v\n", err)

		case <-time.After(5 * time.Second):
			// Heartbeat display
			stats := map[string]interface{}{
				"partial_chan_len": len(h.partialChan),
				"final_chan_len":   len(h.finalChan),
				"error_chan_len":   len(h.errorChan),
			}
			fmt.Printf("💓 Status: %+v\n", stats)
		}
	}
}

// generateTestAudio generates test audio data
func generateTestAudio(text string) []byte {
	// Generate audio data with duration based on text length
	duration := float64(len(text)) // 1 second per character
	samples := int(duration * 16000)     // 16kHz
	audioData := make([]byte, samples*2)

	// Generate simple sine wave
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples)
		frequency := 440.0 + (float64(i%10)*100.0) // Varies from 440Hz-1340Hz
		value := int16(16383 * math.Sin(2*math.Pi*frequency*t))

		// Convert to little-endian bytes
		audioData[i*2] = byte(value & 0xFF)
		audioData[i*2+1] = byte((value >> 8) & 0xFF)
	}

	return audioData
}