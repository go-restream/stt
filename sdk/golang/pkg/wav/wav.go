package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WAVFormat contains WAV file format information
type WAVFormat struct {
	AudioFormat   uint16 // Audio format (1 for PCM)
	NumChannels   uint16 // Number of channels
	SampleRate    uint32 // Sample rate
	ByteRate      uint32 // Byte rate = SampleRate * NumChannels * BitsPerSample/8
	BlockAlign    uint16 // Block alignment = NumChannels * BitsPerSample/8
	BitsPerSample uint16 // Bits per sample
}

// WAVHeader represents WAV file header
type WAVHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // File size - 8
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // Format chunk size (16 bytes)
	AudioFormat   uint16  // Audio format (1 for PCM)
	NumChannels   uint16  // Number of channels
	SampleRate    uint32  // Sample rate
	ByteRate      uint32  // Byte rate
	BlockAlign    uint16  // Block alignment
	BitsPerSample uint16  // Bits per sample
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // Audio data size
}

// NewWAVHeader creates a new WAV header
func NewWAVHeader(format WAVFormat, dataSize uint32) WAVHeader {
	return WAVHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     36 + dataSize, // File size - 8
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 16, // PCM format chunk size is always 16
		AudioFormat:   format.AudioFormat,
		NumChannels:   format.NumChannels,
		SampleRate:    format.SampleRate,
		ByteRate:      format.ByteRate,
		BlockAlign:    format.BlockAlign,
		BitsPerSample: format.BitsPerSample,
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: dataSize,
	}
}

// Validate validates WAV format
func (f *WAVFormat) Validate() error {
	if f.AudioFormat != 1 {
		return fmt.Errorf("unsupported audio format: %d (expected 1 for PCM)", f.AudioFormat)
	}
	if f.BitsPerSample != 16 {
		return fmt.Errorf("unsupported bits per sample: %d (expected 16)", f.BitsPerSample)
	}
	if f.ByteRate != f.SampleRate*uint32(f.NumChannels)*uint32(f.BitsPerSample)/8 {
		return fmt.Errorf("invalid byte rate")
	}
	if f.BlockAlign != f.NumChannels*f.BitsPerSample/8 {
		return fmt.Errorf("invalid block align")
	}
	return nil
}

// Write writes WAV header to writer
func (h *WAVHeader) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, h)
}

// Read reads WAV header from reader
func (h *WAVHeader) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, h)
}

// GetFormat gets WAV format from header
func (h *WAVHeader) GetFormat() WAVFormat {
	return WAVFormat{
		AudioFormat:   h.AudioFormat,
		NumChannels:   h.NumChannels,
		SampleRate:    h.SampleRate,
		ByteRate:      h.ByteRate,
		BlockAlign:    h.BlockAlign,
		BitsPerSample: h.BitsPerSample,
	}
}
