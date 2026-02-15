package types

import "time"

// ProviderType defines the supported AI providers
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderOllama ProviderType = "ollama"
	ProviderGroq   ProviderType = "groq"
)

// Message is the data packet moving through the system
type Message struct {
	Platform  string      // "Telegram", "Discord", "Terminal"
	Sender    string      // User ID or Name
	Content   string      // The prompt
	ReplyChan chan string // Channel to send the answer back
}

// LogEntry allows the TUI to display events without blocking the engine
type LogEntry struct {
	Level   string // INFO, ERROR, SUCCESS, DEBUG
	Source  string // "Engine", "Brain", "Router"
	Message string
	Time    time.Time
}
