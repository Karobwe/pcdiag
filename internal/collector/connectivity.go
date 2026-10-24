package collector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Check kinds.
const (
	KindPing = "ping"
	KindTCP  = "tcp"
	KindDNS  = "dns"
	KindHTTP = "http"
)

// CheckSpec describes a connectivity test.
type CheckSpec struct {
	Name   string `json:"name"`
	Kind   string `json:"type"`
	Target string `json:"target"`
	// Server is an optional DNS server used by "dns" checks instead of the
	// system resolver.
	Server string `json:"server,omitempty"`
	// Expect is an optional substring the HTTP response body must contain.
	Expect string `json:"expect,omitempty"`
}

// checkTimeout bounds each individual check.
const checkTimeout = 4 * time.Second

// BuiltinChecks returns the standard checks derived from the network
// configuration followed by the extra checks.
func BuiltinChecks(n NetworkInfo, opts Options) []CheckSpec {
	var specs []CheckSpec
	if n.DefaultGateway != "" {
		specs = append(specs, CheckSpec{Name: "Default gateway", Kind: KindPing, Target: n.DefaultGateway})
	}
	count := 0
	for _, s := range n.DNSServers {
		if count == 2 {
			break
		}
		if ip := net.ParseIP(s); ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		specs = append(specs, CheckSpec{Name: "DNS server", Kind: KindDNS, Target: opts.DNSTestHost, Server: s})
		count++
	}
	if opts.DNSTestHost != "" {
		specs = append(specs, CheckSpec{Name: "Name resolution", Kind: KindDNS, Target: opts.DNSTestHost})
	}
	if opts.InternetTCP != "" {
		specs = append(specs, CheckSpec{Name: "Internet (TCP)", Kind: KindTCP, Target: opts.InternetTCP})
	}
	if opts.HTTPURL != "" {
		specs = append(specs, CheckSpec{Name: "Internet (HTTP)", Kind: KindHTTP, Target: opts.HTTPURL, Expect: opts.HTTPExpect})
	}
	return append(specs, opts.ExtraChecks...)
}

// RunChecks executes the checks concurrently and returns the results in the
// same order as the specs.
func RunChecks(ctx context.Context, specs []CheckSpec) []CheckResult {
	results := make([]CheckResult, len(specs))
	var wg sync.WaitGroup
	for i, s := range specs {
		wg.Add(1)
		go func(i int, s CheckSpec) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, checkTimeout+time.Second)
			defer cancel()
			results[i] = runCheck(cctx, s)
		}(i, s)
	}
	wg.Wait()
	return results
}

func runCheck(ctx context.Context, s CheckSpec) CheckResult {
	r := CheckResult{Name: s.Name, Kind: s.Kind, Target: s.Target}
	if s.Server != "" {
		r.Target = s.Server
	}
	if r.Name == "" {
		r.Name = s.Target
	}
	start := time.Now()
	var err error
	switch strings.ToLower(s.Kind) {
	case KindPing:
		r.Latency, err = ping(ctx, s.Target)
	case KindTCP:
		err = tcpCheck(ctx, s.Target)
	case KindDNS:
		r.Detail, err = dnsCheck(ctx, s.Target, s.Server)
	case KindHTTP:
		r.Detail, err = httpCheck(ctx, s.Target, s.Expect)
	default:
		r.Status, r.Detail = StatusSkipped, "unknown check type "+strconv.Quote(s.Kind)
		return r
	}
	if r.Latency == 0 {
		r.Latency = time.Since(start)
	}
	var w *warning
	switch {
	case errors.As(err, &w):
		r.Status, r.Detail = StatusWarning, w.msg
	case err != nil:
		r.Status, r.Detail = StatusFailed, err.Error()
		r.Latency = 0
	default:
		r.Status = StatusOK
	}
	return r
}

// warning is an error that should be reported as a partial success.
type warning struct{ msg string }

func (w *warning) Error() string { return w.msg }

func tcpCheck(ctx context.Context, target string) error {
	d := net.Dialer{Timeout: checkTimeout}
	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return cleanNetErr(err)
	}
	return conn.Close()
}

// dnsCheck resolves host with the system resolver, or with server when set.
func dnsCheck(ctx context.Context, host, server string) (string, error) {
	resolver := net.DefaultResolver
	if server != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: checkTimeout}
				return d.DialContext(ctx, network, net.JoinHostPort(server, "53"))
			},
		}
	}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	addrs, err := resolver.LookupHost(ctx, host)
	if err != nil {
		return "", cleanNetErr(err)
	}
	if len(addrs) > 3 {
		addrs = addrs[:3]
	}
	return strings.Join(addrs, ", "), nil
}

func httpCheck(ctx context.Context, url, expect string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pcdiag")
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
		// Do not follow redirects: a redirect usually means a captive portal.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", cleanNetErr(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	detail := fmt.Sprintf("HTTP %d", resp.StatusCode)
	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return detail, &warning{detail + " redirect to " + resp.Header.Get("Location") + " (captive portal or proxy?)"}
	case resp.StatusCode >= 400:
		return detail, &warning{detail + " (filtered or proxy authentication required?)"}
	case expect != "" && !strings.Contains(string(body), expect):
		return detail, &warning{detail + " unexpected content (captive portal or proxy?)"}
	}
	return detail, nil
}

var pingLatency = regexp.MustCompile(`([=<])\s*([0-9]+(?:[.,][0-9]+)?)\s*ms`)

// ping uses the system ping command so that no elevated privilege is needed.
func ping(ctx context.Context, host string) (time.Duration, error) {
	var args []string
	switch runtime.GOOS {
	case "windows":
		args = []string{"-n", "1", "-w", "2000", host}
	case "darwin", "freebsd", "openbsd", "netbsd":
		args = []string{"-c", "1", "-t", "2", host}
	default:
		args = []string{"-c", "1", "-W", "2", host}
	}
	out, err := runCommand(ctx, "ping", args...)
	// Windows ping returns 0 for "Destination host unreachable" replies; a
	// genuine echo reply always contains "TTL=" whatever the language.
	if err != nil || !strings.Contains(strings.ToUpper(out), "TTL=") {
		return 0, errors.New("no reply")
	}
	return parsePingLatency(out), nil
}

func parsePingLatency(out string) time.Duration {
	m := pingLatency.FindStringSubmatch(out)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
	if err != nil {
		return 0
	}
	if m[1] == "<" {
		v /= 2 // "time<1ms"
	}
	return time.Duration(v * float64(time.Millisecond))
}

// cleanNetErr shortens Go network errors to something readable in the UI.
func cleanNetErr(err error) error {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		switch {
		case dnsErr.IsNotFound:
			return errors.New("name not found")
		case dnsErr.IsTimeout:
			return errors.New("DNS timeout")
		}
		return errors.New(dnsErr.Err)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() {
			return errors.New("timeout")
		}
		if opErr.Err != nil {
			msg := opErr.Err.Error()
			if i := strings.LastIndex(msg, ": "); i >= 0 {
				msg = msg[i+2:]
			}
			return errors.New(msg)
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("timeout")
	}
	return err
}
