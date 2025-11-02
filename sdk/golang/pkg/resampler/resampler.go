package resampler

import (
	"errors"

	"github.com/go-audio/audio"
)

// Resample48kTo16k resamples audio data from 48kHz to 16kHz
func Resample48kTo16k(input *audio.IntBuffer) (*audio.IntBuffer, error) {
	if input == nil {
		return nil, errors.New("input buffer cannot be nil")
	}
	if input.Format == nil {
		return nil, errors.New("input format cannot be nil")
	}
	if input.Format.SampleRate != 48000 {
		return nil, errors.New("input sample rate must be 48000Hz")
	}

	resampleRatio := 3
	newNumSamples := len(input.Data) / resampleRatio

	output := &audio.IntBuffer{
		Data: make([]int, newNumSamples),
		Format: &audio.Format{
			NumChannels: input.Format.NumChannels,
			SampleRate:  16000,
		},
		SourceBitDepth: input.SourceBitDepth,
	}

	// Simple averaging downsampling
	for i := 0; i < newNumSamples; i++ {
		start := i * resampleRatio
		end := start + resampleRatio
		if end > len(input.Data) {
			end = len(input.Data)
		}

		sum := 0
		for j := start; j < end; j++ {
			sum += input.Data[j]
		}
		output.Data[i] = sum / resampleRatio
	}

	return output, nil
}

// Resample resamples audio data to target sample rate
func Resample(input *audio.IntBuffer, targetRate int) (*audio.IntBuffer, error) {
	if input == nil || input.Format == nil {
		return nil, errors.New("invalid input buffer")
	}

	if input.Format.SampleRate == targetRate {
		return input, nil
	}

	switch input.Format.SampleRate {
	case 48000:
		if targetRate == 16000 {
			return Resample48kTo16k(input)
		}
	}

	return nil, errors.New("unsupported sample rate conversion")
}