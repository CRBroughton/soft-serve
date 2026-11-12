package ssrf

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTestExemptAddr(t *testing.T) {
	tests := []struct {
		name    string
		testrun string
		addr    string
		want    string
	}{
		{"testrun unset", "", "127.0.0.1:8080", ""},
		{"testrun false", "0", "127.0.0.1:8080", ""},
		{"addr unset", "1", "", ""},
		{"ipv4 loopback", "1", "127.0.0.1:8080", "127.0.0.1:8080"},
		{"ipv6 loopback", "true", "[::1]:8080", "[::1]:8080"},
		{"hostname rejected", "1", "localhost:8080", ""},
		{"private ip rejected", "1", "10.0.0.1:8080", ""},
		{"public ip rejected", "1", "1.1.1.1:80", ""},
		{"metadata ip rejected", "1", "169.254.169.254:80", ""},
		{"missing port", "1", "127.0.0.1", ""},
		{"bad port", "1", "127.0.0.1:http", ""},
		{"port zero", "1", "127.0.0.1:0", ""},
		{"port too large", "1", "127.0.0.1:70000", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTestExemptAddr(tt.testrun, tt.addr); got != tt.want {
				t.Errorf("parseTestExemptAddr(%q, %q) = %q, want %q", tt.testrun, tt.addr, got, tt.want)
			}
		})
	}
}

func withTestExemptAddr(t *testing.T, addr string) {
	t.Helper()
	prev := testExemptAddr
	testExemptAddr = addr
	t.Cleanup(func() { testExemptAddr = prev })
}

func TestValidateURLTestExemption(t *testing.T) {
	// Without the exemption, loopback is blocked.
	withTestExemptAddr(t, "")
	if err := ValidateURL("http://127.0.0.1:8080/hook"); !errors.Is(err, ErrPrivateIP) {
		t.Fatalf("no exemption: expected ErrPrivateIP, got %v", err)
	}

	withTestExemptAddr(t, "127.0.0.1:8080")

	allowed := []string{
		"http://127.0.0.1:8080/hook",
		"https://127.0.0.1:8080/",
	}
	for _, u := range allowed {
		if err := ValidateURL(u); err != nil {
			t.Errorf("ValidateURL(%q) = %v, want nil", u, err)
		}
	}

	blocked := []string{
		"http://127.0.0.1:8081/hook",   // other port
		"http://127.0.0.1/hook",        // default port
		"http://localhost:8080/hook",   // hostname, not the literal
		"http://127.0.0.2:8080/hook",   // other loopback address
		"http://[::1]:8080/hook",       // other family
		"http://10.0.0.1:8080/hook",    // private
		"http://169.254.169.254:8080/", // metadata
	}
	for _, u := range blocked {
		if err := ValidateURL(u); err == nil {
			t.Errorf("ValidateURL(%q) = nil, want error", u)
		}
	}
}

func TestSecureClientTestExemption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	addr := srv.Listener.Addr().String()
	get := func() error {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		if err != nil {
			return err
		}
		res, err := NewSecureClient().Do(req)
		if err != nil {
			return err
		}
		return res.Body.Close()
	}

	// Without the exemption, the dial is refused.
	withTestExemptAddr(t, "")
	if err := get(); !errors.Is(err, ErrPrivateIP) {
		t.Fatalf("no exemption: expected ErrPrivateIP, got %v", err)
	}

	// A different port does not open up the server.
	_, port, _ := net.SplitHostPort(addr)
	other := "1"
	if port == other {
		other = "2"
	}
	withTestExemptAddr(t, net.JoinHostPort("127.0.0.1", other))
	if err := get(); !errors.Is(err, ErrPrivateIP) {
		t.Fatalf("other port exempt: expected ErrPrivateIP, got %v", err)
	}

	// The exact address is allowed.
	withTestExemptAddr(t, addr)
	if err := get(); err != nil {
		t.Fatalf("exempt addr: %v", err)
	}
}
