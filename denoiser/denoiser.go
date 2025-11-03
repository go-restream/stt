package denoiser

import (
	"sync"
	"time"

	"github.com/go-restream/stt/pkg/logger"

	yaml "github.com/go-restream/stt/config"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
	"github.com/sirupsen/logrus"
)

var (
	default_sample_rate = 16000
)

type DenoiserProcessor struct {
	denoiser             *sherpa.OfflineSpeechDenoiser
	sampleRate          int
	config              *yaml.Config
	mutex               sync.RWMutex
	processingStartTime time.Time
	stats               DenoiserStats
}

type DenoiserStats struct {
	TotalSegmentsProcessed int
	TotalProcessingTime   time.Duration
	FailedProcessings     int
	AverageLatency        time.Duration
}

func NewDenoiserProcessor(cfg *yaml.Config) *DenoiserProcessor {
	if !cfg.Denoiser.Enable {
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "denoiser_disabled",
		}).Info("Denoiser is disabled in configuration")
		return &DenoiserProcessor{
			config:     cfg,
			sampleRate: default_sample_rate,
		}
	}

	denoiserCfg := initDenoiserConfig(cfg)
	denoiser := sherpa.NewOfflineSpeechDenoiser(denoiserCfg)
	if denoiser == nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "initialization_failed",
			"model":     cfg.Denoiser.Model,
		}).Error("Failed to initialize denoiser processor - operating in bypass mode")

		// Return processor without denoiser (will operate in bypass mode)
		return &DenoiserProcessor{
			config:     cfg,
			sampleRate: cfg.Denoiser.SampleRate,
			stats: DenoiserStats{
				TotalSegmentsProcessed: 0,
				TotalProcessingTime:   0,
				FailedProcessings:     0,
				AverageLatency:        0,
			},
		}
	}

	logger.WithFields(logrus.Fields{
		"component": "eng_denoiser_audio_sys",
		"action":    "initialization_success",
		"model":     cfg.Denoiser.Model,
		"sampleRate": cfg.Denoiser.SampleRate,
	}).Info("Denoiser processor initialized successfully")

	return &DenoiserProcessor{
		denoiser:   denoiser,
		sampleRate: cfg.Denoiser.SampleRate,
		config:     cfg,
		stats: DenoiserStats{
			TotalSegmentsProcessed: 0,
			TotalProcessingTime:   0,
			FailedProcessings:     0,
			AverageLatency:        0,
		},
	}
}

func (d *DenoiserProcessor) Close() {
	if d.denoiser != nil {
		sherpa.DeleteOfflineSpeechDenoiser(d.denoiser)
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "cleanup_completed",
		}).Info("Denoiser processor cleaned up")
	}
}

func (d *DenoiserProcessor) ProcessSegment(segment *sherpa.SpeechSegment) *sherpa.SpeechSegment {
	if segment == nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "nil_segment_received",
		}).Warn("Received nil segment - returning without processing")
		return nil
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.processingStartTime = time.Now()

	if !d.config.Denoiser.Enable || d.denoiser == nil {
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "denoiser_bypassed",
			"reason":     "disabled_or_not_initialized",
			"sampleCount": len(segment.Samples),
		}).Debug("Denoiser bypassed - returning original segment")
		return segment
	}

	if d.config.Denoiser.BypassForTesting {
		logger.WithFields(logrus.Fields{
			"component": "eng_denoiser_audio_sys",
			"action":    "denoiser_bypass_for_testing",
			"sampleCount": len(segment.Samples),
		}).Warn("Denoiser bypass enabled for testing - returning original segment")
		return segment
	}

	logger.WithFields(logrus.Fields{
		"component":  "eng_denoiser_audio_sys",
		"action":     "processing_segment",
		"sampleCount": len(segment.Samples),
		"sampleRate": d.sampleRate,
	}).Debug("Processing audio segment with denoiser")

	enhancedAudio := d.denoiser.Run(segment.Samples, d.sampleRate)

	processingTime := time.Since(d.processingStartTime)
	d.updateStats(processingTime, true)

	if d.config.Denoiser.Debug > 0 {
		logger.WithFields(logrus.Fields{
			"component":     "eng_denoiser_audio_sys",
			"action":        "segment_processed",
			"originalSamples": len(segment.Samples),
			"enhancedSamples": len(enhancedAudio.Samples),
			"processingTime": processingTime.Milliseconds(),
			"maxProcessingTime": d.config.Denoiser.MaxProcessingTimeMs,
		}).Debug("Audio segment enhanced successfully")
	}

	maxProcessingTime := time.Duration(d.config.Denoiser.MaxProcessingTimeMs) * time.Millisecond
	if processingTime > maxProcessingTime {
		logger.WithFields(logrus.Fields{
			"component":       "eng_denoiser_audio_sys",
			"action":          "processing_timeout_warning",
			"processingTime":  processingTime.Milliseconds(),
			"maxAllowedTime":  maxProcessingTime.Milliseconds(),
			"sampleCount":     len(segment.Samples),
		}).Warn("Denoiser processing exceeded maximum allowed time")

		d.updateStats(processingTime, false)
		return segment
	}

	enhancedSegment := sherpa.SpeechSegment{
		Samples: enhancedAudio.Samples,
	}

	return &enhancedSegment
}

func (d *DenoiserProcessor) GetStats() DenoiserStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.stats
}

func (d *DenoiserProcessor) ResetStats() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.stats = DenoiserStats{
		TotalSegmentsProcessed: 0,
		TotalProcessingTime:   0,
		FailedProcessings:     0,
		AverageLatency:        0,
	}

	logger.WithFields(logrus.Fields{
		"component": "eng_denoiser_audio_sys",
		"action":    "stats_reset",
	}).Info("Denoiser statistics reset")
}

func (d *DenoiserProcessor) updateStats(processingTime time.Duration, success bool) {
	d.stats.TotalSegmentsProcessed++
	d.stats.TotalProcessingTime += processingTime

	if !success {
		d.stats.FailedProcessings++
	}

	if d.stats.TotalSegmentsProcessed > 0 {
		d.stats.AverageLatency = d.stats.TotalProcessingTime / time.Duration(d.stats.TotalSegmentsProcessed)
	}
}

func initDenoiserConfig(cfg *yaml.Config) *sherpa.OfflineSpeechDenoiserConfig {
	config := sherpa.OfflineSpeechDenoiserConfig{}

	config.Model.Gtcrn.Model = cfg.Denoiser.Model
	config.Model.NumThreads = int32(cfg.Denoiser.NumThreads)
	config.Model.Debug = int32(cfg.Denoiser.Debug)
	config.Model.Provider = "cpu"

	return &config
}