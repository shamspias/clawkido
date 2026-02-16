package skills

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TimeSkill returns the current time in UTC or a specified timezone.
type TimeSkill struct{}

func (t TimeSkill) Name() string { return "time" }

func (t TimeSkill) Description() string {
	return "Get the current time. Usage: [!time] or [!time: America/New_York]"
}

func (t TimeSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (t TimeSkill) Execute(_ context.Context, args string) (string, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		return time.Now().UTC().Format(time.RFC3339), nil
	}

	// Try to parse as timezone.
	loc, err := time.LoadLocation(args)
	if err != nil {
		return "", fmt.Errorf("unknown timezone: %s (use IANA format like 'America/New_York')", args)
	}

	return time.Now().In(loc).Format(time.RFC3339), nil
}
