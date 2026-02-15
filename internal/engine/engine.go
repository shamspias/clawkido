package engine

import (
	"clawkido/internal/brain"
	"clawkido/internal/config"
	"clawkido/internal/types"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	Inbox    chan types.Message
	LogChan  chan types.LogEntry
	Brain    *brain.Brain
	Config   *config.Config
	stopChan chan struct{}
}

func NewEngine(cfg *config.Config, b *brain.Brain, logChan chan types.LogEntry) *Engine {
	return &Engine{
		Inbox:    make(chan types.Message, 100),
		LogChan:  logChan,
		Brain:    b,
		Config:   cfg,
		stopChan: make(chan struct{}),
	}
}

func (e *Engine) Start() {
	e.log("INFO", "Engine", "Online & Listening")
	go e.loop()
}

func (e *Engine) Stop() {
	close(e.stopChan)
}

func (e *Engine) loop() {
	for {
		select {
		case msg := <-e.Inbox:
			go e.process(msg)
		case <-e.stopChan:
			return
		}
	}
}

func (e *Engine) process(msg types.Message) {
	e.log("INFO", msg.Platform, fmt.Sprintf("@%s: %s", msg.Sender, msg.Content))

	// 1. Try to find an @agent or @team tag
	target, prompt := e.parseTarget(msg.Content)

	// 2. SMART FALLBACK: If no tag is found, use the First Agent as default
	if target == "" {
		if len(e.Config.Agents) > 0 {
			target = e.Config.Agents[0].Name
			prompt = msg.Content // Use the whole message
			e.log("DEBUG", "Router", fmt.Sprintf("No tag found. Defaulting to '%s'", target))
		} else {
			e.log("ERROR", "Router", "No agents configured!")
			return
		}
	}

	// 3. Check if target is a Team
	for _, team := range e.Config.Teams {
		if strings.EqualFold(team.Name, target) {
			e.handleTeam(team, prompt, msg)
			return
		}
	}

	// 4. Handle Single Agent
	e.handleAgent(target, prompt, msg)
}

func (e *Engine) handleAgent(agentName, prompt string, msg types.Message) {
	resp, err := e.Brain.ProcessRequest(context.Background(), agentName, prompt)
	if err != nil {
		e.log("ERROR", agentName, err.Error())
		msg.ReplyChan <- fmt.Sprintf("❌ %s Error: %s", agentName, err.Error())
		return
	}

	e.log("SUCCESS", agentName, "Replied")
	msg.ReplyChan <- fmt.Sprintf("🤖 **%s**: %s", agentName, resp)
}

func (e *Engine) handleTeam(team config.TeamConfig, prompt string, msg types.Message) {
	e.log("INFO", "Team", fmt.Sprintf("Activating team %s (%d agents)", team.Name, len(team.Members)))

	var wg sync.WaitGroup
	for _, member := range team.Members {
		wg.Add(1)
		go func(agentName string) {
			defer wg.Done()
			e.handleAgent(agentName, prompt, msg)
		}(member)
	}
	wg.Wait()
}

func (e *Engine) parseTarget(input string) (string, string) {
	re := regexp.MustCompile(`@(\w+)`)
	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", input
	}
	// Return the target (e.g., "coder") and the prompt without the tag
	return match[1], strings.TrimSpace(strings.Replace(input, match[0], "", 1))
}

func (e *Engine) log(level, source, msg string) {
	select {
	case e.LogChan <- types.LogEntry{
		Level: level, Source: source, Message: msg, Time: time.Now(),
	}:
	default:
	}
}
