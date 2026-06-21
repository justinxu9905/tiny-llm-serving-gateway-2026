package mappers

import "github.com/xuzixiang/tiny-llm-serving-gateway/internal/entities"

func MessagesEntityToOpenAIAPIMessages(msgs []entities.Message) []entities.OpenAIAPIMessage {
	out := make([]entities.OpenAIAPIMessage, len(msgs))
	for i, m := range msgs {
		out[i] = entities.OpenAIAPIMessage{Role: m.Role, Content: m.Content}
	}
	return out
}

func MessagesEntityToAnthropicAPIMessages(msgs []entities.Message) []entities.AnthropicAPIMessage {
	out := make([]entities.AnthropicAPIMessage, len(msgs))
	for i, m := range msgs {
		out[i] = entities.AnthropicAPIMessage{Role: m.Role, Content: m.Content}
	}
	return out
}
