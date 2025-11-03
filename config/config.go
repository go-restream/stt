package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServicePort string `yaml:"service_port"`

	ASR struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
	} `yaml:"asr"`

	LLM struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
	} `yaml:"llm"`

	// Just for testing purposes
	Audio struct {
		Enable     bool    `yaml:"enable"`
		SaveDir    string `yaml:"save_dir"`
		KeepFiles  int    `yaml:"keep_files"`
		SampleRate int    `yaml:"sample_rate"`
		Channels   int    `yaml:"channels"`
		BitDepth   int    `yaml:"bit_depth"`
		BufferSize int    `yaml:"buffer_size"`
	} `yaml:"audio"`

	Vad struct {
		Enable               bool    `yaml:"enable"`
		Model                string  `yaml:"model"`
		Threshold            float32 `yaml:"threshold"`
		MinSilenceDuration   float32 `yaml:"min_silence_duration"`
		MinSpeechDuration    float32 `yaml:"min_speech_duration"`
		WindowSize           int     `yaml:"window_size"`
		MaxSpeechDuration    float32 `yaml:"max_speech_duration"`
		SampleRate           int     `yaml:"sample_rate"`
		NumThreads           int     `yaml:"num_threads"`
		Provider             string  `yaml:"provider"`
		Debug                int     `yaml:"debug"`
		BypassForTesting     bool    `yaml:"bypass_for_testing"`
	ForceASRAfterSeconds  int    `yaml:"force_asr_after_seconds"`
	} `yaml:"vad"`

	Denoiser struct {
		Enable                bool   `yaml:"enable"`
		Model                 string `yaml:"model"`
		SampleRate            int    `yaml:"sample_rate"`
		NumThreads            int    `yaml:"num_threads"`
		Debug                 int    `yaml:"debug"`
		BypassForTesting      bool   `yaml:"bypass_for_testing"`
		MaxProcessingTimeMs   int    `yaml:"max_processing_time_ms"`
	} `yaml:"denoiser"`

	Logging struct {
		Level  string `yaml:"level"`
		File   string `yaml:"file"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

// validateFilePath safely validates file paths to prevent path traversal attacks
func validateFilePath(filePath, allowedBaseDir string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Clean the path to resolve any ".." or "." elements
	cleanPath := filepath.Clean(filePath)

	// If allowedBaseDir is provided, ensure the path is within it
	if allowedBaseDir != "" {
		absBaseDir, err := filepath.Abs(allowedBaseDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve base directory: %v", err)
		}

		// Join base dir with the relative path and clean it
		joinedPath := filepath.Join(absBaseDir, cleanPath)
		finalPath := filepath.Clean(joinedPath)

		// Ensure the final path is still within the base directory
		if !strings.HasPrefix(finalPath, absBaseDir) {
			return "", fmt.Errorf("path traversal detected: %s attempts to access outside allowed directory %s", filePath, allowedBaseDir)
		}

		return finalPath, nil
	}

	// If no base directory specified, just return the cleaned path
	return cleanPath, nil
}

func LoadConfig(path string) (*Config, error) {
		// Validate config file path to prevent path traversal
	safePath, err := validateFilePath(path, "")
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %v", err)
	}

	absPath, err := filepath.Abs(safePath)
	if err != nil {
		return nil, err
	}

		data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

		var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}