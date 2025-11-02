package service

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-restream/stt/pkg/logger"
	"github.com/go-restream/stt/pkg/resampler"
	"github.com/go-restream/stt/pkg/wav"

	"github.com/go-audio/audio"
)

// AudioUtils provides utilities for Base64 audio encoding/decoding and processing
type AudioUtils struct{}

// safeUint32Audio safely converts int to uint32 with overflow check for audio utilities
func safeUint32Audio(val int) uint32 {
	if val < 0 {
		return 0
	}
	if val > 4294967295 {
		return 4294967295
	}
	return uint32(val)
}

// validateFilePath safely validates file paths to prevent path traversal attacks
func validateFilePath(filePath, allowedBaseDir string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Clean the path to resolve any ".." or "." elements
	cleanPath := filepath.Clean(filePath)

	// If allowedBaseDir is provided, ensure the path is within it
	if allowedBaseDir != "" {
		absBaseDir, err := filepath.Abs(allowedBaseDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve base directory: %v", err)
		}

		// Join base dir with the relative path and clean it
		joinedPath := filepath.Join(absBaseDir, cleanPath)
		finalPath := filepath.Clean(joinedPath)

		// Ensure the final path is still within the base directory
		if !strings.HasPrefix(finalPath, absBaseDir) {
			return "", fmt.Errorf("path traversal detected: %s attempts to access outside allowed directory %s", filePath, allowedBaseDir)
		}

		return finalPath, nil
	}

	// If no base directory specified, just return the cleaned path
	return cleanPath, nil
}

// NewAudioUtils creates a new audio utilities instance
func NewAudioUtils() *AudioUtils {
	return &AudioUtils{}
}

// DecodeBase64Audio decodes Base64 audio data to PCM bytes
// Supports both raw Base64 and data URI formats
func (au *AudioUtils) DecodeBase64Audio(base64Audio string) ([]byte, error) {
	base64Audio = strings.TrimPrefix(base64Audio, "data:audio/wav;base64,")
	data, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64 audio: %v", err)
	}
	return data, nil
}

// EncodeAudioToBase64 encodes PCM audio data to Base64
func (au *AudioUtils) EncodeAudioToBase64(audioData []byte) string {
	return base64.StdEncoding.EncodeToString(audioData)
}

// EncodeAudioToBase64DataURI encodes PCM audio data to Base64 with data URI prefix
func (au *AudioUtils) EncodeAudioToBase64DataURI(audioData []byte) string {
	base64Data := base64.StdEncoding.EncodeToString(audioData)
	return "data:audio/wav;base64," + base64Data
}

// ConvertBase64ToPCM16 converts Base64 audio to 16-bit PCM samples
func (au *AudioUtils) ConvertBase64ToPCM16(base64Audio string) ([]int16, error) {
	pcmBytes, err := au.DecodeBase64Audio(base64Audio)
	if err != nil {
		return nil, err
	}

	// Convert bytes to int16 samples
	if len(pcmBytes)%2 != 0 {
		return nil, fmt.Errorf("audio data length must be even for 16-bit PCM")
	}

	samples := make([]int16, len(pcmBytes)/2)
	for i := range samples {
		// Safely convert uint16 to int16 using proper bit manipulation
		value := binary.LittleEndian.Uint16(pcmBytes[i*2:])
		// Use bit manipulation to avoid overflow - convert unsigned to signed 16-bit
		samples[i] = int16(value) // This is safe in Go - it wraps around as expected for 16-bit audio
	}

	return samples, nil
}

// ConvertPCM16ToBase64 converts 16-bit PCM samples to Base64
func (au *AudioUtils) ConvertPCM16ToBase64(samples []int16) string {
	pcmBytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		// Safe conversion from int16 to uint16 for binary encoding
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(sample)) // This is safe for audio data
	}
	return au.EncodeAudioToBase64(pcmBytes)
}

// ProcessBase64Audio processes Base64 audio data with resampling if needed
func (au *AudioUtils) ProcessBase64Audio(base64Audio string, sourceSampleRate int, targetSampleRate int) ([]int16, error) {
	// Decode Base64 to PCM bytes
	pcmBytes, err := au.DecodeBase64Audio(base64Audio)
	if err != nil {
		return nil, err
	}

	// Convert bytes to int16 samples
	if len(pcmBytes)%2 != 0 {
		return nil, fmt.Errorf("audio data length must be even for 16-bit PCM")
	}

	samples := make([]int16, len(pcmBytes)/2)
	for i := range samples {
		// Safely convert uint16 to int16 using proper bit manipulation
		value := binary.LittleEndian.Uint16(pcmBytes[i*2:])
		// Use bit manipulation to avoid overflow - convert unsigned to signed 16-bit
		samples[i] = int16(value) // This is safe in Go - it wraps around as expected for 16-bit audio
	}

	// Resample if needed
	if sourceSampleRate != targetSampleRate {
		resampledSamples, err := au.ResampleAudio(samples, sourceSampleRate, targetSampleRate)
		if err != nil {
			return nil, fmt.Errorf("failed to resample audio: %v", err)
		}
		return resampledSamples, nil
	}

	return samples, nil
}

// ResampleAudio resamples audio from source to target sample rate
func (au *AudioUtils) ResampleAudio(samples []int16, sourceSampleRate int, targetSampleRate int) ([]int16, error) {
	// Create input buffer
	intBuffer := &audio.IntBuffer{
		Data: make([]int, len(samples)),
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sourceSampleRate,
		},
		SourceBitDepth: 16,
	}
	for i, s := range samples {
		intBuffer.Data[i] = int(s)
	}

	var resampled *audio.IntBuffer
	var err error

	// Handle specific resampling cases
	if sourceSampleRate == 48000 && targetSampleRate == 16000 {
		resampled, err = resampler.Resample48kTo16k(intBuffer)
	} else {
		// Generic resampling (fallback)
		resampled, err = resampler.Resample(intBuffer, targetSampleRate)
	}

	if err != nil {
		return nil, err
	}

	// Convert back to int16 with overflow protection
	resampledSamples := make([]int16, len(resampled.Data))
	for i, v := range resampled.Data {
		// Prevent overflow with proper clipping
		if v > 32767 {
			resampledSamples[i] = 32767  // Clamp to max int16 value
		} else if v < -32768 {
			resampledSamples[i] = -32768 // Clamp to min int16 value
		} else {
			resampledSamples[i] = int16(v)
		}
	}

	return resampledSamples, nil
}

// ValidateAudioFormat validates audio format parameters
func (au *AudioUtils) ValidateAudioFormat(sampleRate int, channels int, bitDepth int) error {
	if sampleRate <= 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	if channels <= 0 {
		return fmt.Errorf("channels must be positive")
	}
	if bitDepth != 8 && bitDepth != 16 && bitDepth != 24 && bitDepth != 32 {
		return fmt.Errorf("bit depth must be 8, 16, 24, or 32")
	}
	return nil
}

// CalculateAudioDuration calculates the duration of audio in milliseconds
func (au *AudioUtils) CalculateAudioDuration(sampleCount int, sampleRate int) int {
	if sampleRate <= 0 {
		return 0
	}
	return (sampleCount * 1000) / sampleRate
}

// CalculateSampleCount calculates the number of samples for a given duration
func (au *AudioUtils) CalculateSampleCount(durationMs int, sampleRate int) int {
	if sampleRate <= 0 {
		return 0
	}
	return (durationMs * sampleRate) / 1000
}

// SplitAudioIntoChunks splits audio data into chunks for processing
func (au *AudioUtils) SplitAudioIntoChunks(samples []int16, chunkSize int) [][]int16 {
	var chunks [][]int16
	for i := 0; i < len(samples); i += chunkSize {
		end := i + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		chunks = append(chunks, samples[i:end])
	}
	return chunks
}

// MergeAudioChunks merges audio chunks into a single array
func (au *AudioUtils) MergeAudioChunks(chunks [][]int16) []int16 {
	totalLength := 0
	for _, chunk := range chunks {
		totalLength += len(chunk)
	}

	merged := make([]int16, totalLength)
	pos := 0
	for _, chunk := range chunks {
		copy(merged[pos:], chunk)
		pos += len(chunk)
	}

	return merged
}

// NormalizeAudio normalizes audio samples to prevent clipping
func (au *AudioUtils) NormalizeAudio(samples []int16, maxAmplitude float64) []int16 {
	if maxAmplitude <= 0 {
		maxAmplitude = 1.0
	}

	// Find the maximum absolute value
	maxVal := float64(0)
	for _, sample := range samples {
		absVal := float64(sample)
		if absVal < 0 {
			absVal = -absVal
		}
		if absVal > maxVal {
			maxVal = absVal
		}
	}

	// If no normalization needed, return original
	if maxVal == 0 || maxVal/maxAmplitude <= 1.0 {
		return samples
	}

	// Apply normalization
	scale := maxAmplitude / maxVal
	normalized := make([]int16, len(samples))
	for i, sample := range samples {
		normalized[i] = int16(float64(sample) * scale)
	}

	return normalized
}

// RemoveSilence removes leading and trailing silence from audio
func (au *AudioUtils) RemoveSilence(samples []int16, silenceThreshold int16) []int16 {
	if len(samples) == 0 {
		return samples
	}

	// Find start of non-silence
	start := 0
	for start < len(samples) && abs(samples[start]) <= silenceThreshold {
		start++
	}

	// Find end of non-silence
	end := len(samples) - 1
	for end >= start && abs(samples[end]) <= silenceThreshold {
		end--
	}

	if start > end {
		// All silence
		return []int16{}
	}

	return samples[start : end+1]
}

// ConvertPCM16ToWAV converts 16-bit PCM samples to WAV format
func (au *AudioUtils) ConvertPCM16ToWAV(samples []int16, sampleRate int) ([]byte, error) {
	// Create WAV format configuration
	wavFormat := wav.WAVFormat{
		AudioFormat:   1, // PCM
		NumChannels:   1, // Mono
		SampleRate:    safeUint32Audio(sampleRate),
		ByteRate:      safeUint32Audio(sampleRate) * 2, // sampleRate * channels * bytesPerSample
		BlockAlign:    2,                      // channels * bytesPerSample
		BitsPerSample: 16,
	}

	// Create a bytes.Buffer to hold the WAV data
	buffer := &bytes.Buffer{}

	// Create WAV header with correct data size
	// Safely calculate data size with overflow check
	samplesLen := len(samples)
	if samplesLen > 2147483647 { // Check for potential overflow before multiplication
		return nil, fmt.Errorf("too many samples: %d exceeds maximum safe limit", samplesLen)
	}
	dataSize := safeUint32Audio(samplesLen * 2) // 16-bit samples, 2 bytes per sample
	header := wav.NewWAVHeader(wavFormat, dataSize)

	// Write WAV header
	if err := header.Write(buffer); err != nil {
		return nil, fmt.Errorf("failed to write WAV header: %v", err)
	}

	// Write PCM samples directly
	for _, sample := range samples {
		// Convert int16 to little-endian bytes
		if err := binary.Write(buffer, binary.LittleEndian, sample); err != nil {
			return nil, fmt.Errorf("failed to write sample: %v", err)
		}
	}

	// Return the complete WAV data
	return buffer.Bytes(), nil
}

// ConvertBase64ToWAV converts Base64 audio directly to WAV format
func (au *AudioUtils) ConvertBase64ToWAV(base64Audio string, sourceSampleRate int, targetSampleRate int) ([]byte, error) {
	// Decode Base64 to PCM samples
	samples, err := au.ConvertBase64ToPCM16(base64Audio)
	if err != nil {
		return nil, err
	}

	// Resample if needed
	if sourceSampleRate != targetSampleRate {
		samples, err = au.ResampleAudio(samples, sourceSampleRate, targetSampleRate)
		if err != nil {
			return nil, fmt.Errorf("failed to resample audio: %v", err)
		}
	}

	// Convert to WAV format
	return au.ConvertPCM16ToWAV(samples, targetSampleRate)
}

// SaveAudioToFile saves audio samples to a WAV file
func (au *AudioUtils) SaveAudioToFile(samples []int16, sampleRate int, filename string) error {
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("audio_%s.wav", timestamp)
	}

	// Ensure audio directory exists
	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0750); err != nil {
		return fmt.Errorf("failed to create audio directory: %v", err)
	}

	// Create and validate full file path to prevent path traversal
	safeFilePath, err := validateFilePath(filename, audioDir)
	if err != nil {
		return fmt.Errorf("invalid file path: %v", err)
	}

	// Create WAV file
	file, err := os.Create(safeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create audio file: %v", err)
	}
	defer file.Close()

	// Create WAV format configuration
	wavFormat := wav.WAVFormat{
		AudioFormat:   1, // PCM
		NumChannels:   1, // Mono
		SampleRate:    safeUint32Audio(sampleRate),
		ByteRate:      safeUint32Audio(sampleRate) * 2, // sampleRate * channels * bytesPerSample
		BlockAlign:    2,                      // channels * bytesPerSample
		BitsPerSample: 16,
	}

	// Create WAV writer
	writer, err := wav.NewWriter(file, wavFormat)
	if err != nil {
		return fmt.Errorf("failed to create WAV writer: %v", err)
	}

	// Write samples
	if err := writer.WriteSamples(samples); err != nil {
		return fmt.Errorf("failed to write WAV samples: %v", err)
	}

	// Close writer to finalize WAV file
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close WAV writer: %v", err)
	}

	logger.WithFields(map[string]interface{}{
		"component":    "ws_audio_core ",
		"action":       "file_saved",
		"filePath":     safeFilePath,
		"sampleCount":  len(samples),
		"sampleRate":   sampleRate,
		"duration":     float64(len(samples)) / float64(sampleRate),
		"fileSize":     func() int64 {
			if info, err := os.Stat(safeFilePath); err == nil {
				return info.Size()
			}
			return 0
		}(),
	}).Info("Audio file saved successfully")

	return nil
}

// SaveAudioFromBase64 saves Base64 audio data to a WAV file
func (au *AudioUtils) SaveAudioFromBase64(base64Audio string, sampleRate int, filename string) error {
	// Convert Base64 to PCM samples
	samples, err := au.ConvertBase64ToPCM16(base64Audio)
	if err != nil {
		return fmt.Errorf("failed to convert Base64 to PCM: %v", err)
	}

	// Log audio analysis
	avgAmplitude := float64(0)
	maxAmplitude := float64(0)
	if len(samples) > 0 {
		for _, sample := range samples {
			absSample := float64(sample)
			if absSample < 0 {
				absSample = -absSample
			}
			avgAmplitude += absSample
			if absSample > maxAmplitude {
				maxAmplitude = absSample
			}
		}
		avgAmplitude /= float64(len(samples))
	}

	logger.WithFields(map[string]interface{}{
		"component":     "audio_analysis",
		"action":        "analyzing_base64_audio",
		"filename":      filename,
		"sampleCount":   len(samples),
		"sampleRate":    sampleRate,
		"avgAmplitude":  avgAmplitude,
		"maxAmplitude":  maxAmplitude,
		"hasAudio":      maxAmplitude > 0,
		"base64Length":  len(base64Audio),
	}).Info("Base64 audio analysis completed")

	// Save to file
	return au.SaveAudioToFile(samples, sampleRate, filename)
}

// CleanOldAudioFiles removes old audio files to prevent disk space issues
func (au *AudioUtils) CleanOldAudioFiles(maxFiles int) error {
	audioDir := "audio"

	// Check if directory exists
	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	// Read directory
	files, err := os.ReadDir(audioDir)
	if err != nil {
		return fmt.Errorf("failed to read audio directory: %v", err)
	}

	// Filter only .wav files and sort by modification time
	var wavFiles []os.FileInfo
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".wav") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			wavFiles = append(wavFiles, info)
		}
	}

	// If we have more files than allowed, remove oldest ones
	if len(wavFiles) > maxFiles {
		// Sort by modification time (oldest first)
		for i := 0; i < len(wavFiles)-1; i++ {
			for j := i + 1; j < len(wavFiles); j++ {
				if wavFiles[i].ModTime().After(wavFiles[j].ModTime()) {
					wavFiles[i], wavFiles[j] = wavFiles[j], wavFiles[i]
				}
			}
		}

		// Remove oldest files
		filesToRemove := len(wavFiles) - maxFiles
		for i := 0; i < filesToRemove; i++ {
			filePath := filepath.Join(audioDir, wavFiles[i].Name())
			if err := os.Remove(filePath); err == nil {
				logger.WithFields(map[string]interface{}{
					"component": "cln_audio_proc",
					"action":    "file_removed",
					"filePath":  filePath,
				}).Info("Old audio file removed")
			}
		}
	}

	return nil
}

// Helper function for absolute value
func abs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}