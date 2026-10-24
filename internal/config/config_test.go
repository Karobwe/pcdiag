package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, used, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	_ = used // a pcdiag.json may exist next to the test binary only in odd setups
	if cfg.Connectivity.DNSTestHost == "" || cfg.Thresholds.DiskPercent != 90 {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}

func TestLoadFileMergesWithDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pcdiag.json")
	data := `{"support":{"name":"IT","phone":"1234"},"connectivity":{"httpUrl":"","checks":[{"name":"File server","type":"tcp","target":"fs01:445"}]}}`
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, used, err := Load(p)
	if err != nil || used != p {
		t.Fatalf("load: %v (%s)", err, used)
	}
	opts := cfg.Options()
	if opts.HTTPURL != "" {
		t.Error("an explicit empty httpUrl must disable the HTTP check")
	}
	if opts.DNSTestHost == "" || cfg.Thresholds.MemoryPercent != 90 {
		t.Error("unspecified values must keep their defaults")
	}
	if len(opts.ExtraChecks) != 1 || opts.ExtraChecks[0].Kind != "tcp" || cfg.Support.IsZero() {
		t.Errorf("unexpected %+v", cfg)
	}
}

func TestLoadErrors(t *testing.T) {
	if _, _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("an explicit missing file must be an error")
	}
	p := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(p, []byte("{"), 0o600)
	if _, _, err := Load(p); err == nil {
		t.Error("invalid JSON must be an error")
	}
}
