package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/config"
)

// sampleSnapshot is a realistic Windows workstation used to exercise every
// widget of the interface (it is also used to produce the screenshots of the
// documentation).
func sampleSnapshot() *collector.Snapshot {
	now := time.Date(2026, 10, 24, 9, 15, 0, 0, time.Local)
	return &collector.Snapshot{
		CollectedAt: now,
		Duration:    2400 * time.Millisecond,
		System: collector.SystemInfo{
			Hostname: "PC-ACCOUNTS-07", User: `CORP\j.martin`, Domain: "corp.example.com", PartOfDomain: true,
			OS: "Microsoft Windows 11 Pro 24H2 (build 26100)", OSVersion: "10.0.26100", KernelVersion: "10.0.26100.4652",
			Arch: "amd64", Manufacturer: "Dell Inc.", Model: "Latitude 5440", SerialNumber: "8XK2Q93",
			BIOSVersion: "1.21.0", CPUModel: "13th Gen Intel(R) Core(TM) i5-1345U", CPUCores: 10, CPUThreads: 12,
			CPUUsage: 18, BootTime: now.Add(-12*24*time.Hour - 3*time.Hour), Uptime: 12*24*time.Hour + 3*time.Hour,
			TimeZone: "Romance Standard Time (UTC+01:00)",
		},
		Memory: collector.MemoryInfo{Total: 16 << 30, Used: 11 << 30, Available: 5 << 30, UsedPercent: 68.7, SwapTotal: 4 << 30, SwapUsed: 1 << 30},
		Disks: []collector.DiskInfo{
			{Mountpoint: "C:", Device: "C:", FSType: "NTFS", Total: 476 << 30, Free: 31 << 30, UsedPercent: 93.5},
			{Mountpoint: "D:", Device: "D:", FSType: "NTFS", Total: 931 << 30, Free: 610 << 30, UsedPercent: 34.5},
		},
		Network: collector.NetworkInfo{
			DefaultGateway: "192.168.10.1", GatewayIface: "Wi-Fi", PrimaryIP: "192.168.10.57",
			DNSServers: []string{"192.168.10.5", "192.168.10.6"},
			Interfaces: []collector.Interface{
				{Name: "Wi-Fi", Description: "Intel(R) Wi-Fi 6E AX211 160MHz", MAC: "A4:C3:F0:12:9B:4E", MTU: 1500, Up: true, Wireless: true,
					IPv4: []string{"192.168.10.57/24"}, IPv6: []string{"fe80::8d1c:2f4a:91b3:7c2e/64"}, Gateway: "192.168.10.1",
					DNSServers: []string{"192.168.10.5", "192.168.10.6"}, DHCP: "enabled", LinkSpeed: "866.7 Mbps", SSID: "CORP-WIFI", Signal: "84%"},
				{Name: "Ethernet", Description: "Intel(R) Ethernet Connection (16) I219-LM", MAC: "C4:5A:B1:7E:20:11", MTU: 1500, DHCP: "enabled"},
				{Name: "Loopback Pseudo-Interface 1", MTU: -1, Up: true, Loopback: true, IPv4: []string{"127.0.0.1/8"}},
			},
		},
		Connectivity: []collector.CheckResult{
			{Name: "Default gateway", Kind: "ping", Target: "192.168.10.1", Status: collector.StatusOK, Latency: 3 * time.Millisecond},
			{Name: "DNS server", Kind: "dns", Target: "192.168.10.5", Status: collector.StatusOK, Latency: 12 * time.Millisecond, Detail: "20.70.246.20"},
			{Name: "Name resolution", Kind: "dns", Target: "www.microsoft.com", Status: collector.StatusOK, Latency: 9 * time.Millisecond, Detail: "20.70.246.20"},
			{Name: "Internet (TCP)", Kind: "tcp", Target: "1.1.1.1:443", Status: collector.StatusOK, Latency: 21 * time.Millisecond},
			{Name: "Internet (HTTP)", Kind: "http", Target: "http://www.msftconnecttest.com/connecttest.txt", Status: collector.StatusOK, Latency: 48 * time.Millisecond, Detail: "HTTP 200"},
			{Name: "File server", Kind: "tcp", Target: "fs01.corp.example.com:445", Status: collector.StatusFailed, Detail: "timeout"},
		},
		Printing: collector.PrintingInfo{SpoolerStatus: "Running", Printers: []collector.Printer{
			{Name: "Canon iR-ADV C5535 - Accounts", Driver: "Canon Generic Plus UFR II", Port: "192.168.10.40", Status: "Idle", Default: true, Network: true},
			{Name: "HP LaserJet M404 - Reception", Driver: "HP Universal Printing PCL 6", Port: "192.168.10.41", Status: "Offline", Network: true, Offline: true},
			{Name: "Microsoft Print to PDF", Driver: "Microsoft Print To PDF", Port: "PORTPROMPT:", Status: "Idle"},
		}},
	}
}

func newTestUI(t *testing.T) *ui {
	t.Helper()
	a := test.NewTempApp(t)
	a.Settings().SetTheme(theme.DefaultTheme())
	cfg := config.Default()
	cfg.Support = config.Support{Name: "IT Service Desk", Phone: "+33 1 23 45 67 89", Email: "helpdesk@example.com"}
	u := newUI(a, Params{Version: "v1.0.0", Config: cfg})
	u.setSnapshot(sampleSnapshot())
	return u
}

func TestRenderAllTabs(t *testing.T) {
	u := newTestUI(t)
	dir := os.Getenv("PCDIAG_SCREENSHOTS") // set to export PNG files
	names := []string{"overview", "system", "network", "connectivity", "printers"}
	if len(names) != len(u.tabs.Items) {
		t.Fatalf("expected %d tabs, got %d", len(names), len(u.tabs.Items))
	}
	for i, item := range u.tabs.Items {
		u.tabs.SelectIndex(i)
		img := u.win.Canvas().Capture()
		if img.Bounds().Dx() == 0 {
			t.Fatalf("tab %q rendered an empty image", item.Text)
		}
		if dir == "" {
			continue
		}
		f, err := os.Create(filepath.Join(dir, names[i]+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
}

func TestReportContainsFindings(t *testing.T) {
	u := newTestUI(t)
	r := u.textReport()
	for _, want := range []string{"PC-ACCOUNTS-07", "a restart is recommended", "Disk C: is almost full", "File server", "HP LaserJet M404 - Reception is offline"} {
		if !strings.Contains(r, want) {
			t.Errorf("report does not contain %q", want)
		}
	}
}

func TestInactiveAdaptersToggle(t *testing.T) {
	u := newTestUI(t)
	visible := len(u.network.Objects)
	u.showInactive = true
	u.renderNetwork()
	if got := len(u.network.Objects); got <= visible {
		t.Errorf("showing inactive adapters should add cards: %d -> %d", visible, got)
	}
}
