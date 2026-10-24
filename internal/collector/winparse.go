package collector

import (
	"bytes"
	"encoding/json"
	"strings"
)

// This file contains the parsers for the JSON emitted by the PowerShell
// scripts used on Windows. It has no build constraint so that it can be unit
// tested on every platform.

// list decodes either a JSON array or a single JSON object, working around
// PowerShell's ConvertTo-Json which unwraps single-element arrays.
type list[T any] []T

func (l *list[T]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*l = nil
		return nil
	}
	if b[0] == '[' {
		var s []T
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*l = s
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*l = []T{v}
	return nil
}

type winHardware struct {
	Manufacturer string `json:"Manufacturer"`
	Model        string `json:"Model"`
	Domain       string `json:"Domain"`
	PartOfDomain bool   `json:"PartOfDomain"`
	Serial       string `json:"Serial"`
	BIOSVersion  string `json:"BIOSVersion"`
	OSCaption    string `json:"OSCaption"`
	OSBuild      string `json:"OSBuild"`
	OSVersion    string `json:"OSVersion"`
}

func parseWinHardware(out string) (hardwareInfo, error) {
	var w winHardware
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &w); err != nil {
		return hardwareInfo{}, err
	}
	osName := strings.TrimSpace(w.OSCaption)
	if w.OSVersion != "" {
		osName += " " + w.OSVersion
	}
	if w.OSBuild != "" {
		osName += " (build " + w.OSBuild + ")"
	}
	return hardwareInfo{
		Manufacturer: strings.TrimSpace(w.Manufacturer),
		Model:        strings.TrimSpace(w.Model),
		SerialNumber: strings.TrimSpace(w.Serial),
		BIOSVersion:  strings.TrimSpace(w.BIOSVersion),
		Domain:       strings.TrimSpace(w.Domain),
		PartOfDomain: w.PartOfDomain,
		OSName:       strings.TrimSpace(osName),
	}, nil
}

type winNetwork struct {
	Adapters list[struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
		Status      string `json:"Status"`
		LinkSpeed   string `json:"LinkSpeed"`
		MediaType   string `json:"MediaType"`
		Physical    string `json:"PhysicalMediaType"`
	}] `json:"Adapters"`
	Routes list[struct {
		Alias   string `json:"Alias"`
		NextHop string `json:"NextHop"`
		Metric  int    `json:"Metric"`
	}] `json:"Routes"`
	DNS list[struct {
		Alias   string       `json:"Alias"`
		Servers list[string] `json:"Servers"`
	}] `json:"DNS"`
	DHCP list[struct {
		Alias string `json:"Alias"`
		Dhcp  string `json:"Dhcp"`
	}] `json:"DHCP"`
	Proxy string `json:"Proxy"`
}

// winIfaceExtra is the per-adapter data only available through PowerShell.
type winIfaceExtra struct {
	Description string
	LinkSpeed   string
	Wireless    bool
	Gateway     string
	DNS         []string
	DHCP        string
}

type winNetworkResult struct {
	Ifaces         map[string]*winIfaceExtra
	DefaultGateway string
	GatewayIface   string
	DNS            []string
	Proxy          string
}

func parseWinNetwork(out string) (winNetworkResult, error) {
	var w winNetwork
	res := winNetworkResult{Ifaces: map[string]*winIfaceExtra{}}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &w); err != nil {
		return res, err
	}
	get := func(alias string) *winIfaceExtra {
		e, ok := res.Ifaces[alias]
		if !ok {
			e = &winIfaceExtra{}
			res.Ifaces[alias] = e
		}
		return e
	}
	for _, a := range w.Adapters {
		e := get(a.Name)
		e.Description = a.Description
		e.LinkSpeed = a.LinkSpeed
		media := strings.ToLower(a.MediaType + " " + a.Physical)
		e.Wireless = strings.Contains(media, "802.11") || strings.Contains(media, "wireless")
	}
	bestMetric := -1
	for _, r := range w.Routes {
		if r.NextHop == "" || r.NextHop == "0.0.0.0" || r.NextHop == "::" {
			continue
		}
		e := get(r.Alias)
		if e.Gateway == "" {
			e.Gateway = r.NextHop
		}
		if strings.Contains(r.NextHop, ".") && (bestMetric < 0 || r.Metric < bestMetric) {
			bestMetric = r.Metric
			res.DefaultGateway = r.NextHop
			res.GatewayIface = r.Alias
		}
	}
	for _, d := range w.DNS {
		e := get(d.Alias)
		for _, s := range d.Servers {
			e.DNS = appendUnique(e.DNS, s)
		}
	}
	if gw, ok := res.Ifaces[res.GatewayIface]; ok {
		res.DNS = append(res.DNS, gw.DNS...)
	}
	for _, d := range w.DNS {
		for _, s := range d.Servers {
			if !strings.HasPrefix(strings.ToLower(s), "fec0:") {
				res.DNS = appendUnique(res.DNS, s)
			}
		}
	}
	for _, d := range w.DHCP {
		get(d.Alias).DHCP = strings.ToLower(d.Dhcp)
	}
	res.Proxy = strings.TrimSpace(w.Proxy)
	return res, nil
}

type winPrinting struct {
	Spooler  string `json:"Spooler"`
	Printers list[struct {
		Name          string `json:"Name"`
		DriverName    string `json:"DriverName"`
		PortName      string `json:"PortName"`
		Default       bool   `json:"Default"`
		Network       bool   `json:"Network"`
		Shared        bool   `json:"Shared"`
		WorkOffline   bool   `json:"WorkOffline"`
		PrinterStatus int    `json:"PrinterStatus"`
	}] `json:"Printers"`
}

// win32PrinterStatus maps Win32_Printer.PrinterStatus values.
var win32PrinterStatus = map[int]string{
	1: "Unknown", 2: "Unknown", 3: "Idle", 4: "Printing",
	5: "Warming up", 6: "Stopped printing", 7: "Offline",
}

func parseWinPrinting(out string) (PrintingInfo, error) {
	var w winPrinting
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &w); err != nil {
		return PrintingInfo{}, err
	}
	info := PrintingInfo{SpoolerStatus: w.Spooler, Printers: []Printer{}}
	for _, p := range w.Printers {
		status := win32PrinterStatus[p.PrinterStatus]
		if status == "" {
			status = "Unknown"
		}
		offline := p.WorkOffline || p.PrinterStatus == 7
		if offline {
			status = "Offline"
		}
		info.Printers = append(info.Printers, Printer{
			Name: p.Name, Driver: p.DriverName, Port: p.PortName,
			Status: status, Default: p.Default, Network: p.Network,
			Shared: p.Shared, Offline: offline,
		})
	}
	return info, nil
}

// parseNetshWlan extracts the SSID and signal per interface from
// "netsh wlan show interfaces". Keys such as "SSID" and "Signal" are not
// translated by Windows, while "Name" is, so interfaces are delimited by the
// first key of each block.
func parseNetshWlan(out string) map[string][2]string {
	res := map[string][2]string{}
	var current string
	var ssid, signal string
	flush := func() {
		if current != "" && ssid != "" {
			res[current] = [2]string{ssid, signal}
		}
		ssid, signal = "", ""
	}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		switch {
		case isNameKey(key):
			flush()
			current = val
		case key == "SSID":
			ssid = val
		case key == "Signal":
			signal = val
		}
	}
	flush()
	return res
}

// isNameKey matches the localized "Name" key of netsh output.
func isNameKey(k string) bool {
	switch strings.ToLower(k) {
	case "name", "nom", "nombre", "nome", "naam":
		return true
	}
	return false
}

func appendUnique(s []string, v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return s
	}
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
