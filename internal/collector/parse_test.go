package collector

import (
	"reflect"
	"testing"
	"time"
)

func TestParseProcNetRoute(t *testing.T) {
	const routes = `Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT
wlp2s0	00000000	010AA8C0	0003	0	0	600	00000000	0	0	0
enp0s31f6	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
enp0s31f6	0001A8C0	00000000	0001	0	0	100	00FFFFFF	0	0	0
`
	gw, iface := parseProcNetRoute(routes)
	if gw != "192.168.1.1" || iface != "enp0s31f6" {
		t.Errorf("got %s via %s, want 192.168.1.1 via enp0s31f6 (lowest metric)", gw, iface)
	}
	all := parseProcNetRouteAll(routes)
	if all["wlp2s0"] != "192.168.10.1" {
		t.Errorf("per interface gateway = %v", all)
	}
}

func TestParseResolvConf(t *testing.T) {
	got := parseResolvConf("# comment\nnameserver 10.0.0.53\nsearch corp.local\nnameserver 1.1.1.1\nnameserver 10.0.0.53\n")
	want := []string{"10.0.0.53", "1.1.1.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseMacOS(t *testing.T) {
	gw, iface := parseRouteGetDefault("   route to: default\ndestination: default\n    gateway: 192.168.1.254\n  interface: en0\n")
	if gw != "192.168.1.254" || iface != "en0" {
		t.Errorf("route: %s %s", gw, iface)
	}
	dns := parseScutilDNS("resolver #1\n  nameserver[0] : 192.168.1.254\n  nameserver[1] : 8.8.8.8\nresolver #2\n  nameserver[0] : 192.168.1.254\n")
	if !reflect.DeepEqual(dns, []string{"192.168.1.254", "8.8.8.8"}) {
		t.Errorf("dns: %v", dns)
	}
}

func TestParseLpstat(t *testing.T) {
	printers := `printer Office_Laser is idle.  enabled since Mon 12 Oct 2026 09:00:00
printer Label_Printer disabled since Tue 13 Oct 2026 10:00:00 -
	Paused
printer PDF now printing PDF-12.  enabled since Fri 23 Oct 2026
system default destination: Office_Laser
`
	devices := "device for Office_Laser: ipp://192.168.1.40/ipp/print\ndevice for Label_Printer: usb://Zebra/ZD420\ndevice for PDF: cups-pdf:/\n"
	got := parseLpstat(printers, devices)
	if len(got) != 3 {
		t.Fatalf("got %d printers", len(got))
	}
	if p := got[0]; !p.Default || !p.Network || p.Status != "Idle" || p.Port != "ipp://192.168.1.40/ipp/print" {
		t.Errorf("Office_Laser = %+v", p)
	}
	if p := got[1]; !p.Offline || p.Network || p.Status != "Disabled" {
		t.Errorf("Label_Printer = %+v", p)
	}
	if got[2].Status != "Printing" {
		t.Errorf("PDF = %+v", got[2])
	}
}

func TestParseWinHardware(t *testing.T) {
	hw, err := parseWinHardware(`{"Manufacturer":"Dell Inc.","Model":"Latitude 5440","Domain":"corp.local","PartOfDomain":true,"Serial":"8XK2Q93","BIOSVersion":"1.21.0","OSCaption":"Microsoft Windows 11 Pro","OSBuild":"26100","OSVersion":"24H2"}`)
	if err != nil {
		t.Fatal(err)
	}
	if hw.OSName != "Microsoft Windows 11 Pro 24H2 (build 26100)" || !hw.PartOfDomain || hw.SerialNumber != "8XK2Q93" {
		t.Errorf("unexpected %+v", hw)
	}
}

func TestParseWinNetwork(t *testing.T) {
	// Single-element collections are emitted as objects by PowerShell 5.1.
	out := `{"Adapters":[{"Name":"Wi-Fi","Description":"Intel Wi-Fi 6E AX211","Status":"Up","LinkSpeed":"866.7 Mbps","MediaType":"Native 802.11","PhysicalMediaType":"Native 802.11"},
{"Name":"Ethernet","Description":"Intel I219-LM","Status":"Disconnected","LinkSpeed":"0 bps","MediaType":"802.3","PhysicalMediaType":"802.3"}],
"Routes":[{"Alias":"Wi-Fi","NextHop":"192.168.10.1","Metric":50},{"Alias":"VPN","NextHop":"10.8.0.1","Metric":5000},{"Alias":"Wi-Fi","NextHop":"fe80::1","Metric":50}],
"DNS":{"Alias":"Wi-Fi","Servers":["192.168.10.5","192.168.10.6"]},
"DHCP":[{"Alias":"Wi-Fi","Dhcp":"Enabled"},{"Alias":"Ethernet","Dhcp":"Disabled"}],
"Proxy":"proxy.corp.local:8080"}`
	res, err := parseWinNetwork(out)
	if err != nil {
		t.Fatal(err)
	}
	if res.DefaultGateway != "192.168.10.1" || res.GatewayIface != "Wi-Fi" {
		t.Errorf("gateway %s via %s", res.DefaultGateway, res.GatewayIface)
	}
	wifi := res.Ifaces["Wi-Fi"]
	if !wifi.Wireless || wifi.DHCP != "enabled" || len(wifi.DNS) != 2 || wifi.Gateway != "192.168.10.1" {
		t.Errorf("wifi = %+v", wifi)
	}
	if res.Ifaces["Ethernet"].Wireless {
		t.Error("ethernet detected as wireless")
	}
	if res.Proxy != "proxy.corp.local:8080" || !reflect.DeepEqual(res.DNS, []string{"192.168.10.5", "192.168.10.6"}) {
		t.Errorf("proxy %q dns %v", res.Proxy, res.DNS)
	}
}

func TestParseWinPrinting(t *testing.T) {
	out := `{"Spooler":"Running","Printers":{"Name":"HP LaserJet","DriverName":"HP Universal","PortName":"192.168.1.41","Default":true,"Network":true,"Shared":false,"WorkOffline":true,"PrinterStatus":3}}`
	info, err := parseWinPrinting(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.SpoolerStatus != "Running" || len(info.Printers) != 1 {
		t.Fatalf("unexpected %+v", info)
	}
	if p := info.Printers[0]; !p.Offline || p.Status != "Offline" || !p.Default {
		t.Errorf("printer = %+v", p)
	}
}

func TestParseNetshWlan(t *testing.T) {
	fr := `
Il existe 1 interface sur le système :

    Nom                    : Wi-Fi
    Description            : Intel(R) Wi-Fi 6E AX211 160MHz
    État                   : connecté
    SSID                   : CORP-WIFI
    BSSID                  : aa:bb:cc:dd:ee:ff
    Signal                 : 84%
`
	got := parseNetshWlan(fr)
	if got["Wi-Fi"] != [2]string{"CORP-WIFI", "84%"} {
		t.Errorf("got %v", got)
	}
}

func TestParsePingLatency(t *testing.T) {
	cases := map[string]time.Duration{
		"64 bytes from 1.1.1.1: icmp_seq=1 ttl=57 time=12.4 ms": 12400 * time.Microsecond,
		"Réponse de 192.168.1.1 : octets=32 temps=3 ms TTL=64":  3 * time.Millisecond,
		"Reply from 192.168.1.1: bytes=32 time<1ms TTL=64":      500 * time.Microsecond,
		"Antwort von 10.0.0.1: Bytes=32 Zeit=7,5 ms TTL=128":    7500 * time.Microsecond,
		"Request timed out.": 0,
	}
	for in, want := range cases {
		if got := parsePingLatency(in); got != want {
			t.Errorf("%q: got %v, want %v", in, got, want)
		}
	}
}

func TestListUnmarshal(t *testing.T) {
	var w winNetwork
	if err := (&w).DNS.UnmarshalJSON([]byte("null")); err != nil || w.DNS != nil {
		t.Errorf("null: %v %v", err, w.DNS)
	}
}

func TestRedactURL(t *testing.T) {
	if got := redactURL("http://alice:s3cret@proxy:3128"); got != "http://%2A%2A%2A@proxy:3128" && got != "http://***@proxy:3128" {
		t.Errorf("credentials leaked: %s", got)
	}
	if got := redactURL("proxy.corp.local:8080"); got != "proxy.corp.local:8080" {
		t.Errorf("got %s", got)
	}
}

func TestFormatZone(t *testing.T) {
	if got := formatZone("Europe/Paris", "CET", 3600); got != "Europe/Paris (UTC+01:00)" {
		t.Errorf("got %s", got)
	}
	if got := formatZone("Local", "", -5*3600-1800); got != "UTC-05:30" {
		t.Errorf("got %s", got)
	}
}
