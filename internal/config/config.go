// Package config loads the optional JSON configuration file that lets an IT
// team tailor PC Diag (support contact, extra connectivity checks, alert
// thresholds) without rebuilding the application.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Karobwe/pcdiag/internal/collector"
)

// FileName is the configuration file looked up next to the executable.
const FileName = "pcdiag.json"

// Config is the user-editable configuration.
type Config struct {
	// Support is displayed in the window footer and in the About dialog.
	Support Support `json:"support"`
	// Connectivity overrides the default Internet checks.
	Connectivity Connectivity `json:"connectivity"`
	// Thresholds above which a value is highlighted as a warning.
	Thresholds Thresholds `json:"thresholds"`
}

// Support describes how users can reach the IT team.
type Support struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// IsZero reports whether no support information is configured.
func (s Support) IsZero() bool { return s == Support{} }

// Connectivity configures the network tests.
type Connectivity struct {
	DNSTestHost string                `json:"dnsTestHost,omitempty"`
	InternetTCP string                `json:"internetTcp,omitempty"`
	HTTPURL     string                `json:"httpUrl,omitempty"`
	HTTPExpect  string                `json:"httpExpect,omitempty"`
	Checks      []collector.CheckSpec `json:"checks,omitempty"`
}

// Thresholds are expressed in percent.
type Thresholds struct {
	MemoryPercent float64 `json:"memoryPercent,omitempty"`
	DiskPercent   float64 `json:"diskPercent,omitempty"`
}

// Default returns the built-in configuration.
func Default() Config {
	d := collector.DefaultOptions()
	return Config{
		Connectivity: Connectivity{
			DNSTestHost: d.DNSTestHost,
			InternetTCP: d.InternetTCP,
			HTTPURL:     d.HTTPURL,
			HTTPExpect:  d.HTTPExpect,
		},
		Thresholds: Thresholds{MemoryPercent: 90, DiskPercent: 90},
	}
}

// Options converts the configuration into collector options.
func (c Config) Options() collector.Options {
	return collector.Options{
		DNSTestHost: c.Connectivity.DNSTestHost,
		InternetTCP: c.Connectivity.InternetTCP,
		HTTPURL:     c.Connectivity.HTTPURL,
		HTTPExpect:  c.Connectivity.HTTPExpect,
		ExtraChecks: c.Connectivity.Checks,
	}
}

// Load reads the configuration from path. When path is empty, the file is
// searched next to the executable then in the user configuration directory;
// if none exists the defaults are returned. The returned string is the file
// actually used ("" for the defaults).
func Load(path string) (Config, string, error) {
	cfg := Default()
	candidates := []string{path}
	if path == "" {
		candidates = searchPaths()
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p) //nolint:gosec // path chosen by the user or the IT team
		if errors.Is(err, fs.ErrNotExist) && path == "" {
			continue
		}
		if err != nil {
			return cfg, p, err
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Default(), p, fmt.Errorf("invalid configuration %s: %w", p, err)
		}
		return cfg, p, nil
	}
	return cfg, "", nil
}

func searchPaths() []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), FileName))
	}
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "pcdiag", "config.json"))
	}
	return paths
}
