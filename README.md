<div align="center">
  <h1>🦞 Clawkido</h1>
  <p><strong>The High-Performance Go AI Swarm</strong></p>
  <p>An optimized, memory-safe, and lightning-fast alternative to TinyClaw.</p>

  <p>
    <img src="https://img.shields.io/badge/Language-Go_1.25.5-00ADD8?style=flat-square" />
    <img src="https://img.shields.io/badge/Architecture-Actor_Model-orange?style=flat-square" />
    <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" />
  </p>
</div>

## 🚀 Overview

**Clawkido** is a multi-agent orchestration system written in pure **Go**. It is designed to run autonomous teams of AI
agents that can collaborate, maintain context, and execute complex workflows with minimal resource usage.

Unlike Node.js alternatives that rely on file-system polling for queues, Clawkido utilizes Go's native **Channels** and*
*Goroutines** to create a non-blocking, event-driven Swarm architecture. This results in:

- **Zero Latency** agent-to-agent communication.
- **Tiny Memory Footprint** (<50MB RAM typically).
- **Single Binary** deployment (no complex npm dependencies).

## ✨ Key Features

- **🧠 Stateful Agents**: Every agent maintains its own conversation history and context indefinitely (until reset).
- **⚡ Neural Swarm Networking**: Agents can message each other using `[@agent_name]` tags. The Swarm handles the routing
  instantly.
- **🤝 Team Formations**: Broadcast a single message to a "DevTeam" and watch multiple agents (Coder, Reviewer, QA) work
  in parallel.
- **🔌 Multi-Provider**: Seamlessly mix agents powered by OpenAI, Groq, and Ollama in the same swarm.
- **🖥️ TUI Dashboard**: A built-in Terminal User Interface to visualize the swarm's thought process in real-time.
- **📡 Multi-Channel**: Connects simultaneously to Telegram, Discord, and Terminal.

## 🛠️ Installation

```bash
# Clone the repository
git clone https://github.com/shamspias/clawkido.git

# Build the binary
cd clawkido
go build -o clawkido cmd/clawkido/main.go

# Run
./clawkido

```

## ⚙️ Configuration

Create a `config.json` in the root directory:

```json
{
  "ai": {
    "openai_key": "sk-...",
    "groq_key": "gsk-..."
  },
  "agents": [
    {
      "name": "manager",
      "provider": "openai",
      "model": "gpt-4o",
      "system_prompt": "You are the project manager. Delegate tasks using [@agent: instructions]."
    },
    {
      "name": "coder",
      "provider": "groq",
      "model": "llama3-70b-8192",
      "system_prompt": "You are a Golang expert. Write concise, optimized code."
    }
  ],
  "teams": [
    {
      "name": "dev",
      "members": [
        "manager",
        "coder"
      ]
    }
  ]
}

```

## 🤖 How to Use

### 1. Direct Messaging

Simply talk to an agent:

```
User: @coder write a hello world in Rust
Coder: Here is the code...

```

### 2. Swarm Handoffs (The Power Move)

You can ask the Manager to handle it. If the Manager decides it needs code, it will autonomously tag the Coder.

```
User: @manager build a web scraper
Manager: (Thinking...) I need code for this.
         [@coder: Write a web scraper in Python using BeautifulSoup]
Swarm:   (Routes internal message to Coder)
Coder:   Here is the python script...
Manager: (Receives code) Here is the solution you asked for.

```

## 📐 Architecture

Clawkido implements the **Actor Model**:

1. **Inbox**: Central channel receiving signals from Telegram/Discord.
2. **The Hive**: A goroutine acting as the central router.
3. **Agents**: Individual goroutines with isolated memory stacks.
4. **TUI**: A separate listener visualization layer.

```mermaid
graph TD
    User -->|Msg| Inbox
    Inbox --> Hive
    Hive -->|Msg| AgentA
    Hive -->|Msg| AgentB
    AgentA -->|[@agentB]| Hive

```