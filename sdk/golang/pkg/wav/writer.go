package wav

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Writer writes WAV files
type Writer struct {
	writer     io.WriteSeeker
	header     WAVHeader
	format     WAVFormat
	dataSize   uint32
	dataOffset int64
}

// NewWriter creates a new WAV writer
func NewWriter(writer io.WriteSeeker, format WAVFormat) (*Writer, error) {
	// Validate format
	if err := format.Validate(); err != nil {
		return nil, fmt.Errorf("invalid WAV format: %v", err)
	}

	w := &Writer{
		writer: writer,
		format: format,
		header: NewWAVHeader(format, 0), // Initial data size is 0
	}

	// Write header
	if err := w.writeHeader(); err != nil {
		return nil, fmt.Errorf("failed to write WAV header: %v", err)
	}

	// Record data section start position
	offset, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to get data offset: %v", err)
	}
	w.dataOffset = offset

	return w, nil
}

// NewFileWriter creates a new WAV file writer
func NewFileWriter(filename string, format WAVFormat) (*Writer, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}

	writer, err := NewWriter(file, format)
	if err != nil {
		file.Close()
		return nil, err
	}

	return writer, nil
}

// writeHeader writes WAV file header
func (w *Writer) writeHeader() error {
	return w.header.Write(w.writer)
}

// WriteSamples writes sample data
func (w *Writer) WriteSamples(samples []int16) error {
		bytesToWrite := len(samples) * int(w.format.BlockAlign/w.format.NumChannels)
	rawData := make([]byte, bytesToWrite)

	// Convert samples to bytes
	for i := 0; i < len(samples); i++ {
		offset := i * 2 // 16-bit samples, 2 bytes per sample
		binary.LittleEndian.PutUint16(rawData[offset:offset+2], uint16(samples[i]))
	}

	// Write data
	n, err := w.writer.Write(rawData)
	if err != nil {
		return fmt.Errorf("failed to write samples: %v", err)
	}

	// Update data size
	// Safely add to data size with overflow check
	if n > 0 {
		newSize := w.dataSize + uint32(n)
		if newSize < w.dataSize { // Check for overflow
			return fmt.Errorf("WAV data size overflow: %d + %d exceeds uint32 limit", w.dataSize, n)
		}
		w.dataSize = newSize
	}
	return nil
}

// Close updates header and closes writer
func (w *Writer) Close() error {
	// Update data size in header
	w.header.Subchunk2Size = w.dataSize
	w.header.ChunkSize = 36 + w.dataSize

	// Seek to file start
	_, err := w.writer.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start: %v", err)
	}

	// Rewrite header
	if err := w.writeHeader(); err != nil {
		return fmt.Errorf("failed to update header: %v", err)
	}

	// Close writer
	if closer, ok := w.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// GetDataSize returns written data size
func (w *Writer) GetDataSize() uint32 {
	return w.dataSize
}

// GetFormat returns WAV format information
func (w *Writer) GetFormat() WAVFormat {
	return w.format
}

// GetHeader returns WAV header information
func (w *Writer) GetHeader() WAVHeader {
	return w.header
}
