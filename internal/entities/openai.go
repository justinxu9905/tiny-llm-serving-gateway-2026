package entities

type OpenAIAPIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIAPIRequest struct {
	Model    string             `json:"model"`
	Messages []OpenAIAPIMessage `json:"messages"`
	Temp     float32            `json:"temperature,omitempty"`
	MaxToks  int32              `json:"max_tokens,omitempty"`
	Stream   bool               `json:"stream,omitempty"`
}

type OpenAIAPIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int32 `json:"prompt_tokens"`
		CompletionTokens int32 `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type OpenAIAPIChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}
