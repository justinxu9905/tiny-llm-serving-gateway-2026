package main

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	gatewayv1 "github.com/xuzixiang/tiny-llm-serving-gateway/gen/gateway/v1"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/config"
	handler "github.com/xuzixiang/tiny-llm-serving-gateway/internal/handlers"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/providers"
	"github.com/xuzixiang/tiny-llm-serving-gateway/internal/router"
)

func main() {
	fx.New(
		config.Module,
		providers.Module,
		fx.Provide(
			router.New,
			handler.New,
		),
		fx.Invoke(registerAndServe),
	).Run()
}

func registerAndServe(lc fx.Lifecycle, cfg *config.Config, h *handler.LLMHandler) {
	srv := grpc.NewServer()
	gatewayv1.RegisterGatewayServiceServer(srv, h)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
			if err != nil {
				return fmt.Errorf("listen on :%d: %w", cfg.Server.Port, err)
			}
			go func() {
				if err := srv.Serve(lis); err != nil {
					fmt.Printf("grpc serve error: %v\n", err)
				}
			}()
			fmt.Printf("gRPC server listening on :%d\n", cfg.Server.Port)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})
}
