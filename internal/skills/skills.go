package skills

import (
	"clawkido/internal/types"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Skill is the interface every tool must implement.
type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args string) (string, error)
}

// Registry holds all registered skills.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]Skill)}
}

func (r *Registry) Register(s Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[strings.ToLower(s.Name())] = s
}

func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[strings.ToLower(name)]
	return s, ok
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, fmt.Sprintf("- [!%s]: %s", s.Name(), s.Description()))
	}
	return out
}

func (r *Registry) Execute(ctx context.Context, name, args string) types.SkillResult {
	start := time.Now()
	s, ok := r.Get(name)
	if !ok {
		return types.SkillResult{Name: name, Error: fmt.Errorf("unknown skill: %s", name), Latency: time.Since(start)}
	}
	output, err := s.Execute(ctx, args)
	return types.SkillResult{Name: name, Output: output, Error: err, Latency: time.Since(start)}
}

// ─── Built-in Skills ─────────────────────────────────────────────────────────

type ShellSkill struct{}

func (s ShellSkill) Name() string { return "shell" }
func (s ShellSkill) Description() string {
	return "Execute a shell command. Usage: [!shell: <command>]"
}
func (s ShellSkill) Execute(ctx context.Context, args string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("shell: %w: %s", err, string(out))
	}
	result := string(out)
	if len(result) > 4000 {
		result = result[:4000] + "\n...(truncated)"
	}
	return result, nil
}

type TimeSkill struct{}

func (t TimeSkill) Name() string        { return "time" }
func (t TimeSkill) Description() string { return "Get the current UTC time. Usage: [!time]" }
func (t TimeSkill) Execute(_ context.Context, _ string) (string, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}

type MemoryResetSkill struct{}

func (m MemoryResetSkill) Name() string { return "memory_reset" }
func (m MemoryResetSkill) Description() string {
	return "Reset conversation history. Usage: [!memory_reset]"
}
func (m MemoryResetSkill) Execute(_ context.Context, _ string) (string, error) {
	return "__MEMORY_RESET__", nil
}

func RegisterDefaults(r *Registry) {
	r.Register(ShellSkill{})
	r.Register(TimeSkill{})
	r.Register(MemoryResetSkill{})
}
