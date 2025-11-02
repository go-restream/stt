package config

import (
	"os"
	"path/filepath"

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

	Logging struct {
		Level  string `yaml:"level"`
		File   string `yaml:"file"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

func LoadConfig(path string) (*Config, error) {
		absPath, err := filepath.Abs(path)
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