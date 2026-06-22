package entities

type Message struct {
	Role    string
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float32
	MaxTokens   int32
}

type ChatResponse struct {
	ID      string
	Model   string
	Content string
	Usage   Usage
}

// ChatChunk is a single streamed token delta. Err is set if streaming failed mid-flight.
type ChatChunk struct {
	ID    string
	Delta string
	Done  bool
	Err   error
}

type Usage struct {
	InputTokens  int32
	OutputTokens int32
}

type EmbedRequest struct {
	Model string
	Input []string
}

type Embedding struct {
	Index  int32
	Vector []float32
}

type EmbedResponse struct {
	Model        string
	Embeddings   []Embedding
	PromptTokens int32
	TotalTokens  int32
}
