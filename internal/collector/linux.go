//go:build linux

package collector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func readTrim(path string) string {
	b, err := os.ReadFile(path) //nolint:gosec // fixed system paths
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func collectHardware(_ context.Context) (hardwareInfo, error) {
	const dmi = "/sys/class/dmi/id/"
	hw := hardwareInfo{
		Manufacturer: readTrim(dmi + "sys_vendor"),
		Model:        strings.TrimSpace(readTrim(dmi+"product_name") + " " + readTrim(dmi+"product_version")),
		SerialNumber: readTrim(dmi + "product_serial"), // usually root only
		BIOSVersion:  readTrim(dmi + "bios_version"),
	}
	if d := readTrim("/proc/sys/kernel/domainname"); d != "" && d != "(none)" {
		hw.Domain = d
	}
	if pretty := osReleasePrettyName(readTrim("/etc/os-release")); pretty != "" {
		hw.OSName = pretty
	}
	return hw, nil
}

func osReleasePrettyName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`)
		}
	}
	return ""
}

func enrichNetwork(ctx context.Context, n *NetworkInfo) error {
	routes := readTrim("/proc/net/route")
	n.DefaultGateway, n.GatewayIface = parseProcNetRoute(routes)
	perIface := parseProcNetRouteAll(routes)

	// Prefer the real upstream servers over the systemd-resolved stub.
	for _, f := range []string{"/run/systemd/resolve/resolv.conf", "/etc/resolv.conf"} {
		if s := parseResolvConf(readTrim(f)); len(s) > 0 {
			n.DNSServers = s
			break
		}
	}

	wifi := nmcliWifi(ctx)
	for i := range n.Interfaces {
		it := &n.Interfaces[i]
		base := filepath.Join("/sys/class/net", it.Name)
		if _, err := os.Stat(filepath.Join(base, "wireless")); err == nil {
			it.Wireless = true
		}
		if sp := readTrim(filepath.Join(base, "speed")); sp != "" && !strings.HasPrefix(sp, "-") {
			it.LinkSpeed = sp + " Mbps"
		}
		it.Gateway = perIface[it.Name]
		if ssid, ok := wifi[it.Name]; ok {
			it.SSID, it.Signal = ssid[0], ssid[1]
		}
	}
	return nil
}

// nmcliWifi returns SSID and signal for active Wi-Fi connections when
// NetworkManager is available.
func nmcliWifi(ctx context.Context) map[string][2]string {
	res := map[string][2]string{}
	out, err := runCommand(ctx, "nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL,DEVICE", "dev", "wifi")
	if err != nil {
		return res
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, ":")
		if len(f) == 4 && f[0] == "yes" {
			res[f[3]] = [2]string{f[1], f[2] + " %"}
		}
	}
	return res
}

func collectPrinting(ctx context.Context) (PrintingInfo, error) {
	return collectCUPS(ctx)
}
