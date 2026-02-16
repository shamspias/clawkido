package brain

import (
	"clawkido/internal/config"
	"clawkido/internal/middleware"
	"clawkido/pkg/provider"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/algolyzer/groq-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Brain manages all LLM providers and dispatches calls with retry/fallback.
type Brain struct {
	providers map[string]provider.Provider
	retryCfg  middleware.RetryConfig
	mu        sync.RWMutex
}

func NewBrain(cfg *config.Config) (*Brain, error) {
	b := &Brain{
		providers: make(map[string]provider.Provider),
		retryCfg:  middleware.DefaultRetryConfig(),
	}

	if cfg.AI.OpenAIKey != "" {
		client := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIKey))
		b.Register(&OpenAIProvider{client: &client, name: "openai"})
	}

	if cfg.AI.OllamaURL != "" {
		client := openai.NewClient(
			option.WithBaseURL(fmt.Sprintf("%s/v1", strings.TrimRight(cfg.AI.OllamaURL, "/"))),
			option.WithAPIKey("ollama"),
		)
		b.Register(&OpenAIProvider{client: &client, name: "ollama"})
	}

	if cfg.AI.GroqKey != "" {
		b.Register(&GroqProvider{client: groq.NewClient(cfg.AI.GroqKey)})
	}

	return b, nil
}

func (b *Brain) Register(p provider.Provider) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.providers[p.Name()] = p
}

func (b *Brain) GetProvider(name string) (provider.Provider, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	p, ok := b.providers[strings.ToLower(name)]
	return p, ok
}

func (b *Brain) ChatCompletion(ctx context.Context, agent config.AgentConfig, messages []provider.ChatMessage) (provider.CompletionResponse, error) {
	req := provider.CompletionRequest{
		Model:       agent.ModelName,
		Messages:    messages,
		Temperature: agent.Temperature,
	}

	resp, err := b.callWithRetry(ctx, agent.Provider, req)
	if err == nil {
		return resp, nil
	}

	if agent.Fallback != "" && agent.Fallback != agent.Provider {
		resp, fallbackErr := b.callWithRetry(ctx, agent.Fallback, req)
		if fallbackErr == nil {
			return resp, nil
		}
		return provider.CompletionResponse{}, fmt.Errorf(
			"primary (%s): %w; fallback (%s): %v", agent.Provider, err, agent.Fallback, fallbackErr,
		)
	}

	return provider.CompletionResponse{}, err
}

func (b *Brain) callWithRetry(ctx context.Context, providerName string, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	p, ok := b.GetProvider(providerName)
	if !ok {
		return provider.CompletionResponse{}, fmt.Errorf("provider '%s' not registered (check .env)", providerName)
	}

	var resp provider.CompletionResponse
	err := middleware.Retry(ctx, b.retryCfg, func() error {
		var callErr error
		resp, callErr = p.ChatCompletion(ctx, req)
		return callErr
	})
	return resp, err
}

func (b *Brain) AvailableProviders() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.providers))
	for name := range b.providers {
		names = append(names, name)
	}
	return names
}

// ─── OpenAI-compatible Provider (OpenAI + Ollama) ────────────────────────────

type OpenAIProvider struct {
	client *openai.Client
	name   string
}

func (p *OpenAIProvider) Name() string { return p.name }

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	if p.client == nil {
		return provider.CompletionResponse{}, fmt.Errorf("%s: client is nil", p.name)
	}

	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		default:
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(req.Model),
		Messages:    messages,
		Temperature: openai.Float(req.Temperature),
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return provider.CompletionResponse{}, fmt.Errorf("%s: %w", p.name, err)
	}
	if len(resp.Choices) == 0 {
		return provider.CompletionResponse{}, fmt.Errorf("%s: empty response", p.name)
	}

	return provider.CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		TokensUsed: provider.TokenUsage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}, nil
}

func (p *OpenAIProvider) Healthy(ctx context.Context) bool {
	_, err := p.ChatCompletion(ctx, provider.CompletionRequest{
		Model:       "gpt-4o-mini",
		Messages:    []provider.ChatMessage{{Role: "user", Content: "ping"}},
		Temperature: 0,
	})
	return err == nil
}

// ─── Groq Provider ───────────────────────────────────────────────────────────

type GroqProvider struct {
	client *groq.Client
}

func (p *GroqProvider) Name() string { return "groq" }

func (p *GroqProvider) ChatCompletion(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	if p.client == nil {
		return provider.CompletionResponse{}, fmt.Errorf("groq: client is nil")
	}

	var messages []groq.ChatMessage
	for _, msg := range req.Messages {
		role := groq.RoleUser
		switch msg.Role {
		case "system":
			role = groq.RoleSystem
		case "assistant":
			role = groq.RoleAssistant
		}
		messages = append(messages, groq.ChatMessage{Role: role, Content: msg.Content})
	}

	temp := req.Temperature
	greq := groq.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: &temp,
	}

	resp, err := p.client.CreateChatCompletion(ctx, greq)
	if err != nil {
		return provider.CompletionResponse{}, fmt.Errorf("groq: %w", err)
	}
	if len(resp.Choices) == 0 {
		return provider.CompletionResponse{}, fmt.Errorf("groq: empty response")
	}

	return provider.CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: resp.Choices[0].FinishReason,
		TokensUsed: provider.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (p *GroqProvider) Healthy(ctx context.Context) bool {
	_, err := p.ChatCompletion(ctx, provider.CompletionRequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    []provider.ChatMessage{{Role: "user", Content: "ping"}},
		Temperature: 0,
	})
	return err == nil
}
