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
	fmt.Println("🎤 ASR SDK - Basic Usage Example")
	fmt.Println("Press Ctrl+C to exit")

	// Create event handler
	handler := &BasicEventHandler{}

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

	fmt.Println("✅ Recognizer started, sending audio data...")

	// Simulate audio data sending
	go func() {
		for i := 0; i < 10; i++ {
			// Simulate 1 second of audio data (16kHz, 16-bit, mono)
			audioData := make([]byte, 16000*2) // 1 second = 16000 samples * 2 bytes
			for j := 0; j < len(audioData); j += 2 {
				// Generate simple test audio (sine wave)
				t := float64(j) / float64(len(audioData))
				value := int16(32767 * math.Sin(2*math.Pi*440*t)) // 440Hz sine wave
				audioData[j] = byte(value & 0xFF)
				audioData[j+1] = byte((value >> 8) & 0xFF)
			}

			fmt.Printf("📡 Sending audio chunk %d/%d (size: %d bytes)\n", i+1, 10, len(audioData))

			if err := recognizer.Write(audioData); err != nil {
				log.Printf("❌ Failed to send audio: %v", err)
				return
			}

			time.Sleep(1 * time.Second)
		}

		fmt.Println("📤 Audio sending completed")
	}()

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 Exiting...")
}

// BasicEventHandler implements basic event handling
type BasicEventHandler struct{}

func (h *BasicEventHandler) OnSessionCreated(event *asr.SessionCreatedEvent) {
	fmt.Printf("✅ Session created: %s (model: %s, modality: %v)\n",
		event.Session.ID, event.Session.Model, event.Session.Modalities)
}

func (h *BasicEventHandler) OnSessionUpdated(event *asr.SessionUpdatedEvent) {
	fmt.Printf("🔄 Session updated: %s\n", event.Session.ID)
}

func (h *BasicEventHandler) OnConversationCreated(event *asr.ConversationCreatedEvent) {
	fmt.Printf("💬 Conversation created: %s\n", event.Conversation.ID)
}

func (h *BasicEventHandler) OnConversationItemCreated(event *asr.ConversationItemCreatedEvent) {
	fmt.Printf("🎤 Conversation item created: %s (type: %s)\n", event.Item.ID, event.Item.Type)
}

func (h *BasicEventHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
	if len(event.Item.Content) > 0 {
		fmt.Printf("✅ Transcription completed: %s\n", event.Item.Content[0].Transcript)
	} else {
		fmt.Println("⚠️ Transcription completed but no content")
	}
}

func (h *BasicEventHandler) OnTranscriptionFailed(event *asr.ConversationItemInputAudioTranscriptionFailedEvent) {
	fmt.Printf("❌ Transcription failed: %s - %s\n", event.Error.Code, event.Error.Message)
}

func (h *BasicEventHandler) OnError(event *asr.ErrorEvent) {
	fmt.Printf("💥 Error event: %s - %s\n", event.Error.Type, event.Error.Message)
}

func (h *BasicEventHandler) OnConnected() {
	fmt.Println("🔗 Connected to server")
}

func (h *BasicEventHandler) OnDisconnected() {
	fmt.Println("🔌 Disconnected from server")
}

func (h *BasicEventHandler) OnPing(event *asr.HeartbeatPingEvent) {
	fmt.Println("💓 Received heartbeat ping")
}

func (h *BasicEventHandler) OnPong(event *asr.HeartbeatPongEvent) {
	fmt.Println("💓 Received heartbeat pong")
}

// Empty implementations for other callback methods
func (h *BasicEventHandler) OnConversationItemDeleted(event *asr.ConversationItemDeletedEvent) {}
func (h *BasicEventHandler) OnAudioBufferAppended(event *asr.InputAudioBufferAppendEvent) {}
func (h *BasicEventHandler) OnAudioBufferCommitted(event *asr.InputAudioBufferCommittedEvent) {}
func (h *BasicEventHandler) OnAudioBufferCleared(event *asr.InputAudioBufferClearedEvent) {}
func (h *BasicEventHandler) OnSpeechStarted(event *asr.InputAudioBufferSpeechStartedEvent) {}
func (h *BasicEventHandler) OnSpeechStopped(event *asr.InputAudioBufferSpeechStoppedEvent) {}