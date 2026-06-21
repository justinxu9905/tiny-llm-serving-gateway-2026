package entities

type AnthropicAPIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicAPIRequest struct {
	Model     string                `json:"model"`
	Messages  []AnthropicAPIMessage `json:"messages"`
	MaxTokens int32                 `json:"max_tokens"`
	Temp      float32               `json:"temperature,omitempty"`
	Stream    bool                  `json:"stream,omitempty"`
}

type AnthropicAPIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int32 `json:"input_tokens"`
		OutputTokens int32 `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type AnthropicSSEEvent struct {
	Type    string `json:"type"`
	Message struct {
		ID string `json:"id"`
	} `json:"message"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
