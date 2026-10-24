// Package report renders a Snapshot as plain text (to paste into a ticket)
// or JSON (for tooling). Reports are intentionally written in English so
// that they can be read by any support team.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/format"
	"github.com/Karobwe/pcdiag/internal/health"
)

// JSON writes the snapshot as indented JSON.
func JSON(w io.Writer, s *collector.Snapshot) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// Text writes a human-readable report.
func Text(w io.Writer, s *collector.Snapshot, findings []health.Finding, appVersion string) error {
	b := &builder{}
	b.title(fmt.Sprintf("PC Diag report - %s", s.System.Hostname))
	b.kv("Generated", s.CollectedAt.Format("2006-01-02 15:04:05 MST"))
	b.kv("Tool version", appVersion)

	b.section("Findings")
	if len(findings) == 0 {
		b.line("  " + health.MsgAllGood)
	}
	for _, f := range findings {
		b.line(fmt.Sprintf("  [%s] %s", f.Level, f.Text()))
	}

	sys := s.System
	b.section("System")
	b.kv("Hostname", sys.Hostname)
	b.kv("User", sys.User)
	if sys.Domain != "" {
		kind := "Workgroup"
		if sys.PartOfDomain {
			kind = "Domain"
		}
		b.kv(kind, sys.Domain)
	}
	b.kv("Operating system", sys.OS)
	b.kv("Kernel", sys.KernelVersion)
	b.kv("Architecture", sys.Arch)
	b.kv("Manufacturer", sys.Manufacturer)
	b.kv("Model", sys.Model)
	b.kv("Serial number", sys.SerialNumber)
	b.kv("BIOS", sys.BIOSVersion)
	b.kv("CPU", fmt.Sprintf("%s (%d cores / %d threads)", sys.CPUModel, sys.CPUCores, sys.CPUThreads))
	b.kv("CPU usage", format.Percent(sys.CPUUsage))
	if !sys.BootTime.IsZero() {
		b.kv("Last boot", sys.BootTime.Format("2006-01-02 15:04"))
	}
	b.kv("Uptime", format.Uptime(sys.Uptime))
	b.kv("Time zone", sys.TimeZone)

	m := s.Memory
	b.section("Memory")
	b.kv("Total", format.Bytes(m.Total))
	b.kv("Used", fmt.Sprintf("%s (%s)", format.Bytes(m.Used), format.Percent(m.UsedPercent)))
	b.kv("Available", format.Bytes(m.Available))
	if m.SwapTotal > 0 {
		b.kv("Swap / page file", fmt.Sprintf("%s / %s", format.Bytes(m.SwapUsed), format.Bytes(m.SwapTotal)))
	}

	b.section("Storage")
	for _, d := range s.Disks {
		b.kv(d.Mountpoint, fmt.Sprintf("%s free of %s (%s used, %s)",
			format.Bytes(d.Free), format.Bytes(d.Total), format.Percent(d.UsedPercent), d.FSType))
	}

	n := s.Network
	b.section("Network")
	b.kv("Primary IP", n.PrimaryIP)
	b.kv("Default gateway", strings.TrimSpace(n.DefaultGateway+" "+paren(n.GatewayIface)))
	b.kv("DNS servers", format.Join(n.DNSServers, ", "))
	b.kv("Proxy", n.Proxy)
	for _, it := range n.Interfaces {
		if it.Loopback || (!it.Up && !it.HasIPv4()) {
			continue
		}
		b.sub(it.Name + " " + paren(it.Description))
		b.kv("Status", upDown(it.Up))
		b.kv("MAC", it.MAC)
		b.kv("IPv4", strings.Join(it.IPv4, ", "))
		b.kv("IPv6", strings.Join(it.IPv6, ", "))
		b.kv("Gateway", it.Gateway)
		b.kv("DNS", strings.Join(it.DNSServers, ", "))
		b.kv("DHCP", it.DHCP)
		b.kv("Link speed", it.LinkSpeed)
		if it.SSID != "" {
			b.kv("Wi-Fi", strings.TrimSpace(it.SSID+" "+paren(it.Signal)))
		}
	}

	b.section("Connectivity")
	if len(s.Connectivity) == 0 {
		b.line("  (not run)")
	}
	for _, c := range s.Connectivity {
		b.line(fmt.Sprintf("  [%-7s] %-18s %-40s %8s  %s",
			strings.ToUpper(string(c.Status)), c.Name, c.Target, format.Latency(c.Latency), c.Detail))
	}

	p := s.Printing
	b.section("Printers")
	b.kv("Spooler", p.SpoolerStatus)
	if len(p.Printers) == 0 {
		b.line("  (no printer installed)")
	}
	for _, pr := range p.Printers {
		flags := []string{pr.Status}
		if pr.Default {
			flags = append(flags, "default")
		}
		if pr.Network {
			flags = append(flags, "network")
		}
		if pr.Shared {
			flags = append(flags, "shared")
		}
		b.kv(pr.Name, fmt.Sprintf("%s - port %s - driver %s",
			strings.Join(flags, ", "), format.OrDash(pr.Port), format.OrDash(pr.Driver)))
	}

	if len(s.Errors) > 0 {
		b.section("Collection errors")
		for k, v := range s.Errors {
			b.kv(k, v)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func paren(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}

func upDown(up bool) string {
	if up {
		return "Connected"
	}
	return "Disconnected"
}

// builder aligns key/value pairs within each section.
type builder struct {
	out strings.Builder
	tw  *tabwriter.Writer
	buf strings.Builder
}

func (b *builder) flush() {
	if b.tw != nil {
		_ = b.tw.Flush()
		b.out.WriteString(b.buf.String())
		b.buf.Reset()
		b.tw = nil
	}
}

func (b *builder) title(t string) {
	b.flush()
	b.out.WriteString(t + "\n" + strings.Repeat("=", len(t)) + "\n")
}

func (b *builder) section(t string) {
	b.flush()
	b.out.WriteString("\n" + t + "\n" + strings.Repeat("-", len(t)) + "\n")
}

func (b *builder) sub(t string) {
	b.flush()
	b.out.WriteString("  > " + strings.TrimSpace(t) + "\n")
}

func (b *builder) line(t string) {
	b.flush()
	b.out.WriteString(t + "\n")
}

func (b *builder) kv(k, v string) {
	if strings.TrimSpace(v) == "" {
		return
	}
	if b.tw == nil {
		b.tw = tabwriter.NewWriter(&b.buf, 0, 0, 2, ' ', 0)
	}
	fmt.Fprintf(b.tw, "  %s:\t%s\n", k, v)
}

func (b *builder) String() string {
	b.flush()
	return b.out.String()
}
