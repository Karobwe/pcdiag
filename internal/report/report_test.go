package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/health"
)

func snapshot() *collector.Snapshot {
	return &collector.Snapshot{
		System: collector.SystemInfo{Hostname: "PC-01", User: "jdoe", Domain: "CORP", PartOfDomain: true},
		Network: collector.NetworkInfo{PrimaryIP: "10.0.0.5", Interfaces: []collector.Interface{
			{Name: "Ethernet", Up: true, IPv4: []string{"10.0.0.5/24"}},
			{Name: "lo", Up: true, Loopback: true, IPv4: []string{"127.0.0.1/8"}},
		}},
		Connectivity: []collector.CheckResult{{Name: "Default gateway", Target: "10.0.0.1", Status: collector.StatusOK}},
		Printing:     collector.PrintingInfo{SpoolerStatus: "Running", Printers: []collector.Printer{{Name: "Laser", Default: true, Status: "Idle"}}},
	}
}

func TestText(t *testing.T) {
	var buf bytes.Buffer
	if err := Text(&buf, snapshot(), nil, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"PC Diag report - PC-01", "Domain:", "CORP", "> Ethernet", "[OK     ] Default gateway", "Idle, default", health.MsgAllGood} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in report:\n%s", want, out)
		}
	}
	if strings.Contains(out, "> lo") {
		t.Error("loopback adapters must not be listed")
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, snapshot()); err != nil {
		t.Fatal(err)
	}
	var back collector.Snapshot
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatal(err)
	}
	if back.System.Hostname != "PC-01" || len(back.Printing.Printers) != 1 {
		t.Errorf("round trip failed: %+v", back)
	}
}
