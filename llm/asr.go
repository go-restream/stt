package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/go-restream/stt/pkg/logger"
	"github.com/sirupsen/logrus"
)

var  (
	asrApiKey = os.Getenv("OPENAI_API_KEY")
	asrBaseURL = "http://localhost:3000/v1"
	asrModel = "FunAudioLLM/SenseVoiceSmall"
)

func SetAsrBaseURL(url string) {
	asrBaseURL = url
}
func SetAsrApiKey(ak string) {
	asrApiKey = ak
}
func SetAsrModel(model string) {
	asrModel = model
}

// CallOpenaiAPI calls OpenAI-compatible speech recognition API at "$BaseURL + /audio/transcriptions"
func CallOpenaiAPI(audioData []byte) (string, error) {
	startTime := time.Now()

	logger.WithFields(logrus.Fields{
		"component": "api_asr_service",
		"action":        "call_start",
		"audioSize":     len(audioData),
		"baseURL":       asrBaseURL,
		"model":         asrModel,
		"hasApiKey":     asrApiKey != "",
	}).Info("Starting ASR API call")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":    "create_form_file_failed",
			"error":     err,
		}).Error("Failed to create form file")
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	if _, err := part.Write(audioData); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":    "write_audio_data_failed",
			"error":     err,
		}).Error("Failed to write audio data")
		return "", fmt.Errorf("failed to write audio data: %v", err)
	}

	if err := writer.WriteField("model", asrModel); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":    "write_model_field_failed",
			"error":     err,
			"model":     asrModel,
		}).Error("Failed to write model field")
		return "", fmt.Errorf("failed to write model field: %v", err)
	}

	if err := writer.Close(); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":    "close_writer_failed",
			"error":     err,
		}).Error("Failed to close multipart writer")
		return "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	requestURL := asrBaseURL + "/audio/transcriptions"
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":      "create_request_failed",
			"error":       err,
			"requestURL":  requestURL,
		}).Error("Failed to create HTTP request")
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+asrApiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	logger.WithFields(logrus.Fields{
		"component": "api_asr_service",
		"action":          "sending_request",
		"requestURL":      requestURL,
		"bodySize":        body.Len(),
		"contentType":     writer.FormDataContentType(),
		"hasAuthorization": asrApiKey != "",
	}).Info("Sending ASR API request")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":      "request_failed",
			"error":       err,
			"requestURL":  requestURL,
			"duration":    time.Since(startTime).Milliseconds(),
		}).Error("ASR API request failed")
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	logger.WithFields(logrus.Fields{
		"component": "api_asr_service",
		"action":       "response_received",
		"statusCode":   resp.StatusCode,
		"status":       resp.Status,
		"duration":     time.Since(startTime).Milliseconds(),
	}).Info("ASR API response received")

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":      "api_error",
			"statusCode":  resp.StatusCode,
			"status":      resp.Status,
			"response":    string(responseBody),
			"duration":    time.Since(startTime).Milliseconds(),
		}).Error("ASR API returned error response")
		return "", fmt.Errorf("API error: %s, response: %s", resp.Status, string(responseBody))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":    "read_response_failed",
			"error":     err,
			"duration":  time.Since(startTime).Milliseconds(),
		}).Error("Failed to read ASR API response")
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	logger.WithFields(logrus.Fields{
		"component": "api_asr_service",
		"action":      "response_body_read",
		"responseSize": len(responseBody),
		"duration":    time.Since(startTime).Milliseconds(),
	}).Debug("ASR API response body read")

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "api_asr_service",
			"action":      "decode_response_failed",
			"error":       err,
			"response":    string(responseBody),
			"duration":    time.Since(startTime).Milliseconds(),
		}).Error("Failed to decode ASR API response")
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	totalDuration := time.Since(startTime)
	logger.WithFields(logrus.Fields{
		"component": "api_asr_service",
		"action":         "call_completed",
		"recognizedText": result.Text,
		"textLength":     len(result.Text),
		"totalDuration":  totalDuration.Milliseconds(),
		"audioSize":      len(audioData),
	}).Info("ASR API call completed successfully")

	return result.Text, nil
}