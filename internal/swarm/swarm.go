package swarm

import (
	"clawkido/internal/agent"
	"clawkido/internal/brain"
	"clawkido/internal/config"
	"clawkido/internal/types"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Swarm struct {
	Inbox   chan types.Message  // From Channels
	Router  chan types.Envelope // Internal High-Speed Bus
	LogChan chan types.LogEntry
	Agents  map[string]*agent.Agent
	Config  *config.Config
}

func NewSwarm(cfg *config.Config, b *brain.Brain, logChan chan types.LogEntry) *Swarm {
	s := &Swarm{
		Inbox:   make(chan types.Message, 100),
		Router:  make(chan types.Envelope, 100),
		LogChan: logChan,
		Agents:  make(map[string]*agent.Agent),
		Config:  cfg,
	}

	// Initialize Agents
	for _, agentCfg := range cfg.Agents {
		a := agent.New(agentCfg, b, s.Router)
		s.Agents[strings.ToLower(agentCfg.Name)] = a
		a.Start() // Boot the actor
		s.log("INFO", "Swarm", fmt.Sprintf("Agent '%s' online", agentCfg.Name))
	}

	return s
}

func (s *Swarm) Start() {
	s.log("INFO", "Swarm", "Hive Mind Active")

	// 1. Process External Messages (User -> Swarm)
	go func() {
		for msg := range s.Inbox {
			s.log("INFO", msg.Platform, fmt.Sprintf("@%s: %s", msg.Sender, msg.Content))

			// Wrap in Envelope
			s.route(types.Envelope{
				Sender:    msg.Sender,
				Content:   msg.Content,
				ReplyChan: msg.ReplyChan,
			})
		}
	}()

	// 2. Process Internal Routing (Agent -> Swarm -> Agent)
	go func() {
		for env := range s.Router {
			s.log("DEBUG", "Router", fmt.Sprintf("Handoff from %s", env.Sender))
			s.route(env)
		}
	}()
}

func (s *Swarm) route(env types.Envelope) {
	targetName, content := parseTarget(env.Content)

	// Default Fallback
	if targetName == "" {
		if len(s.Config.Agents) > 0 {
			targetName = s.Config.Agents[0].Name // Default to first agent (Manager)
		}
	}

	targetName = strings.ToLower(targetName)

	// 1. Direct Agent Message
	if a, exists := s.Agents[targetName]; exists {
		env.Content = content
		a.Inbox <- env // Non-blocking send
		s.log("SUCCESS", "Router", fmt.Sprintf("Routed to '%s'", targetName))
		return
	}

	// 2. Team Broadcast
	for _, team := range s.Config.Teams {
		if strings.EqualFold(team.Name, targetName) {
			s.log("INFO", "Router", fmt.Sprintf("Broadcasting to Team '%s'", team.Name))
			for _, memberName := range team.Members {
				if a, exists := s.Agents[strings.ToLower(memberName)]; exists {
					// Clone envelope for each member
					a.Inbox <- types.Envelope{
						Sender:    env.Sender,
						Content:   content,
						ReplyChan: env.ReplyChan,
					}
				}
			}
			return
		}
	}

	env.ReplyChan <- fmt.Sprintf("🚫 Swarm: Unknown target '@%s'", targetName)
}

func (s *Swarm) log(level, source, msg string) {
	s.LogChan <- types.LogEntry{
		Level: level, Source: source, Message: msg, Time: time.Now(),
	}
}

// Extracts @name from string
func parseTarget(input string) (string, string) {
	re := regexp.MustCompile(`@(\w+)`)
	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", input
	}
	// Return "name", "rest of message"
	return match[1], strings.TrimSpace(strings.Replace(input, match[0], "", 1))
}
