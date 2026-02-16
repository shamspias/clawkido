package agent

import (
	"clawkido/internal/brain"
	"clawkido/internal/config"
	"clawkido/internal/eventbus"
	"clawkido/internal/skills"
	"clawkido/internal/types"
	"clawkido/pkg/provider"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Agent is a stateful actor that maintains its own conversation memory,
// processes messages in its own goroutine, and can hand off to other agents.
type Agent struct {
	Name     string
	Config   config.AgentConfig
	Brain    *brain.Brain
	Skills   *skills.Registry
	EventBus *eventbus.Bus
	Inbox    chan types.Envelope
	Router   chan types.Envelope
	LogChan  chan<- types.LogEntry

	mu      sync.Mutex
	history []provider.ChatMessage

	messagesTotal atomic.Int64
	errorsTotal   atomic.Int64
	totalLatency  atomic.Int64 // nanoseconds
	lastActiveAt  atomic.Int64 // unix nano

	maxDepth int
	cancel   context.CancelFunc
}

func New(
	cfg config.AgentConfig,
	b *brain.Brain,
	router chan types.Envelope,
	logChan chan<- types.LogEntry,
	skillReg *skills.Registry,
	bus *eventbus.Bus,
	maxDepth int,
) *Agent {
	return &Agent{
		Name:     cfg.Name,
		Config:   cfg,
		Brain:    b,
		Skills:   skillReg,
		EventBus: bus,
		Inbox:    make(chan types.Envelope, 128),
		Router:   router,
		LogChan:  logChan,
		maxDepth: maxDepth,
	}
}

// Start boots the agent's processing loop.
func (a *Agent) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	a.cancel = cancel

	go func() {
		defer a.log("INFO", "Shutting down")
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-a.Inbox:
				if !ok {
					return
				}
				a.process(ctx, env)
			}
		}
	}()

	a.EventBus.Publish(types.Event{
		Type: types.EventAgentStarted, Source: a.Name, Timestamp: time.Now(),
	})
}

func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.EventBus.Publish(types.Event{
		Type: types.EventAgentStopped, Source: a.Name, Timestamp: time.Now(),
	})
}

func (a *Agent) Health() types.AgentHealth {
	a.mu.Lock()
	histLen := len(a.history)
	a.mu.Unlock()

	total := a.messagesTotal.Load()
	var avg time.Duration
	if total > 0 {
		avg = time.Duration(a.totalLatency.Load() / total)
	}

	return types.AgentHealth{
		Name:          a.Name,
		IsAlive:       true,
		MessagesTotal: total,
		ErrorsTotal:   a.errorsTotal.Load(),
		AvgLatency:    avg,
		LastActiveAt:  time.Unix(0, a.lastActiveAt.Load()),
		QueueDepth:    len(a.Inbox),
		HistoryLen:    histLen,
	}
}

func (a *Agent) process(ctx context.Context, env types.Envelope) {
	start := time.Now()
	a.messagesTotal.Add(1)
	a.lastActiveAt.Store(time.Now().UnixNano())

	a.mu.Lock()

	// Initialize with system prompt on first message.
	if len(a.history) == 0 {
		sysPrompt := a.buildSystemPrompt()
		a.history = append(a.history, provider.ChatMessage{Role: "system", Content: sysPrompt})
	}

	// Append incoming message with sender context.
	userMsg := fmt.Sprintf("[From %s]: %s", env.Sender, env.Content)
	a.history = append(a.history, provider.ChatMessage{Role: "user", Content: userMsg})

	a.trimHistory()

	// Snapshot for thread safety.
	snapshot := make([]provider.ChatMessage, len(a.history))
	copy(snapshot, a.history)
	a.mu.Unlock()

	// Call LLM.
	resp, err := a.Brain.ChatCompletion(ctx, a.Config, snapshot)
	if err != nil {
		a.errorsTotal.Add(1)
		a.log("ERROR", fmt.Sprintf("LLM call failed: %v", err))
		a.safeSend(env.ReplyChan, fmt.Sprintf("💥 **%s** error: %v", a.Name, err))
		return
	}

	content := resp.Content
	a.totalLatency.Add(int64(time.Since(start)))

	a.mu.Lock()
	a.history = append(a.history, provider.ChatMessage{Role: "assistant", Content: content})
	a.mu.Unlock()

	// Execute skill calls: [!skill_name: args]
	content = a.executeSkills(ctx, content)

	// Parse handoff tags: [@agent_name: instruction]
	handoffs := parseHandoffs(content)
	cleanResp := cleanResponse(content)

	// Reply to user with cleaned response.
	if cleanResp != "" {
		a.safeSend(env.ReplyChan, fmt.Sprintf("🤖 **%s**: %s", a.Name, cleanResp))
	}

	// Execute handoffs with depth check.
	for target, instruction := range handoffs {
		if env.Depth >= a.maxDepth {
			a.log("WARN", fmt.Sprintf("Max handoff depth (%d) reached, blocking handoff to @%s", a.maxDepth, target))
			a.safeSend(env.ReplyChan, fmt.Sprintf("⚠️ **%s**: Handoff chain too deep (max %d). Stopping.", a.Name, a.maxDepth))
			continue
		}

		a.log("INFO", fmt.Sprintf("Handing off to @%s (depth=%d)", target, env.Depth+1))

		a.Router <- types.Envelope{
			Sender:    a.Name,
			Content:   fmt.Sprintf("@%s %s", target, instruction),
			ReplyChan: env.ReplyChan,
			Depth:     env.Depth + 1,
			ParentID:  env.ID,
			Ctx:       ctx,
		}

		a.EventBus.Publish(types.Event{
			Type: types.EventHandoffCreated, Source: a.Name,
			Payload:   map[string]string{"target": target},
			Timestamp: time.Now(),
		})
	}
}

func (a *Agent) buildSystemPrompt() string {
	prompt := a.Config.SystemPrompt

	if a.Skills != nil {
		skillDescs := a.Skills.List()
		if len(skillDescs) > 0 {
			prompt += "\n\nYou have access to the following skills. To use one, include it in your response:\n"
			prompt += strings.Join(skillDescs, "\n")
		}
	}

	prompt += "\n\nTo delegate tasks to other agents, use: [@agent_name: instruction]"
	return prompt
}

func (a *Agent) trimHistory() {
	maxMsgs := a.Config.MaxHistory * 2 // user + assistant per turn
	if maxMsgs <= 0 || len(a.history) <= maxMsgs+1 {
		return
	}
	trimmed := make([]provider.ChatMessage, 0, maxMsgs+1)
	trimmed = append(trimmed, a.history[0])
	trimmed = append(trimmed, a.history[len(a.history)-maxMsgs:]...)
	a.history = trimmed
}

func (a *Agent) executeSkills(ctx context.Context, text string) string {
	if a.Skills == nil {
		return text
	}

	re := regexp.MustCompile(`\[!(\w+)(?::\s*([^\]]*))?\]`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		name := sub[1]
		args := ""
		if len(sub) >= 3 {
			args = strings.TrimSpace(sub[2])
		}

		result := a.Skills.Execute(ctx, name, args, a.Name)

		a.EventBus.Publish(types.Event{
			Type: types.EventSkillExecuted, Source: a.Name,
			Payload: result, Timestamp: time.Now(),
		})

		if result.Error != nil {
			a.log("ERROR", fmt.Sprintf("Skill '%s' failed: %v", name, result.Error))
			return fmt.Sprintf("[Skill %s error: %v]", name, result.Error)
		}

		if result.Output == "__MEMORY_RESET__" {
			a.mu.Lock()
			a.history = nil
			a.mu.Unlock()
			a.log("INFO", "Memory reset by skill")
			return "[Memory cleared]"
		}

		a.log("DEBUG", fmt.Sprintf("Skill '%s' completed in %s", name, result.Latency))
		return result.Output
	})
}

func (a *Agent) safeSend(ch chan string, msg string) {
	select {
	case ch <- msg:
	default:
		a.log("WARN", "Reply channel full, dropping message")
	}
}

func (a *Agent) log(level, msg string) {
	select {
	case a.LogChan <- types.LogEntry{
		Level: level, Source: fmt.Sprintf("Agent:%s", a.Name), Message: msg, Time: time.Now(),
	}:
	default:
	}
}

// ─── Parsing Helpers ─────────────────────────────────────────────────────────

var handoffRe = regexp.MustCompile(`\[@(\w+):\s*([^\]]+)\]`)

func parseHandoffs(text string) map[string]string {
	matches := handoffRe.FindAllStringSubmatch(text, -1)
	results := make(map[string]string, len(matches))
	for _, m := range matches {
		results[m[1]] = strings.TrimSpace(m[2])
	}
	return results
}

func cleanResponse(text string) string {
	text = handoffRe.ReplaceAllString(text, "")
	skillRe := regexp.MustCompile(`\[!(\w+)(?::\s*([^\]]*))?\]`)
	text = skillRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}
