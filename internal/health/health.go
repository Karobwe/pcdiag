// Package health analyses a Snapshot and highlights the issues a support
// technician would look for first.
package health

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/format"
)

// Level is the severity of a finding.
type Level int

// Severity levels, from least to most severe.
const (
	Info Level = iota
	Warning
	Critical
)

func (l Level) String() string {
	switch l {
	case Warning:
		return "WARNING"
	case Critical:
		return "CRITICAL"
	default:
		return "INFO"
	}
}

// Finding is one diagnostic remark. Message is an English text/template
// that also serves as the translation key; Params holds its values.
type Finding struct {
	Level   Level
	Message string
	Params  map[string]any
}

// Text renders the English message.
func (f Finding) Text() string { return Render(f.Message, f.Params) }

// Render executes a message template, falling back to the raw template.
func Render(tmpl string, params map[string]any) string {
	t, err := template.New("m").Parse(tmpl)
	if err != nil {
		return tmpl
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, params); err != nil {
		return tmpl
	}
	return buf.String()
}

// Thresholds configures the analysis.
type Thresholds struct {
	MemoryPercent float64
	DiskPercent   float64
	MaxUptime     time.Duration
}

// DefaultThresholds returns the defaults used when none are configured.
func DefaultThresholds() Thresholds {
	return Thresholds{MemoryPercent: 90, DiskPercent: 90, MaxUptime: 7 * 24 * time.Hour}
}

// Message templates. They are exported so that translation files can be
// checked against them in tests.
const (
	MsgUptime         = "The computer has been running for {{.Uptime}}: a restart is recommended."
	MsgMemory         = "Memory usage is high ({{.Percent}})."
	MsgCPU            = "CPU usage is high ({{.Percent}})."
	MsgDiskFull       = "Disk {{.Disk}} is almost full ({{.Free}} free)."
	MsgNoNetwork      = "No active network connection with an IPv4 address."
	MsgAPIPA          = "{{.Iface}} has a self-assigned address ({{.IP}}): the DHCP server did not answer."
	MsgNoGateway      = "No default gateway is configured."
	MsgCheckFailed    = "Connectivity check failed: {{.Name}} ({{.Target}}) - {{.Detail}}."
	MsgCheckWarning   = "Connectivity check degraded: {{.Name}} - {{.Detail}}."
	MsgSpoolerStopped = "The print spooler is not running ({{.Status}})."
	MsgPrinterOffline = "Printer {{.Name}} is offline."
	MsgNoDefault      = "No default printer is set."
	MsgAllGood        = "No issue detected."
)

// Analyze returns the findings sorted from most to least severe.
func Analyze(s *collector.Snapshot, th Thresholds) []Finding {
	var out []Finding
	add := func(l Level, msg string, kv ...any) {
		p := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			p[fmt.Sprint(kv[i])] = kv[i+1]
		}
		out = append(out, Finding{Level: l, Message: msg, Params: p})
	}

	if th.MaxUptime > 0 && s.System.Uptime > th.MaxUptime {
		add(Warning, MsgUptime, "Uptime", format.Uptime(s.System.Uptime))
	}
	if th.MemoryPercent > 0 && s.Memory.UsedPercent >= th.MemoryPercent {
		add(Warning, MsgMemory, "Percent", format.Percent(s.Memory.UsedPercent))
	}
	if s.System.CPUUsage >= 90 {
		add(Warning, MsgCPU, "Percent", format.Percent(s.System.CPUUsage))
	}
	for _, d := range s.Disks {
		if th.DiskPercent > 0 && d.UsedPercent >= th.DiskPercent {
			lvl := Warning
			if d.UsedPercent >= 97 {
				lvl = Critical
			}
			add(lvl, MsgDiskFull, "Disk", d.Mountpoint, "Free", format.Bytes(d.Free))
		}
	}

	active := 0
	for _, it := range s.Network.Interfaces {
		if it.Loopback || !it.Up {
			continue
		}
		for _, ip := range it.IPv4 {
			if strings.HasPrefix(ip, "169.254.") {
				add(Critical, MsgAPIPA, "Iface", it.Name, "IP", strings.Split(ip, "/")[0])
			} else {
				active++
			}
		}
	}
	if active == 0 {
		add(Critical, MsgNoNetwork)
	} else if s.Network.DefaultGateway == "" {
		add(Critical, MsgNoGateway)
	}

	for _, c := range s.Connectivity {
		switch c.Status {
		case collector.StatusFailed:
			add(Critical, MsgCheckFailed, "Name", c.Name, "Target", c.Target, "Detail", c.Detail)
		case collector.StatusWarning:
			add(Warning, MsgCheckWarning, "Name", c.Name, "Detail", c.Detail)
		}
	}

	sp := strings.ToLower(s.Printing.SpoolerStatus)
	if sp != "" && sp != "running" && sp != "not installed" {
		add(Critical, MsgSpoolerStopped, "Status", s.Printing.SpoolerStatus)
	}
	hasDefault := false
	for _, p := range s.Printing.Printers {
		hasDefault = hasDefault || p.Default
		if p.Offline {
			add(Warning, MsgPrinterOffline, "Name", p.Name)
		}
	}
	if len(s.Printing.Printers) > 0 && !hasDefault {
		add(Info, MsgNoDefault)
	}

	// Stable sort by severity, most severe first.
	sorted := make([]Finding, 0, len(out))
	for _, l := range []Level{Critical, Warning, Info} {
		for _, f := range out {
			if f.Level == l {
				sorted = append(sorted, f)
			}
		}
	}
	return sorted
}

// Worst returns the highest level among the findings, or -1 if none.
func Worst(fs []Finding) Level {
	w := Level(-1)
	for _, f := range fs {
		if f.Level > w {
			w = f.Level
		}
	}
	return w
}
