package llm

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var TestbaseURL = "http://localhost:3001/v1/"

func TestCallOpenaiAPI(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	tests := []struct {
		name        string
		audioFile   string
		model       string
		mockHandler http.HandlerFunc
		wantErr     bool
	}{
		{
			name:      "Successful audio transcription",
			audioFile: "./samples/audio_ok.wav",
			model:     "FunAudioLLM/SenseVoiceSmall",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"text": "你好你好，我就做一个测试吧。这一个。"}`))
			},
			wantErr: false,
		},
		{
			name:      "Invalid audio file",
			audioFile: "./samples/audio_invalid.wav",
			model:     "FunAudioLLM/SenseVoiceSmall",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Invalid audio format"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			ts := httptest.NewServer(tt.mockHandler)
			defer ts.Close()

			var audioData []byte
			if !tt.wantErr {
				var err error
				audioData, err = ioutil.ReadFile(tt.audioFile)
				if err != nil {
					t.Fatalf("Failed to read audio file: %v", err)
				}
			}

			got, err := CallOpenaiAPI(audioData)

			if (err != nil) != tt.wantErr {
				t.Errorf("callOpenaiAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != "你好你好，我就做一个测试吧。这一个。" {
				t.Errorf("callOpenaiAPI() = %v, want %v", got, "你好你好，我就做一个测试吧。这一个。")
			}
		})
	}
}
