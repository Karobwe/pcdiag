package collector

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"net"
	"regexp"
	"strings"
)

// The parsers in this file have no build constraint so that they can be
// unit tested on every platform.

// parseProcNetRoute parses /proc/net/route and returns the default IPv4
// gateway with the lowest metric and the interface it belongs to.
func parseProcNetRoute(content string) (gateway, iface string) {
	best := -1
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 7 || f[0] == "Iface" || f[1] != "00000000" {
			continue
		}
		gw := hexToIPv4(f[2])
		if gw == "" || gw == "0.0.0.0" {
			continue
		}
		metric := atoiSafe(f[6])
		if best < 0 || metric < best {
			best = metric
			gateway, iface = gw, f[0]
		}
	}
	return gateway, iface
}

// parseProcNetRouteAll returns the default gateway of every interface.
func parseProcNetRouteAll(content string) map[string]string {
	res := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 3 || f[1] != "00000000" {
			continue
		}
		if gw := hexToIPv4(f[2]); gw != "" && gw != "0.0.0.0" {
			if _, ok := res[f[0]]; !ok {
				res[f[0]] = gw
			}
		}
	}
	return res
}

func hexToIPv4(h string) string {
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 4 {
		return ""
	}
	v := binary.LittleEndian.Uint32(b)
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, v)
	return ip.String()
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// parseResolvConf returns the name servers listed in a resolv.conf file.
func parseResolvConf(content string) []string {
	var servers []string
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 2 && f[0] == "nameserver" {
			servers = appendUnique(servers, f[1])
		}
	}
	return servers
}

// parseRouteGetDefault parses macOS "route -n get default".
func parseRouteGetDefault(out string) (gateway, iface string) {
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "gateway":
			gateway = strings.TrimSpace(v)
		case "interface":
			iface = strings.TrimSpace(v)
		}
	}
	return gateway, iface
}

var scutilNameserver = regexp.MustCompile(`nameserver\[\d+\]\s*:\s*(\S+)`)

// parseScutilDNS parses macOS "scutil --dns".
func parseScutilDNS(out string) []string {
	var servers []string
	for _, m := range scutilNameserver.FindAllStringSubmatch(out, -1) {
		servers = appendUnique(servers, m[1])
	}
	return servers
}

// parseLpstat parses the output of "lpstat -p -d" (CUPS).
func parseLpstat(printers, devices string) []Printer {
	var defaultName string
	byName := map[string]*Printer{}
	var order []string

	for _, line := range strings.Split(printers, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "system default destination:"):
			defaultName = strings.TrimSpace(strings.TrimPrefix(line, "system default destination:"))
		case strings.HasPrefix(line, "printer "):
			f := strings.Fields(line)
			if len(f) < 3 {
				continue
			}
			p := &Printer{Name: f[1], Status: "Unknown"}
			rest := strings.Join(f[2:], " ")
			switch {
			case strings.Contains(rest, "disabled"):
				p.Status, p.Offline = "Disabled", true
			case strings.Contains(rest, "now printing"):
				p.Status = "Printing"
			case strings.Contains(rest, "idle"):
				p.Status = "Idle"
			}
			byName[p.Name] = p
			order = append(order, p.Name)
		}
	}
	for _, line := range strings.Split(devices, "\n") {
		// "device for NAME: URI"
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "device for ") {
			continue
		}
		name, uri, ok := strings.Cut(strings.TrimPrefix(line, "device for "), ":")
		if !ok {
			continue
		}
		if p, ok := byName[name]; ok {
			p.Port = strings.TrimSpace(uri)
			scheme, _, _ := strings.Cut(p.Port, ":")
			switch scheme {
			case "ipp", "ipps", "http", "https", "socket", "lpd", "smb", "dnssd":
				p.Network = true
			}
		}
	}
	out := make([]Printer, 0, len(order))
	for _, n := range order {
		p := byName[n]
		p.Default = n == defaultName
		out = append(out, *p)
	}
	return out
}
