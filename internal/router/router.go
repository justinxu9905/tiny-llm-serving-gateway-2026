package router

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/entities"
	providers "github.com/xuzixiang/tiny-llm-serving-gateway/internal/providers"
)

type Router struct {
	providers []providers.Provider
}

type Params struct {
	fx.In
	Providers []providers.Provider `group:"providers"`
}

func New(p Params) *Router {
	return &Router{providers: p.Providers}
}

func (r *Router) route(model string) (providers.Provider, error) {
	for _, p := range r.providers {
		if p.SupportsModel(model) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no provider found for model %q", model)
}

func (r *Router) Chat(ctx context.Context, req *entities.ChatRequest) (*entities.ChatResponse, error) {
	p, err := r.route(req.Model)
	if err != nil {
		return nil, err
	}
	return p.Chat(ctx, req)
}

func (r *Router) Stream(ctx context.Context, req *entities.ChatRequest) (<-chan entities.ChatChunk, error) {
	p, err := r.route(req.Model)
	if err != nil {
		return nil, err
	}
	return p.Stream(ctx, req)
}
