package provider

import "context"

// ChatMessage is a provider-agnostic message.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// CompletionRequest holds everything needed for a single LLM call.
type CompletionRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// CompletionResponse is the provider-agnostic response.
type CompletionResponse struct {
	Content      string
	FinishReason string
	TokensUsed   TokenUsage
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Provider is the interface every LLM backend must implement.
type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Healthy(ctx context.Context) bool
}
