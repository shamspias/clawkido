package skills

import "context"

// MemoryResetSkill allows an agent to clear its own conversation history.
// The agent's process loop detects the magic output "__MEMORY_RESET__" and
// wipes its history slice.
type MemoryResetSkill struct{}

func (m MemoryResetSkill) Name() string { return "memory_reset" }

func (m MemoryResetSkill) Description() string {
	return "Clear your conversation history and start fresh. Usage: [!memory_reset]"
}

func (m MemoryResetSkill) Safety() SafetyLevel { return SafetyDestructive }

func (m MemoryResetSkill) Execute(_ context.Context, _ string) (string, error) {
	return "__MEMORY_RESET__", nil
}
