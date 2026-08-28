// Package netutil matches a client IP against an allowlist of exact IPs or
// CIDR ranges, used to restrict API clients and IoT devices to specific
// hosts/networks.
package netutil

import "net"

// IPAllowed reports whether ip matches any entry in allowed. An empty
// allowlist means "no restriction" (matches everything), so this stays
// opt-in per client/device.
func IPAllowed(ip string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}
	for _, entry := range allowed {
		if entry == ip {
			return true
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(clientIP) {
			return true
		}
	}
	return false
}
