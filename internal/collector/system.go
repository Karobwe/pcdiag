package collector

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// cpuSampleWindow is how long CPU usage is measured for.
const cpuSampleWindow = 500 * time.Millisecond

// hardwareInfo holds platform-specific details filled by collectHardware.
type hardwareInfo struct {
	Manufacturer string
	Model        string
	SerialNumber string
	BIOSVersion  string
	Domain       string
	PartOfDomain bool
	OSName       string
}

func collectSystem(ctx context.Context) (SystemInfo, error) {
	info := SystemInfo{Arch: runtime.GOARCH}
	var errs multiError

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	} else {
		errs.add("hostname", err)
	}
	info.User = currentUser()
	zone, offset := time.Now().Zone()
	info.TimeZone = formatZone(time.Local.String(), zone, offset)

	if hi, err := host.InfoWithContext(ctx); err == nil {
		info.OS = strings.TrimSpace(hi.Platform + " " + hi.PlatformVersion)
		info.OSVersion = hi.PlatformVersion
		info.KernelVersion = hi.KernelVersion
		info.BootTime = time.Unix(int64(hi.BootTime), 0)     //nolint:gosec // boot time fits in int64
		info.Uptime = time.Duration(hi.Uptime) * time.Second //nolint:gosec // uptime fits in int64
		if info.OS == "" {
			info.OS = hi.OS
		}
	} else {
		errs.add("host", err)
	}

	if cpus, err := cpu.InfoWithContext(ctx); err == nil && len(cpus) > 0 {
		info.CPUModel = strings.TrimSpace(cpus[0].ModelName)
	} else if err != nil {
		errs.add("cpu", err)
	}
	if n, err := cpu.CountsWithContext(ctx, false); err == nil {
		info.CPUCores = n
	}
	if n, err := cpu.CountsWithContext(ctx, true); err == nil {
		info.CPUThreads = n
	}
	if pct, err := cpu.PercentWithContext(ctx, cpuSampleWindow, false); err == nil && len(pct) > 0 {
		info.CPUUsage = pct[0]
	}

	hw, err := collectHardware(ctx)
	if err != nil {
		errs.add("hardware", err)
	}
	info.Manufacturer = hw.Manufacturer
	info.Model = hw.Model
	info.SerialNumber = hw.SerialNumber
	info.BIOSVersion = hw.BIOSVersion
	info.Domain = hw.Domain
	info.PartOfDomain = hw.PartOfDomain
	if hw.OSName != "" {
		info.OS = hw.OSName
	}

	return info, errs.err()
}

func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	for _, k := range []string{"USERNAME", "USER", "LOGNAME"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func formatZone(name, abbr string, offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	off := fmt.Sprintf("UTC%s%02d:%02d", sign, offsetSeconds/3600, (offsetSeconds%3600)/60)
	if name == "" || name == "Local" {
		name = abbr
	}
	if name == "" || name == off {
		return off
	}
	return name + " (" + off + ")"
}

func collectMemory(ctx context.Context) (MemoryInfo, error) {
	var info MemoryInfo
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return info, err
	}
	info.Total = vm.Total
	info.Used = vm.Used
	info.Available = vm.Available
	info.UsedPercent = vm.UsedPercent
	if sw, err := mem.SwapMemoryWithContext(ctx); err == nil {
		info.SwapTotal = sw.Total
		info.SwapUsed = sw.Used
	}
	return info, nil
}

// ignoredFS lists pseudo or read-only file systems that are noise for a
// helpdesk technician.
var ignoredFS = map[string]bool{
	"squashfs": true, "tmpfs": true, "devtmpfs": true, "overlay": true,
	"autofs": true, "fuse.snapfuse": true, "iso9660": true, "udf": true,
}

func collectDisks(ctx context.Context) ([]DiskInfo, error) {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	disks := make([]DiskInfo, 0, len(parts))
	for _, p := range parts {
		if ignoredFS[strings.ToLower(p.Fstype)] || seen[p.Mountpoint] {
			continue
		}
		if runtime.GOOS == "darwin" && strings.HasPrefix(p.Mountpoint, "/System/Volumes/") {
			continue
		}
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue // empty card reader, unmounted CD drive...
		}
		seen[p.Mountpoint] = true
		disks = append(disks, DiskInfo{
			Mountpoint:  p.Mountpoint,
			Device:      p.Device,
			FSType:      p.Fstype,
			Total:       u.Total,
			Free:        u.Free,
			UsedPercent: u.UsedPercent,
		})
	}
	sort.Slice(disks, func(i, j int) bool { return disks[i].Mountpoint < disks[j].Mountpoint })
	return disks, nil
}
