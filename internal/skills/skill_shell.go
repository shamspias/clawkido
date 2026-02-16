package skills

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ShellSkill executes shell commands with safety checks.
// Destructive commands (rm -rf, dd, etc.) are detected and blocked.
// The registry's confirmation flow gates them further.
type ShellSkill struct{}

func (s ShellSkill) Name() string { return "shell" }

func (s ShellSkill) Description() string {
	return "Execute a shell command (30s timeout). Usage: [!shell: <command>]"
}

func (s ShellSkill) Safety() SafetyLevel {
	// Marked as Write — the registry + IsDangerousCommand handle
	// escalation to Destructive per-invocation.
	return SafetyWrite
}

func (s ShellSkill) Execute(ctx context.Context, args string) (string, error) {
	if args == "" {
		return "", fmt.Errorf("shell: no command provided")
	}

	// Check for dangerous commands before execution.
	if dangerous, reason := IsDangerousCommand(args); dangerous {
		return "", fmt.Errorf("🛑 BLOCKED: %s\nCommand: %s", reason, args)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", args)
	out, err := cmd.CombinedOutput()
	result := string(out)

	if err != nil {
		// Include output even on error — it often contains the actual error message.
		if len(result) > 0 {
			return "", fmt.Errorf("shell: %w\n%s", err, truncateOutput(result))
		}
		return "", fmt.Errorf("shell: %w", err)
	}

	return truncateOutput(result), nil
}

func truncateOutput(s string) string {
	const maxLen = 4000
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...(truncated)"
}
