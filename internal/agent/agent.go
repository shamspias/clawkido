package agent

import (
	"clawkido/internal/brain"
	"clawkido/internal/config"
	"clawkido/internal/types"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type Agent struct {
	Name    string
	Config  config.AgentConfig
	Brain   *brain.Brain
	History []brain.Message     // Persistent memory
	Inbox   chan types.Envelope // Messages incoming
	Router  chan types.Envelope // Messages outgoing (to other agents)
	mu      sync.Mutex
}

// New creates a stateful agent
func New(cfg config.AgentConfig, b *brain.Brain, router chan types.Envelope) *Agent {
	return &Agent{
		Name:    cfg.Name,
		Config:  cfg,
		Brain:   b,
		Inbox:   make(chan types.Envelope, 100), // Buffer for scalability
		Router:  router,
		History: []brain.Message{},
	}
}

// Start runs the Agent in its own Goroutine (lightweight thread)
func (a *Agent) Start() {
	go func() {
		for env := range a.Inbox {
			a.process(env)
		}
	}()
}

func (a *Agent) process(env types.Envelope) {
	a.mu.Lock()
	// 1. Initialize Memory with System Prompt if new
	if len(a.History) == 0 {
		a.History = append(a.History, brain.Message{Role: "system", Content: a.Config.SystemPrompt})
	}

	// 2. Add incoming message
	// If it's a handoff, we prefix the sender name so the AI knows who is talking
	userMsg := fmt.Sprintf("From %s: %s", env.Sender, env.Content)
	a.History = append(a.History, brain.Message{Role: "user", Content: userMsg})

	// 3. Create context snapshot for thread safety
	msgs := make([]brain.Message, len(a.History))
	copy(msgs, a.History)
	a.mu.Unlock()

	// 4. Think (Call LLM)
	resp, err := a.Brain.ChatCompletion(context.Background(), a.Config, msgs)
	if err != nil {
		env.ReplyChan <- fmt.Sprintf("💥 **%s** Failed: %v", a.Name, err)
		return
	}

	// 5. Update Memory with own response
	a.mu.Lock()
	a.History = append(a.History, brain.Message{Role: "assistant", Content: resp})
	a.mu.Unlock()

	// 6. Detect Handoffs: [@agent: instruction]
	handoffs := parseHandoffs(resp)
	cleanResp := cleanResponse(resp)

	// 7. Reply to User
	if cleanResp != "" {
		env.ReplyChan <- fmt.Sprintf("🤖 **%s**: %s", a.Name, cleanResp)
	}

	// 8. Execute Handoffs
	for target, instruction := range handoffs {
		// Send a new Envelope to the Swarm Router
		a.Router <- types.Envelope{
			Sender:    a.Name,
			Content:   fmt.Sprintf("@%s %s", target, instruction), // Re-tag for routing
			ReplyChan: env.ReplyChan,                              // Pass the user's channel so the next agent replies to the user
		}
	}
}

// Extract [@agent: instruction]
func parseHandoffs(text string) map[string]string {
	re := regexp.MustCompile(`\[@(\w+):\s*([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(text, -1)
	results := make(map[string]string)
	for _, m := range matches {
		results[m[1]] = m[2]
	}
	return results
}

// Remove tags from text to keep chat clean
func cleanResponse(text string) string {
	re := regexp.MustCompile(`\[@(\w+):\s*([^\]]+)\]`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}
