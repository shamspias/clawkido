# 🦞 Clawkido Skills

Skills are tools that agents can invoke inline during their responses. When an LLM outputs `[!skill_name: args]`,
Clawkido intercepts it, executes the skill, and replaces the tag with the result before delivering the response.

## Built-in Skills

| Skill          | Safety         | Usage                                    | Description                           |
|----------------|----------------|------------------------------------------|---------------------------------------|
| `help`         | 🟢 ReadOnly    | `[!help]`                                | List all available skills             |
| `time`         | 🟢 ReadOnly    | `[!time]` or `[!time: America/New_York]` | Current time (UTC or timezone)        |
| `sysinfo`      | 🟢 ReadOnly    | `[!sysinfo]`                             | OS, arch, memory, goroutines, uptime  |
| `ls`           | 🟢 ReadOnly    | `[!ls: /path/to/dir]`                    | List directory contents (max 50)      |
| `file_read`    | 🟢 ReadOnly    | `[!file_read: /path/to/file]`            | Read file contents (max 8KB)          |
| `env`          | 🟢 ReadOnly    | `[!env: PATH]` or `[!env]`               | Read env vars (redacts secrets)       |
| `http_get`     | 🟢 ReadOnly    | `[!http_get: https://example.com]`       | Fetch a URL (10s timeout, 8KB max)    |
| `shell`        | ⚠️ Write       | `[!shell: ls -la]`                       | Run shell command (30s timeout)       |
| `file_write`   | 🔴 Destructive | `[!file_write: /path \|\|\| content]`    | Write to file (confirmation required) |
| `file_delete`  | 🔴 Destructive | `[!file_delete: /path/to/file]`          | Delete a file (confirmation required) |
| `memory_reset` | 🔴 Destructive | `[!memory_reset]`                        | Clear agent conversation history      |

## Safety System

Every skill declares a safety level:

### 🟢 ReadOnly

No side effects. Always executes immediately.

### ⚠️ Write

Can create or modify data. Executes immediately but has built-in safety checks:

- **Shell**: blocks known destructive commands (`rm -rf /`, `dd`, fork bombs, etc.)
- **File operations**: blocks writes to system paths (`/etc`, `/usr`, `/boot`, etc.)
- **Env**: redacts sensitive variables (`*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`)

### 🔴 Destructive

Can delete data or cause irreversible changes. **Requires confirmation:**

1. First invocation → skill is blocked, returns a confirmation prompt
2. User/agent must invoke the exact same skill+args again within 60 seconds
3. Second invocation → executes

This prevents accidental deletions from a single LLM hallucination.

#### Blocked Commands (Shell)

These patterns are **always blocked**, even with confirmation:

```
rm -rf /          sudo rm -rf /
rm -rf /*         mv / /dev/null
:(){ :|:& };:     > /dev/sda
```

Plus regex patterns for: `rm -r`, `rm -f`, `dd`, `mkfs`, `shutdown`, `reboot`, `kill -9`, `killall`, `chmod 777`, SQL
`DROP TABLE`, `DELETE FROM`, etc.

#### Protected Paths (File skills)

These paths are blocked for all file operations:

```
/  /etc  /usr  /bin  /sbin  /boot  /dev  /proc  /sys  /var  /root  /home
```

Plus anything starting with `/etc/`, `/boot/`, `/dev/`, `/proc/`, `/sys/`.

## Configuration

Enable skills per agent in `config.json`:

```json
{
  "name": "coder",
  "skills": [
    "shell",
    "time",
    "file_read",
    "ls",
    "sysinfo"
  ],
  "system_prompt": "You are a Senior Engineer..."
}
```

Only skills listed in the agent's `skills` array are mentioned in its system prompt. However, all skills are available
in the registry — the list controls prompt injection, not execution access.

## Creating a Custom Skill

### Step 1: Create the file

Create `internal/skills/skill_yourname.go`:

```go
package skills

import (
	"context"
	"fmt"
)

type YourSkill struct{}

func (y YourSkill) Name() string { return "yourskill" }

func (y YourSkill) Description() string {
	return "What it does. Usage: [!yourskill: args]"
}

func (y YourSkill) Safety() SafetyLevel {
	// Choose one:
	// SafetyReadOnly    - no side effects
	// SafetyWrite       - creates/modifies data
	// SafetyDestructive - deletes data (confirmation required)
	return SafetyReadOnly
}

func (y YourSkill) Execute(ctx context.Context, args string) (string, error) {
	if args == "" {
		return "", fmt.Errorf("yourskill: no args provided")
	}

	// Your logic here.
	result := fmt.Sprintf("Result: %s", args)
	return result, nil
}
```

### Step 2: Register it

Add to `internal/skills/defaults.go`:

```go
func RegisterDefaults(r *Registry) {
// ... existing skills ...
r.Register(YourSkill{})
}
```

Or register in `cmd/clawkido/main.go` after `RegisterDefaults`:

```go
skillReg.Register(skills.YourSkill{})
```

### Step 3: Enable for agents

Add to agent config:

```json
{
  "name": "coder",
  "skills": [
    "shell",
    "yourskill"
  ]
}
```

### Step 4: Rebuild

```bash
make run
```

## Skill Interface

Every skill implements:

```go
type Skill interface {
Name() string        // Unique ID
Description() string // Injected into prompts
Safety() SafetyLevel // ReadOnly | Write | Destructive
Execute(ctx context.Context, args string) (string, error) // The actual logic
}
```

### Rules

1. **Name**: lowercase, no spaces, unique across the registry
2. **Description**: include `Usage: [!name: args]` — this is what the LLM reads
3. **Safety**: be honest. If it can delete data, mark it `Destructive`
4. **Execute**: respect `ctx` cancellation. Timeout long operations. Truncate large outputs
5. **Errors**: return `("", error)` on failure — the agent sees the error in its response

### Skills That Need State

If your skill needs initialization (e.g., API clients, start time):

```go
type MySkill struct {
client *http.Client
apiKey string
}

func NewMySkill(apiKey string) *MySkill {
return &MySkill{
client: &http.Client{Timeout: 10 * time.Second},
apiKey: apiKey,
}
}
```

Register with: `r.Register(NewMySkill(os.Getenv("MY_API_KEY")))`

Each file is a self-contained skill. To add a new skill, add a single file and one line in `defaults.go`.