package collector

import (
	"context"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
)

func collectNetwork(ctx context.Context) (NetworkInfo, error) {
	var info NetworkInfo
	ifaces, err := net.Interfaces()
	if err != nil {
		return info, err
	}
	for _, ifc := range ifaces {
		it := Interface{
			Name:     ifc.Name,
			MAC:      strings.ToUpper(ifc.HardwareAddr.String()),
			MTU:      ifc.MTU,
			Up:       ifc.Flags&net.FlagUp != 0 && ifc.Flags&net.FlagRunning != 0,
			Loopback: ifc.Flags&net.FlagLoopback != 0,
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.IP.To4() != nil {
				it.IPv4 = append(it.IPv4, ipnet.String())
			} else {
				it.IPv6 = append(it.IPv6, ipnet.String())
			}
		}
		info.Interfaces = append(info.Interfaces, it)
	}

	info.Proxy = proxyFromEnv()
	enrichErr := enrichNetwork(ctx, &info)

	info.PrimaryIP = primaryIP()
	if info.PrimaryIP == "" {
		for _, it := range info.Interfaces {
			if it.Name == info.GatewayIface && it.HasIPv4() {
				info.PrimaryIP = stripMask(it.IPv4[0])
			}
		}
	}

	sortInterfaces(info.Interfaces, info.PrimaryIP)
	return info, enrichErr
}

// primaryIP returns the local address used to reach the Internet. No packet
// is sent: dialing UDP only selects a route.
func primaryIP() string {
	conn, err := net.Dial("udp4", "192.0.2.1:9") // TEST-NET-1, never routed publicly
	if err != nil {
		return ""
	}
	defer conn.Close()
	if a, ok := conn.LocalAddr().(*net.UDPAddr); ok && !a.IP.IsUnspecified() {
		return a.IP.String()
	}
	return ""
}

// sortInterfaces puts the most relevant adapters first: the one carrying the
// primary IP, then active adapters with IPv4, then the rest, loopback last.
func sortInterfaces(ifs []Interface, primary string) {
	rank := func(it Interface) int {
		switch {
		case it.Loopback:
			return 4
		case primary != "" && containsIP(it.IPv4, primary):
			return 0
		case it.Up && it.HasIPv4():
			return 1
		case it.Up:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(ifs, func(i, j int) bool {
		ri, rj := rank(ifs[i]), rank(ifs[j])
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(ifs[i].Name) < strings.ToLower(ifs[j].Name)
	})
}

func containsIP(cidrs []string, ip string) bool {
	for _, c := range cidrs {
		if stripMask(c) == ip {
			return true
		}
	}
	return false
}

func stripMask(cidr string) string {
	if i := strings.IndexByte(cidr, '/'); i >= 0 {
		return cidr[:i]
	}
	return cidr
}

func proxyFromEnv() string {
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if v := os.Getenv(k); v != "" {
			return redactURL(v)
		}
	}
	return ""
}

// redactURL removes credentials from a proxy URL so they never end up in a
// report pasted into a ticket.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = url.User("***")
	return u.String()
}
