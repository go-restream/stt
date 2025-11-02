package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	openAIAPIURL = "https://api.deepseek.com/v1"
	defaultTimeout = 30 * time.Second
)


type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	MaxTokens int         `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP float64         `json:"top_p,omitempty"`
	N int                `json:"n,omitempty"`
	Stream bool          `json:"stream,omitempty"`
	Stop []string       `json:"stop,omitempty"`
	PresencePenalty float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`
	User string         `json:"user,omitempty"`
}


type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}


type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		Index        int         `json:"index"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}


type LLMClient interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)
	CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
}


func NewClient(apiKey string) LLMClient {
	return &openAIClient{
		apiKey:  apiKey,
		baseURL: openAIAPIURL,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

type openAIClient struct {
	apiKey    string
	baseURL   string
	client    *http.Client
}

func (c *openAIClient) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	url := c.baseURL + path
	
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request failed: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s, body: %s", resp.Status, string(errorBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *openAIClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	path := "/chat/completions"
	respData, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &response, nil
}

func (c *openAIClient) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	path := "/completions"
	respData, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	var response CompletionResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &response, nil
}

func (c *openAIClient) CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	path := "/embeddings"
	respData, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	var response EmbeddingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &response, nil
}