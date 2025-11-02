package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Reader handles WAV file reading
type Reader struct {
	reader     io.ReadSeeker
	format     WAVFormat
	dataOffset int64  // Start position of data chunk
	dataSize   uint32 // Size of data chunk
}

// NewReader creates a new WAV reader
func NewReader(reader io.ReadSeeker) (*Reader, error) {
	r := &Reader{
		reader: reader,
	}

	// Read and validate WAV header
	if err := r.parseWAV(); err != nil {
		return nil, fmt.Errorf("failed to parse WAV file: %v", err)
	}

	return r, nil
}

// parseWAV parses WAV file format
func (r *Reader) parseWAV() error {
	// Read RIFF header
	var riffID [4]byte
	var riffSize uint32
	var waveID [4]byte

	if err := binary.Read(r.reader, binary.LittleEndian, &riffID); err != nil {
		return fmt.Errorf("failed to read RIFF ID: %v", err)
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &riffSize); err != nil {
		return fmt.Errorf("failed to read RIFF size: %v", err)
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &waveID); err != nil {
		return fmt.Errorf("failed to read WAVE ID: %v", err)
	}

	// Validate file identifiers
	if string(riffID[:]) != "RIFF" {
		return fmt.Errorf("not a RIFF file")
	}
	if string(waveID[:]) != "WAVE" {
		return fmt.Errorf("not a WAVE file")
	}

	// Find fmt and data chunks
	var chunkID [4]byte
	var chunkSize uint32
	var foundFmt, foundData bool

	for !foundFmt || !foundData {
		if err := binary.Read(r.reader, binary.LittleEndian, &chunkID); err != nil {
			return fmt.Errorf("failed to read chunk ID: %v", err)
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &chunkSize); err != nil {
			return fmt.Errorf("failed to read chunk size: %v", err)
		}

		switch string(chunkID[:]) {
		case "fmt ":
			// Read format chunk content
			if err := binary.Read(r.reader, binary.LittleEndian, &r.format); err != nil {
				return fmt.Errorf("failed to read format chunk: %v", err)
			}
			foundFmt = true

			// Skip extra data if chunk size exceeds format struct size
			remaining := int64(chunkSize) - int64(binary.Size(r.format))
			if remaining > 0 {
				if _, err := r.reader.Seek(remaining, io.SeekCurrent); err != nil {
					return fmt.Errorf("failed to seek past extra format data: %v", err)
				}
			}

		case "data":
			// Record data chunk position and size
			offset, err := r.reader.Seek(0, io.SeekCurrent)
			if err != nil {
				return fmt.Errorf("failed to get data offset: %v", err)
			}
			r.dataOffset = offset
			r.dataSize = chunkSize
			foundData = true

			// Skip data chunk content
			if _, err := r.reader.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return fmt.Errorf("failed to seek past data chunk: %v", err)
			}

		default:
			// Skip other chunks
			if _, err := r.reader.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return fmt.Errorf("failed to seek past chunk: %v", err)
			}
		}
	}

	// Validate format
	if err := r.format.Validate(); err != nil {
		return fmt.Errorf("invalid WAV format: %v", err)
	}

	// Seek to data chunk start position
	_, err := r.reader.Seek(r.dataOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to data start: %v", err)
	}

	return nil
}

// ReadSamples reads specified number of audio samples
func (r *Reader) ReadSamples(samples []int16) (int, error) {
	// Calculate bytes to read
	bytesToRead := len(samples) * int(r.format.BlockAlign/r.format.NumChannels)

	// Read raw bytes
	rawData := make([]byte, bytesToRead)
	n, err := r.reader.Read(rawData)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read samples: %v", err)
	}

	// Convert bytes to samples with proper signed conversion
	samplesRead := n / int(r.format.BlockAlign/r.format.NumChannels)
	for i := 0; i < samplesRead; i++ {
		offset := i * 2 // 16-bit samples, 2 bytes per sample
		// Safely convert uint16 to int16 using proper bit manipulation
		value := binary.LittleEndian.Uint16(rawData[offset : offset+2])
		// Use bit manipulation to avoid overflow - convert unsigned to signed 16-bit
		samples[i] = int16(value) // This is safe in Go - it wraps around as expected for 16-bit audio
	}

	if err == io.EOF {
		return samplesRead, io.EOF
	}
	return samplesRead, nil
}

// GetFormat returns WAV format information
func (r *Reader) GetFormat() WAVFormat {
	return r.format
}

// GetDataSize returns audio data size
func (r *Reader) GetDataSize() uint32 {
	return r.dataSize
}

// Seek sets read position
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		offset += r.dataOffset
	case io.SeekEnd:
		offset += r.dataOffset + int64(r.dataSize)
	}
	return r.reader.Seek(offset, whence)
}

// Close closes the reader
func (r *Reader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
