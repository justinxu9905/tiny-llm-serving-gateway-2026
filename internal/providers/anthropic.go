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

const anthropicVersion = "2023-06-01"

type anthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider(cfg *config.Config) Provider {
	return &anthropicProvider{
		apiKey:  cfg.Providers.Anthropic.APIKey,
		baseURL: cfg.Providers.Anthropic.BaseURL,
		client:  &http.Client{},
	}
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) SupportsModel(model string) bool {
	return strings.HasPrefix(model, "claude")
}

func (p *anthropicProvider) doRequest(ctx context.Context, payload any, stream bool) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
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
		return nil, fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(b))
	}
	return resp, nil
}

func (p *anthropicProvider) Chat(ctx context.Context, req *entities.ChatRequest) (*entities.ChatResponse, error) {
	payload := entities.AnthropicAPIRequest{
		Model:     req.Model,
		Messages:  mappers.MessagesEntityToAnthropicAPIMessages(req.Messages),
		MaxTokens: req.MaxTokens,
		Temp:      req.Temperature,
	}
	resp, err := p.doRequest(ctx, payload, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ar entities.AnthropicAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if ar.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", ar.Error.Message)
	}
	content := ""
	for _, c := range ar.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}
	return &entities.ChatResponse{
		ID:      ar.ID,
		Model:   ar.Model,
		Content: content,
		Usage: entities.Usage{
			InputTokens:  ar.Usage.InputTokens,
			OutputTokens: ar.Usage.OutputTokens,
		},
	}, nil
}

func (p *anthropicProvider) Stream(ctx context.Context, req *entities.ChatRequest) (<-chan entities.ChatChunk, error) {
	payload := entities.AnthropicAPIRequest{
		Model:     req.Model,
		Messages:  mappers.MessagesEntityToAnthropicAPIMessages(req.Messages),
		MaxTokens: req.MaxTokens,
		Temp:      req.Temperature,
		Stream:    true,
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

func (p *anthropicProvider) consumeStream(r io.Reader, ch chan<- entities.ChatChunk) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var msgID string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var ev entities.AnthropicSSEEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "message_start":
			msgID = ev.Message.ID
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" {
				ch <- entities.ChatChunk{ID: msgID, Delta: ev.Delta.Text}
			}
		case "message_stop":
			ch <- entities.ChatChunk{ID: msgID, Done: true}
		case "error":
			if ev.Error != nil {
				ch <- entities.ChatChunk{Err: fmt.Errorf("anthropic stream error: %s", ev.Error.Message)}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- entities.ChatChunk{Err: err}
	}
}
