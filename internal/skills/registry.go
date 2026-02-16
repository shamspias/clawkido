package skills

import (
	"clawkido/internal/types"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SafetyLevel indicates how dangerous a skill's action is.
type SafetyLevel int

const (
	SafetyReadOnly    SafetyLevel = iota // No side effects
	SafetyWrite                          // Creates or modifies data
	SafetyDestructive                    // Can delete data or cause irreversible changes
)

// Skill is the interface every tool must implement.
type Skill interface {
	// Name returns the unique identifier (lowercase, no spaces).
	Name() string

	// Description returns a human-readable summary injected into agent prompts.
	Description() string

	// Safety returns the skill's danger level.
	Safety() SafetyLevel

	// Execute runs the skill. Args is the text after the colon in [!skill: args].
	Execute(ctx context.Context, args string) (string, error)
}

// Registry holds all registered skills with thread-safe access.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill

	// Safety: patterns that require confirmation before execution.
	// If a destructive skill is invoked, the registry blocks it and
	// returns a confirmation prompt instead.
	confirmationPending sync.Map // key: "sender:skill:args" → bool
}

// NewRegistry creates an empty skill registry.
func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]Skill)}
}

// Register adds a skill. Duplicate names overwrite (last write wins).
func (r *Registry) Register(s Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[strings.ToLower(s.Name())] = s
}

// Get returns a skill by name.
func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[strings.ToLower(name)]
	return s, ok
}

// List returns formatted descriptions for all registered skills (for prompt injection).
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.skills))
	for _, s := range r.skills {
		safety := ""
		switch s.Safety() {
		case SafetyWrite:
			safety = " ⚠️ writes data"
		case SafetyDestructive:
			safety = " 🔴 destructive — requires confirmation"
		}
		out = append(out, fmt.Sprintf("- [!%s]: %s%s", s.Name(), s.Description(), safety))
	}
	return out
}

// Names returns all registered skill names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// Execute runs a skill by name with safety checks.
// For destructive skills, the first call returns a confirmation prompt.
// The caller must re-invoke with the same args to confirm.
func (r *Registry) Execute(ctx context.Context, name, args, sender string) types.SkillResult {
	start := time.Now()

	s, ok := r.Get(name)
	if !ok {
		return types.SkillResult{
			Name:    name,
			Error:   fmt.Errorf("unknown skill: %s", name),
			Latency: time.Since(start),
		}
	}

	// Safety gate for destructive skills.
	if s.Safety() == SafetyDestructive {
		confirmKey := fmt.Sprintf("%s:%s:%s", sender, name, args)

		// Check if this exact operation was already confirmed.
		if _, confirmed := r.confirmationPending.LoadAndDelete(confirmKey); !confirmed {
			// First call: block and request confirmation.
			r.confirmationPending.Store(confirmKey, true)

			// Auto-expire confirmation after 60 seconds.
			go func() {
				time.Sleep(60 * time.Second)
				r.confirmationPending.Delete(confirmKey)
			}()

			return types.SkillResult{
				Name: name,
				Output: fmt.Sprintf(
					"⚠️ **Safety Check** — `%s` is a destructive operation.\n"+
						"Command: `%s`\n"+
						"To confirm, run the same command again within 60 seconds.\n"+
						"To cancel, do nothing.",
					name, args,
				),
				Latency: time.Since(start),
			}
		}
		// Second call: confirmed — fall through to execution.
	}

	output, err := s.Execute(ctx, args)
	return types.SkillResult{
		Name:    name,
		Output:  output,
		Error:   err,
		Latency: time.Since(start),
	}
}
