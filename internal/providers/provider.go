package providers

import (
	"context"

	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/entities"
)

type Provider interface {
	Name() string
	// SupportsModel returns true if this provider should handle the given model name.
	SupportsModel(model string) bool
	Chat(ctx context.Context, req *entities.ChatRequest) (*entities.ChatResponse, error)
	// Stream returns a channel that emits chunks until Done=true or Err is set.
	Stream(ctx context.Context, req *entities.ChatRequest) (<-chan entities.ChatChunk, error)
}
