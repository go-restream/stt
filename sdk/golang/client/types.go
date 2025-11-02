package asr

import "time"

type RecognitionListener interface {
	OnRecognitionStart(*RecognitionResponse)
	OnSentenceBegin(*RecognitionResponse) 
	OnRecognitionResultChange(*RecognitionResponse)
	OnSentenceEnd(*RecognitionResponse)
	OnRecognitionComplete(*RecognitionResponse)
	OnFail(*RecognitionResponse, error)
}

type RecognitionResponse struct {
	Code      int                 `json:"code"`
	Message   string              `json:"message"`
	VoiceID   string              `json:"voice_id,omitempty"`
	Final     bool                `json:"final,omitempty"`
	Result    RecognitionResult   `json:"result,omitempty"`
}

type RecognitionResult struct {
	SliceType    int                `json:"slice_type"`
	Index        int                `json:"index"`
	StartTime    time.Duration      `json:"start_time"`
	EndTime      time.Duration      `json:"end_time"`
	Text         string             `json:"text"`
	WordList     []RecognitionWord  `json:"word_list,omitempty"`
}

type RecognitionWord struct {
	Word       string        `json:"word"`
	StartTime  time.Duration `json:"start_time"`
	EndTime    time.Duration `json:"end_time"`
	Confidence float32       `json:"confidence,omitempty"`
}

const (
	AudioFormatPCM = 1
	AudioFormatWAV = 2
)