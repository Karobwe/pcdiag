//go:build darwin

package collector

import (
	"context"
	"regexp"
	"strings"
)

var ioregSerial = regexp.MustCompile(`"IOPlatformSerialNumber"\s*=\s*"([^"]+)"`)

func collectHardware(ctx context.Context) (hardwareInfo, error) {
	hw := hardwareInfo{Manufacturer: "Apple"}
	if out, err := runCommand(ctx, "sysctl", "-n", "hw.model"); err == nil {
		hw.Model = strings.TrimSpace(out)
	}
	if out, err := runCommand(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice"); err == nil {
		if m := ioregSerial.FindStringSubmatch(out); m != nil {
			hw.SerialNumber = m[1]
		}
	}
	if name, err := runCommand(ctx, "sw_vers", "-productName"); err == nil {
		ver, _ := runCommand(ctx, "sw_vers", "-productVersion")
		hw.OSName = strings.TrimSpace(strings.TrimSpace(name) + " " + strings.TrimSpace(ver))
	}
	return hw, nil
}

func enrichNetwork(ctx context.Context, n *NetworkInfo) error {
	if out, err := runCommand(ctx, "route", "-n", "get", "default"); err == nil {
		n.DefaultGateway, n.GatewayIface = parseRouteGetDefault(out)
	}
	if out, err := runCommand(ctx, "scutil", "--dns"); err == nil {
		n.DNSServers = parseScutilDNS(out)
	}
	for i := range n.Interfaces {
		if n.Interfaces[i].Name == n.GatewayIface {
			n.Interfaces[i].Gateway = n.DefaultGateway
		}
	}
	return nil
}

func collectPrinting(ctx context.Context) (PrintingInfo, error) {
	return collectCUPS(ctx)
}
