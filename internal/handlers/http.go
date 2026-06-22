package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	gatewayv1 "github.com/xuzixiang/tiny-llm-serving-gateway/gen/gateway/v1"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/mappers"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/router"
)

type HttpHandler struct {
	grpc   *GrpcHandler
	router *router.Router
}

func NewHttpHandler(grpcHandler *GrpcHandler, r *router.Router) *HttpHandler {
	return &HttpHandler{grpc: grpcHandler, router: r}
}

func (h *HttpHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", h.handleChatCompletions)
	mux.HandleFunc("/v1/embeddings", h.handleEmbeddings)
	mux.HandleFunc("/v1/models", h.handleModels)
	mux.HandleFunc("/v1/models/", h.handleModel)
	return mux
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int32         `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type chatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []chunkChoice `json:"choices"`
}

type chunkChoice struct {
	Index        int        `json:"index"`
	Delta        chunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

type chunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

func (r *embeddingRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Model string          `json:"model"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Model = raw.Model

	var single string
	if err := json.Unmarshal(raw.Input, &single); err == nil {
		r.Input = []string{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(raw.Input, &many); err == nil {
		r.Input = many
		return nil
	}
	return fmt.Errorf("input must be a string or array of strings")
}

type embeddingResponse struct {
	Object string            `json:"object"`
	Data   []embeddingObject `json:"data"`
	Model  string            `json:"model"`
	Usage  embeddingUsage    `json:"usage"`
}

type embeddingObject struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int32     `json:"index"`
}

type embeddingUsage struct {
	PromptTokens int32 `json:"prompt_tokens"`
	TotalTokens  int32 `json:"total_tokens"`
}

type modelListResponse struct {
	Object string  `json:"object"`
	Data   []model `json:"data"`
}

type model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type errorResponse struct {
	Error openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (h *HttpHandler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Stream {
		h.handleChatCompletionsStream(w, r, req)
		return
	}

	resp, err := h.grpc.Chat(r.Context(), chatRequestToProto(req))
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id := resp.Id
	if id == "" {
		id = fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	}
	writeJSON(w, http.StatusOK, chatCompletionResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []chatChoice{{
			Index: 0,
			Message: chatMessage{
				Role:    "assistant",
				Content: resp.Content,
			},
			FinishReason: "stop",
		}},
		Usage: chatUsage{
			PromptTokens:     resp.Usage.GetInputTokens(),
			CompletionTokens: resp.Usage.GetOutputTokens(),
			TotalTokens:      resp.Usage.GetInputTokens() + resp.Usage.GetOutputTokens(),
		},
	})
}

func (h *HttpHandler) handleChatCompletionsStream(w http.ResponseWriter, r *http.Request, req chatCompletionRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	chunks, err := h.router.Stream(r.Context(), mappers.ChatRequestProtoToEntity(chatRequestToProto(req)))
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	created := time.Now().Unix()
	wroteRole := false
	for chunk := range chunks {
		if chunk.Err != nil {
			writeSSE(w, errorResponse{Error: openAIError{Message: chunk.Err.Error(), Type: "server_error"}})
			flusher.Flush()
			return
		}
		if chunk.Done {
			reason := "stop"
			writeSSE(w, chatCompletionChunk{
				ID:      chunk.ID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   req.Model,
				Choices: []chunkChoice{{
					Index:        0,
					Delta:        chunkDelta{},
					FinishReason: &reason,
				}},
			})
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		delta := chunkDelta{Content: chunk.Delta}
		if !wroteRole {
			delta.Role = "assistant"
			wroteRole = true
		}
		writeSSE(w, chatCompletionChunk{
			ID:      chunk.ID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []chunkChoice{{
				Index: 0,
				Delta: delta,
			}},
		})
		flusher.Flush()
	}
}

func (h *HttpHandler) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req embeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.grpc.Embed(r.Context(), &gatewayv1.EmbedRequest{
		Model: req.Model,
		Input: req.Input,
	})
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := make([]embeddingObject, len(resp.Embeddings))
	for i, e := range resp.Embeddings {
		data[i] = embeddingObject{
			Object:    "embedding",
			Embedding: e.Vector,
			Index:     e.Index,
		}
	}
	writeJSON(w, http.StatusOK, embeddingResponse{
		Object: "list",
		Data:   data,
		Model:  resp.Model,
		Usage: embeddingUsage{
			PromptTokens: resp.PromptTokens,
			TotalTokens:  resp.TotalTokens,
		},
	})
}

func (h *HttpHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, modelListResponse{
		Object: "list",
		Data:   supportedModels(),
	})
}

func (h *HttpHandler) handleModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/models/")
	for _, model := range supportedModels() {
		if model.ID == id {
			writeJSON(w, http.StatusOK, model)
			return
		}
	}
	if owner, ok := supportedModelOwner(id); ok {
		writeJSON(w, http.StatusOK, model{
			ID:      id,
			Object:  "model",
			Created: 0,
			OwnedBy: owner,
		})
		return
	}
	writeOpenAIError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", id))
}

func chatRequestToProto(req chatCompletionRequest) *gatewayv1.ChatRequest {
	msgs := make([]*gatewayv1.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = &gatewayv1.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return &gatewayv1.ChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}

func supportedModels() []model {
	created := int64(0)
	return []model{
		{ID: "gpt-*", Object: "model", Created: created, OwnedBy: "openai"},
		{ID: "o1*", Object: "model", Created: created, OwnedBy: "openai"},
		{ID: "o3*", Object: "model", Created: created, OwnedBy: "openai"},
		{ID: "o4*", Object: "model", Created: created, OwnedBy: "openai"},
		{ID: "text-embedding-*", Object: "model", Created: created, OwnedBy: "openai"},
		{ID: "claude*", Object: "model", Created: created, OwnedBy: "anthropic"},
	}
}

func supportedModelOwner(id string) (string, bool) {
	switch {
	case strings.HasPrefix(id, "gpt"),
		strings.HasPrefix(id, "o1"),
		strings.HasPrefix(id, "o3"),
		strings.HasPrefix(id, "o4"),
		strings.HasPrefix(id, "text-embedding"):
		return "openai", true
	case strings.HasPrefix(id, "claude"):
		return "anthropic", true
	default:
		return "", false
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOpenAIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{
		Error: openAIError{
			Message: msg,
			Type:    "invalid_request_error",
		},
	})
}

func writeSSE(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
}
