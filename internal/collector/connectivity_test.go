package collector

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuiltinChecks(t *testing.T) {
	n := NetworkInfo{DefaultGateway: "192.168.1.1", DNSServers: []string{"127.0.0.53", "192.168.1.1", "fe80::1", "9.9.9.9", "1.1.1.1"}}
	opts := DefaultOptions()
	opts.ExtraChecks = []CheckSpec{{Name: "Intranet", Kind: KindHTTP, Target: "http://intranet"}}
	specs := BuiltinChecks(n, opts)

	var dnsServers []string
	for _, s := range specs {
		if s.Server != "" {
			dnsServers = append(dnsServers, s.Server)
		}
	}
	if fmt.Sprint(dnsServers) != "[192.168.1.1 9.9.9.9]" {
		t.Errorf("DNS servers tested: %v (loopback/link-local must be skipped, max 2)", dnsServers)
	}
	if specs[0].Kind != KindPing || specs[len(specs)-1].Name != "Intranet" {
		t.Errorf("unexpected order: %+v", specs)
	}
}

func TestTCPAndHTTPChecks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			fmt.Fprint(w, "Microsoft Connect Test")
		case "/portal":
			http.Redirect(w, r, "http://login.hotspot/", http.StatusFound)
		default:
			fmt.Fprint(w, "<html>login</html>")
		}
	}))
	defer srv.Close()

	closed, _ := net.Listen("tcp", "127.0.0.1:0")
	closedAddr := closed.Addr().String()
	_ = closed.Close()

	specs := []CheckSpec{
		{Name: "tcp ok", Kind: KindTCP, Target: srv.Listener.Addr().String()},
		{Name: "tcp closed", Kind: KindTCP, Target: closedAddr},
		{Name: "http ok", Kind: KindHTTP, Target: srv.URL + "/ok", Expect: "Microsoft Connect Test"},
		{Name: "captive", Kind: KindHTTP, Target: srv.URL + "/portal"},
		{Name: "content", Kind: KindHTTP, Target: srv.URL + "/other", Expect: "Microsoft Connect Test"},
		{Name: "bad kind", Kind: "smtp", Target: "x"},
	}
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	want := []Status{StatusOK, StatusFailed, StatusOK, StatusWarning, StatusWarning, StatusSkipped}
	for i, r := range RunChecks(context.Background(), specs) {
		if r.Status != want[i] {
			t.Errorf("%s: status %s (%s), want %s", r.Name, r.Status, r.Detail, want[i])
		}
	}
}
