package brain

import (
	"clawkido/internal/config"
	"clawkido/internal/types"
	"context"
	"fmt"
	"strings"

	"github.com/algolyzer/groq-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Brain struct {
	Config *config.Config
	OpenAI *openai.Client
	Ollama *openai.Client
	Groq   *groq.Client
}

func NewBrain(cfg *config.Config) (*Brain, error) {
	b := &Brain{Config: cfg}

	// 1. Initialize OpenAI (Only if Key is present)
	if cfg.AI.OpenAIKey != "" {
		client := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIKey))
		b.OpenAI = &client
	}

	// 2. Initialize Ollama (Always if URL is present)
	if cfg.AI.OllamaURL != "" {
		client := openai.NewClient(
			option.WithBaseURL(fmt.Sprintf("%s/v1", cfg.AI.OllamaURL)),
			option.WithAPIKey("ollama"), // Dummy key required by library
		)
		b.Ollama = &client
	}

	// 3. Initialize Groq (Only if Key is present)
	if cfg.AI.GroqKey != "" {
		b.Groq = groq.NewClient(cfg.AI.GroqKey)
	}

	return b, nil
}

func (b *Brain) ProcessRequest(ctx context.Context, agentName, prompt string) (string, error) {
	// Find the agent in config
	var agent config.AgentConfig
	found := false
	for _, a := range b.Config.Agents {
		if strings.EqualFold(a.Name, agentName) {
			agent = a
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("agent '%s' not found", agentName)
	}

	// Determine Provider (Robust Case-Insensitive Cast)
	provider := types.ProviderType(strings.ToLower(agent.Provider))

	switch provider {
	case types.ProviderOpenAI:
		return b.askOpenAI(ctx, b.OpenAI, agent, prompt)
	case types.ProviderOllama:
		return b.askOpenAI(ctx, b.Ollama, agent, prompt) // Ollama uses OpenAI compatible client
	case types.ProviderGroq:
		return b.askGroq(ctx, agent, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", agent.Provider)
	}
}

func (b *Brain) askOpenAI(ctx context.Context, client *openai.Client, agent config.AgentConfig, prompt string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("provider client is not initialized (check API key in .env)")
	}

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(agent.ModelName),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(agent.SystemPrompt),
			openai.UserMessage(prompt),
		},
		Temperature: openai.Float(agent.Temperature),
	}

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from AI")
}

func (b *Brain) askGroq(ctx context.Context, agent config.AgentConfig, prompt string) (string, error) {
	if b.Groq == nil {
		return "", fmt.Errorf("groq client is not initialized (check GROQ_API_KEY in .env)")
	}

	temp := agent.Temperature
	req := groq.ChatCompletionRequest{
		Model: agent.ModelName,
		Messages: []groq.ChatMessage{
			{Role: groq.RoleSystem, Content: agent.SystemPrompt},
			{Role: groq.RoleUser, Content: prompt},
		},
		Temperature: &temp,
	}

	resp, err := b.Groq.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from Groq")
}
