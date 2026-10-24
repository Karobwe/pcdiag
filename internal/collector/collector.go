package collector

import (
	"context"
	"sync"
	"time"
)

// Options configures a collection run.
type Options struct {
	// DNSTestHost is the host name resolved by the DNS checks.
	DNSTestHost string
	// InternetTCP is a host:port reachable on the Internet.
	InternetTCP string
	// HTTPURL is fetched to detect captive portals and proxies.
	HTTPURL string
	// HTTPExpect is a substring expected in the HTTP response body.
	HTTPExpect string
	// ExtraChecks are additional checks, typically defined by the IT team
	// (file servers, domain controllers, intranet...).
	ExtraChecks []CheckSpec
	// SkipConnectivity disables the network tests.
	SkipConnectivity bool
}

// DefaultOptions returns sensible defaults for an Internet-connected office.
func DefaultOptions() Options {
	return Options{
		DNSTestHost: "www.microsoft.com",
		InternetTCP: "1.1.1.1:443",
		HTTPURL:     "http://www.msftconnecttest.com/connecttest.txt",
		HTTPExpect:  "Microsoft Connect Test",
	}
}

// Collect gathers every area concurrently. It never returns an error: the
// failures of individual areas are reported in Snapshot.Errors.
func Collect(ctx context.Context, opts Options) *Snapshot {
	start := time.Now()
	s := &Snapshot{CollectedAt: start, Errors: map[string]string{}}

	var mu sync.Mutex
	record := func(area string, err error) {
		if err == nil {
			return
		}
		mu.Lock()
		s.Errors[area] = err.Error()
		mu.Unlock()
	}

	var wg sync.WaitGroup
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	run(func() { var err error; s.System, err = collectSystem(ctx); record("system", err) })
	run(func() { var err error; s.Memory, err = collectMemory(ctx); record("memory", err) })
	run(func() { var err error; s.Disks, err = collectDisks(ctx); record("disks", err) })
	run(func() { var err error; s.Printing, err = collectPrinting(ctx); record("printing", err) })
	run(func() {
		var err error
		s.Network, err = collectNetwork(ctx)
		record("network", err)
		if !opts.SkipConnectivity {
			s.Connectivity = RunChecks(ctx, BuiltinChecks(s.Network, opts))
		}
	})
	wg.Wait()

	if s.Printing.Printers == nil {
		s.Printing.Printers = []Printer{}
	}
	if len(s.Errors) == 0 {
		s.Errors = nil
	}
	s.Duration = time.Since(start)
	return s
}
