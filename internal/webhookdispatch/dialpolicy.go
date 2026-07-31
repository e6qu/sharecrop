//go:build !wasip1

package webhookdispatch

import (
	"fmt"
	"net/netip"
)

// DialPolicy decides, at connection time, whether an outbound webhook dial
// may proceed. The dialer calls it after DNS resolution with the literal
// "ip:port" it is about to connect to, so a receiver hostname that resolves
// to an internal address is caught here regardless of what the stored URL
// said. Production uses RejectNonPublicAddress; tests inject AllowEveryAddress
// to reach local httptest receivers.
type DialPolicy func(network string, address string) error

// StrictDialPolicy is the production policy: only public unicast addresses.
func StrictDialPolicy() DialPolicy {
	return func(_ string, address string) error {
		return RejectNonPublicAddress(address)
	}
}

// AllowEveryAddress is the test-only policy that admits loopback receivers.
func AllowEveryAddress() DialPolicy {
	return func(_ string, _ string) error { return nil }
}

// RejectNonPublicAddress returns an error unless address ("ip:port") is a
// public unicast IP. Rejected classes: loopback, RFC 1918 private ranges
// (10/8, 172.16/12, 192.168/16), link-local v4/v6, IPv6 unique-local
// (fc00::/7), unspecified, multicast of every flavor, and 4-in-6 mapped
// forms of all of the above (the address is unmapped before classification).
func RejectNonPublicAddress(address string) error {
	parsed, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("webhook dial blocked: address %q is not an ip:port", address)
	}
	addr := parsed.Addr().Unmap()
	if !addr.IsValid() {
		return fmt.Errorf("webhook dial blocked: address %q is invalid", address)
	}
	// IsGlobalUnicast is false for loopback, link-local unicast, multicast
	// (including interface- and link-local multicast), and the unspecified
	// address.
	if !addr.IsGlobalUnicast() {
		return fmt.Errorf("webhook dial blocked: %s is not a global unicast address", addr)
	}
	// IsPrivate covers RFC 1918 (10/8, 172.16/12, 192.168/16) and RFC 4193
	// unique-local IPv6 (fc00::/7); the explicit prefix check keeps the
	// unique-local rejection independent of that coupling.
	if addr.IsPrivate() {
		return fmt.Errorf("webhook dial blocked: %s is a private address", addr)
	}
	if addr.Is6() && (addr.As16()[0]&0xfe) == 0xfc {
		return fmt.Errorf("webhook dial blocked: %s is a unique-local address", addr)
	}
	return nil
}
