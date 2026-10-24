// Package collector gathers the diagnostic information displayed by PC Diag.
//
// Every collector is best effort: a failure in one area (for example the
// printer subsystem) is recorded in Snapshot.Errors and never prevents the
// other areas from being reported.
package collector

import "time"

// Snapshot is a point-in-time view of the workstation.
type Snapshot struct {
	CollectedAt  time.Time         `json:"collectedAt"`
	Duration     time.Duration     `json:"durationNs"`
	System       SystemInfo        `json:"system"`
	Memory       MemoryInfo        `json:"memory"`
	Disks        []DiskInfo        `json:"disks"`
	Network      NetworkInfo       `json:"network"`
	Connectivity []CheckResult     `json:"connectivity"`
	Printing     PrintingInfo      `json:"printing"`
	Errors       map[string]string `json:"errors,omitempty"`
}

// SystemInfo describes the machine, its operating system and the session.
type SystemInfo struct {
	Hostname      string        `json:"hostname"`
	User          string        `json:"user"`
	Domain        string        `json:"domain,omitempty"`
	PartOfDomain  bool          `json:"partOfDomain"`
	OS            string        `json:"os"`
	OSVersion     string        `json:"osVersion"`
	KernelVersion string        `json:"kernelVersion"`
	Arch          string        `json:"arch"`
	Manufacturer  string        `json:"manufacturer,omitempty"`
	Model         string        `json:"model,omitempty"`
	SerialNumber  string        `json:"serialNumber,omitempty"`
	BIOSVersion   string        `json:"biosVersion,omitempty"`
	CPUModel      string        `json:"cpuModel"`
	CPUCores      int           `json:"cpuCores"`
	CPUThreads    int           `json:"cpuThreads"`
	CPUUsage      float64       `json:"cpuUsagePercent"`
	BootTime      time.Time     `json:"bootTime"`
	Uptime        time.Duration `json:"uptimeNs"`
	TimeZone      string        `json:"timeZone"`
}

// MemoryInfo describes physical memory and swap / page file usage.
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"usedPercent"`
	SwapTotal   uint64  `json:"swapTotal"`
	SwapUsed    uint64  `json:"swapUsed"`
}

// DiskInfo describes a mounted volume.
type DiskInfo struct {
	Mountpoint  string  `json:"mountpoint"`
	Device      string  `json:"device"`
	FSType      string  `json:"fsType"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
}

// NetworkInfo groups the network adapters and the global network settings.
type NetworkInfo struct {
	Interfaces     []Interface `json:"interfaces"`
	DefaultGateway string      `json:"defaultGateway,omitempty"`
	GatewayIface   string      `json:"gatewayInterface,omitempty"`
	DNSServers     []string    `json:"dnsServers,omitempty"`
	PrimaryIP      string      `json:"primaryIp,omitempty"`
	Proxy          string      `json:"proxy,omitempty"`
}

// Interface describes a single network adapter.
type Interface struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	MAC         string   `json:"mac,omitempty"`
	MTU         int      `json:"mtu"`
	Up          bool     `json:"up"`
	Loopback    bool     `json:"loopback"`
	Wireless    bool     `json:"wireless"`
	IPv4        []string `json:"ipv4,omitempty"`
	IPv6        []string `json:"ipv6,omitempty"`
	Gateway     string   `json:"gateway,omitempty"`
	DNSServers  []string `json:"dnsServers,omitempty"`
	DHCP        string   `json:"dhcp,omitempty"`
	LinkSpeed   string   `json:"linkSpeed,omitempty"`
	SSID        string   `json:"ssid,omitempty"`
	Signal      string   `json:"signal,omitempty"`
}

// HasIPv4 reports whether the interface carries at least one IPv4 address.
func (i Interface) HasIPv4() bool { return len(i.IPv4) > 0 }

// Status is the outcome of a connectivity check.
type Status string

// Possible check outcomes.
const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// CheckResult is the outcome of one connectivity test.
type CheckResult struct {
	Name    string        `json:"name"`
	Kind    string        `json:"kind"`
	Target  string        `json:"target"`
	Status  Status        `json:"status"`
	Latency time.Duration `json:"latencyNs,omitempty"`
	Detail  string        `json:"detail,omitempty"`
}

// PrintingInfo describes the print spooler and the installed printers.
type PrintingInfo struct {
	SpoolerStatus string    `json:"spoolerStatus,omitempty"`
	Printers      []Printer `json:"printers"`
}

// Printer describes an installed printer queue.
type Printer struct {
	Name    string `json:"name"`
	Driver  string `json:"driver,omitempty"`
	Port    string `json:"port,omitempty"`
	Status  string `json:"status,omitempty"`
	Default bool   `json:"default"`
	Network bool   `json:"network"`
	Shared  bool   `json:"shared"`
	Offline bool   `json:"offline"`
}
