package safety

import (
	"fmt"
	"net"
	"strings"
)

// ValidateListen allows only a host:port whose host is a loopback IP.
// Names such as localhost are refused so a resolver cannot point the UI elsewhere.
func ValidateListen(addr string) error {
	addr = strings.TrimSpace(addr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen address %q must be host:port", addr)
	}
	if port == "" {
		return fmt.Errorf("listen address %q is missing a port", addr)
	}
	if host == "" {
		return fmt.Errorf("listen address %q is missing a host", addr)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("refusing non-loopback listen address %q", addr)
	}
	return nil
}
