package llm

import (
	"context"
	"testing"
)

const (
	testAPIKey = "sk-xxxxx-xxxxxxxxxxxxxxxxx"
	testAPIURL = "https://api.deepseek.com/v1"
	testModel  = "deepseek-coder"
)

func TestChatCompletion(t *testing.T) {
	client := NewMockClient()
	
	req := ChatCompletionRequest{
		Model: testModel,
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateChatCompletion failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice in response")
	}
}

func TestCompletion(t *testing.T) {
	client := NewMockClient()
	
	req := CompletionRequest{
		Model:  testModel,
		Prompt: "Once upon a time",
	}

	resp, err := client.CreateCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateCompletion failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice in response")
	}
}

type mockClient struct{}

func (c *mockClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	return &ChatCompletionResponse{
		ID:      "mock-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []struct {
			Message      ChatMessage `json:"message"`
			Index        int         `json:"index"`
			FinishReason string      `json:"finish_reason"`
		}{
			{
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello! How can I help you today?",
				},
				Index:        0,
				FinishReason: "stop",
			},
		},
	}, nil
}

func (c *mockClient) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{
		ID:      "mock-id",
		Object:  "text_completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []struct {
			Text         string `json:"text"`
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Text:         "Once upon a time, in a land far far away...",
				Index:        0,
				FinishReason: "stop",
			},
		},
	}, nil
}

func (c *mockClient) CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return &EmbeddingResponse{
		Object: "list",
		Data: []struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}{
			{
				Object:    "embedding",
				Embedding: make([]float64, 1536),
				Index:     0,
			},
		},
		Model: req.Model,
	}, nil
}

func NewMockClient() LLMClient {
	return &mockClient{}
}