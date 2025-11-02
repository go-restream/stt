package vad

import (
	"github.com/go-restream/stt/pkg/logger"

	yaml "github.com/go-restream/stt/config"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
	"github.com/sirupsen/logrus"
)

var (
	default_sample_rate = 16000
)

type VADDetector struct {
	vad         *sherpa.VoiceActivityDetector
	sampleRate  int
	sampleBuffer []float32
	speechSegments []sherpa.SpeechSegment
	printed     bool
	config      *yaml.Config
}

func NewVADDetector(cfg *yaml.Config) *VADDetector {
	vadCfg := initVADConfig(cfg)
	bufferSize := float32(20)
	vad := sherpa.NewVoiceActivityDetector(vadCfg, bufferSize)
	if vad == nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":    "initialization_failed",
		}).Fatal("Failed to initialize VAD detector")
	}
	return &VADDetector{
		vad:        vad,
		sampleRate: default_sample_rate,
		config:     cfg,
	}
}

func (v *VADDetector) Close() {
	sherpa.DeleteVoiceActivityDetector(v.vad)
}

// ProcessSamples processes audio samples and returns speech segments
func (v *VADDetector) ProcessSamples(samples []float32) *sherpa.SpeechSegment {
	if len(samples) == 0 {
		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":    "empty_samples_received",
		}).Warn("Warning: empty samples passed to ProcessSamples")
		return nil
	}
	if v.vad == nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":    "detector_not_initialized",
		}).Error("Error: VAD detector not initialized")
		return nil
	}

	var maxAmplitude float32
	var sumAmplitude float32
	for _, sample := range samples {
		if sample < 0 {
			sumAmplitude -= sample
		} else {
			sumAmplitude += sample
		}
		if sample < 0 {
			sample = -sample
		}
		if sample > maxAmplitude {
			maxAmplitude = sample
		}
	}
	avgAmplitude := sumAmplitude / float32(len(samples))

	logger.WithFields(logrus.Fields{
		"component": "eng_vad_audio_sys",
		"action":        "processing_samples",
		"sampleCount":   len(samples),
		"sampleRate":    v.sampleRate,
		"maxAmplitude":  maxAmplitude,
		"avgAmplitude":  avgAmplitude,
		"vadBypass":     v.config.Vad.BypassForTesting,
	}).Debug("Processing samples with audio analysis")

	if v.config.Vad.BypassForTesting {
		if len(samples) > 0 {
			segment := sherpa.SpeechSegment{
				Samples: samples,
			}
			logger.WithFields(logrus.Fields{
				"component": "eng_vad_audio_sys",
				"action":       "vad_bypass_triggered",
				"sampleCount":  len(samples),
				"maxAmplitude": maxAmplitude,
				"bypassReason": "forced_for_testing",
			}).Warn("VAD bypass enabled - forcing speech segment creation for testing")
			return &segment
		}
	}

	v.vad.AcceptWaveform(samples)

	isSpeech := v.vad.IsSpeech()
	isEmpty := v.vad.IsEmpty()

	logger.WithFields(logrus.Fields{
		"component": "eng_vad_audio_sys",
		"action":       "vad_state_check",
		"isSpeech":     isSpeech,
		"isEmpty":      isEmpty,
		"hasPrinted":   v.printed,
		"sampleCount":  len(samples),
	}).Debug("VAD state after processing")

	if isSpeech && !v.printed {
		v.printed = true
		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":    "speech_detected",
			"threshold": 0.5,
		}).Info("Speech detected by VAD engine")
	}

	if !isSpeech {
		v.printed = false
	}

	segmentsCollected := 0
	for !isEmpty {
		segment := v.vad.Front()
		v.vad.Pop()
		v.speechSegments = append(v.speechSegments, *segment)
		segmentsCollected++
		isEmpty = v.vad.IsEmpty()

		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":        "segment_collected",
			"segmentIndex":  segmentsCollected,
			"segmentSamples": len(segment.Samples),
			"isEmpty":       isEmpty,
		}).Debug("Collected VAD speech segment")
	}

	if segmentsCollected > 0 {
		logger.WithFields(logrus.Fields{
			"component": "eng_vad_audio_sys",
			"action":          "segments_collected_total",
			"segmentsCount":   segmentsCollected,
			"bufferedSegments": len(v.speechSegments),
		}).Info("VAD segments collected in this batch")
	}

	if len(v.speechSegments) > 0 {
		segment := v.speechSegments[0]
		v.speechSegments = v.speechSegments[1:]

		if len(segment.Samples) > 0 {
			duration := float32(len(segment.Samples)) / float32(v.sampleRate)
			logger.WithFields(logrus.Fields{
				"component": "eng_vad_audio_sys",
				"action":        "segment_processed",
				"duration":      duration,
				"sampleCount":   len(segment.Samples),
				"sampleRate":    v.sampleRate,
			}).Info("Processed speech segment")
			v.vad.Reset()
			v.printed = false
			return &segment
		}
	}

	logger.WithFields(logrus.Fields{
		"component": "eng_vad_audio_sys",
		"action":      "no_speech_detected",
		"sampleCount": len(samples),
	}).Debug("No speech segments detected")
	return nil
}

func (v *VADDetector) ProcessSample(sample float32) *sherpa.SpeechSegment {
	v.sampleBuffer = append(v.sampleBuffer, sample)

	if len(v.sampleBuffer) >= 160 {
		defer func() { v.sampleBuffer = v.sampleBuffer[:0] }()
		return v.ProcessSamples(v.sampleBuffer)
	}
	return nil
}

// Reset resets the VAD detector state
func (v *VADDetector) Reset() {
	v.vad.Reset()
	v.speechSegments = nil
}

// IsSpeech checks if speech activity is currently detected
func (v *VADDetector) IsSpeech() bool {
	return v.vad.IsSpeech()
}

func initVADConfig(cfg *yaml.Config) *sherpa.VadModelConfig{
	config := sherpa.VadModelConfig{}

	config.SileroVad.Model = cfg.Vad.Model
	config.SileroVad.Threshold = cfg.Vad.Threshold
	config.SileroVad.MinSilenceDuration = cfg.Vad.MinSilenceDuration
	config.SileroVad.MinSpeechDuration = cfg.Vad.MinSpeechDuration
	config.SileroVad.WindowSize = cfg.Vad.WindowSize
	config.SileroVad.MaxSpeechDuration = cfg.Vad.MaxSpeechDuration

	config.SampleRate = cfg.Vad.SampleRate
	config.NumThreads = cfg.Vad.NumThreads
	config.Provider = cfg.Vad.Provider
	config.Debug = cfg.Vad.Debug

	return &config
}
