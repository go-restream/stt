package asr

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// AudioBuffer manages audio data for streaming
type AudioBuffer struct {
	buffer      []byte
	bufferMutex sync.RWMutex
	maxSize     int
	sampleRate  int
	channels    int
}

// NewAudioBuffer creates a new audio buffer
func NewAudioBuffer(maxSize int, sampleRate, channels int) *AudioBuffer {
	return &AudioBuffer{
		buffer:     make([]byte, 0, maxSize),
		maxSize:    maxSize,
		sampleRate: sampleRate,
		channels:   channels,
	}
}

// Write adds audio data to the buffer
func (ab *AudioBuffer) Write(data []byte) error {
	ab.bufferMutex.Lock()
	defer ab.bufferMutex.Unlock()

	currentSize := len(ab.buffer)
	newSize := currentSize + len(data)

	if newSize > ab.maxSize {
		return ErrAudioBufferFull
	}

	// Ensure buffer capacity
	if cap(ab.buffer) < newSize {
		newBuffer := make([]byte, currentSize, newSize)
		copy(newBuffer, ab.buffer)
		ab.buffer = newBuffer
	}

	// Append new data
	ab.buffer = append(ab.buffer, data...)
	return nil
}

// ReadAndClear reads all data from buffer and clears it
func (ab *AudioBuffer) ReadAndClear() []byte {
	ab.bufferMutex.Lock()
	defer ab.bufferMutex.Unlock()

	if len(ab.buffer) == 0 {
		return nil
	}

	data := make([]byte, len(ab.buffer))
	copy(data, ab.buffer)
	ab.buffer = ab.buffer[:0] // Clear buffer
	return data
}

// Size returns the current buffer size
func (ab *AudioBuffer) Size() int {
	ab.bufferMutex.RLock()
	defer ab.bufferMutex.RUnlock()
	return len(ab.buffer)
}

// IsFull checks if the buffer is full
func (ab *AudioBuffer) IsFull() bool {
	ab.bufferMutex.RLock()
	defer ab.bufferMutex.RUnlock()
	return len(ab.buffer) >= ab.maxSize
}

// Clear empties the buffer
func (ab *AudioBuffer) Clear() {
	ab.bufferMutex.Lock()
	defer ab.bufferMutex.Unlock()
	ab.buffer = ab.buffer[:0]
}

// GetDuration returns the duration of audio in buffer
func (ab *AudioBuffer) GetDuration() time.Duration {
	ab.bufferMutex.RLock()
	defer ab.bufferMutex.RUnlock()

	bytesPerSample := 2 // 16-bit PCM
	samples := len(ab.buffer) / bytesPerSample
	duration := time.Duration(samples) * time.Second / time.Duration(ab.sampleRate)

	return duration
}

// PCM16ToBase64 converts 16-bit PCM audio data to Base64
func PCM16ToBase64(audioData []int16) string {
	if len(audioData) == 0 {
		return ""
	}

	// Convert int16 slice to byte slice
	byteData := make([]byte, len(audioData)*2)
	for i, sample := range audioData {
		binary.LittleEndian.PutUint16(byteData[i*2:], uint16(sample))
	}

	return base64.StdEncoding.EncodeToString(byteData)
}

// Base64ToPCM16 converts Base64 audio data to 16-bit PCM
func Base64ToPCM16(base64Audio string) ([]int16, error) {
	if base64Audio == "" {
		return nil, nil
	}

	// Remove data URL prefix if present
	if len(base64Audio) > 22 && base64Audio[:22] == "data:audio/wav;base64," {
		base64Audio = base64Audio[22:]
	}

	// Decode Base64
	byteData, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64: %v", err)
	}

	// Ensure even length for 16-bit samples
	if len(byteData)%2 != 0 {
		return nil, fmt.Errorf("invalid PCM data length: not divisible by 2")
	}

	// Convert byte slice to int16 slice
	samples := make([]int16, len(byteData)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(byteData[i*2:]))
	}

	return samples, nil
}

// AudioUtils provides utility functions for audio processing
type AudioUtils struct {
	sampleRate   int
	channels    int
	bufferPool  sync.Pool
}

// NewAudioUtils creates a new audio utility instance
func NewAudioUtils(sampleRate, channels int) *AudioUtils {
	return &AudioUtils{
		sampleRate: sampleRate,
		channels:   channels,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024*10) // 10KB buffer
			},
		},
	}
}

// ResampleAudio resamples audio from one sample rate to another
// This is a simplified linear interpolation resampling
// For production use, consider using a proper resampling library
func (au *AudioUtils) ResampleAudio(inputSamples []int16, inputRate, outputRate int) ([]int16, error) {
	if inputRate == outputRate {
		return inputSamples, nil
	}

	if len(inputSamples) == 0 {
		return nil, nil
	}

	log.Printf("[🎛 Audio] Resampling from %dHz to %dHz (samples: %d)", inputRate, outputRate, len(inputSamples))

	ratio := float64(outputRate) / float64(inputRate)
	outputLength := int(math.Ceil(float64(len(inputSamples)) * ratio))
	outputSamples := make([]int16, outputLength)

	for i := 0; i < outputLength; i++ {
		inputIndex := float64(i) / ratio
		if inputIndex >= float64(len(inputSamples)-1) {
			outputSamples[i] = inputSamples[len(inputSamples)-1]
		} else {
			index := int(inputIndex)
			fraction := inputIndex - float64(index)

			if index+1 < len(inputSamples) {
				// Linear interpolation
				sample1 := float64(inputSamples[index])
				sample2 := float64(inputSamples[index+1])
				outputSamples[i] = int16(sample1 + fraction*(sample2-sample1))
			} else {
				outputSamples[i] = inputSamples[index]
			}
		}
	}

	log.Printf("[✅ Audio] Resampling completed: %d -> %d samples", len(inputSamples), len(outputSamples))
	return outputSamples, nil
}

// ValidateAudioFormat checks if audio format is supported
func (au *AudioUtils) ValidateAudioFormat(sampleRate, channels int) error {
	// Check sample rate
	if sampleRate != 16000 && sampleRate != 48000 {
		return ErrInvalidSampleRate
	}

	// Check channels
	if channels != 1 && channels != 2 {
		return ErrInvalidChannels
	}

	return nil
}

// ConvertToMono converts stereo audio to mono by averaging channels
func (au *AudioUtils) ConvertToMono(inputSamples []int16, channels int) []int16 {
	if channels == 1 {
		return inputSamples // Already mono
	}

	if channels != 2 {
		return inputSamples // Only support stereo to mono conversion
	}

	monoSamples := make([]int16, len(inputSamples)/channels)
	for i := 0; i < len(monoSamples); i++ {
		// Average left and right channels
		left := inputSamples[i*2]
		right := inputSamples[i*2+1]
		monoSamples[i] = (left + right) / 2
	}

	return monoSamples
}

// GetAudioBufferFromPool gets a buffer from the pool
func (au *AudioUtils) GetAudioBufferFromPool() []byte {
	return au.bufferPool.Get().([]byte)
}

// PutAudioBufferToPool returns a buffer to the pool
func (au *AudioUtils) PutAudioBufferToPool(buffer []byte) {
	if cap(buffer) == 1024*10 { // Only pool buffers of expected size
		// Reset buffer before returning to pool
		buffer = buffer[:0]
		au.bufferPool.Put(&buffer)
	}
}

// CalculateAudioDuration calculates duration of PCM audio data
func CalculateAudioDuration(sampleCount, sampleRate int) time.Duration {
	samplesPerSecond := float64(sampleRate)
	duration := float64(sampleCount) / samplesPerSecond
	return time.Duration(duration * float64(time.Second))
}

// DetectSilence detects silence in audio data using simple energy threshold
func DetectSilence(audioData []int16, threshold int16) bool {
	if len(audioData) == 0 {
		return true
	}

	// Calculate RMS energy
	var sum int64
	for _, sample := range audioData {
		sum += int64(sample * sample)
	}

	rms := int16(math.Sqrt(float64(sum) / float64(len(audioData))))
	return abs(rms) < threshold
}

// abs returns absolute value of int16
func abs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}

// NormalizeAudio normalizes audio data to prevent clipping
func NormalizeAudio(audioData []int16) []int16 {
	if len(audioData) == 0 {
		return audioData
	}

	// Find peak value
	var peak int16
	for _, sample := range audioData {
		if sample > peak {
			peak = sample
		} else if -sample > peak {
			peak = -sample
		}
	}

	if peak == 0 {
		return audioData
	}

	// Calculate normalization factor
	targetPeak := int16(math.Round(float64(math.MaxInt16) * 0.8)) // Target 80% of max
	factor := float64(targetPeak) / float64(peak)

	// Apply normalization
	normalized := make([]int16, len(audioData))
	for i, sample := range audioData {
		normalized[i] = int16(float64(sample) * factor)
	}

	return normalized
}

// ApplyWindow applies a window function to audio data to reduce spectral leakage
func ApplyWindow(audioData []int16, windowType string) []int16 {
	if len(audioData) == 0 {
		return audioData
	}

	windowed := make([]int16, len(audioData))
	n := len(audioData)

	switch windowType {
	case "hann":
		for i := 0; i < n; i++ {
			window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
			windowed[i] = int16(float64(audioData[i]) * window)
		}
	case "hamming":
		for i := 0; i < n; i++ {
			window := 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
			windowed[i] = int16(float64(audioData[i]) * window)
		}
	default:
		// No windowing
		copy(windowed, audioData)
	}

	return windowed
}