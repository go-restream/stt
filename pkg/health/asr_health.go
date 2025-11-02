package health

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-restream/stt/pkg/logger"

	"github.com/sirupsen/logrus"
)

// HealthChecker performs ASR engine health checks
type HealthChecker struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// NewHealthChecker creates a health checker
func NewHealthChecker(baseURL, apiKey, model string) *HealthChecker {
	return &HealthChecker{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckResult represents health check result
type CheckResult struct {
	Service string        `json:"service"`
	Status  string        `json:"status"`  // "ok", "error"
	Error   string        `json:"error,omitempty"`
	Latency time.Duration `json:"latency"`
}

// OverallHealth represents overall health status
type OverallHealth struct {
	Status       string       `json:"status"`        // "ok", "degraded", "error"
	ASREngineURL string       `json:"asr_engine_url"`
	Checks       []CheckResult `json:"checks"`
	Error        string       `json:"error,omitempty"`
}

// checkHealth checks /health endpoint
func (hc *HealthChecker) checkHealth(ctx context.Context) CheckResult {
	start := time.Now()
	url := hc.BaseURL + "/health"

	logger.WithFields(logrus.Fields{
		"component": "mont_health_chk",
		"action":    "check_health_endpoint",
		"url":       url,
	}).Debug("Checking ASR health endpoint")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return CheckResult{
			Service: "health",
			Status:  "error",
			Error:   fmt.Sprintf("create request failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if hc.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+hc.APIKey)
	}

	resp, err := hc.Client.Do(req)
	if err != nil {
		return CheckResult{
			Service: "health",
			Status:  "error",
			Error:   fmt.Sprintf("request failed: %v", err),
			Latency: time.Since(start),
		}
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode == http.StatusOK {
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":    "health_check_success",
			"url":       url,
			"latency":   latency.Milliseconds(),
		}).Debug("Health endpoint check successful")

		return CheckResult{
			Service: "health",
			Status:  "ok",
			Latency: latency,
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return CheckResult{
		Service: "health",
		Status:  "error",
		Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		Latency: latency,
	}
}

// checkModels checks /models endpoint
func (hc *HealthChecker) checkModels(ctx context.Context) CheckResult {
	start := time.Now()
	url := hc.BaseURL + "/models"

	logger.WithFields(logrus.Fields{
		"component": "mont_health_chk",
		"action":    "check_models_endpoint",
		"url":       url,
	}).Debug("Checking ASR models endpoint")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return CheckResult{
			Service: "models",
			Status:  "error",
			Error:   fmt.Sprintf("create request failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if hc.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+hc.APIKey)
	}

	resp, err := hc.Client.Do(req)
	if err != nil {
		return CheckResult{
			Service: "models",
			Status:  "error",
			Error:   fmt.Sprintf("request failed: %v", err),
			Latency: time.Since(start),
		}
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode == http.StatusOK {
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":    "models_check_success",
			"url":       url,
			"latency":   latency.Milliseconds(),
		}).Debug("Models endpoint check successful")

		return CheckResult{
			Service: "models",
			Status:  "ok",
			Latency: latency,
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return CheckResult{
		Service: "models",
		Status:  "error",
		Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		Latency: latency,
	}
}

// checkTranscriptions checks /audio/transcriptions endpoint
func (hc *HealthChecker) checkTranscriptions(ctx context.Context) CheckResult {
	start := time.Now()
	url := hc.BaseURL + "/audio/transcriptions"

	logger.WithFields(logrus.Fields{
		"component": "mont_health_chk",
		"action":    "check_transcriptions_endpoint",
		"url":       url,
	}).Debug("Checking ASR transcriptions endpoint")

	// Try using actual sample.wav file for testing
	samplePath := "./samples/sample.wav"
	audioData, err := os.ReadFile(samplePath)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":    "read_sample_file_failed",
			"samplePath": samplePath,
			"error":     err,
		}).Warn("Failed to read sample.wav file, falling back to test audio")

		// If read fails, create small dummy audio data for testing
		// Generate 100ms silence PCM data (16kHz, 16bit, mono)
		sampleRate := 16000
		duration := 0.1 // 100ms
		numSamples := int(float64(sampleRate) * duration)
		audioData = make([]byte, numSamples*2) // 16-bit samples
	} else {
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":    "read_sample_file_success",
			"samplePath": samplePath,
			"audioSize":  len(audioData),
		}).Debug("Successfully read sample.wav file")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Use actual filename
	filename := filepath.Base(samplePath)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("create form file failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if _, err := part.Write(audioData); err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("write audio data failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if err := writer.WriteField("model", hc.Model); err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("write model field failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if err := writer.Close(); err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("close writer failed: %v", err),
			Latency: time.Since(start),
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("create request failed: %v", err),
			Latency: time.Since(start),
		}
	}

	if hc.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+hc.APIKey)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := hc.Client.Do(req)
	if err != nil {
		return CheckResult{
			Service: "transcriptions",
			Status:  "error",
			Error:   fmt.Sprintf("request failed: %v", err),
			Latency: time.Since(start),
		}
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	// For transcriptions endpoint, check if service can handle requests
	// Consider OK if service can process request (even with errors)
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":    "transcriptions_check_success",
			"url":       url,
			"statusCode": resp.StatusCode,
			"latency":   latency.Milliseconds(),
		}).Debug("Transcriptions endpoint check successful")

		return CheckResult{
			Service: "transcriptions",
			Status:  "ok",
			Latency: latency,
		}
	}

	responseBody, _ := io.ReadAll(resp.Body)
	return CheckResult{
		Service: "transcriptions",
		Status:  "error",
		Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)),
		Latency: latency,
	}
}

// CheckASREngineHealth performs complete ASR engine health check
func (hc *HealthChecker) CheckASREngineHealth() OverallHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.WithFields(logrus.Fields{
		"component": "mont_health_chk",
		"action":    "starting_health_check",
		"baseURL":   hc.BaseURL,
	}).Info("Starting ASR engine health check")

	var checks []CheckResult

	// Execute all checks concurrently
	checkChan := make(chan CheckResult, 3)

	go func() {
		checkChan <- hc.checkHealth(ctx)
	}()

	go func() {
		checkChan <- hc.checkModels(ctx)
	}()

	go func() {
		checkChan <- hc.checkTranscriptions(ctx)
	}()

	// Collect results
	for i := 0; i < 3; i++ {
		checks = append(checks, <-checkChan)
	}

	// ASR engine is OK if at least one check succeeds
	successCount := 0
	var totalLatency time.Duration
	for _, check := range checks {
		if check.Status == "ok" {
			successCount++
		}
		totalLatency += check.Latency
	}

	avgLatency := totalLatency / time.Duration(len(checks))

	result := OverallHealth{
		ASREngineURL: hc.BaseURL,
		Checks:       checks,
	}

	// Determine overall status
	if successCount > 0 {
		result.Status = "ok"
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":        "health_check_completed",
			"status":        result.Status,
			"successCount":  successCount,
			"totalChecks":   len(checks),
			"avgLatency":    avgLatency.Milliseconds(),
		}).Info("ASR engine health check completed successfully")
	} else {
		result.Status = "error"
		result.Error = "All health checks failed"
		logger.WithFields(logrus.Fields{
			"component": "mont_health_chk",
			"action":        "health_check_failed",
			"status":        result.Status,
			"successCount":  successCount,
			"totalChecks":   len(checks),
			"avgLatency":    avgLatency.Milliseconds(),
		}).Error("ASR engine health check failed")
	}

	return result
}