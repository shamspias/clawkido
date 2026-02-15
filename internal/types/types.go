package types

import "time"

type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderOllama ProviderType = "ollama"
	ProviderGroq   ProviderType = "groq"
)

// Message represents an input from the outside world (User)
type Message struct {
	Platform  string
	Sender    string
	Content   string
	ReplyChan chan string
}

// Envelope is the internal carrier for Agent-to-Agent communication
type Envelope struct {
	Sender    string      // Name of the sender (User or Agent)
	Content   string      // The instruction/prompt
	ReplyChan chan string // The channel to send the final result back to the user
}

// LogEntry for the TUI Dashboard
type LogEntry struct {
	Level   string // INFO, ERROR, SUCCESS, DEBUG
	Source  string // "Swarm", "Agent:Coder", "Telegram"
	Message string
	Time    time.Time
}
