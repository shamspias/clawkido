package health

import (
	"clawkido/internal/agent"
	"clawkido/internal/types"
	"context"
	"fmt"
	"sync"
	"time"
)

// Monitor periodically collects health metrics from all agents.
type Monitor struct {
	agents  map[string]*agent.Agent
	logChan chan<- types.LogEntry
	mu      sync.RWMutex
	latest  map[string]types.AgentHealth
}

func NewMonitor(agents map[string]*agent.Agent, logChan chan<- types.LogEntry) *Monitor {
	return &Monitor{
		agents:  agents,
		logChan: logChan,
		latest:  make(map[string]types.AgentHealth),
	}
}

func (m *Monitor) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collect()
		}
	}
}

func (m *Monitor) collect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, a := range m.agents {
		m.latest[name] = a.Health()
	}
}

func (m *Monitor) Status() map[string]types.AgentHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]types.AgentHealth, len(m.latest))
	for k, v := range m.latest {
		out[k] = v
	}
	return out
}

func (m *Monitor) Summary() string {
	statuses := m.Status()
	if len(statuses) == 0 {
		return "No agents registered"
	}
	result := "Agent Health:\n"
	for _, h := range statuses {
		status := "🟢"
		if h.ErrorsTotal > 0 && h.MessagesTotal > 0 && float64(h.ErrorsTotal)/float64(h.MessagesTotal) > 0.5 {
			status = "🔴"
		} else if h.QueueDepth > 50 {
			status = "🟡"
		}
		result += fmt.Sprintf("  %s %-12s | msgs:%d errs:%d avg:%s queue:%d hist:%d\n",
			status, h.Name, h.MessagesTotal, h.ErrorsTotal, h.AvgLatency, h.QueueDepth, h.HistoryLen)
	}
	return result
}
