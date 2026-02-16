package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AI          AIConfig            `json:"ai"`
	Telegram    TelegramConfig      `json:"telegram"`
	Discord     DiscordConfig       `json:"discord"`
	SkillGroups map[string][]string `json:"skill_groups"`
	Agents      []AgentConfig       `json:"agents"`
	Teams       []TeamConfig        `json:"teams"`
	Swarm       SwarmConfig         `json:"swarm"`
}

type AIConfig struct {
	OpenAIKey    string `json:"-"`
	GroqKey      string `json:"-"`
	AnthropicKey string `json:"-"`
	OllamaURL    string `json:"ollama_url"`
}

type TelegramConfig struct {
	Token        string  `json:"-"`
	AllowedUsers []int64 `json:"allowed_users"`
}

type DiscordConfig struct {
	Token string `json:"-"`
}

type AgentConfig struct {
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	ModelName    string   `json:"model_name"`
	Temperature  float64  `json:"temperature"`
	SystemPrompt string   `json:"system_prompt"`
	MaxHistory   int      `json:"max_history"`
	Groups       []string `json:"groups"`
	Skills       []string `json:"skills"`
	Fallback     string   `json:"fallback"`
}

type TeamConfig struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
	Leader  string   `json:"leader"`
}

type SwarmConfig struct {
	MaxHandoffDepth int `json:"max_handoff_depth"`
	InboxBufferSize int `json:"inbox_buffer_size"`
	RouterBuffer    int `json:"router_buffer"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse json: %w", err)
	}

	// Inject secrets from environment.
	cfg.AI.OpenAIKey = os.Getenv("OPENAI_API_KEY")
	cfg.AI.GroqKey = os.Getenv("GROQ_API_KEY")
	cfg.AI.AnthropicKey = os.Getenv("ANTHROPIC_API_KEY")
	cfg.Telegram.Token = os.Getenv("TELEGRAM_BOT_TOKEN")
	cfg.Discord.Token = os.Getenv("DISCORD_BOT_TOKEN")

	cfg.applyDefaults()
	cfg.expandSkillGroups() // Merge groups into skills
	cfg.normalize()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	return &cfg, nil
}

// expandSkillGroups merges skills from 'groups' into the 'skills' list for each agent.
func (c *Config) expandSkillGroups() {
	for i := range c.Agents {
		agent := &c.Agents[i]

		// Use a map to deduplicate skills
		skillSet := make(map[string]bool)

		// 1. Add existing individual skills
		for _, s := range agent.Skills {
			skillSet[s] = true
		}

		// 2. Add skills from referenced groups
		for _, groupName := range agent.Groups {
			if groupSkills, ok := c.SkillGroups[groupName]; ok {
				for _, s := range groupSkills {
					skillSet[s] = true
				}
			}
		}

		// 3. Rebuild the Skills slice
		merged := make([]string, 0, len(skillSet))
		for s := range skillSet {
			merged = append(merged, s)
		}
		agent.Skills = merged
	}
}

func (c *Config) applyDefaults() {
	if c.Swarm.MaxHandoffDepth <= 0 {
		c.Swarm.MaxHandoffDepth = 5
	}
	if c.Swarm.InboxBufferSize <= 0 {
		c.Swarm.InboxBufferSize = 256
	}
	if c.Swarm.RouterBuffer <= 0 {
		c.Swarm.RouterBuffer = 256
	}
	for i := range c.Agents {
		if c.Agents[i].Temperature == 0 {
			c.Agents[i].Temperature = 0.7
		}
		if c.Agents[i].MaxHistory <= 0 {
			c.Agents[i].MaxHistory = 50
		}
	}
}

func (c *Config) normalize() {
	for i := range c.Agents {
		c.Agents[i].Name = strings.ToLower(strings.TrimSpace(c.Agents[i].Name))
		c.Agents[i].Provider = strings.ToLower(strings.TrimSpace(c.Agents[i].Provider))
	}
	for i := range c.Teams {
		c.Teams[i].Name = strings.ToLower(strings.TrimSpace(c.Teams[i].Name))
		for j := range c.Teams[i].Members {
			c.Teams[i].Members[j] = strings.ToLower(strings.TrimSpace(c.Teams[i].Members[j]))
		}
	}
}

func (c *Config) validate() error {
	if len(c.Agents) == 0 {
		return errors.New("at least one agent must be configured")
	}

	names := make(map[string]bool)
	for _, a := range c.Agents {
		if a.Name == "" {
			return errors.New("agent name cannot be empty")
		}
		if names[a.Name] {
			return fmt.Errorf("duplicate agent name: %s", a.Name)
		}
		names[a.Name] = true

		if a.Provider == "" {
			return fmt.Errorf("agent '%s' has no provider", a.Name)
		}
		if a.ModelName == "" {
			return fmt.Errorf("agent '%s' has no model_name", a.Name)
		}
		if a.SystemPrompt == "" {
			return fmt.Errorf("agent '%s' has no system_prompt", a.Name)
		}
	}

	for _, t := range c.Teams {
		for _, m := range t.Members {
			if !names[m] {
				return fmt.Errorf("team '%s' references unknown agent '%s'", t.Name, m)
			}
		}
	}

	return nil
}

func (c *Config) AgentByName(name string) *AgentConfig {
	name = strings.ToLower(name)
	for i := range c.Agents {
		if c.Agents[i].Name == name {
			return &c.Agents[i]
		}
	}
	return nil
}
