package channels

import (
	"clawkido/internal/config"
	"clawkido/internal/types"
	"context"
	"sync"
	"time"
)

// Channel defines the contract every platform adapter must implement.
type Channel interface {
	Name() string
	Start(ctx context.Context) error
}

// Manager orchestrates all configured channel adapters.
type Manager struct {
	channels []Channel
	logChan  chan<- types.LogEntry
}

func NewManager(cfg *config.Config, inbox chan<- types.Message, logChan chan<- types.LogEntry) *Manager {
	mgr := &Manager{logChan: logChan}

	if cfg.Telegram.Token != "" {
		mgr.channels = append(mgr.channels, NewTelegram(cfg, inbox, logChan))
		mgr.log("INFO", "Channel:Telegram registered")
	}

	if cfg.Discord.Token != "" {
		mgr.channels = append(mgr.channels, NewDiscord(cfg.Discord.Token, inbox, logChan))
		mgr.log("INFO", "Channel:Discord registered")
	}

	if len(mgr.channels) == 0 {
		mgr.log("WARN", "No channels configured — running headless")
	}

	return mgr
}

func (m *Manager) StartAll(ctx context.Context) {
	var wg sync.WaitGroup

	for _, ch := range m.channels {
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			m.log("INFO", "Starting "+c.Name())
			if err := c.Start(ctx); err != nil {
				m.log("ERROR", c.Name()+" failed: "+err.Error())
			}
		}(ch)
	}

	<-ctx.Done()
	wg.Wait()
}

func (m *Manager) log(level, msg string) {
	select {
	case m.logChan <- types.LogEntry{Level: level, Source: "ChannelMgr", Message: msg, Time: time.Now()}:
	default:
	}
}
