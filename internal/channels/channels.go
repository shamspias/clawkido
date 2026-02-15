package channels

import (
	"clawkido/internal/config"
	"clawkido/internal/types"
	"sync"
)

// Channel defines the interface every platform must follow
type Channel interface {
	Start()
}

type Manager struct {
	channels []Channel
}

func NewManager(cfg *config.Config, inbox chan<- types.Message, logs chan<- types.LogEntry) *Manager {
	mgr := &Manager{}

	// 1. Add Telegram if configured
	if cfg.Telegram.Token != "" {
		mgr.channels = append(mgr.channels, NewTelegram(cfg.Telegram.Token, inbox, logs))
	}

	// 2. Add Discord if configured
	if cfg.Discord.Token != "" {
		mgr.channels = append(mgr.channels, NewDiscord(cfg.Discord.Token, inbox, logs))
	}

	return mgr
}

func (m *Manager) StartAll() {
	var wg sync.WaitGroup
	for _, ch := range m.channels {
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			c.Start()
		}(ch)
	}
	// We don't wait here because channels run forever.
	// The main function keeps the program alive.
}
