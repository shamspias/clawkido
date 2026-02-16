package tui

import (
	"clawkido/internal/types"
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

type Dashboard struct {
	logChan <-chan types.LogEntry
}

func NewDashboard(logChan <-chan types.LogEntry) *Dashboard {
	return &Dashboard{logChan: logChan}
}

func (d *Dashboard) Run(ctx context.Context) {
	fmt.Print("\033[H\033[2J")

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.FgHiBlack).SprintFunc()

	fmt.Println(cyan("╔══════════════════════════════════════════════════╗"))
	fmt.Println(cyan("║       🦞 CLAWKIDO — AI AGENT SWARM ENGINE       ║"))
	fmt.Println(cyan("╚══════════════════════════════════════════════════╝"))
	fmt.Println(dim("  Actor-model orchestration • Go • Zero-latency"))
	fmt.Println(dim("  Press Ctrl+C to shutdown gracefully"))
	fmt.Println(strings.Repeat("─", 52))
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			color.Yellow("⏹  Swarm shutting down...")
			return
		case entry, ok := <-d.logChan:
			if !ok {
				return
			}
			d.render(entry)
		}
	}
}

func (d *Dashboard) render(e types.LogEntry) {
	ts := color.HiBlackString(e.Time.Format("15:04:05.000"))

	var level string
	switch e.Level {
	case "INFO":
		level = color.BlueString("%-7s", "INFO")
	case "SUCCESS":
		level = color.GreenString("%-7s", "OK")
	case "ERROR":
		level = color.RedString("%-7s", "ERROR")
	case "WARN":
		level = color.YellowString("%-7s", "WARN")
	case "DEBUG":
		level = color.HiBlackString("%-7s", "DEBUG")
	default:
		level = color.WhiteString("%-7s", e.Level)
	}

	source := color.CyanString("%-14s", e.Source)

	msg := e.Message
	if strings.Contains(msg, "@") {
		msg = highlightMentions(msg)
	}

	fmt.Printf("%s │ %s │ %s │ %s\n", ts, level, source, msg)
}

func highlightMentions(text string) string {
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	words := strings.Fields(text)
	for i, w := range words {
		if strings.HasPrefix(w, "@") {
			words[i] = yellow(w)
		}
	}
	return strings.Join(words, " ")
}
