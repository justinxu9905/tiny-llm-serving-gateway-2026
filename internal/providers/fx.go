package providers

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewAnthropicProvider,
			fx.ResultTags(`group:"providers"`),
			fx.As(new(Provider)),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewOpenAIProvider,
			fx.ResultTags(`group:"providers"`),
			fx.As(new(Provider)),
		),
	),
)
