package main

import (
	"clawkido/internal/brain"
	"clawkido/internal/channels"
	"clawkido/internal/config"
	"clawkido/internal/swarm"
	"clawkido/internal/tui"
	"clawkido/internal/types"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found (relying on system env vars)")
	}

	// 2. Load Config
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Critical: Config load failed: %v", err)
	}

	// 3. Setup Logging Channel
	logChan := make(chan types.LogEntry, 1000)

	// 4. Initialize Brain (LLM Connection)
	brn, err := brain.NewBrain(cfg)
	if err != nil {
		log.Fatalf("Critical: Brain init failed: %v", err)
	}

	// 5. Initialize Swarm (The Engine)
	hive := swarm.NewSwarm(cfg, brn, logChan)

	// 6. Initialize Channels (Telegram, Discord, etc.)
	// Note: channels.NewManager must accept chan types.Message now
	mgr := channels.NewManager(cfg, hive.Inbox, logChan)

	// 7. Initialize TUI (Dashboard)
	dash := tui.NewDashboard(logChan)

	// 8. Start Everything
	go dash.Run()     // Visuals
	hive.Start()      // Logic
	go mgr.StartAll() // Connectivity

	// 9. Keep Alive until Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Clawkido shutting down...")
}
