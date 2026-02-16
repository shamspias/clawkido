package eventbus

import (
	"clawkido/internal/types"
	"sync"
)

// Handler processes an event. Must be non-blocking.
type Handler func(types.Event)

// Bus is a lightweight, thread-safe pub/sub event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func New() *Bus {
	return &Bus{handlers: make(map[string][]Handler)}
}

// Subscribe registers a handler. Use "*" for all events.
func (b *Bus) Subscribe(eventType string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

// Publish dispatches an event to all matching subscribers.
func (b *Bus) Publish(e types.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, h := range b.handlers[e.Type] {
		go h(e)
	}
	if e.Type != "*" {
		for _, h := range b.handlers["*"] {
			go h(e)
		}
	}
}
