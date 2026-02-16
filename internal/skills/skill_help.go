package skills

import (
	"context"
	"fmt"
	"strings"
)

// HelpSkill lists all registered skills and their descriptions.
// It holds a reference to the registry to introspect at runtime.
type HelpSkill struct {
	registry *Registry
}

// NewHelpSkill creates a help skill that can introspect the registry.
func NewHelpSkill(r *Registry) *HelpSkill {
	return &HelpSkill{registry: r}
}

func (h *HelpSkill) Name() string { return "help" }

func (h *HelpSkill) Description() string {
	return "List all available skills and their usage. Usage: [!help]"
}

func (h *HelpSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (h *HelpSkill) Execute(_ context.Context, _ string) (string, error) {
	descs := h.registry.List()
	if len(descs) == 0 {
		return "No skills registered.", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🦞 Available Skills (%d)\n\n", len(descs)))
	for _, desc := range descs {
		b.WriteString(desc)
		b.WriteString("\n")
	}
	b.WriteString("\nSafety levels: ReadOnly (safe), Write ⚠️, Destructive 🔴 (requires confirmation)")

	return b.String(), nil
}
