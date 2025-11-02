package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	asr "gosdk/client"
	"gosdk/pkg/resampler"

	"github.com/go-audio/audio"

	"gosdk/pkg/wav"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cli <audio_dir>")
		return
	}

	audioDir := os.Args[1]
	files, err := os.ReadDir(audioDir)
	if err != nil {
		log.Fatalf("Error reading audio dir: %v", err)
	}

	// Create recognition listener
	listener := &RecognitionListener{
		doneChan: make(chan struct{}),
	}

	// Create configuration
	config := asr.DefaultConfig()
	config.URL = "ws://localhost:8088/v1/realtime"
	config.TranscriptionLanguage = "zh"
	config.InputSampleRate = 16000
	config.InputChannels = 1
	config.Timeout = 30 * time.Second
	config.EnableReconnect = true
	config.MaxReconnectAttempts = 3

	// Create compatibility wrapper
	wrapper := asr.NewCompatibilityWrapper(config)

	// Start recognition session with retry mechanism
	var maxRetries = 3
	var retryDelay = 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		if err = wrapper.Start(); err == nil {
			break
		}
		log.Printf("Connection attempt %d failed: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		log.Fatalf("Failed to start recognizer after %d attempts: %v", maxRetries, err)
	}
	defer func() {
		if err := wrapper.Stop(); err != nil {
			log.Printf("Error stopping recognizer: %v", err)
		}
	}()

	// Process each audio file
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".wav" {
			continue
		}

		// Reset listener state
		listener.doneChan = make(chan struct{})
		filePath := filepath.Join(audioDir, file.Name())
		log.Printf("[ AudioFile ] Processing file: %s\n", filePath)

		// Process audio file
		if err := processAudioFile(wrapper, filePath, file.Name()); err != nil {
			log.Printf("Error processing file %s: %v", filePath, err)
			continue
		}
	}


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-listener.doneChan:
			log.Printf("Recognition completed")
			// Send end marker immediately after successful recognition
			if err := wrapper.Write([]byte{0}); err != nil {
				log.Printf("Error sending end marker: %v", err)
			}
			return

		case <-time.After(30 * time.Second):
			log.Printf("Timeout waiting for recognition result")
			return

		case <-ctx.Done():
			log.Printf("Context canceled, stopping recognition")
			return
		}
	}

}

// processAudioFile processes a single audio file
func processAudioFile(wrapper *asr.CompatibilityWrapper, filePath, fileName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Monitor processing progress
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Still processing file: %s", fileName)
			case <-ctx.Done():
				return
			}
		}
	}()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening WAV file: %v", err)
	}
	defer file.Close()

	wavReader, err := wav.NewReader(file)
	if err != nil {
		return fmt.Errorf("error creating WAV reader: %v", err)
	}

	format := wavReader.GetFormat()
	if err := format.Validate(); err != nil {
		return fmt.Errorf("invalid WAV format: %v", err)
	}

	bytesPerSample := uint32(format.BitsPerSample / 8)
	numSamples := wavReader.GetDataSize() / (bytesPerSample * uint32(format.NumChannels))

	// Read audio PCM data
	pcmData := make([]int16, numSamples)
	if _, err := wavReader.ReadSamples(pcmData); err != nil && err != io.EOF {
		return fmt.Errorf("error reading PCM data: %v", err)
	}

	if format.NumChannels == 2 {
		monoData := make([]int16, numSamples/2)
		for i := 0; i < len(monoData); i++ {
			left := int32(pcmData[i*2])
			right := int32(pcmData[i*2+1])
			monoData[i] = int16((left + right) / 2)
		}
		pcmData = monoData
	}


	var reSamples []int16
	var byteData []byte
	if len(pcmData) > 0 {
		// Resample audio if needed
		if format.SampleRate== 48000 {
			intBuffer := &audio.IntBuffer{
				Data: make([]int, len(pcmData)),
				Format: &audio.Format{
					NumChannels: 1,
					SampleRate:  48000,
				},
				SourceBitDepth: 16,
			}
			for i, s := range pcmData {
				intBuffer.Data[i] = int(s)
			}
			
			var resampled  *audio.IntBuffer
			var err error

			log.Println("[ DEBUG ] Starting 48k->16k resampling...")
			resampled, err = resampler.Resample48kTo16k(intBuffer)
			if err != nil {
				return fmt.Errorf("failed to resample audio: %v", err)
			}
			reSamples = make([]int16, len(resampled.Data))
			for i, v := range resampled.Data {
				reSamples[i] = int16(v)
			}
			// add silence to the end
			silence := make([]int16, 48000) 
			reSamples = append(reSamples, silence...)
			byteData,err = samplesToBytes(reSamples)
			if err != nil {
				return fmt.Errorf("error converting samples to bytes: %v", err)
			}

		}  else {
			// add silence to the end
			silence := make([]int16, 48000) 
			pcmData = append(pcmData, silence...)
			byteData,err = samplesToBytes(pcmData)
			if err != nil {
				return fmt.Errorf("error converting samples to bytes: %v", err)
			}
		}

		// DEBUG Audio data
		// saveAsWAV(byteData,16000)

		// Send audio data in chunks to avoid buffer overflow
		chunkSize := 1024 * 16 // 16KB chunks
		for i := 0; i < len(byteData); i += chunkSize {
			end := i + chunkSize
			if end > len(byteData) {
				end = len(byteData)
			}

			chunk := byteData[i:end]
			if err := wrapper.Write(chunk); err != nil {
				return fmt.Errorf("error sending audio chunk at position %d: %v", i, err)
			}

			// Small delay to allow processing
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Add delay to ensure data processing completes
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		return nil
	}
}