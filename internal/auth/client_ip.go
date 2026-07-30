/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Client IP sources, as accepted by the http.client_ip.source setting.
const (
	// ClientIPSourceSocket takes the address from the connection's peer,
	// ignoring any forwarding headers. This is the default.
	ClientIPSourceSocket = "socket"

	// ClientIPSourceHeader takes the address from a forwarding header, but
	// only when the connection's peer is a configured trusted proxy.
	ClientIPSourceHeader = "header"
)

// DefaultClientIPHeader is the header consulted when http.client_ip.source is
// "header" and no header name is configured. X-Real-IP is preferred over
// X-Forwarded-For because the reverse proxy configurations we document set it
// to the address the proxy actually observed, replacing rather than appending
// to whatever the caller sent.
const DefaultClientIPHeader = "X-Real-IP"

// ClientIPResolver determines the address a request came from.
//
// A forwarding header is only ever honoured when the immediate peer is a
// configured trusted proxy, and the header list is then read from right to
// left, skipping trusted entries. This matters because X-Forwarded-For is a
// history list to which each proxy appends the address it received the request
// from: anything the caller invents lands at the far left and everything the
// infrastructure adds lands to the right of it, so counting from the right is
// the only reliable direction. A nil resolver resolves from the socket, which
// makes the safe behaviour the zero value.
type ClientIPResolver struct {
	useHeader bool
	header    string
	trusted   []netip.Prefix
}

// NewClientIPResolver builds a resolver from configuration.
//
// The source must be "socket" (the default when empty) or "header". When it is
// "header", at least one trusted proxy is required, since a header carries no
// authority unless we know which peer is entitled to set it.
func NewClientIPResolver(source, header string, trustedProxies []string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{}

	switch strings.ToLower(strings.TrimSpace(source)) {
	case "", ClientIPSourceSocket:
		// Socket only; the header and trusted proxy settings are unused.
		return resolver, nil
	case ClientIPSourceHeader:
		resolver.useHeader = true
	default:
		return nil, fmt.Errorf(
			"invalid client IP source %q: must be %q or %q",
			source, ClientIPSourceSocket, ClientIPSourceHeader)
	}

	resolver.header = strings.TrimSpace(header)
	if resolver.header == "" {
		resolver.header = DefaultClientIPHeader
	}

	if len(trustedProxies) == 0 {
		return nil, fmt.Errorf(
			"client IP source is %q but no trusted proxies are configured; "+
				"a forwarding header can only be trusted when it arrives from a known proxy",
			ClientIPSourceHeader)
	}

	for _, entry := range trustedProxies {
		prefix, err := parseTrustedProxy(entry)
		if err != nil {
			return nil, err
		}
		resolver.trusted = append(resolver.trusted, prefix)
	}

	return resolver, nil
}

// parseTrustedProxy parses a trusted proxy entry, which may be either a CIDR
// block or a single address; a single address becomes a host-sized prefix.
func parseTrustedProxy(entry string) (netip.Prefix, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return netip.Prefix{}, fmt.Errorf("trusted proxy entry is empty")
	}

	if strings.Contains(entry, "/") {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("invalid trusted proxy CIDR %q: %w", entry, err)
		}

		// An IPv4-mapped IPv6 prefix such as ::ffff:192.0.2.0/120 describes a
		// range of IPv4 addresses, and every candidate is unmapped before it is
		// compared, so the prefix has to be unmapped as well. Without this,
		// netip.Prefix.Contains rejects each candidate on the address family
		// alone and the entry silently never matches: the configuration is
		// accepted, no error is raised, and the forwarding header is quietly
		// ignored, which is the most misleading way for this to fail. The
		// mapped range occupies the final 32 bits, hence the shift of 96.
		//
		// A shorter prefix than /96 is left alone, because it spans more than
		// the mapped-IPv4 range and so does not describe an IPv4 block.
		if addr := prefix.Addr(); addr.Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(addr.Unmap(), prefix.Bits()-96)
		}

		// Masking discards any host bits, which ParsePrefix preserves but
		// Contains ignores; normalising keeps comparisons predictable.
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(entry)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid trusted proxy address %q: %w", entry, err)
	}
	addr = normaliseAddr(addr)

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

// Resolve returns the address the request should be attributed to, for rate
// limiting and for logging alike. The result is always either the connection's
// peer or an address a trusted proxy vouched for, and it is normalised so that
// one client cannot occupy two different keys.
func (r *ClientIPResolver) Resolve(req *http.Request) string {
	socket, socketOK := parseCandidateAddr(req.RemoteAddr)

	// RemoteAddr is always host:port for TCP connections, so this is
	// defensive. Falling back to the raw value keeps a stable per-peer key
	// rather than returning nothing, and the value is not caller-controlled.
	if !socketOK {
		return strings.TrimSpace(req.RemoteAddr)
	}

	if r == nil || !r.useHeader {
		return socket.String()
	}

	// The header carries no authority unless the peer that sent it is trusted.
	if !r.isTrusted(socket) {
		return socket.String()
	}

	// A request may carry the header more than once, and every instance has to
	// be treated as one list in order; reading only the first instance would
	// mean reading only what a caller chose to send.
	candidates := make([]string, 0, 4)
	for _, value := range req.Header.Values(r.header) {
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				candidates = append(candidates, part)
			}
		}
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		addr, ok := parseCandidateAddr(candidates[i])
		if !ok {
			// An unparseable entry means the positions to its left can no
			// longer be trusted to mean what they appear to, so stop here and
			// use the peer instead of guessing.
			return socket.String()
		}
		if r.isTrusted(addr) {
			continue
		}
		return addr.String()
	}

	// Either the header was absent or every entry was a trusted proxy, so the
	// nearest thing to a client address we have is the peer itself.
	return socket.String()
}

// isTrusted reports whether an address belongs to a configured trusted proxy.
func (r *ClientIPResolver) isTrusted(addr netip.Addr) bool {
	for _, prefix := range r.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// parseCandidateAddr parses an address that may arrive bare, bracketed, or
// with a port attached, and rejects anything that is not an address at all.
func parseCandidateAddr(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Addr{}, false
	}

	if addr, err := netip.ParseAddr(value); err == nil {
		return normaliseAddr(addr), true
	}

	// "192.0.2.1:8080" and "[2001:db8::1]:8080"; SplitHostPort also handles
	// the bracketed IPv6 form correctly, which manual index arithmetic on the
	// last colon does not.
	if host, _, err := net.SplitHostPort(value); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			return normaliseAddr(addr), true
		}
	}

	// A bracketed IPv6 address with no port, as some proxies emit.
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		if addr, err := netip.ParseAddr(value[1 : len(value)-1]); err == nil {
			return normaliseAddr(addr), true
		}
	}

	return netip.Addr{}, false
}

// normaliseAddr reduces an address to one canonical form, so that the same
// client cannot end up under two separate rate limiter keys. IPv4-mapped IPv6
// addresses collapse to their IPv4 form, and interface zones are dropped.
func normaliseAddr(addr netip.Addr) netip.Addr {
	return addr.Unmap().WithZone("")
}

// ClientIPMiddleware resolves the client address once per request and places it
// in the request context, so that everything downstream (rate limiting, logs)
// agrees on which address a request came from.
func ClientIPMiddleware(resolver *ClientIPResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), IPAddressContextKey, resolver.Resolve(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
