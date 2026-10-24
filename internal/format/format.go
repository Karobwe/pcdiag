// Package format holds the human-readable formatting helpers shared by the
// user interface and the text report.
package format

import (
	"fmt"
	"strings"
	"time"
)

// Bytes formats a size using binary units (KiB, MiB...) with the
// conventional short labels used by Windows Explorer (KB, MB...).
func Bytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Uptime formats a duration as "3d 4h 12m".
func Uptime(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	d = d.Round(time.Minute)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %02dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %02dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// Latency formats a round-trip time.
func Latency(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d < time.Millisecond:
		return "<1 ms"
	case d < 10*time.Millisecond:
		return fmt.Sprintf("%.1f ms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
}

// Percent formats a percentage with no decimals.
func Percent(p float64) string { return fmt.Sprintf("%.0f %%", p) }

// Join joins non-empty values, returning "-" when there is none.
func Join(values []string, sep string) string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, sep)
}

// OrDash returns "-" for empty strings.
func OrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// YesNo formats a boolean.
func YesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
