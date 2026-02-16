package skills

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// SysInfoSkill returns system information (OS, arch, memory, uptime).
type SysInfoSkill struct {
	startedAt time.Time
}

// NewSysInfoSkill creates a SysInfoSkill that tracks uptime from now.
func NewSysInfoSkill() *SysInfoSkill {
	return &SysInfoSkill{startedAt: time.Now()}
}

func (s *SysInfoSkill) Name() string { return "sysinfo" }

func (s *SysInfoSkill) Description() string {
	return "Show system info (OS, arch, memory, goroutines, uptime). Usage: [!sysinfo]"
}

func (s *SysInfoSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (s *SysInfoSkill) Execute(_ context.Context, _ string) (string, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, _ := os.Hostname()
	uptime := time.Since(s.startedAt).Round(time.Second)

	var b strings.Builder
	b.WriteString("🖥️ System Info\n")
	b.WriteString(fmt.Sprintf("  Host:       %s\n", hostname))
	b.WriteString(fmt.Sprintf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH))
	b.WriteString(fmt.Sprintf("  Go:         %s\n", runtime.Version()))
	b.WriteString(fmt.Sprintf("  CPUs:       %d\n", runtime.NumCPU()))
	b.WriteString(fmt.Sprintf("  Goroutines: %d\n", runtime.NumGoroutine()))
	b.WriteString(fmt.Sprintf("  Heap:       %.1f MB\n", float64(m.Alloc)/1024/1024))
	b.WriteString(fmt.Sprintf("  Sys mem:    %.1f MB\n", float64(m.Sys)/1024/1024))
	b.WriteString(fmt.Sprintf("  Uptime:     %s\n", uptime))

	return b.String(), nil
}
