package skills

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// sensitiveKeys are env var names that should never be exposed.
var sensitiveKeys = map[string]bool{
	"OPENAI_API_KEY":        true,
	"GROQ_API_KEY":          true,
	"ANTHROPIC_API_KEY":     true,
	"TELEGRAM_BOT_TOKEN":    true,
	"DISCORD_BOT_TOKEN":     true,
	"AWS_SECRET_ACCESS_KEY": true,
	"DATABASE_URL":          true,
	"DB_PASSWORD":           true,
}

// EnvSkill reads environment variables (redacts sensitive ones).
type EnvSkill struct{}

func (e EnvSkill) Name() string { return "env" }

func (e EnvSkill) Description() string {
	return "Read an environment variable (sensitive keys are redacted). Usage: [!env: PATH] or [!env] to list all"
}

func (e EnvSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (e EnvSkill) Execute(_ context.Context, args string) (string, error) {
	key := strings.TrimSpace(args)

	// Single variable lookup.
	if key != "" {
		upperKey := strings.ToUpper(key)
		if isSensitive(upperKey) {
			return fmt.Sprintf("%s = [REDACTED]", upperKey), nil
		}
		val := os.Getenv(key)
		if val == "" {
			return fmt.Sprintf("%s is not set", key), nil
		}
		return fmt.Sprintf("%s = %s", key, val), nil
	}

	// List all (redact sensitive).
	var b strings.Builder
	b.WriteString("Environment Variables:\n")
	count := 0
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name, val := parts[0], parts[1]
		if isSensitive(strings.ToUpper(name)) {
			val = "[REDACTED]"
		}
		if len(val) > 80 {
			val = val[:80] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s = %s\n", name, val))
		count++
		if count >= 50 {
			b.WriteString("  ...(truncated at 50)\n")
			break
		}
	}

	return b.String(), nil
}

func isSensitive(key string) bool {
	if sensitiveKeys[key] {
		return true
	}
	// Catch common patterns: *_KEY, *_SECRET, *_TOKEN, *_PASSWORD
	for _, suffix := range []string{"_KEY", "_SECRET", "_TOKEN", "_PASSWORD", "_PASS"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}
