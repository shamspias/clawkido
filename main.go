package main

import (
	"clawkido/internal/brain"
	"clawkido/internal/channels"
	"clawkido/internal/config"
	"clawkido/internal/engine"
	"clawkido/internal/tui"
	"clawkido/internal/types"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. Load Configuration (Env -> JSON -> Env Override)
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Setup Log Channel (Buffer 1000 events)
	logChan := make(chan types.LogEntry, 1000)

	// 3. Initialize Brain (Thinking)
	brn, err := brain.NewBrain(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Brain: %v", err)
	}

	// 4. Initialize Engine (Processing)
	eng := engine.NewEngine(cfg, brn, logChan)

	// 5. Initialize Channels (Input/Output)
	mgr := channels.NewManager(cfg, eng.Inbox, logChan)

	// 6. Initialize TUI (Dashboard)
	dash := tui.NewDashboard(logChan)

	// 7. Start everything
	go dash.Run()     // Visuals
	eng.Start()       // Logic
	go mgr.StartAll() // Connectivity

	// 8. Wait for Exit Signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	eng.Stop()
	log.Println("System shutting down...")
}
