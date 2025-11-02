# ASR SDK Documentation

## Overview
This SDK provides speech recognition (ASR) capabilities with support for:
- Real-time audio streaming (WebSocket)
- WAV file processing (processAudioFile for testing)
- Recognition result callbacks
- Automatic resampling (48kHz→16kHz)
- Multi-threading and heartbeat mechanism

## Installation
```bash
go get asr
```

## Quick Start
```go
package main

import (
	"log"

	asr "asr/client"
)

func main() {
	// 1. Create recognition listener
	listener := &RecognitionListener{}

	// 2. Initialize recognizer
	recognizer := asr.NewRecognizer(listener)
	recognizer.AudioFormat = asr.AudioFormatPCM
	recognizer.SampleRate = 16000  // 16kHz
	recognizer.Language = "zh-CN"  // Chinese recognition

	// 3. Start recognition session
	if err := recognizer.Start(); err != nil {
		log.Fatal(err)
	}
	defer recognizer.Stop()

	// 4. Real-time audio processing example
	// Get data from microphone or other audio source
	audioChunk := getAudioData() // PCM data in []byte format
	if err := recognizer.Write(audioChunk); err != nil {
		log.Fatal(err)
	}
}
```

## Real-time Audio Streaming
```go
// Continuously write audio data
func streamAudio(recognizer *asr.Recognizer) {
	for {
		select {
		case data := <-audioSource.Chan():
			if err := recognizer.Write(data); err != nil {
				// 50ms timeout will be skipped automatically
				if err != asr.ErrWriteTimeout {
					log.Printf("Write failed: %v", err)
					return
				}
			}
		case <-doneChan:
			return
		}
	}
}
```

## Configuration
| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| AudioFormat | int | Audio format (1=PCM) | 1 |
| SampleRate | int | Sampling rate (Hz) | 16000 |
| Language | string | Recognition language | "zh-CN" |
| MaxRetries | int | Maximum retry attempts | 3 |
| RetryDelay | time.Duration | Retry interval | 2s |

## Event Callbacks
Implement `RecognitionListener` interface to handle recognition events:
```go
type RecognitionListener struct{}

func (l *RecognitionListener) OnRecognitionStart(resp *asr.RecognitionResponse) {
	// Recognition started
}

func (l *RecognitionListener) OnSentenceBegin(resp *asr.RecognitionResponse) {
	// Sentence begin
}

func (l *RecognitionListener) OnRecognitionResultChange(resp *asr.RecognitionResponse) {
	// Partial result update
	log.Printf("Partial: %s", resp.Result.Text)
}

func (l *RecognitionListener) OnSentenceEnd(resp *asr.RecognitionResponse) {
	// Sentence end
	log.Printf("Final: %s", resp.Result.Text)
}

func (l *RecognitionListener) OnRecognitionComplete(resp *asr.RecognitionResponse) {
	// Recognition completed
}

func (l *RecognitionListener) OnFail(resp *asr.RecognitionResponse, err error) {
	// Recognition failed
}
```

## Audio Requirements
- Real-time stream format: Raw PCM data ([]byte)
- File test format: WAV(PCM)
- Sample rate: 16kHz or 48kHz (auto-resampled)
- Bit depth: 16bit
- Channels: Mono or stereo (auto-converted)

## Notes
1. Write() method is designed for real-time streaming with 50ms timeout
2. processAudioFile is only for local file testing
3. Stable network connection required for real-time recognition
4. Control write frequency for high-throughput scenarios
5. 48kHz audio will be automatically resampled to 16kHz