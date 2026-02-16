package swarm

import (
	"clawkido/internal/agent"
	"clawkido/internal/brain"
	"clawkido/internal/config"
	"clawkido/internal/eventbus"
	"clawkido/internal/health"
	"clawkido/internal/skills"
	"clawkido/internal/types"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// Swarm is the central orchestration engine.
type Swarm struct {
	Inbox    chan types.Message
	Router   chan types.Envelope
	LogChan  chan types.LogEntry
	Agents   map[string]*agent.Agent
	Config   *config.Config
	EventBus *eventbus.Bus
	Health   *health.Monitor
	Skills   *skills.Registry

	routedTotal  atomic.Int64
	droppedTotal atomic.Int64
}

func NewSwarm(
	cfg *config.Config,
	b *brain.Brain,
	logChan chan types.LogEntry,
	bus *eventbus.Bus,
	skillReg *skills.Registry,
) *Swarm {
	s := &Swarm{
		Inbox:    make(chan types.Message, cfg.Swarm.InboxBufferSize),
		Router:   make(chan types.Envelope, cfg.Swarm.RouterBuffer),
		LogChan:  logChan,
		Agents:   make(map[string]*agent.Agent),
		Config:   cfg,
		EventBus: bus,
		Skills:   skillReg,
	}

	for _, agentCfg := range cfg.Agents {
		a := agent.New(agentCfg, b, s.Router, logChan, skillReg, bus, cfg.Swarm.MaxHandoffDepth)
		s.Agents[strings.ToLower(agentCfg.Name)] = a
		s.log("INFO", "Swarm", fmt.Sprintf("Agent '%s' registered [provider=%s model=%s]",
			agentCfg.Name, agentCfg.Provider, agentCfg.ModelName))
	}

	s.Health = health.NewMonitor(s.Agents, logChan)
	return s
}

func (s *Swarm) Start(ctx context.Context) {
	// Boot all agents.
	for _, a := range s.Agents {
		a.Start(ctx)
	}

	s.log("INFO", "Swarm", fmt.Sprintf("Hive active: %d agents, %d teams, providers: %s",
		len(s.Agents), len(s.Config.Teams), s.providerSummary()))

	// External message processor (User -> Swarm).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-s.Inbox:
				if !ok {
					return
				}
				s.log("INFO", msg.Platform, fmt.Sprintf("@%s: %s", msg.Sender, truncate(msg.Content, 100)))
				s.route(ctx, types.Envelope{
					Sender:    msg.Sender,
					Content:   msg.Content,
					ReplyChan: msg.ReplyChan,
					Depth:     0,
					Ctx:       ctx,
				})
			}
		}
	}()

	// Internal routing processor (Agent -> Swarm -> Agent).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-s.Router:
				if !ok {
					return
				}
				s.log("DEBUG", "Router", fmt.Sprintf("Handoff from '%s' (depth=%d)", env.Sender, env.Depth))
				s.route(ctx, env)
			}
		}
	}()

	// Health monitor.
	go s.Health.Start(ctx, 10*time.Second)
}

func (s *Swarm) Stop() {
	for _, a := range s.Agents {
		a.Stop()
	}
	s.log("INFO", "Swarm", "All agents stopped")
}

func (s *Swarm) route(ctx context.Context, env types.Envelope) {
	targetName, content := parseTarget(env.Content)

	if targetName == "" {
		if len(s.Config.Agents) > 0 {
			targetName = s.Config.Agents[0].Name
		} else {
			s.safeSend(env.ReplyChan, "🚫 No agents configured")
			return
		}
	}

	targetName = strings.ToLower(targetName)
	env.Content = content

	// Direct agent routing.
	if a, exists := s.Agents[targetName]; exists {
		select {
		case a.Inbox <- env:
			s.routedTotal.Add(1)
			s.log("SUCCESS", "Router", fmt.Sprintf("→ '%s'", targetName))
		default:
			s.droppedTotal.Add(1)
			s.log("WARN", "Router", fmt.Sprintf("Agent '%s' inbox full, message dropped", targetName))
			s.safeSend(env.ReplyChan, fmt.Sprintf("⚠️ Agent @%s is overloaded. Try again.", targetName))
		}

		s.EventBus.Publish(types.Event{
			Type: types.EventMessageRouted, Source: "Swarm",
			Payload:   map[string]string{"target": targetName, "sender": env.Sender},
			Timestamp: time.Now(),
		})
		return
	}

	// Team broadcast.
	for _, team := range s.Config.Teams {
		if strings.EqualFold(team.Name, targetName) {
			s.log("INFO", "Router", fmt.Sprintf("Broadcasting to team '%s' (%d members)", team.Name, len(team.Members)))
			for _, memberName := range team.Members {
				if a, exists := s.Agents[strings.ToLower(memberName)]; exists {
					clone := types.Envelope{
						Sender:    env.Sender,
						Content:   env.Content,
						ReplyChan: env.ReplyChan,
						Depth:     env.Depth,
						ParentID:  env.ParentID,
						Ctx:       ctx,
					}
					select {
					case a.Inbox <- clone:
						s.routedTotal.Add(1)
					default:
						s.droppedTotal.Add(1)
					}
				}
			}
			return
		}
	}

	s.safeSend(env.ReplyChan, fmt.Sprintf("🚫 Unknown target '@%s'. Available: %s",
		targetName, s.availableTargets()))
}

func (s *Swarm) availableTargets() string {
	targets := make([]string, 0, len(s.Agents)+len(s.Config.Teams))
	for name := range s.Agents {
		targets = append(targets, "@"+name)
	}
	for _, t := range s.Config.Teams {
		targets = append(targets, "@"+t.Name)
	}
	return strings.Join(targets, ", ")
}

func (s *Swarm) providerSummary() string {
	seen := make(map[string]bool)
	for _, a := range s.Config.Agents {
		seen[a.Provider] = true
	}
	names := make([]string, 0, len(seen))
	for p := range seen {
		names = append(names, p)
	}
	return strings.Join(names, ", ")
}

func (s *Swarm) safeSend(ch chan string, msg string) {
	select {
	case ch <- msg:
	default:
	}
}

func (s *Swarm) log(level, source, msg string) {
	select {
	case s.LogChan <- types.LogEntry{Level: level, Source: source, Message: msg, Time: time.Now()}:
	default:
	}
}

// ─── Parsing ─────────────────────────────────────────────────────────────────

var targetRe = regexp.MustCompile(`@(\w+)`)

func parseTarget(input string) (string, string) {
	match := targetRe.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", input
	}
	rest := strings.TrimSpace(strings.Replace(input, match[0], "", 1))
	return match[1], rest
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
