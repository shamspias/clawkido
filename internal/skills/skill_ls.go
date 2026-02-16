package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirListSkill lists files and directories at a given path.
type DirListSkill struct{}

func (d DirListSkill) Name() string { return "ls" }

func (d DirListSkill) Description() string {
	return "List files in a directory (max 50 entries). Usage: [!ls: /path/to/dir]"
}

func (d DirListSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (d DirListSkill) Execute(_ context.Context, args string) (string, error) {
	path := strings.TrimSpace(args)
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("ls: bad path: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("ls: %w", err)
	}

	const maxEntries = 50
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📂 %s (%d entries)\n", absPath, len(entries)))

	for i, entry := range entries {
		if i >= maxEntries {
			b.WriteString(fmt.Sprintf("... and %d more\n", len(entries)-maxEntries))
			break
		}

		info, err := entry.Info()
		if err != nil {
			b.WriteString(fmt.Sprintf("  ❓ %s (stat error)\n", entry.Name()))
			continue
		}

		icon := "📄"
		suffix := fmt.Sprintf(" (%d bytes)", info.Size())
		if entry.IsDir() {
			icon = "📁"
			suffix = "/"
		}

		b.WriteString(fmt.Sprintf("  %s %s%s\n", icon, entry.Name(), suffix))
	}

	return b.String(), nil
}
