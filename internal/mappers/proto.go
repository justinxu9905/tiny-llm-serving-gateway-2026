package mappers

import (
	gatewayv1 "github.com/xuzixiang/tiny-llm-serving-gateway/gen/gateway/v1"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/entities"
)

func ChatRequestProtoToEntity(req *gatewayv1.ChatRequest) *entities.ChatRequest {
	msgs := make([]entities.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = entities.Message{Role: m.Role, Content: m.Content}
	}
	return &entities.ChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}

func ChatResponseEntityToProto(resp *entities.ChatResponse) *gatewayv1.ChatResponse {
	return &gatewayv1.ChatResponse{
		Id:      resp.ID,
		Model:   resp.Model,
		Content: resp.Content,
		Usage: &gatewayv1.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}
}

func ChatChunkEntityToProto(chunk entities.ChatChunk) *gatewayv1.ChatChunk {
	return &gatewayv1.ChatChunk{
		Id:    chunk.ID,
		Delta: chunk.Delta,
		Done:  chunk.Done,
	}
}

func EmbedRequestProtoToEntity(req *gatewayv1.EmbedRequest) *entities.EmbedRequest {
	return &entities.EmbedRequest{
		Model: req.Model,
		Input: req.Input,
	}
}

func EmbedResponseEntityToProto(resp *entities.EmbedResponse) *gatewayv1.EmbedResponse {
	embeddings := make([]*gatewayv1.Embedding, len(resp.Embeddings))
	for i, e := range resp.Embeddings {
		embeddings[i] = &gatewayv1.Embedding{
			Index:  e.Index,
			Vector: e.Vector,
		}
	}
	return &gatewayv1.EmbedResponse{
		Model:        resp.Model,
		Embeddings:   embeddings,
		PromptTokens: resp.PromptTokens,
		TotalTokens:  resp.TotalTokens,
	}
}
