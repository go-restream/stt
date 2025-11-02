package main

import (
	"gosdk/pkg/wav"
	"encoding/binary"
	"fmt"
)

// Convert []int16 to []byte (little-endian)
func samplesToBytes(samples []int16) ([]byte, error)  {
    buf := make([]byte, 2*len(samples))
    for i, v := range samples {
        binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
    }
    return buf,nil
}

func BytesToSamples(data []byte) ([]int16, error) {
    if len(data)%2 != 0 {
        return nil, fmt.Errorf("input byte length must be even, got %d bytes", len(data))
    }

    samples := make([]int16, len(data)/2)
    for i := range samples {
        // 安全边界检查（虽然理论上不需要，因为前面已经验证过长度）
        offset := i * 2
        if offset+2 > len(data) {
            break
        }
        samples[i] = int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
    }
    return samples, nil
}

// saveAsWAV saves PCM byte data as WAV file
func saveAsWAV(byteData []byte,sampleRate uint32) error {
	format := wav.WAVFormat{
		AudioFormat:   1,    // PCM
		NumChannels:   1,    // Mono
		SampleRate:    sampleRate, // Sample rate
		BitsPerSample: 16,   // 16-bit
		ByteRate:      sampleRate * 1 * 16 / 8, // SampleRate * NumChannels * BitsPerSample/8
		BlockAlign:    1 * 16 / 8,         // NumChannels * BitsPerSample/8
	}

	writer, err := wav.NewFileWriter("debug_output.wav", format)
	if err != nil {
		return fmt.Errorf("failed to create WAV writer: %v", err)
	}
	defer writer.Close()

	samples, err := BytesToSamples(byteData)
	if err != nil {
		return fmt.Errorf("invalid PCM byte data: %v", err)
	}

	if err := writer.WriteSamples(samples); err != nil {
		return fmt.Errorf("failed to write samples: %v", err)
	}

	return nil
}
