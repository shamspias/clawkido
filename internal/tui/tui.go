package tui

import (
	"clawkido/internal/types"
	"fmt"

	"github.com/fatih/color"
)

type Dashboard struct {
	LogChan <-chan types.LogEntry
}

func NewDashboard(logChan <-chan types.LogEntry) *Dashboard {
	return &Dashboard{LogChan: logChan}
}

func (d *Dashboard) Run() {
	// Clear screen
	fmt.Print("\033[H\033[2J")

	// Banner
	color.Cyan("========================================")
	color.Cyan("   CLAWKIDO - AI AGENT SWARM SYSTEM     ")
	color.Cyan("========================================")
	fmt.Println()

	// Listen for logs
	for entry := range d.LogChan {
		d.printEntry(entry)
	}
}

func (d *Dashboard) printEntry(e types.LogEntry) {
	ts := e.Time.Format("15:04:05")

	var levelColor func(a ...interface{}) string

	switch e.Level {
	case "INFO":
		levelColor = color.New(color.FgBlue).SprintFunc()
	case "SUCCESS":
		levelColor = color.New(color.FgGreen).SprintFunc()
	case "ERROR":
		levelColor = color.New(color.FgRed).SprintFunc()
	case "DEBUG":
		levelColor = color.New(color.FgHiBlack).SprintFunc()
	default:
		levelColor = color.New(color.FgWhite).SprintFunc()
	}

	// Format: [TIME] [LEVEL] [SOURCE] Message
	fmt.Printf("%s | %s | %-10s | %s\n",
		color.WhiteString(ts),
		levelColor(fmt.Sprintf("%-7s", e.Level)),
		color.YellowString(e.Source),
		e.Message,
	)
}
