package ssrf

import (
	"net"
	"net/url"
	"os"
	"strconv"
)

// testExemptAddr is a single loopback host:port that the test suite may send
// webhooks to. It is empty unless both SOFT_SERVE_TESTRUN is true and
// SOFT_SERVE_TESTRUN_WEBHOOK_ADDR names a loopback IP literal and port.
//
// The integration tests run a local HTTP server to receive webhooks so they
// do not depend on an external service. Every local address is blocked by
// the SSRF checks, so this one exact address is let through. Nothing else is:
// other ports on loopback, "localhost", and all private ranges stay blocked.
var testExemptAddr = parseTestExemptAddr(
	os.Getenv("SOFT_SERVE_TESTRUN"),
	os.Getenv("SOFT_SERVE_TESTRUN_WEBHOOK_ADDR"),
)

func parseTestExemptAddr(testrun, addr string) string {
	if ok, _ := strconv.ParseBool(testrun); !ok || addr == "" {
		return ""
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return ""
	}

	if p, err := strconv.Atoi(port); err != nil || p <= 0 || p > 65535 {
		return ""
	}

	return net.JoinHostPort(ip.String(), port)
}

// isTestExemptAddr reports whether hostport ("host:port") is the test
// exemption address.
func isTestExemptAddr(hostport string) bool {
	if testExemptAddr == "" {
		return false
	}

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return net.JoinHostPort(ip.String(), port) == testExemptAddr
}

// isTestExemptURL reports whether u points at the test exemption address.
func isTestExemptURL(u *url.URL) bool {
	if testExemptAddr == "" {
		return false
	}

	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	return isTestExemptAddr(net.JoinHostPort(u.Hostname(), port))
}
