package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/format"
	"github.com/Karobwe/pcdiag/internal/health"
)

func setContent(c *fyne.Container, objs ...fyne.CanvasObject) {
	c.Objects = objs
	c.Refresh()
}

// tr translates a value coming from the collectors (status names, check
// names...). Unknown values are returned unchanged.
func tr(s string) string {
	if s == "" {
		return ""
	}
	return lang.L(s)
}

func (u *ui) renderOverview() {
	s := u.snap
	sys, n := s.System, s.Network

	// Findings first: this is what the technician is looking for.
	findings := container.NewVBox()
	if len(u.findings) == 0 {
		findings.Add(message(iconOK(), lang.L(health.MsgAllGood)))
	}
	for _, f := range u.findings {
		findings.Add(message(levelIcon(f.Level), lang.L(f.Message, f.Params)))
	}
	findingsCard := widget.NewCard(lang.L("Diagnosis"), "", findings)

	// Computer tile.
	computer := tile(theme.ComputerIcon(), sys.Hostname, form(
		kv{lang.L("User"), sys.User},
		kv{domainLabel(sys), sys.Domain},
		kv{lang.L("System"), sys.OS},
		kv{lang.L("Model"), strings.TrimSpace(sys.Manufacturer + " " + sys.Model)},
		kv{lang.L("Uptime"), format.Uptime(sys.Uptime)},
	))

	// Network tile.
	netTile := tile(networkIcon(), lang.L("Network"), form(
		kv{lang.L("IP address"), format.OrDash(n.PrimaryIP)},
		kv{lang.L("Gateway"), format.OrDash(n.DefaultGateway)},
		kv{lang.L("DNS"), format.Join(n.DNSServers, ", ")},
		kv{lang.L("Adapter"), n.GatewayIface},
		kv{lang.L("Wi-Fi"), wifiSummary(n)},
	))

	// Connectivity tile.
	ok, total := 0, len(s.Connectivity)
	conn := container.NewVBox()
	for _, c := range s.Connectivity {
		if c.Status == collector.StatusOK {
			ok++
		}
		conn.Add(container.NewHBox(widget.NewIcon(statusIcon(c.Status)), widget.NewLabel(tr(c.Name))))
	}
	connTitle := lang.L("Connectivity") + fmt.Sprintf(" (%d/%d)", ok, total)
	connTile := tile(theme.UploadIcon(), connTitle, conn)

	// Resources tile.
	m := s.Memory
	res := container.NewVBox(
		widget.NewLabel(lang.L("Memory")),
		usageBar(m.UsedPercent, fmt.Sprintf("%s / %s", format.Bytes(m.Used), format.Bytes(m.Total))),
		widget.NewLabel(lang.L("CPU")),
		usageBar(sys.CPUUsage, format.Percent(sys.CPUUsage)),
	)
	for _, d := range s.Disks {
		res.Add(widget.NewLabel(lang.L("Disk") + " " + d.Mountpoint))
		res.Add(usageBar(d.UsedPercent, lang.L("{{.Free}} free of {{.Total}}",
			map[string]any{"Free": format.Bytes(d.Free), "Total": format.Bytes(d.Total)})))
	}
	resTile := tile(theme.StorageIcon(), lang.L("Resources"), res)

	// Printers tile.
	pr := container.NewVBox()
	p := s.Printing
	pr.Add(form(kv{lang.L("Spooler"), tr(p.SpoolerStatus)},
		kv{lang.L("Printers"), fmt.Sprint(len(p.Printers))},
		kv{lang.L("Default"), defaultPrinter(p.Printers)}))
	prTile := tile(theme.DocumentPrintIcon(), lang.L("Printers"), pr)

	setContent(u.overview, findingsCard,
		container.NewGridWithColumns(3, computer, netTile, resTile),
		container.NewGridWithColumns(2, connTile, prTile))
}

func (u *ui) renderSystem() {
	s := u.snap
	sys := s.System
	m := s.Memory

	boot := ""
	if !sys.BootTime.IsZero() {
		boot = sys.BootTime.Format("2006-01-02 15:04")
	}
	sysCard := widget.NewCard(lang.L("Computer"), "", form(
		kv{lang.L("Hostname"), sys.Hostname},
		kv{lang.L("User"), sys.User},
		kv{domainLabel(sys), sys.Domain},
		kv{lang.L("Operating system"), sys.OS},
		kv{lang.L("Kernel"), sys.KernelVersion},
		kv{lang.L("Architecture"), sys.Arch},
		kv{lang.L("Manufacturer"), sys.Manufacturer},
		kv{lang.L("Model"), sys.Model},
		kv{lang.L("Serial number"), sys.SerialNumber},
		kv{lang.L("BIOS version"), sys.BIOSVersion},
		kv{lang.L("Last boot"), boot},
		kv{lang.L("Uptime"), format.Uptime(sys.Uptime)},
		kv{lang.L("Time zone"), sys.TimeZone},
	))

	cpuCard := widget.NewCard(lang.L("Processor"), "", container.NewVBox(
		form(
			kv{lang.L("Model"), sys.CPUModel},
			kv{lang.L("Cores / threads"), fmt.Sprintf("%d / %d", sys.CPUCores, sys.CPUThreads)},
		),
		usageBar(sys.CPUUsage, lang.L("Usage")+" "+format.Percent(sys.CPUUsage)),
	))

	swap := ""
	if m.SwapTotal > 0 {
		swap = fmt.Sprintf("%s / %s", format.Bytes(m.SwapUsed), format.Bytes(m.SwapTotal))
	}
	memCard := widget.NewCard(lang.L("Memory"), "", container.NewVBox(
		form(
			kv{lang.L("Total"), format.Bytes(m.Total)},
			kv{lang.L("Used"), format.Bytes(m.Used)},
			kv{lang.L("Available"), format.Bytes(m.Available)},
			kv{lang.L("Swap / page file"), swap},
		),
		usageBar(m.UsedPercent, format.Percent(m.UsedPercent)),
	))

	disks := container.NewVBox()
	for _, d := range s.Disks {
		disks.Add(heading(fmt.Sprintf("%s  (%s, %s)", d.Mountpoint, d.FSType, d.Device)))
		disks.Add(usageBar(d.UsedPercent, lang.L("{{.Free}} free of {{.Total}}",
			map[string]any{"Free": format.Bytes(d.Free), "Total": format.Bytes(d.Total)})))
	}
	diskCard := widget.NewCard(lang.L("Storage"), "", disks)

	objs := []fyne.CanvasObject{sysCard, container.NewGridWithColumns(2, cpuCard, memCard), diskCard}
	if len(s.Errors) > 0 {
		errs := container.NewVBox()
		for k, v := range s.Errors {
			errs.Add(row(iconWarning(), k, v, ""))
		}
		objs = append(objs, widget.NewCard(lang.L("Collection errors"), "", errs))
	}
	setContent(u.system, objs...)
}

func (u *ui) renderNetwork() {
	if u.snap == nil {
		return
	}
	n := u.snap.Network
	global := widget.NewCard(lang.L("Network configuration"), "", form(
		kv{lang.L("IP address"), format.OrDash(n.PrimaryIP)},
		kv{lang.L("Default gateway"), format.OrDash(strings.TrimSpace(n.DefaultGateway + " " + paren(n.GatewayIface)))},
		kv{lang.L("DNS servers"), format.Join(n.DNSServers, ", ")},
		kv{lang.L("Proxy"), n.Proxy},
	))
	objs := []fyne.CanvasObject{global}

	hidden := 0
	for _, it := range n.Interfaces {
		if !u.showInactive && (it.Loopback || !it.Up || !it.HasIPv4()) {
			hidden++
			continue
		}
		icon := iconOK()
		state := lang.L("Connected")
		if !it.Up {
			icon, state = iconSkipped(), lang.L("Disconnected")
		}
		kind := lang.L("Wired")
		switch {
		case it.Loopback:
			kind = lang.L("Loopback")
		case it.Wireless:
			kind = lang.L("Wireless")
		}
		wifi := ""
		if it.SSID != "" {
			wifi = strings.TrimSpace(it.SSID + " " + paren(it.Signal))
		}
		mtu := ""
		if it.MTU > 0 {
			mtu = fmt.Sprint(it.MTU)
		}
		body := container.NewVBox(
			container.NewHBox(widget.NewIcon(icon), widget.NewLabel(state+" · "+kind)),
			form(
				kv{lang.L("IPv4"), strings.Join(it.IPv4, "\n")},
				kv{lang.L("IPv6"), strings.Join(it.IPv6, "\n")},
				kv{lang.L("MAC address"), it.MAC},
				kv{lang.L("Gateway"), it.Gateway},
				kv{lang.L("DNS"), strings.Join(it.DNSServers, ", ")},
				kv{lang.L("DHCP"), tr(dhcpLabel(it.DHCP))},
				kv{lang.L("Link speed"), it.LinkSpeed},
				kv{lang.L("Wi-Fi"), wifi},
				kv{lang.L("MTU"), mtu},
			),
		)
		objs = append(objs, widget.NewCard(it.Name, it.Description, body))
	}
	if hidden > 0 {
		objs = append(objs, widget.NewLabel(lang.L("{{.Count}} inactive or virtual adapters hidden.",
			map[string]any{"Count": hidden})))
	}
	setContent(u.network, objs...)
}

func (u *ui) renderConnectivity() {
	list := container.NewVBox()
	if len(u.snap.Connectivity) == 0 {
		list.Add(widget.NewLabel(lang.L("No check was run.")))
	}
	for i, c := range u.snap.Connectivity {
		if i > 0 {
			list.Add(widget.NewSeparator())
		}
		sub := c.Target
		if c.Detail != "" {
			sub += " — " + c.Detail
		}
		list.Add(styledRow(statusIcon(c.Status), tr(c.Name), sub, format.Latency(c.Latency), rowStyle{bold: true, monoTrailing: true}))
	}
	help := label(lang.L("Checks can be customized with a pcdiag.json file placed next to the executable (see the documentation)."))
	help.Importance = widget.LowImportance
	setContent(u.connectivity, widget.NewCard(lang.L("Connectivity checks"), "", list), help)
}

func (u *ui) renderPrinters() {
	p := u.snap.Printing
	spoolIcon := iconOK()
	if st := strings.ToLower(p.SpoolerStatus); st != "running" && st != "" {
		spoolIcon = iconError()
	}
	objs := []fyne.CanvasObject{
		widget.NewCard(lang.L("Print spooler"), "", row(spoolIcon, tr(format.OrDash(p.SpoolerStatus)), "", "")),
	}
	list := container.NewVBox()
	if len(p.Printers) == 0 {
		list.Add(widget.NewLabel(lang.L("No printer installed.")))
	}
	for i, pr := range p.Printers {
		if i > 0 {
			list.Add(widget.NewSeparator())
		}
		icon := iconOK()
		if pr.Offline {
			icon = iconWarning()
		}
		var tags []string
		if pr.Default {
			tags = append(tags, lang.L("Default"))
		}
		if pr.Network {
			tags = append(tags, lang.L("Network"))
		}
		if pr.Shared {
			tags = append(tags, lang.L("Shared"))
		}
		sub := fmt.Sprintf("%s: %s\n%s: %s", lang.L("Port"), format.OrDash(pr.Port), lang.L("Driver"), format.OrDash(pr.Driver))
		list.Add(row(icon, pr.Name, sub, strings.Join(append([]string{tr(pr.Status)}, tags...), " · ")))
	}
	objs = append(objs, widget.NewCard(lang.L("Installed printers"), "", list))
	setContent(u.printers, objs...)
}

func domainLabel(sys collector.SystemInfo) string {
	if sys.PartOfDomain {
		return lang.L("Domain")
	}
	return lang.L("Workgroup")
}

func dhcpLabel(v string) string {
	switch v {
	case "enabled":
		return "Enabled"
	case "disabled":
		return "Disabled (static IP)"
	}
	return v
}

func wifiSummary(n collector.NetworkInfo) string {
	for _, it := range n.Interfaces {
		if it.SSID != "" && it.Up {
			return strings.TrimSpace(it.SSID + " " + paren(it.Signal))
		}
	}
	return ""
}

func defaultPrinter(ps []collector.Printer) string {
	for _, p := range ps {
		if p.Default {
			return p.Name
		}
	}
	return "-"
}

func paren(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}
