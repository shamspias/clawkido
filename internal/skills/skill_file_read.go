package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileReadSkill reads a file and returns its contents.
// Blocks access to sensitive system paths.
type FileReadSkill struct{}

func (f FileReadSkill) Name() string { return "file_read" }

func (f FileReadSkill) Description() string {
	return "Read a file's contents (max 8KB, blocks system paths). Usage: [!file_read: /path/to/file]"
}

func (f FileReadSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (f FileReadSkill) Execute(_ context.Context, args string) (string, error) {
	path := strings.TrimSpace(args)
	if path == "" {
		return "", fmt.Errorf("file_read: no path provided")
	}

	// Resolve to absolute path.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("file_read: bad path: %w", err)
	}

	// Block sensitive paths.
	if IsPathDangerous(absPath) {
		return "", fmt.Errorf("file_read: 🛑 access to '%s' is blocked for safety", absPath)
	}

	// Check file exists and is not a directory.
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("file_read: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("file_read: '%s' is a directory, not a file", absPath)
	}

	// Block very large files.
	const maxSize = 8192
	if info.Size() > maxSize {
		return "", fmt.Errorf("file_read: file too large (%d bytes, max %d)", info.Size(), maxSize)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("file_read: %w", err)
	}

	return string(data), nil
}
