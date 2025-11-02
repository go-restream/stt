package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Reader reads WAV files
type Reader struct {
	reader     io.ReadSeeker
	format     WAVFormat
	dataOffset int64  // data chunk start position
	dataSize   uint32 // data chunk size
}

// NewReader creates a new WAV reader
func NewReader(reader io.ReadSeeker) (*Reader, error) {
	r := &Reader{
		reader: reader,
	}

	// Parse and validate WAV header
	if err := r.parseWAV(); err != nil {
		return nil, fmt.Errorf("failed to parse WAV file: %v", err)
	}

	return r, nil
}

// parseWAV parses WAV file format
func (r *Reader) parseWAV() error {
    // Read RIFF header
    var riffHeader struct {
        ChunkID   [4]byte
        ChunkSize uint32
        Format    [4]byte
    }
    if err := binary.Read(r.reader, binary.LittleEndian, &riffHeader); err != nil {
        return fmt.Errorf("failed to read RIFF header: %v", err)
    }

    // Validate RIFF and WAVE identifiers
    if string(riffHeader.ChunkID[:]) != "RIFF" {
        return fmt.Errorf("invalid RIFF header")
    }
    if string(riffHeader.Format[:]) != "WAVE" {
        return fmt.Errorf("not a WAVE file")
    }

    // Iterate through chunks
    var (
        foundFmt  bool
        foundData bool
        chunkPos  int64 = 12 // Position after RIFF header
    )

    for chunkPos < int64(riffHeader.ChunkSize)+8 {
        // Read chunk header
        var chunkHeader struct {
            ChunkID   [4]byte
            ChunkSize uint32
        }
        if err := binary.Read(r.reader, binary.LittleEndian, &chunkHeader); err != nil {
            return fmt.Errorf("failed to read chunk header at pos %d: %v", chunkPos, err)
        }

        chunkStart := chunkPos + 8
        chunkEnd := chunkStart + int64(chunkHeader.ChunkSize)

        switch string(chunkHeader.ChunkID[:]) {
        case "fmt ":
            // Read fmt chunk
            if err := binary.Read(r.reader, binary.LittleEndian, &r.format); err != nil {
                return fmt.Errorf("failed to read fmt chunk: %v", err)
            }
            foundFmt = true

            // Validate audio format
            if r.format.AudioFormat != 1 { // 1 = PCM
                return fmt.Errorf("unsupported audio format: %d", r.format.AudioFormat)
            }

            // Skip possible extension data
            if _, err := r.reader.Seek(chunkEnd, io.SeekStart); err != nil {
                return fmt.Errorf("failed to seek past fmt chunk: %v", err)
            }

        case "data":
            r.dataOffset = chunkStart
            r.dataSize = chunkHeader.ChunkSize
            foundData = true

            // Seek to data area start
            if _, err := r.reader.Seek(r.dataOffset, io.SeekStart); err != nil {
                return fmt.Errorf("failed to seek to data start: %v", err)
            }
            return nil

        default: // Other chunks (LIST/JUNK/etc.)
            // Note: WAV spec requires chunks to be 2-byte aligned
            if chunkHeader.ChunkSize%2 != 0 {
                chunkHeader.ChunkSize++
            }
            if _, err := r.reader.Seek(int64(chunkHeader.ChunkSize), io.SeekCurrent); err != nil {
                return fmt.Errorf("failed to skip chunk %q: %v", chunkHeader.ChunkID, err)
            }
        }

        chunkPos = chunkEnd
        if chunkHeader.ChunkSize%2 != 0 {
            chunkPos++ // Alignment padding
        }
    }

    if !foundFmt {
        return fmt.Errorf("fmt chunk not found")
    }
    if !foundData {
        return fmt.Errorf("data chunk not found")
    }
    return nil
}

// ReadSamples reads specified number of samples
func (r *Reader) ReadSamples(samples []int16) (int, error) {
		bytesToRead := len(samples) * int(r.format.BlockAlign/r.format.NumChannels)

	// Read raw bytes
	rawData := make([]byte, bytesToRead)
	n, err := r.reader.Read(rawData)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read samples: %v", err)
	}

	// Convert bytes to samples
	samplesRead := n / int(r.format.BlockAlign/r.format.NumChannels)
	for i := 0; i < samplesRead; i++ {
		offset := i * 2 // 16-bit samples, 2 bytes per sample
		samples[i] = int16(binary.LittleEndian.Uint16(rawData[offset : offset+2]))
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

// ReadSamplesPCM reads entire WAV file PCM data
func (r *Reader) ReadSamplesPCM() ([]int16, error) {
		totalSamples := int(r.dataSize) / int(r.format.BlockAlign/r.format.NumChannels)
	samples := make([]int16, totalSamples)

	// Read all samples
	_, err := r.ReadSamples(samples)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read PCM samples: %v", err)
	}

	return samples, nil
}
