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

// Message represents a single turn in conversation history
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

type Brain struct {
	Config *config.Config
	OpenAI *openai.Client
	Ollama *openai.Client
	Groq   *groq.Client
}

func NewBrain(cfg *config.Config) (*Brain, error) {
	b := &Brain{Config: cfg}

	// Initialize Clients only if keys/URLs are present
	if cfg.AI.OpenAIKey != "" {
		client := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIKey))
		b.OpenAI = &client
	}

	if cfg.AI.OllamaURL != "" {
		client := openai.NewClient(
			option.WithBaseURL(fmt.Sprintf("%s/v1", cfg.AI.OllamaURL)),
			option.WithAPIKey("ollama"), // Dummy key
		)
		b.Ollama = &client
	}

	if cfg.AI.GroqKey != "" {
		b.Groq = groq.NewClient(cfg.AI.GroqKey)
	}

	return b, nil
}

// ChatCompletion generates a response based on full history
func (b *Brain) ChatCompletion(ctx context.Context, agent config.AgentConfig, history []Message) (string, error) {
	provider := types.ProviderType(strings.ToLower(agent.Provider))

	switch provider {
	case types.ProviderOpenAI:
		return b.askOpenAI(ctx, b.OpenAI, agent, history)
	case types.ProviderOllama:
		return b.askOpenAI(ctx, b.Ollama, agent, history) // Ollama uses OpenAI format
	case types.ProviderGroq:
		return b.askGroq(ctx, agent, history)
	default:
		return "", fmt.Errorf("unsupported provider: %s", agent.Provider)
	}
}

func (b *Brain) askOpenAI(ctx context.Context, client *openai.Client, agent config.AgentConfig, history []Message) (string, error) {
	if client == nil {
		return "", fmt.Errorf("provider client is nil (check .env)")
	}

	// Convert internal history to OpenAI format
	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range history {
		if msg.Role == "system" {
			messages = append(messages, openai.SystemMessage(msg.Content))
		} else if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(agent.ModelName),
		Messages:    messages,
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

func (b *Brain) askGroq(ctx context.Context, agent config.AgentConfig, history []Message) (string, error) {
	if b.Groq == nil {
		return "", fmt.Errorf("groq client is nil (check .env)")
	}

	// Convert internal history to Groq format
	var messages []groq.ChatMessage
	for _, msg := range history {
		role := groq.RoleUser
		if msg.Role == "system" {
			role = groq.RoleSystem
		} else if msg.Role == "assistant" {
			role = groq.RoleAssistant
		}
		messages = append(messages, groq.ChatMessage{Role: role, Content: msg.Content})
	}

	temp := agent.Temperature
	req := groq.ChatCompletionRequest{
		Model:       agent.ModelName,
		Messages:    messages,
		Temperature: &temp,
	}

	resp, err := b.Groq.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from AI")
}
