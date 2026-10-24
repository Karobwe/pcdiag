//go:build windows

package collector

import (
	"context"
	"strings"
)

const psHardware = `$cs=Get-CimInstance Win32_ComputerSystem;$b=Get-CimInstance Win32_BIOS;$o=Get-CimInstance Win32_OperatingSystem;` +
	`$v=(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion' -EA SilentlyContinue).DisplayVersion;` +
	`[pscustomobject]@{Manufacturer=[string]$cs.Manufacturer;Model=[string]$cs.Model;Domain=[string]$cs.Domain;PartOfDomain=[bool]$cs.PartOfDomain;` +
	`Serial=[string]$b.SerialNumber;BIOSVersion=[string]$b.SMBIOSBIOSVersion;OSCaption=[string]$o.Caption;OSBuild=[string]$o.BuildNumber;OSVersion=[string]$v}|ConvertTo-Json -Compress`

const psNetwork = `$a=@(Get-NetAdapter -EA SilentlyContinue|%{[pscustomobject]@{Name=$_.Name;Description=$_.InterfaceDescription;Status=[string]$_.Status;LinkSpeed=[string]$_.LinkSpeed;MediaType=[string]$_.MediaType;PhysicalMediaType=[string]$_.PhysicalMediaType}});` +
	`$r=@(Get-NetRoute -DestinationPrefix '0.0.0.0/0','::/0' -EA SilentlyContinue|%{[pscustomobject]@{Alias=$_.InterfaceAlias;NextHop=$_.NextHop;Metric=[int]$_.RouteMetric+[int]$_.InterfaceMetric}});` +
	`$d=@(Get-DnsClientServerAddress -EA SilentlyContinue|?{$_.ServerAddresses}|%{[pscustomobject]@{Alias=$_.InterfaceAlias;Servers=@($_.ServerAddresses)}});` +
	`$h=@(Get-NetIPInterface -AddressFamily IPv4 -EA SilentlyContinue|%{[pscustomobject]@{Alias=$_.InterfaceAlias;Dhcp=[string]$_.Dhcp}});` +
	`$i=Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -EA SilentlyContinue;$px=@();` +
	`if($i.ProxyEnable -eq 1){$px+=[string]$i.ProxyServer};if($i.AutoConfigURL){$px+='PAC '+$i.AutoConfigURL};` +
	`@{Adapters=$a;Routes=$r;DNS=$d;DHCP=$h;Proxy=($px -join ' | ')}|ConvertTo-Json -Compress -Depth 4`

const psPrinting = `$s=(Get-Service Spooler -EA SilentlyContinue).Status;` +
	`$p=@(Get-CimInstance Win32_Printer -EA SilentlyContinue|%{[pscustomobject]@{Name=$_.Name;DriverName=$_.DriverName;PortName=$_.PortName;` +
	`Default=[bool]$_.Default;Network=[bool]$_.Network;Shared=[bool]$_.Shared;WorkOffline=[bool]$_.WorkOffline;PrinterStatus=[int]$_.PrinterStatus}});` +
	`@{Spooler=[string]$s;Printers=$p}|ConvertTo-Json -Compress -Depth 3`

func collectHardware(ctx context.Context) (hardwareInfo, error) {
	out, err := runPowerShell(ctx, psHardware)
	if err != nil {
		return hardwareInfo{}, err
	}
	return parseWinHardware(out)
}

func enrichNetwork(ctx context.Context, n *NetworkInfo) error {
	out, err := runPowerShell(ctx, psNetwork)
	if err != nil {
		return err
	}
	res, err := parseWinNetwork(out)
	if err != nil {
		return err
	}
	n.DefaultGateway = res.DefaultGateway
	n.GatewayIface = res.GatewayIface
	n.DNSServers = res.DNS
	if res.Proxy != "" {
		n.Proxy = res.Proxy
	}

	wlan := map[string][2]string{}
	if out, err := runCommand(ctx, "netsh", "wlan", "show", "interfaces"); err == nil {
		wlan = parseNetshWlan(out)
	}

	for i := range n.Interfaces {
		it := &n.Interfaces[i]
		if e, ok := res.Ifaces[it.Name]; ok {
			it.Description = e.Description
			it.LinkSpeed = e.LinkSpeed
			it.Wireless = it.Wireless || e.Wireless
			it.Gateway = e.Gateway
			it.DNSServers = e.DNS
			it.DHCP = e.DHCP
		}
		if w, ok := wlan[it.Name]; ok {
			it.Wireless = true
			it.SSID, it.Signal = w[0], w[1]
		}
		if strings.HasPrefix(strings.ToLower(it.Name), "loopback") {
			it.Loopback = true
		}
	}
	return nil
}

func collectPrinting(ctx context.Context) (PrintingInfo, error) {
	out, err := runPowerShell(ctx, psPrinting)
	if err != nil {
		return PrintingInfo{}, err
	}
	return parseWinPrinting(out)
}
