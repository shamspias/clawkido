package config

import (
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AI       AIConfig       `json:"ai"`
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Agents   []AgentConfig  `json:"agents"`
	Teams    []TeamConfig   `json:"teams"`
}

type AIConfig struct {
	OpenAIKey string `json:"openai_key"`
	GroqKey   string `json:"groq_key"`
	OllamaURL string `json:"ollama_url"`
}

type TelegramConfig struct {
	Token string `json:"token"`
}

type DiscordConfig struct {
	Token string `json:"token"`
}

type AgentConfig struct {
	Name         string  `json:"name"`
	Provider     string  `json:"provider"` // "openai", "groq", "ollama"
	ModelName    string  `json:"model_name"`
	SystemPrompt string  `json:"system_prompt"`
	Temperature  float64 `json:"temperature"`
}

type TeamConfig struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func Load(path string) (*Config, error) {
	// 1. Load .env file (if present)
	_ = godotenv.Load()

	// 2. Parse the JSON config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 3. Override with Environment Variables (Security Best Practice)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.AI.OpenAIKey = key
	}
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		cfg.AI.GroqKey = key
	}
	if url := os.Getenv("OLLAMA_URL"); url != "" {
		cfg.AI.OllamaURL = url
	}
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		cfg.Discord.Token = token
	}

	return &cfg, nil
}
