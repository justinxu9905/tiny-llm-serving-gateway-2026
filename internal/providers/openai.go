package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/config"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/entities"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/mappers"
)

type openaiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(cfg *config.Config) Provider {
	return &openaiProvider{
		apiKey:  cfg.Providers.OpenAI.APIKey,
		baseURL: cfg.Providers.OpenAI.BaseURL,
		client:  &http.Client{},
	}
}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) SupportsModel(model string) bool {
	return strings.HasPrefix(model, "gpt") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4")
}

func (p *openaiProvider) doRequest(ctx context.Context, payload any, stream bool) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+p.apiKey)
	req.Header.Set("content-type", "application/json")
	if stream {
		req.Header.Set("accept", "text/event-stream")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai %d: %s", resp.StatusCode, string(b))
	}
	return resp, nil
}

func (p *openaiProvider) Chat(ctx context.Context, req *entities.ChatRequest) (*entities.ChatResponse, error) {
	payload := entities.OpenAIAPIRequest{
		Model:    req.Model,
		Messages: mappers.MessagesEntityToOpenAIAPIMessages(req.Messages),
		Temp:     req.Temperature,
		MaxToks:  req.MaxTokens,
	}
	resp, err := p.doRequest(ctx, payload, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ar entities.OpenAIAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if ar.Error != nil {
		return nil, fmt.Errorf("openai error: %s", ar.Error.Message)
	}
	content := ""
	if len(ar.Choices) > 0 {
		content = ar.Choices[0].Message.Content
	}
	return &entities.ChatResponse{
		ID:      ar.ID,
		Model:   ar.Model,
		Content: content,
		Usage: entities.Usage{
			InputTokens:  ar.Usage.PromptTokens,
			OutputTokens: ar.Usage.CompletionTokens,
		},
	}, nil
}

func (p *openaiProvider) Stream(ctx context.Context, req *entities.ChatRequest) (<-chan entities.ChatChunk, error) {
	payload := entities.OpenAIAPIRequest{
		Model:    req.Model,
		Messages: mappers.MessagesEntityToOpenAIAPIMessages(req.Messages),
		Temp:     req.Temperature,
		MaxToks:  req.MaxTokens,
		Stream:   true,
	}
	resp, err := p.doRequest(ctx, payload, true)
	if err != nil {
		return nil, err
	}

	ch := make(chan entities.ChatChunk, 16)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		p.consumeStream(resp.Body, ch)
	}()
	return ch, nil
}

func (p *openaiProvider) consumeStream(r io.Reader, ch chan<- entities.ChatChunk) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- entities.ChatChunk{Done: true}
			return
		}

		var chunk entities.OpenAIAPIChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta != "" {
			ch <- entities.ChatChunk{ID: chunk.ID, Delta: delta}
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- entities.ChatChunk{Err: err}
	}
}
