package skills

import (
	"regexp"
	"strings"
)

// dangerousPatterns are shell command patterns that indicate destructive intent.
// Each pattern is checked against the normalized (lowercase, trimmed) command.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+(-[a-zA-Z]*[rf])`),     // rm -rf, rm -r, rm -f
	regexp.MustCompile(`\brm\s+/`),                    // rm /anything
	regexp.MustCompile(`\brmdir\b`),                   // rmdir
	regexp.MustCompile(`\bmkfs\b`),                    // mkfs (format disk)
	regexp.MustCompile(`\bdd\s+`),                     // dd (disk destroyer)
	regexp.MustCompile(`>\s*/dev/`),                   // redirect to device
	regexp.MustCompile(`\bshutdown\b`),                // shutdown
	regexp.MustCompile(`\breboot\b`),                  // reboot
	regexp.MustCompile(`\bkill\s+-9`),                 // kill -9
	regexp.MustCompile(`\bkillall\b`),                 // killall
	regexp.MustCompile(`\bchmod\s+777`),               // chmod 777
	regexp.MustCompile(`\bchown\s+-R\s+`),             // chown -R
	regexp.MustCompile(`\b:(){ :|:& };:`),             // fork bomb
	regexp.MustCompile(`/dev/s[dh]`),                  // raw disk access
	regexp.MustCompile(`\bdrop\s+(table|database)\b`), // SQL DROP
	regexp.MustCompile(`\btruncate\s+table\b`),        // SQL TRUNCATE
	regexp.MustCompile(`\bdelete\s+from\b`),           // SQL DELETE
	regexp.MustCompile(`\bformat\s+[a-z]:`),           // Windows format
}

// blockedCommands are exact command prefixes that are always blocked.
var blockedCommands = []string{
	"sudo rm -rf /",
	"rm -rf /",
	"rm -rf /*",
	":(){ :|:& };:",
	"> /dev/sda",
	"mv / /dev/null",
}

// IsDangerousCommand checks if a shell command matches known destructive patterns.
func IsDangerousCommand(cmd string) (bool, string) {
	normalized := strings.ToLower(strings.TrimSpace(cmd))

	// Check exact blocklist first.
	for _, blocked := range blockedCommands {
		if strings.HasPrefix(normalized, blocked) {
			return true, "This command is permanently blocked for safety."
		}
	}

	// Check regex patterns.
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(normalized) {
			return true, "This command matches a destructive pattern: " + pattern.String()
		}
	}

	return false, ""
}

// IsPathDangerous checks if a file path targets sensitive locations.
func IsPathDangerous(path string) bool {
	path = strings.TrimSpace(path)
	dangerous := []string{
		"/", "/etc", "/usr", "/bin", "/sbin",
		"/boot", "/dev", "/proc", "/sys",
		"/var", "/root", "/home",
	}
	for _, d := range dangerous {
		if path == d || path == d+"/" {
			return true
		}
	}
	// Block paths starting with /etc/, /usr/, etc.
	prefixes := []string{"/etc/", "/boot/", "/dev/", "/proc/", "/sys/"}
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
