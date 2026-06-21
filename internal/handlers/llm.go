package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gatewayv1 "github.com/xuzixiang/tiny-llm-serving-gateway/gen/gateway/v1"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/mappers"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/router"
)

type LLMHandler struct {
	gatewayv1.UnimplementedGatewayServiceServer
	router *router.Router
}

func New(r *router.Router) *LLMHandler {
	return &LLMHandler{router: r}
}

func (h *LLMHandler) Chat(ctx context.Context, req *gatewayv1.ChatRequest) (*gatewayv1.ChatResponse, error) {
	resp, err := h.router.Chat(ctx, mappers.ChatRequestProtoToEntity(req))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mappers.ChatResponseEntityToProto(resp), nil
}

func (h *LLMHandler) ChatStream(req *gatewayv1.ChatRequest, stream gatewayv1.GatewayService_ChatStreamServer) error {
	chunks, err := h.router.Stream(stream.Context(), mappers.ChatRequestProtoToEntity(req))
	if err != nil {
		return status.Errorf(codes.Internal, "%v", err)
	}
	for chunk := range chunks {
		if chunk.Err != nil {
			return status.Errorf(codes.Internal, "%v", chunk.Err)
		}
		if err := stream.Send(mappers.ChatChunkEntityToProto(chunk)); err != nil {
			return err
		}
	}
	return nil
}
