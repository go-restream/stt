package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	llm "github.com/go-restream/stt/llm"
	"github.com/go-restream/stt/pkg/logger"
	"github.com/go-restream/stt/pkg/resampler"
	"github.com/go-restream/stt/pkg/wav"
	vad "github.com/go-restream/stt/vad"
	"sync"
	"time"

	"github.com/go-restream/stt/config"

	"github.com/go-audio/audio"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var CHANNELS = 1
var BITS_PER_SAMPLE = 16
var SAMPLE_RATE = 48000

// safeUint16 safely converts int to uint16 with overflow check
func safeUint16(val int) uint16 {
	if val < 0 {
		return 0
	}
	if val > 65535 {
		return 65535
	}
	return uint16(val)
}

// safeUint32 safely converts int to uint32 with overflow check
func safeUint32(val int) uint32 {
	if val < 0 {
		return 0
	}
	if val > 4294967295 {
		return 4294967295
	}
	return uint32(val)
}

type SpeechRecognizer struct {
	conn            *websocket.Conn
	audioChan       chan int16        // Audio data channel with buffer (20 seconds capacity)
	isSpeaking      bool              // VAD speaking detection flag
	stopChan        chan struct{}     // Stop signal channel
	wavFormat       wav.WAVFormat     // WAV format configuration
	consumerRunning bool              // Consumer thread running status
	consumerStop    chan struct{}     // Consumer thread stop signal
	consumerMu      sync.Mutex        // Consumer thread state mutex
	samplesConsumed int               // Number of samples consumed
	vad 			bool 			  // VAD enabled flag
	vadDetector     *vad.VADDetector  // VAD detector instance
	sampleBuffer    []float32         // Sample buffer for batch processing
	voiceID 		string			  // Voice session ID
	savePath 		string	   	      // Save path
}

func (sr *SpeechRecognizer) sendEvent(event map[string]interface{}) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event error: %v", err)
	}
	return sr.conn.WriteMessage(websocket.TextMessage, jsonData)
}

// NewSpeechRecognizer creates and initializes a speech recognizer
func NewSpeechRecognizer(conn *websocket.Conn, configPath string) *SpeechRecognizer {
	var err error
	AppConfig, err := config.LoadConfig(configPath)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_audio_rcger",
			"action":    "load_config_failed",
			"error":     err,
		}).Fatal("load config failed")
	}
	
	if AppConfig.Audio.SampleRate > 0 {
		SAMPLE_RATE = AppConfig.Audio.SampleRate
	}

	if AppConfig.Audio.Channels > 0 {
		CHANNELS = AppConfig.Audio.Channels
	}

	if AppConfig.Audio.BitDepth > 0 {
		BITS_PER_SAMPLE = AppConfig.Audio.BitDepth
	}

	if AppConfig.ASR.APIKey != "" {
	   llm.SetAsrApiKey(AppConfig.ASR.APIKey)
	}

	if AppConfig.ASR.BaseURL != "" {
	   llm.SetAsrBaseURL(AppConfig.ASR.BaseURL)
	}

	if AppConfig.ASR.Model != "" {
	   llm.SetAsrModel(AppConfig.ASR.Model)
	}

	dir:= "."
    if AppConfig.Audio.SaveDir != "" {
		if err := os.MkdirAll(AppConfig.Audio.SaveDir, 0750); err != nil {
			logger.WithFields(logrus.Fields{
				"component": "eng_audio_rcger",
				"action":    "create_save_dir_failed",
				"saveDir":   AppConfig.Audio.SaveDir,
				"error":     err,
			}).Fatal("Failed to create save directory")
		}
		dir = AppConfig.Audio.SaveDir
	}

	// Channel Capacity (sampleRate * 1channel * 20s)
	chanCapacity := SAMPLE_RATE * 1 * 20
	return &SpeechRecognizer{
		conn:         conn,
		audioChan:    make(chan int16, chanCapacity),
		stopChan:     make(chan struct{}),
		consumerStop: make(chan struct{}),
		vad:          AppConfig.Vad.Enable ,
		vadDetector:  vad.NewVADDetector(AppConfig),
		wavFormat: wav.WAVFormat{
			AudioFormat:   1, // PCM
			NumChannels:   safeUint16(CHANNELS),
			SampleRate:    safeUint32(SAMPLE_RATE),
			ByteRate:      safeUint32(SAMPLE_RATE) * safeUint32(CHANNELS) * safeUint32(BITS_PER_SAMPLE) / 8,
			BlockAlign:    safeUint16(CHANNELS) * safeUint16(BITS_PER_SAMPLE) / 8,
			BitsPerSample: safeUint16(BITS_PER_SAMPLE),
		},
		savePath: dir,
	}
}

func (sr *SpeechRecognizer) Stream(audioData []byte) error {
	if len(audioData) == 0 {
		logger.WithFields(logrus.Fields{
			"component": "eng_audio_rcger",
			"action":    "empty_audio_data",
		}).Warn("Warning: empty audio data received")
		return nil
	}
	if len(audioData)%2 != 0 {
		audioData = append(audioData, 0)
		logger.WithFields(logrus.Fields{
			"component":    "recognizer",
			"action":       "fix_odd_length_data",
			"originalSize": len(audioData) - 1,
			"newSize":      len(audioData),
		}).Warn("Warning: fixed odd-length audio data by padding")
	}

	samples := make([]int16, len(audioData)/2)
	for i := range samples {
		if len(audioData) < 2*(i+1) {
			return fmt.Errorf("audio data truncated")
		}
		// Safely convert uint16 to int16 using proper bit manipulation
		value := binary.LittleEndian.Uint16(audioData[i*2:])
		// Use bit manipulation to avoid overflow - convert unsigned to signed 16-bit
		samples[i] = int16(value) // This is safe in Go - it wraps around as expected for 16-bit audio
	}

	intBuffer := &audio.IntBuffer{
		Data: make([]int, len(samples)),
		Format: &audio.Format{
			NumChannels: int(sr.wavFormat.NumChannels),
			SampleRate:  int(sr.wavFormat.SampleRate),
		},
		SourceBitDepth: BITS_PER_SAMPLE,
	}
	for i, s := range samples {
		intBuffer.Data[i] = int(s)
	}
	
	var resampled  *audio.IntBuffer
	var err error
	if intBuffer.Format.SampleRate == 48000	 && intBuffer.Format.NumChannels == 1 {
	    resampled, err = resampler.Resample48kTo16k(intBuffer)
		if err != nil {
			return fmt.Errorf("failed to resample audio: %v", err)
		}
		samples = make([]int16, len(resampled.Data))
		for i, v := range resampled.Data {
			// Prevent overflow with proper clipping
			if v > 32767 {
				samples[i] = 32767  // Clamp to max int16 value
			} else if v < -32768 {
				samples[i] = -32768 // Clamp to min int16 value
			} else {
				samples[i] = int16(v)
			}
		}
	}

	for _, sample := range samples {
		select {
		case sr.audioChan <- sample:
		default:
			<-sr.audioChan
			sr.audioChan <- sample
			logger.WithFields(logrus.Fields{
				"component": "sys_debug_tool",
				"action":    "drop_old_data",
				"bufferSize": len(sr.audioChan),
			}).Debug("Drop old data")
		}
	}

	return nil
}



func (sr *SpeechRecognizer) StartVADConsumer() {
	if !sr.consumerRunning {
		sr.consumerRunning = true
		go sr.consumerVADLoop()
		logger.WithFields(logrus.Fields{
			"component": "eng_stt_audio_sys",
			"action":    "start_vad_consumer_thread",
		}).Info("Starting VAD audio data consumer thread")
	}
}

func (sr *SpeechRecognizer) consumerVADLoop() {
	for {
			select {
			case <-sr.consumerStop:
				return
			case sample := <-sr.audioChan:
				floatSample := float32(sample) / 32768.0
				sr.sampleBuffer = append(sr.sampleBuffer, floatSample)

				if len(sr.sampleBuffer) >= 160 {
					startTime := time.Now()

					segment := sr.vadDetector.ProcessSamples(sr.sampleBuffer)
					sr.sampleBuffer = sr.sampleBuffer[:0]

					if segment != nil {
						sr.isSpeaking = true
						samples := make([]int16, len(segment.Samples))
						for i, s := range segment.Samples {
							samples[i] = int16(s * 32768.0)
						}

						go func(samples []int16) {
							if err := sr.sendToASREngine(samples); err != nil {
								logger.WithFields(logrus.Fields{
									"component": "eng_stt_audio_sys",
									"action":    "process_speech_segment",
									"error":     err,
								}).Error("Error processing speech segment")
							}
							duration := time.Since(startTime).Seconds()
							logger.WithFields(logrus.Fields{
								"component":       "stt_engine",
								"action":          "asr_processing_time",
								"processingTime":  duration,
								"sampleCount":     len(samples),
							}).Info("ASR engine processing completed")
						}(samples)
					} else {
						sr.isSpeaking = false
					}
				}
			}
		}
}


func (sr *SpeechRecognizer) StartConsumer() {
	sr.consumerMu.Lock()
	defer sr.consumerMu.Unlock()

	if !sr.consumerRunning {
		sr.consumerRunning = true
		go sr.consumerLoop()
		logger.WithFields(logrus.Fields{
			"component": "eng_stt_audio_sys",
			"action":    "consumer_started",
		}).Info("Starting audio data consumer thread")
	}
}
func (sr *SpeechRecognizer) consumerLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	targetSamples := 16000 * 1 * 2
	logger.WithFields(logrus.Fields{
		"component":      "consumer",
		"action":         "target_samples_config",
		"targetSamples":  targetSamples,
		"duration":       "2 seconds audio",
	}).Info("Target sample count configured")

	for {
		select {
		case <-sr.consumerStop:
			return
		case <-ticker.C:
			sr.consumerMu.Lock()
			speaking := sr.isSpeaking
			if speaking && len(sr.audioChan) >= targetSamples {
				samples := make([]int16, targetSamples)
				for i := 0; i < targetSamples; i++ {
					samples[i] = <-sr.audioChan
				}
				logger.WithFields(logrus.Fields{
					"component":    "consumer",
					"action":       "samples_collected",
					"sampleCount":  len(samples),
					"duration":     "2 seconds audio",
				}).Info("Collected samples, starting processing")
				go func(data []int16) {
					if err := sr.sendToASREngine(data); err != nil {
						logger.WithFields(logrus.Fields{
							"component":   "consumer",
							"action":      "asr_error",
							"sampleCount": len(data),
							"error":       err,
						}).Error("Consumer thread ASR error")
					}
				}(samples)
			} else if !speaking {
				sr.samplesConsumed = 0
				for len(sr.audioChan) > 0 {
					<-sr.audioChan
				}
			}
			sr.consumerMu.Unlock()
		}
	}
}
func (sr *SpeechRecognizer) StopConsumer() {
	sr.consumerMu.Lock()
	defer sr.consumerMu.Unlock()

	if sr.consumerRunning {
		if len(sr.audioChan) > 0 {
			logger.WithFields(logrus.Fields{
				"component":    "consumer",
				"action":       "processing_remaining_samples",
				"remainingCount": len(sr.audioChan),
			}).Info("Starting to process remaining samples")

			remainingSamples := make([]int16, 0, len(sr.audioChan))
			for len(sr.audioChan) > 0 {
				remainingSamples = append(remainingSamples, <-sr.audioChan)
			}

			if len(remainingSamples) > 0 {
				logger.WithFields(logrus.Fields{
					"component":   "consumer",
					"action":      "sending_final_samples",
					"sampleCount": len(remainingSamples),
				}).Info("Sending final samples to ASR engine")
				if err := sr.sendToASREngine(remainingSamples); err != nil {
					logger.WithFields(logrus.Fields{
						"component":   "consumer",
						"action":      "process_remaining_error",
						"sampleCount": len(remainingSamples),
						"error":       err,
					}).Error("Error processing remaining data")
				}
			}
		}

		close(sr.consumerStop)
		sr.consumerRunning = false
		sr.consumerStop = make(chan struct{})
		logger.WithFields(logrus.Fields{
			"component": "eng_stt_audio_sys",
			"action":    "consumer_stopped",
		}).Info("Audio data consumer thread stopped")
	}
}

// sendToASREngine calls the speech recognition engine
func (sr *SpeechRecognizer) sendToASREngine(audioData []int16) error {
	sr.consumerMu.Lock()
	defer sr.consumerMu.Unlock()

	startEvent := map[string]interface{}{
		"code":    0,
		"message": "Recognition started",
		"voiceID": sr.voiceID,
	}
	if err := sr.sendEvent(startEvent); err != nil {
		return fmt.Errorf("send start event failed: %v", err)
	}

	wavData, err := sr.saveAsWAV(audioData)
	if err != nil {
		errorEvent := map[string]interface{}{
			"code":    -1,
			"message": "failed to encode WAV",
			"voiceID": "",
		}
		_ = sr.sendEvent(errorEvent)
		return fmt.Errorf("failed to encode WAV: %v", err)
	}

	text, err := llm.CallOpenaiAPI(wavData)
	if err != nil {
		errorEvent := map[string]interface{}{
			"code":    -1,
			"message": "ASR processing failed",
			"voiceID": "",
			"error":   err.Error(),
		}
		if sendErr := sr.sendEvent(errorEvent); sendErr != nil {
			logger.WithFields(logrus.Fields{
				"component":   "recognizer",
				"action":      "send_error_event_failed",
				"voiceID":     sr.voiceID,
				"error":       sendErr,
			}).Error("Failed to send error event")
		}
		return fmt.Errorf("ASR processing failed: %v", err)
	}

	logger.WithFields(logrus.Fields{
		"component": "svc_stt_audio_main",
		"action":    "recognition_result",
		"voiceID":   sr.voiceID,
		"text":      text,
	}).Info("🚀 STT speech text result")

	completeEvent := map[string]interface{}{
		"code":    0,
		"message": "Recognition complete",
		"voiceID": "",
		"result": map[string]interface{}{
			"text":  text,
			"final": true,
		},
	}
	return sr.sendEvent(completeEvent)
}

func (sr *SpeechRecognizer) saveAsWAV(audioData []int16) ([]byte, error) {
    tmpfile, err := os.CreateTemp(sr.savePath, "audio_*.wav")
    if err != nil {
        return nil, fmt.Errorf("failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())
    defer tmpfile.Close()

    wavFormat := wav.WAVFormat{
        AudioFormat:   1,
        BitsPerSample: 16,
        BlockAlign:    2,
        ByteRate:      16000 * 2,
        NumChannels:   1,
        SampleRate:    16000,
    }

    writer, err := wav.NewWriter(tmpfile, wavFormat)
    if err != nil {
        return nil, fmt.Errorf("create WAV writer failed: %v", err)
    }

    if err := writer.WriteSamples(audioData); err != nil {
        return nil, fmt.Errorf("write samples failed: %v", err)
    }

    if err := writer.Close(); err != nil {
        return nil, fmt.Errorf("close WAV writer failed: %v", err)
    }

    if err := tmpfile.Sync(); err != nil {
        return nil, fmt.Errorf("failed to sync file: %v", err)
    }

    tmpfile.Seek(0, 0)
    return io.ReadAll(tmpfile)
}






