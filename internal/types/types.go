package types

import (
	"context"
	"time"
)

// ProviderType enumerates supported LLM backends.
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderOllama    ProviderType = "ollama"
	ProviderGroq      ProviderType = "groq"
	ProviderAnthropic ProviderType = "anthropic"
)

// Message represents an inbound signal from the outside world.
type Message struct {
	ID        string
	Platform  string
	ChannelID string
	Sender    string
	Content   string
	ReplyChan chan string
	Timestamp time.Time
}

// Envelope is the internal carrier for Agent-to-Agent communication.
type Envelope struct {
	ID        string
	Sender    string
	Target    string
	Content   string
	ReplyChan chan string
	Depth     int
	ParentID  string
	Ctx       context.Context
}

// LogEntry carries structured log information for the TUI.
type LogEntry struct {
	Level   string
	Source  string
	Message string
	Time    time.Time
}

// Event is dispatched through the EventBus.
type Event struct {
	Type      string
	Source    string
	Payload   interface{}
	Timestamp time.Time
}

const (
	EventAgentStarted   = "agent.started"
	EventAgentStopped   = "agent.stopped"
	EventMessageRouted  = "message.routed"
	EventHandoffCreated = "handoff.created"
	EventProviderError  = "provider.error"
	EventSkillExecuted  = "skill.executed"
)

// SkillResult holds the output of a skill execution.
type SkillResult struct {
	Name    string
	Output  string
	Error   error
	Latency time.Duration
}

// AgentHealth holds runtime health metrics for a single agent.
type AgentHealth struct {
	Name          string
	IsAlive       bool
	MessagesTotal int64
	ErrorsTotal   int64
	AvgLatency    time.Duration
	LastActiveAt  time.Time
	QueueDepth    int
	HistoryLen    int
}
