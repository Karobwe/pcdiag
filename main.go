// Command pcdiag displays the information a helpdesk technician needs to
// troubleshoot a workstation: system, memory, storage, network adapters,
// connectivity and printers.
//
// Run without arguments to open the window, or with -report to print a
// report on the console (useful in scripts or remote shells).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/config"
	"github.com/Karobwe/pcdiag/internal/health"
	"github.com/Karobwe/pcdiag/internal/report"
	"github.com/Karobwe/pcdiag/internal/ui"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	var (
		reportFormat = flag.String("report", "", "print a report without opening the window: `text` or json")
		output       = flag.String("o", "", "write the report to `file` instead of the standard output")
		configPath   = flag.String("config", "", "configuration `file` (default: "+config.FileName+" next to the executable)")
		noChecks     = flag.Bool("no-checks", false, "skip the connectivity checks in report mode")
		showVersion  = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVersion || *reportFormat != "" {
		attachConsole()
	}
	if *showVersion {
		fmt.Println("pcdiag", resolveVersion())
		return
	}

	cfg, cfgFile, cfgErr := config.Load(*configPath)

	if *reportFormat != "" {
		if cfgErr != nil {
			fmt.Fprintln(os.Stderr, "warning:", cfgErr)
		}
		if err := runReport(*reportFormat, *output, cfg, *noChecks); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	ui.Run(ui.Params{
		Version:    resolveVersion(),
		Config:     cfg,
		ConfigPath: cfgFile,
		ConfigErr:  cfgErr,
	})
}

func runReport(kind, output string, cfg config.Config, noChecks bool) error {
	if kind != "text" && kind != "json" {
		return fmt.Errorf("unknown report format %q (expected text or json)", kind)
	}
	opts := cfg.Options()
	opts.SkipConnectivity = noChecks

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	snap := collector.Collect(ctx, opts)

	var w io.Writer = os.Stdout
	if output != "" {
		f, err := os.Create(output) //nolint:gosec // path provided by the user
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if kind == "json" {
		return report.JSON(w, snap)
	}
	th := health.DefaultThresholds()
	if v := cfg.Thresholds.MemoryPercent; v > 0 {
		th.MemoryPercent = v
	}
	if v := cfg.Thresholds.DiskPercent; v > 0 {
		th.DiskPercent = v
	}
	return report.Text(w, snap, health.Analyze(snap, th), resolveVersion())
}

// resolveVersion falls back to the module version when the binary was
// installed with "go install ...@vX.Y.Z".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}
