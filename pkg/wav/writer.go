package wav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Writer handles WAV file writing
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

// NewBufferWriter creates a new WAV memory buffer writer
// Returns Writer and underlying bytes.Buffer
func NewBufferWriter(format WAVFormat) (*Writer, *bytes.Buffer, error) {
	buffer := &bytes.Buffer{}

	// Create wrapper for bytes.Buffer to implement io.WriteSeeker interface
	bufferSeeker := &bufferWriteSeeker{buffer: buffer}

	writer, err := NewWriter(bufferSeeker, format)
	if err != nil {
		return nil, nil, err
	}

	return writer, buffer, nil
}

// bufferWriteSeeker wraps bytes.Buffer to implement io.WriteSeeker interface
type bufferWriteSeeker struct {
	buffer *bytes.Buffer
	pos    int64
}

func (b *bufferWriteSeeker) Write(p []byte) (n int, err error) {
	// Handle writes when position is not at end
	if b.pos < int64(b.buffer.Len()) {
		// Memory buffer only supports sequential writes
		// Return error if position is not at end
		if b.pos != int64(b.buffer.Len()) {
			return 0, fmt.Errorf("bufferWriteSeeker only supports sequential writes")
		}
	}

	n, err = b.buffer.Write(p)
	b.pos += int64(n)
	return n, err
}

func (b *bufferWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		b.pos = offset
	case io.SeekCurrent:
		b.pos += offset
	case io.SeekEnd:
		b.pos = int64(b.buffer.Len()) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	// Ensure position is valid
	if b.pos < 0 {
		b.pos = 0
	}
	if b.pos > int64(b.buffer.Len()) {
		b.pos = int64(b.buffer.Len())
	}

	return b.pos, nil
}

// writeHeader writes WAV file header
func (w *Writer) writeHeader() error {
	return w.header.Write(w.writer)
}

// WriteSamples writes sample data
func (w *Writer) WriteSamples(samples []int16) error {
	// Calculate bytes to write
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
	w.dataSize += uint32(n)
	return nil
}

// Close updates file header and closes writer
func (w *Writer) Close() error {
	// Update data size in file header
	w.header.Subchunk2Size = w.dataSize
	w.header.ChunkSize = 36 + w.dataSize

	// Seek to file start
	_, err := w.writer.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start: %v", err)
	}

	// Rewrite file header
	if err := w.writeHeader(); err != nil {
		return fmt.Errorf("failed to update header: %v", err)
	}

	// Close writer (only for closable types)
	if closer, ok := w.writer.(io.Closer); ok {
		// Close resources that need closing (like files)
		return closer.Close()
	}
	// For non-closable types (like memory buffers), return directly
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
