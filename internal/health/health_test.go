package health

import (
	"strings"
	"testing"
	"time"

	"github.com/Karobwe/pcdiag/internal/collector"
)

func healthy() *collector.Snapshot {
	return &collector.Snapshot{
		System: collector.SystemInfo{Uptime: 2 * time.Hour, CPUUsage: 10},
		Memory: collector.MemoryInfo{UsedPercent: 40},
		Disks:  []collector.DiskInfo{{Mountpoint: "C:", UsedPercent: 50, Free: 100 << 30}},
		Network: collector.NetworkInfo{
			DefaultGateway: "192.168.1.1",
			Interfaces:     []collector.Interface{{Name: "Ethernet", Up: true, IPv4: []string{"192.168.1.20/24"}}},
		},
		Connectivity: []collector.CheckResult{{Name: "Default gateway", Status: collector.StatusOK}},
		Printing:     collector.PrintingInfo{SpoolerStatus: "Running", Printers: []collector.Printer{{Name: "P1", Default: true}}},
	}
}

func TestHealthySnapshot(t *testing.T) {
	if f := Analyze(healthy(), DefaultThresholds()); len(f) != 0 {
		t.Errorf("expected no finding, got %+v", f)
	}
}

func TestFindings(t *testing.T) {
	s := healthy()
	s.System.Uptime = 30 * 24 * time.Hour
	s.Disks[0].UsedPercent = 98
	s.Network.Interfaces[0].IPv4 = []string{"169.254.12.3/16"}
	s.Connectivity = append(s.Connectivity, collector.CheckResult{Name: "DNS", Target: "10.0.0.1", Status: collector.StatusFailed, Detail: "timeout"})
	s.Printing = collector.PrintingInfo{SpoolerStatus: "Stopped", Printers: []collector.Printer{{Name: "P1", Offline: true}}}

	f := Analyze(s, DefaultThresholds())
	if Worst(f) != Critical || f[0].Level != Critical {
		t.Fatalf("findings must be sorted by severity: %+v", f)
	}
	var texts []string
	for _, x := range f {
		texts = append(texts, x.Text())
	}
	all := strings.Join(texts, "\n")
	for _, want := range []string{"restart is recommended", "Disk C: is almost full", "self-assigned address (169.254.12.3)",
		"No active network connection", "DNS (10.0.0.1) - timeout", "spooler is not running (Stopped)", "P1 is offline", "No default printer"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in:\n%s", want, all)
		}
	}
}

func TestNoGateway(t *testing.T) {
	s := healthy()
	s.Network.DefaultGateway = ""
	f := Analyze(s, DefaultThresholds())
	if len(f) != 1 || f[0].Message != MsgNoGateway {
		t.Errorf("got %+v", f)
	}
}

func TestRenderFallback(t *testing.T) {
	if Render("{{.Broken", nil) != "{{.Broken" {
		t.Error("invalid templates must be returned unchanged")
	}
}
