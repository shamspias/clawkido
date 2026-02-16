package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileDeleteSkill deletes a single file.
// Marked as Destructive — the registry's confirmation flow blocks the first
// invocation and requires re-invocation within 60 seconds to confirm.
// NEVER deletes directories. NEVER touches system paths.
type FileDeleteSkill struct{}

func (f FileDeleteSkill) Name() string { return "file_delete" }

func (f FileDeleteSkill) Description() string {
	return "Delete a single file (not directories). Requires confirmation. Usage: [!file_delete: /path/to/file]"
}

func (f FileDeleteSkill) Safety() SafetyLevel { return SafetyDestructive }

func (f FileDeleteSkill) Execute(_ context.Context, args string) (string, error) {
	path := strings.TrimSpace(args)
	if path == "" {
		return "", fmt.Errorf("file_delete: no path provided")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("file_delete: bad path: %w", err)
	}

	// Block sensitive paths.
	if IsPathDangerous(absPath) {
		return "", fmt.Errorf("file_delete: 🛑 deleting '%s' is blocked for safety", absPath)
	}

	// Only delete files, never directories.
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("file_delete: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("file_delete: 🛑 refusing to delete directory '%s' — only files are allowed", absPath)
	}

	if err := os.Remove(absPath); err != nil {
		return "", fmt.Errorf("file_delete: %w", err)
	}

	return fmt.Sprintf("🗑️ Deleted: %s", absPath), nil
}
