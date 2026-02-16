<div align="center">
  <h1>🦞 Clawkido</h1>
  <p><strong>The High-Performance Go AI Swarm</strong></p>
  <p>Multi-agent orchestration with zero-latency routing, extensible skills, and multi-channel support.</p>

  <p>
    <img src="https://img.shields.io/badge/Language-Go_1.25-00ADD8?style=flat-square" />
    <img src="https://img.shields.io/badge/Architecture-Actor_Model-orange?style=flat-square" />
    <img src="https://img.shields.io/badge/Providers-OpenAI·Groq·Ollama-purple?style=flat-square" />
    <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" />
  </p>
</div>

---

## What Is Clawkido?

Clawkido is a **multi-agent AI swarm** written in Go. You define agents (each with their own LLM provider, personality,
and tools), organize them into teams, and talk to them through Telegram or Discord. Agents collaborate autonomously — a
manager delegates to a coder, the coder writes code, the reviewer checks it — all from a single message.

**Why Go instead of Python/Node?**

- Go channels = zero-latency agent-to-agent routing (no file-system polling)
- Single binary, ~50MB RAM — runs on a Raspberry Pi
- Goroutines = each agent runs in its own lightweight thread
- `context.Context` flows everywhere = clean graceful shutdown on Ctrl+C

---

## Quick Start

### 1. Clone & Build

```bash
git clone https://github.com/shamspias/clawkido.git
cd clawkido
make deps
make build
```

### 2. Configure Secrets

```bash
cp .env.example .env
```

Edit `.env` with your real keys:

```bash
GROQ_API_KEY=gsk_your_real_key_here
TELEGRAM_BOT_TOKEN=123456789:AAF-your_real_token_here

# Optional — only needed if you configure agents with these providers:
OPENAI_API_KEY=sk-proj-...
DISCORD_BOT_TOKEN=...
```

> **How to get a Telegram bot token:** Message [@BotFather](https://t.me/BotFather) on Telegram → `/newbot` → follow
> prompts → copy the token.

> **How to get a Groq API key:** Sign up at [console.groq.com](https://console.groq.com) → API Keys → Create.

### 3. Configure Agents

Edit `config.json`. The default ships with 3 agents (manager, coder, reviewer). No changes needed to start.

### 4. Run

```bash
make run
```

You should see:

```
🦞 CLAWKIDO — AI AGENT SWARM ENGINE
──────────────────────────────────────
10:04:35 │ INFO    │ Swarm          │ Agent 'manager' registered
10:04:35 │ INFO    │ Swarm          │ Agent 'coder' registered
10:04:35 │ INFO    │ Swarm          │ Agent 'reviewer' registered
10:04:35 │ INFO    │ Swarm          │ Hive active: 3 agents, 1 teams
10:04:35 │ INFO    │ Telegram       │ Connected as @your_bot
```

Now message your bot on Telegram. It works.

---

## Configuration Reference

### `config.json` Structure

```json
{
  "ai": {
    "ollama_url": "http://localhost:11434"
  },
  "telegram": {
    "allowed_users": []
  },
  "discord": {},
  "swarm": {
    "max_handoff_depth": 5,
    "inbox_buffer_size": 256,
    "router_buffer": 256
  },
  "agents": [
    ...
  ],
  "teams": [
    ...
  ]
}
```

### Access Control

The `allowed_users` field controls who can talk to your bot:

| Value                    | Behavior                                                   |
|--------------------------|------------------------------------------------------------|
| `[]` (empty)             | **Allow everyone** — any Telegram user can message the bot |
| `[910739932]`            | Only user ID `910739932` can message the bot               |
| `[910739932, 123456789]` | Only these two users can message the bot                   |

> **How to find your Telegram user ID:** Message [@userinfobot](https://t.me/userinfobot) on Telegram — it replies with
> your numeric ID.

### Agent Configuration

Each agent entry in the `agents` array:

```json
{
  "name": "coder",
  "provider": "groq",
  "model_name": "openai/gpt-oss-120b",
  "temperature": 0.2,
  "max_history": 80,
  "skills": [
    "shell",
    "time"
  ],
  "fallback": "ollama",
  "system_prompt": "You are a Senior Software Engineer..."
}
```

| Field           | Type     | Description                                           |
|-----------------|----------|-------------------------------------------------------|
| `name`          | string   | Unique agent name (used for `@mentions`)              |
| `provider`      | string   | `"groq"`, `"openai"`, or `"ollama"`                   |
| `model_name`    | string   | Model ID for the provider                             |
| `temperature`   | float    | 0.0 (deterministic) to 2.0 (creative). Default: 0.7   |
| `max_history`   | int      | Max conversation turns to keep in memory. Default: 50 |
| `skills`        | string[] | Skill names this agent can invoke                     |
| `fallback`      | string   | Fallback provider if primary fails (optional)         |
| `system_prompt` | string   | The agent's personality and instructions              |

### Team Configuration

```json
{
  "name": "dev",
  "members": [
    "manager",
    "coder",
    "reviewer"
  ],
  "leader": "manager"
}
```

Message `@dev` to broadcast to all members simultaneously.

### Swarm Tunables

| Field               | Default | Description                            |
|---------------------|---------|----------------------------------------|
| `max_handoff_depth` | 5       | Prevents infinite agent→agent loops    |
| `inbox_buffer_size` | 256     | Buffer size for external message queue |
| `router_buffer`     | 256     | Buffer size for internal handoff queue |

---

## How to Use

### Direct Agent Messaging

Talk to a specific agent by prefixing with `@name`:

```
You:     @coder write a fibonacci function in Go
Coder:   🤖 Here's an optimized implementation...
```

### Default Agent (No @mention)

Messages without `@` go to the **first agent** in the config (typically the manager):

```
You:     Build me a REST API for a todo app
Manager: 🤖 I'll coordinate this. [@coder: Build a REST API in Go with...]
```

### Swarm Handoffs

This is the core power. The manager can autonomously delegate:

```
You:       @manager I need a Python web scraper
Manager:   🤖 I'll have the coder handle this.
             [@coder: Write a Python web scraper using BeautifulSoup]
  ↓ (Swarm routes internally — you see both responses)
Coder:     🤖 Here's the scraper:
             ```python
             import requests
             from bs4 import BeautifulSoup
             ...
             ```
```

The Swarm routes `[@coder: ...]` tags internally. You receive all responses from the chain in your chat.

### Multi-Agent Chains

Agents can chain through multiple handoffs:

```
You:      @manager review and optimize my sorting algorithm
Manager:  [@coder: Optimize this sorting algorithm]
Coder:    Here's the optimized version. [@reviewer: Check this for edge cases]
Reviewer: 🤖 Found 2 issues: ...
```

The `max_handoff_depth` (default: 5) prevents infinite loops.

### Team Broadcasts

```
You:      @dev what's the status of the auth module?
  ↓ (All 3 agents receive the message in parallel)
Manager:  🤖 From a project perspective...
Coder:    🤖 Implementation is at 80%...
Reviewer: 🤖 I've flagged 3 security concerns...
```

---

## Skills System

Skills are tools agents can invoke inline during responses. The LLM includes `[!skill_name: args]` in its output, and
Clawkido executes it before delivering the response.

### Built-in Skills

| Skill          | Usage in LLM output | Description                                                |
|----------------|---------------------|------------------------------------------------------------|
| `shell`        | `[!shell: ls -la]`  | Run a shell command (30s timeout, output truncated at 4KB) |
| `time`         | `[!time]`           | Current UTC timestamp                                      |
| `memory_reset` | `[!memory_reset]`   | Clear the agent's conversation history                     |

### How Skills Work

1. You configure which skills each agent can use in `config.json`:
   ```json
   { "name": "coder", "skills": ["shell", "time"], ... }
   ```

2. The agent's system prompt is automatically augmented with skill descriptions.

3. When the LLM outputs `[!shell: ls -la]`, Clawkido:
    - Parses the tag
    - Executes the skill
    - Replaces the tag with the output in the response

### Creating Custom Skills

Create a new file (e.g., `internal/skills/weather.go`):

```go
package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type WeatherSkill struct{}

func (w WeatherSkill) Name() string {
	return "weather"
}

func (w WeatherSkill) Description() string {
	return "Get weather for a city. Usage: [!weather: London]"
}

func (w WeatherSkill) Execute(ctx context.Context, args string) (string, error) {
	url := fmt.Sprintf("https://wttr.in/%s?format=3", args)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}
```

Register it in `cmd/clawkido/main.go` after `skills.RegisterDefaults(skillReg)`:

```go
skillReg.Register(skills.WeatherSkill{})
```

Add it to agent configs:

```json
{
  "name": "manager",
  "skills": [
    "time",
    "weather"
  ],
  ...
}
```

Rebuild and run. The manager can now say `[!weather: Tokyo]` and it works.

### Skill Ideas

| Skill        | What it does                                     |
|--------------|--------------------------------------------------|
| `http_get`   | Fetch a URL and return the body                  |
| `calculator` | Evaluate a math expression                       |
| `file_read`  | Read a local file                                |
| `file_write` | Write content to a file                          |
| `git_status` | Run `git status` in a project directory          |
| `docker_ps`  | List running containers                          |
| `db_query`   | Execute a read-only SQL query                    |
| `web_search` | Search the web via an API (SerpAPI, Brave, etc.) |

---

## Architecture

```
┌─────────────┐     ┌─────────────┐
│  Telegram    │     │   Discord   │
└──────┬──────┘     └──────┬──────┘
       │                   │
       └───────┬───────────┘
               ▼
       ┌───────────────┐
       │    Inbox       │  ← Buffered channel (256)
       └───────┬───────┘
               ▼
       ┌───────────────┐
       │   The Hive     │  ← Router goroutine
       │   (Swarm)      │
       └──┬──────┬──┬──┘
          │      │  │
          ▼      ▼  ▼
       ┌─────┐ ┌─────┐ ┌─────┐
       │ Mgr │ │Coder│ │ QA  │  ← Agent goroutines (actors)
       │     │ │     │ │     │     Each has: inbox, history, skills
       └──┬──┘ └──┬──┘ └──┬──┘
          │       │       │
          └───┬───┘───┬───┘
              ▼       ▼
       ┌───────────────┐
       │  Router Bus    │  ← Handoff channel (256)
       └───────┬───────┘
               ▼
       ┌───────────────┐
       │  Event Bus     │  ← Pub/sub for extensions
       └───────┬───────┘
               ▼
       ┌───────────────┐
       │  TUI + Health  │  ← Dashboard + metrics
       └───────────────┘
```

### Key Design Decisions

| Decision                        | Why                                                                                                                      |
|---------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| **Buffered reply channel (16)** | A handoff chain (manager → coder → reviewer) produces multiple responses. Buffer holds them all without blocking.        |
| **Rolling idle timeout (90s)**  | After receiving a response, wait 90s for more before closing. Lets slow LLMs in the chain finish.                        |
| **Depth limiting**              | `max_handoff_depth=5` prevents agent A tagging agent B tagging agent A in a loop.                                        |
| **History trimming**            | `max_history=50` keeps the last 50 turns. System prompt (index 0) is never trimmed.                                      |
| **Non-blocking sends**          | Every channel send uses `select/default`. If a buffer is full, the message is dropped with a warning — never deadlocked. |
| **Atomic metrics**              | Message counts and latency use `atomic.Int64` — no mutex contention on the hot path.                                     |
| **Provider fallback**           | If an agent's primary provider (Groq) fails after retries, it automatically tries the fallback (Ollama).                 |
| **Markdown fallback**           | If Telegram rejects a message due to bad Markdown, it retries without formatting instead of failing silently.            |

---

## Use Cases

### 1. Personal Coding Assistant

Configure a coder agent with `shell` skill. Ask it to write code, and it can run tests:

```
@coder write a Go function to reverse a string, then test it with [!shell: go test ./...]
```

### 2. Code Review Pipeline

```
@manager review my auth implementation
→ Manager delegates to coder for analysis
→ Coder reviews and delegates to reviewer for security check
→ You get all three perspectives in your chat
```

### 3. DevOps Bot

Add a `docker_ps` skill and a devops agent:

```json
{
  "name": "devops",
  "provider": "groq",
  "model_name": "openai/gpt-oss-120b",
  "skills": [
    "shell"
  ],
  "system_prompt": "You are a DevOps engineer. Use [!shell: command] to check system status."
}
```

```
@devops check if nginx is running and show disk usage
```

### 4. Research Team

```json
{
  "agents": [
    {
      "name": "researcher",
      "system_prompt": "Find information and cite sources..."
    },
    {
      "name": "writer",
      "system_prompt": "Write clear, engaging content..."
    },
    {
      "name": "editor",
      "system_prompt": "Edit for grammar, clarity, and accuracy..."
    }
  ],
  "teams": [
    {
      "name": "content",
      "members": [
        "researcher",
        "writer",
        "editor"
      ]
    }
  ]
}
```

```
@content write a blog post about Go's concurrency model
```

---

## Troubleshooting

### "Blocked user 910739946 (sleepomi)"

Your Telegram user ID is not in `allowed_users`. Fix:

**Option A:** Allow everyone (recommended for personal use):

```json
"telegram": {"allowed_users": []}
```

**Option B:** Add your specific ID:

```json
"telegram": {"allowed_users": [910739946]}
```

### "Discord failed: Authentication failed"

Your `DISCORD_BOT_TOKEN` in `.env` is invalid or empty. If you don't use Discord, just leave it empty — the error is
non-fatal and Telegram still works.

### Agent responds but Telegram shows nothing

This was a bug in v1 — the reply goroutine only read one message and then exited. v2 uses a **draining loop** that
collects all responses from handoff chains. Make sure you're running the latest code.

### "provider 'groq' not registered"

Your `GROQ_API_KEY` in `.env` is missing or empty. The brain only registers providers that have valid keys.

### Agent seems to forget context

Check `max_history` in config.json. Default is 50 turns. For long conversations, increase it (cost goes up).

---

## Development

```bash
make fmt      # gofmt
make vet      # go vet
make lint     # golangci-lint (if installed)
make test     # Tests with race detector
make release  # Cross-compile for Linux/macOS/Windows
```

---

## License

MIT — see [LICENSE](LICENSE).
