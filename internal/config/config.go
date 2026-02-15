package config

import (
	"encoding/json"
	"os"
	"strings"
)

type Config struct {
	AI       AIConfig       `json:"ai"`
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Agents   []AgentConfig  `json:"agents"`
	Teams    []TeamConfig   `json:"teams"`
}

type AIConfig struct {
	OpenAIKey string `json:"-"` // Loaded from ENV
	GroqKey   string `json:"-"` // Loaded from ENV
	OllamaURL string `json:"ollama_url"`
}

type TelegramConfig struct {
	Token        string  `json:"-"` // Loaded from ENV
	AllowedUsers []int64 `json:"allowed_users"`
}

type DiscordConfig struct {
	Token string `json:"-"` // Loaded from ENV
}

type AgentConfig struct {
	Name         string  `json:"name"`
	Provider     string  `json:"provider"`
	ModelName    string  `json:"model_name"`
	Temperature  float64 `json:"temperature"`
	SystemPrompt string  `json:"system_prompt"`
}

type TeamConfig struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func Load(path string) (*Config, error) {
	// 1. Read JSON Config
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	// 2. Inject Secrets from Environment
	cfg.AI.OpenAIKey = os.Getenv("OPENAI_API_KEY")
	cfg.AI.GroqKey = os.Getenv("GROQ_API_KEY")
	cfg.Telegram.Token = os.Getenv("TELEGRAM_BOT_TOKEN")
	cfg.Discord.Token = os.Getenv("DISCORD_BOT_TOKEN")

	// 3. Normalize Data
	for i := range cfg.Agents {
		cfg.Agents[i].Name = strings.ToLower(cfg.Agents[i].Name)
	}

	return &cfg, nil
}
