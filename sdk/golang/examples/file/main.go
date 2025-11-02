package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"time"

	asr "gosdk/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <audio_file.wav>")
		os.Exit(1)
	}

	wavFile := os.Args[1]
	fmt.Printf("🎵 Processing audio file: %s\n", wavFile)

	// Create simple callback handler
	handler := &FileHandler{}

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

	// Read and process WAV file
	if err := processWAVFile(wavFile, recognizer); err != nil {
		log.Fatalf("Failed to process audio file: %v", err)
	}

	fmt.Println("✅ File processing completed, waiting for recognition results...")
	time.Sleep(10 * time.Second)
}

// FileHandler handles file processing callbacks
type FileHandler struct {
	fileName string
}

func (h *FileHandler) OnSessionCreated(event *asr.SessionCreatedEvent) {
	fmt.Printf("📝 Processing file: %s - Session created: %s\n", h.fileName, event.Session.ID)
}

func (h *FileHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
	if len(event.Item.Content) > 0 {
		fmt.Printf("✅ File %s transcription completed: %s\n", h.fileName, event.Item.Content[0].Transcript)
	}
}

func (h *FileHandler) OnTranscriptionFailed(event *asr.ConversationItemInputAudioTranscriptionFailedEvent) {
	fmt.Printf("❌ File %s transcription failed: %s\n", h.fileName, event.Error.Message)
}

func (h *FileHandler) OnError(event *asr.ErrorEvent) {
	fmt.Printf("💥 File %s processing error: %s\n", h.fileName, event.Error.Message)
}

func (h *FileHandler) OnConnected() {
	fmt.Println("🔗 Connected to server")
}

func (h *FileHandler) OnDisconnected() {
	fmt.Println("🔌 Disconnected from server")
}

// Empty implementations for other callback methods
func (h *FileHandler) OnSessionUpdated(event *asr.SessionUpdatedEvent) {}
func (h *FileHandler) OnConversationCreated(event *asr.ConversationCreatedEvent) {}
func (h *FileHandler) OnConversationItemCreated(event *asr.ConversationItemCreatedEvent) {}
func (h *FileHandler) OnConversationItemDeleted(event *asr.ConversationItemDeletedEvent) {}
func (h *FileHandler) OnAudioBufferAppended(event *asr.InputAudioBufferAppendEvent) {}
func (h *FileHandler) OnAudioBufferCommitted(event *asr.InputAudioBufferCommittedEvent) {}
func (h *FileHandler) OnAudioBufferCleared(event *asr.InputAudioBufferClearedEvent) {}
func (h *FileHandler) OnSpeechStarted(event *asr.InputAudioBufferSpeechStartedEvent) {}
func (h *FileHandler) OnSpeechStopped(event *asr.InputAudioBufferSpeechStoppedEvent) {}
func (h *FileHandler) OnPing(event *asr.HeartbeatPingEvent) {}
func (h *FileHandler) OnPong(event *asr.HeartbeatPongEvent) {}

// processWAVFile reads and sends WAV file
func processWAVFile(filename string, recognizer *asr.Recognizer) error {
	// Read WAV file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Check WAV header
	var wavHeader struct {
		RIFF       [4]byte
		FileSize   uint32
		WAVE       [4]byte
		Fmt        [4]byte
		Subchunk1Size uint32
		AudioFormat   uint16
		NumChannels  uint16
		SampleRate   uint32
		ByteRate    uint32
		BlockAlign   uint16
		BitsPerSample uint16
	}

	if err := binary.Read(file, binary.LittleEndian, &wavHeader); err != nil {
		return fmt.Errorf("failed to read WAV header: %v", err)
	}

	// Check if it's a valid WAV file
	if string(wavHeader.RIFF[:]) != "RIFF" ||
		string(wavHeader.WAVE[:]) != "WAVE" ||
		wavHeader.AudioFormat != 1 || // PCM
		wavHeader.BitsPerSample != 16 {
		return fmt.Errorf("unsupported WAV format, requires 16-bit PCM")
	}

	fmt.Printf("📊 WAV Info: SampleRate=%dHz, Channels=%d, DataSize=%d bytes\n",
		wavHeader.SampleRate, wavHeader.NumChannels, wavHeader.FileSize-36)

	// Calculate audio data start position
	dataStartPos := 12 + 8 + wavHeader.Subchunk1Size
	if _, err := file.Seek(int64(dataStartPos), 0); err != nil {
		return fmt.Errorf("failed to seek to audio data: %v", err)
	}

	// Read audio data
	audioData := make([]byte, wavHeader.FileSize-36)
	if _, err := file.Read(audioData); err != nil {
		return fmt.Errorf("failed to read audio data: %v", err)
	}

	fmt.Printf("📡 Read audio data: %d bytes\n", len(audioData))

	// Send audio data in chunks
	chunkSize := 1024 // 1KB chunks
	totalChunks := (len(audioData) + chunkSize - 1) / chunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		chunk := audioData[start:end]

		fmt.Printf("📤 Sending audio chunk %d/%d (size: %d bytes)\n", i+1, totalChunks, len(chunk))

		// Send audio chunk
		if err := recognizer.Write(chunk); err != nil {
			return fmt.Errorf("failed to send audio chunk %d: %v", i+1, err)
		}

		// Commit the last chunk
		if end == len(audioData) {
			fmt.Println("📤 Committing audio buffer")
			if err := recognizer.CommitAudio(); err != nil {
				return fmt.Errorf("failed to commit audio buffer: %v", err)
			}
		}

		// Brief delay to avoid sending too fast
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}