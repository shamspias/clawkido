package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileWriteSkill writes content to a file.
// Marked as Destructive — the registry requires confirmation before overwriting.
// Format: [!file_write: /path/to/file ||| content here]
type FileWriteSkill struct{}

func (f FileWriteSkill) Name() string { return "file_write" }

func (f FileWriteSkill) Description() string {
	return "Write content to a file. Usage: [!file_write: /path/to/file ||| content goes here]"
}

func (f FileWriteSkill) Safety() SafetyLevel { return SafetyDestructive }

func (f FileWriteSkill) Execute(_ context.Context, args string) (string, error) {
	// Parse: "/path/to/file ||| content"
	parts := strings.SplitN(args, "|||", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("file_write: bad format — use: /path/to/file ||| content")
	}

	path := strings.TrimSpace(parts[0])
	content := strings.TrimSpace(parts[1])

	if path == "" {
		return "", fmt.Errorf("file_write: no path provided")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("file_write: bad path: %w", err)
	}

	// Block sensitive paths.
	if IsPathDangerous(absPath) {
		return "", fmt.Errorf("file_write: 🛑 writing to '%s' is blocked for safety", absPath)
	}

	// Check if file already exists — warn in output.
	existed := false
	if _, err := os.Stat(absPath); err == nil {
		existed = true
	}

	// Create parent directories if needed.
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("file_write: cannot create directory '%s': %w", dir, err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("file_write: %w", err)
	}

	action := "Created"
	if existed {
		action = "Overwritten"
	}
	return fmt.Sprintf("✅ %s: %s (%d bytes)", action, absPath, len(content)), nil
}
