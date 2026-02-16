package main

import (
	"clawkido/internal/brain"
	"clawkido/internal/channels"
	"clawkido/internal/config"
	"clawkido/internal/eventbus"
	"clawkido/internal/skills"
	"clawkido/internal/swarm"
	"clawkido/internal/tui"
	"clawkido/internal/types"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Environment
	if err := godotenv.Load(); err != nil {
		log.Println("⚠  No .env file found (using system environment)")
	}

	// 2. Configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("❌ Config: %v", err)
	}

	// 3. Root context — cancellation propagates to all subsystems
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Shared infrastructure
	logChan := make(chan types.LogEntry, 2048)
	bus := eventbus.New()

	// 5. Skills registry
	skillReg := skills.NewRegistry()
	skills.RegisterDefaults(skillReg)

	// 6. Brain (LLM provider registry)
	brn, err := brain.NewBrain(cfg)
	if err != nil {
		log.Fatalf("❌ Brain: %v", err)
	}
	log.Printf("🧠 Brain online: providers=%v", brn.AvailableProviders())

	// 7. Swarm (orchestration engine)
	hive := swarm.NewSwarm(cfg, brn, logChan, bus, skillReg)
	hive.Start(ctx)

	// 8. Channels (Telegram, Discord, etc.)
	mgr := channels.NewManager(cfg, hive.Inbox, logChan)
	go mgr.StartAll(ctx)

	// 9. TUI Dashboard
	dash := tui.NewDashboard(logChan)
	go dash.Run(ctx)

	// 10. Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutdown signal received...")
	cancel()

	hive.Stop()
	time.Sleep(500 * time.Millisecond)
	log.Println("👋 Clawkido stopped.")
}
