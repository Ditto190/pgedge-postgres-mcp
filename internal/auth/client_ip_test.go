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
	"net/http"
	"net/http/httptest"
	"testing"
)

// newRequest builds a request with the given peer address and headers, where
// each header entry may be repeated to simulate a caller sending its own copy.
func newRequest(remoteAddr string, headers map[string][]string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/mcp/v1", nil)
	req.RemoteAddr = remoteAddr
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	return req
}

func TestExtractIPAddressIgnoresForwardingHeaders(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string][]string
		want       string
	}{
		{
			name:       "plain IPv4 peer",
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
		},
		{
			name:       "X-Forwarded-For is not trusted by default",
			remoteAddr: "192.0.2.10:54321",
			headers:    map[string][]string{"X-Forwarded-For": {"9.9.9.9"}},
			want:       "192.0.2.10",
		},
		{
			name:       "appended X-Forwarded-For is not trusted by default",
			remoteAddr: "192.0.2.10:54321",
			headers:    map[string][]string{"X-Forwarded-For": {"9.9.9.9, 198.51.100.7"}},
			want:       "192.0.2.10",
		},
		{
			name:       "X-Real-IP is not trusted by default",
			remoteAddr: "192.0.2.10:54321",
			headers:    map[string][]string{"X-Real-IP": {"9.9.9.9"}},
			want:       "192.0.2.10",
		},
		{
			name:       "IPv6 peer is returned unbracketed",
			remoteAddr: "[2001:db8::1]:54321",
			want:       "2001:db8::1",
		},
		{
			name:       "IPv6 loopback peer is returned unbracketed",
			remoteAddr: "[::1]:54321",
			want:       "::1",
		},
		{
			name:       "IPv4-mapped IPv6 peer collapses to its IPv4 form",
			remoteAddr: "[::ffff:192.0.2.10]:54321",
			want:       "192.0.2.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIPAddress(newRequest(tt.remoteAddr, tt.headers))
			if got != tt.want {
				t.Errorf("ExtractIPAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractIPAddressIPv6KeyConsistency guards the specific defect where port
// stripping by last colon left IPv6 peers bracketed, so the same client could
// occupy two different rate limiter keys depending on the code path.
func TestExtractIPAddressIPv6KeyConsistency(t *testing.T) {
	withPort := ExtractIPAddress(newRequest("[2001:db8::1]:54321", nil))
	if withPort == "[2001:db8::1]" {
		t.Fatal("IPv6 address is still bracketed; port stripping must use net.SplitHostPort")
	}
	if withPort != "2001:db8::1" {
		t.Fatalf("got %q, want %q", withPort, "2001:db8::1")
	}
}

func TestParseCandidateAddr(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "bare IPv4", value: "192.0.2.1", want: "192.0.2.1", ok: true},
		{name: "bare IPv6", value: "2001:db8::1", want: "2001:db8::1", ok: true},
		{name: "IPv4 with port", value: "192.0.2.1:8080", want: "192.0.2.1", ok: true},
		{name: "bracketed IPv6 with port", value: "[2001:db8::1]:8080", want: "2001:db8::1", ok: true},
		{name: "bracketed IPv6 without port", value: "[2001:db8::1]", want: "2001:db8::1", ok: true},
		{name: "IPv4-mapped IPv6 collapses", value: "::ffff:192.0.2.1", want: "192.0.2.1", ok: true},
		{name: "zone is dropped", value: "fe80::1%eth0", want: "fe80::1", ok: true},
		{name: "empty", value: ""},
		{name: "whitespace only", value: "   "},
		{name: "not an address", value: "not-an-address"},
		{name: "bracketed rubbish", value: "[not-an-address]"},
		{name: "hostname with port", value: "example.com:8080"},
		{name: "truncated IPv4", value: "192.0.2."},
		{name: "SQL fragment", value: "'; DROP TABLE users; --"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, ok := parseCandidateAddr(tt.value)
			if ok != tt.ok {
				t.Fatalf("parseCandidateAddr(%q) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && addr.String() != tt.want {
				t.Errorf("parseCandidateAddr(%q) = %q, want %q", tt.value, addr.String(), tt.want)
			}
		})
	}
}

func TestNewClientIPResolverValidation(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		header         string
		trustedProxies []string
		wantErr        bool
	}{
		{name: "empty source defaults to socket", source: ""},
		{name: "socket source", source: "socket"},
		{name: "socket source is case insensitive", source: "Socket"},
		{name: "socket source ignores missing trusted proxies", source: "socket", header: "X-Forwarded-For"},
		{
			name:           "header source with trusted proxy",
			source:         "header",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
		},
		{
			name:           "header source with CIDR",
			source:         "header",
			trustedProxies: []string{"192.0.2.0/24", "2001:db8::/32"},
		},
		{name: "unknown source is rejected", source: "proxy_protocol", wantErr: true},
		{
			name:    "header source without trusted proxies is rejected",
			source:  "header",
			wantErr: true,
		},
		{
			name:           "header source with empty trusted proxy entry is rejected",
			source:         "header",
			trustedProxies: []string{""},
			wantErr:        true,
		},
		{
			name:           "malformed trusted proxy address is rejected",
			source:         "header",
			trustedProxies: []string{"not-an-address"},
			wantErr:        true,
		},
		{
			name:           "malformed trusted proxy CIDR is rejected",
			source:         "header",
			trustedProxies: []string{"192.0.2.0/99"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewClientIPResolver(tt.source, tt.header, tt.trustedProxies)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolver == nil {
				t.Fatal("expected a resolver, got nil")
			}
		})
	}
}

func TestClientIPResolverHeaderSource(t *testing.T) {
	tests := []struct {
		name           string
		header         string
		trustedProxies []string
		remoteAddr     string
		headers        map[string][]string
		want           string
	}{
		{
			name:           "spoofed leftmost entry is ignored in favour of the appended one",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"9.9.9.9, 198.51.100.7"}},
			want:           "198.51.100.7",
		},
		{
			name:           "chain of trusted proxies is walked right to left",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.0/24"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"9.9.9.9, 198.51.100.7, 192.0.2.9"}},
			want:           "198.51.100.7",
		},
		{
			name:           "untrusted peer means the header is ignored entirely",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "203.0.113.5:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"9.9.9.9"}},
			want:           "203.0.113.5",
		},
		{
			name:           "repeated header instances are treated as one list",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"9.9.9.9", "198.51.100.7"}},
			want:           "198.51.100.7",
		},
		{
			name:           "absent header falls back to the peer",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			want:           "192.0.2.1",
		},
		{
			name:           "a list of only trusted proxies falls back to the peer",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.0/24"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"192.0.2.8, 192.0.2.9"}},
			want:           "192.0.2.1",
		},
		{
			name:           "unparseable entry falls back to the peer",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"not-an-address"}},
			want:           "192.0.2.1",
		},
		{
			name:           "arbitrary strings cannot become rate limiter keys",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"'; DROP TABLE users; --"}},
			want:           "192.0.2.1",
		},
		{
			name:           "X-Real-IP set by a trusted proxy is honoured",
			header:         "X-Real-IP",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Real-IP": {"198.51.100.7"}},
			want:           "198.51.100.7",
		},
		{
			name:           "the configured header is the only one consulted",
			header:         "X-Real-IP",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers: map[string][]string{
				"X-Real-IP":       {"198.51.100.7"},
				"X-Forwarded-For": {"9.9.9.9"},
			},
			want: "198.51.100.7",
		},
		{
			name:           "header name matching is case insensitive",
			header:         "x-forwarded-for",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"198.51.100.7"}},
			want:           "198.51.100.7",
		},
		{
			name:           "entries carrying a port are accepted",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"198.51.100.7:1234"}},
			want:           "198.51.100.7",
		},
		{
			name:           "bracketed IPv6 entries are accepted and normalised",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"[2001:db8::7]"}},
			want:           "2001:db8::7",
		},
		{
			name:           "an IPv6 trusted proxy is matched by CIDR",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"2001:db8::/32"},
			remoteAddr:     "[2001:db8::1]:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"198.51.100.7, 2001:db8::9"}},
			want:           "198.51.100.7",
		},
		{
			name:           "whitespace around entries is tolerated",
			header:         "X-Forwarded-For",
			trustedProxies: []string{"192.0.2.1"},
			remoteAddr:     "192.0.2.1:44444",
			headers:        map[string][]string{"X-Forwarded-For": {"  9.9.9.9 ,   198.51.100.7  "}},
			want:           "198.51.100.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewClientIPResolver("header", tt.header, tt.trustedProxies)
			if err != nil {
				t.Fatalf("failed to build resolver: %v", err)
			}
			got := resolver.Resolve(newRequest(tt.remoteAddr, tt.headers))
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestClientIPResolverRotationCannotEvadeLimiter is the brute force case: with
// the header honoured only from a trusted peer, a caller rotating the value it
// sends still resolves to one stable address, so the rate limiter keeps counting.
// TestClientIPResolverMappedCIDRTrustedProxy covers a trusted proxy written as
// an IPv4-mapped IPv6 CIDR. Candidates are unmapped before comparison, so the
// prefix must be unmapped too; otherwise netip.Prefix.Contains rejects every
// candidate on the address family and the entry silently never matches, leaving
// the operator with an accepted configuration whose header is ignored.
func TestClientIPResolverMappedCIDRTrustedProxy(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		want           string
	}{
		{
			name:           "mapped CIDR trusts a plain IPv4 peer",
			trustedProxies: []string{"::ffff:192.0.2.0/120"},
			remoteAddr:     "192.0.2.1:44444",
			want:           "198.51.100.7",
		},
		{
			name:           "mapped CIDR trusts a mapped peer",
			trustedProxies: []string{"::ffff:192.0.2.0/120"},
			remoteAddr:     "[::ffff:192.0.2.1]:44444",
			want:           "198.51.100.7",
		},
		{
			// /120 mapped is a /24, so .255 is inside the block; the peer
			// outside it has to differ in the third octet.
			name:           "mapped CIDR still excludes an address outside it",
			trustedProxies: []string{"::ffff:192.0.2.0/120"},
			remoteAddr:     "192.0.3.1:44444",
			want:           "192.0.3.1",
		},
		{
			// ::ffff:0.0.0.0/96 unmaps to 0.0.0.0/0, which trusts every
			// address, so each header entry counts as a trusted proxy too and
			// the walk exhausts the list and falls back to the peer. Trusting
			// everything therefore makes the header useless rather than
			// authoritative, which is the safe direction to fail in.
			name:           "trusting the whole mapped range falls back to the peer",
			trustedProxies: []string{"::ffff:0.0.0.0/96"},
			remoteAddr:     "203.0.113.5:44444",
			want:           "203.0.113.5",
		},
		{
			name:           "plain IPv4 CIDR is unaffected",
			trustedProxies: []string{"192.0.2.0/24"},
			remoteAddr:     "192.0.2.1:44444",
			want:           "198.51.100.7",
		},
		{
			name:           "a genuine IPv6 CIDR is unaffected",
			trustedProxies: []string{"2001:db8::/32"},
			remoteAddr:     "[2001:db8::1]:44444",
			want:           "198.51.100.7",
		},
		{
			name:           "a mapped prefix shorter than /96 is left as IPv6",
			trustedProxies: []string{"::ffff:0.0.0.0/64"},
			remoteAddr:     "192.0.2.1:44444",
			want:           "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewClientIPResolver("header", "X-Forwarded-For", tt.trustedProxies)
			if err != nil {
				t.Fatalf("failed to build resolver: %v", err)
			}
			headers := map[string][]string{"X-Forwarded-For": {"9.9.9.9, 198.51.100.7"}}
			if got := resolver.Resolve(newRequest(tt.remoteAddr, headers)); got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseTrustedProxyMappedCIDREquivalence checks that a mapped CIDR and its
// plain IPv4 equivalent parse to exactly the same prefix, so the two spellings
// cannot drift apart in behaviour.
func TestParseTrustedProxyMappedCIDREquivalence(t *testing.T) {
	pairs := [][2]string{
		{"::ffff:192.0.2.0/120", "192.0.2.0/24"},
		{"::ffff:10.0.0.0/104", "10.0.0.0/8"},
		{"::ffff:0.0.0.0/96", "0.0.0.0/0"},
	}

	for _, pair := range pairs {
		mapped, err := parseTrustedProxy(pair[0])
		if err != nil {
			t.Fatalf("parseTrustedProxy(%q): %v", pair[0], err)
		}
		plain, err := parseTrustedProxy(pair[1])
		if err != nil {
			t.Fatalf("parseTrustedProxy(%q): %v", pair[1], err)
		}
		if mapped != plain {
			t.Errorf("parseTrustedProxy(%q) = %v, want it to equal parseTrustedProxy(%q) = %v",
				pair[0], mapped, pair[1], plain)
		}
	}
}

func TestClientIPResolverRotationCannotEvadeLimiter(t *testing.T) {
	resolver, err := NewClientIPResolver("header", "X-Forwarded-For", []string{"192.0.2.1"})
	if err != nil {
		t.Fatalf("failed to build resolver: %v", err)
	}

	spoofed := []string{"9.9.9.9", "10.0.0.1", "203.0.113.99", "not-an-address", ""}
	for _, value := range spoofed {
		headers := map[string][]string{"X-Forwarded-For": {value + ", 198.51.100.7"}}
		if got := resolver.Resolve(newRequest("192.0.2.1:44444", headers)); got != "198.51.100.7" {
			t.Errorf("with spoofed prefix %q, Resolve() = %q, want %q", value, got, "198.51.100.7")
		}
	}

	// Directly connected callers get their own address regardless of what they send
	for _, value := range spoofed {
		headers := map[string][]string{"X-Forwarded-For": {value}}
		if got := resolver.Resolve(newRequest("203.0.113.5:44444", headers)); got != "203.0.113.5" {
			t.Errorf("with spoofed value %q, Resolve() = %q, want %q", value, got, "203.0.113.5")
		}
	}
}

func TestClientIPResolverNilResolvesFromSocket(t *testing.T) {
	var resolver *ClientIPResolver
	headers := map[string][]string{"X-Forwarded-For": {"9.9.9.9"}}
	if got := resolver.Resolve(newRequest("192.0.2.10:54321", headers)); got != "192.0.2.10" {
		t.Errorf("Resolve() = %q, want %q", got, "192.0.2.10")
	}
}

func TestClientIPResolverMalformedRemoteAddr(t *testing.T) {
	// RemoteAddr is always host:port for TCP, so this only exercises the
	// defensive path; the value must still be stable rather than empty, since an
	// empty address disables rate limiting for the request.
	resolver, err := NewClientIPResolver("socket", "", nil)
	if err != nil {
		t.Fatalf("failed to build resolver: %v", err)
	}
	if got := resolver.Resolve(newRequest("@", nil)); got != "@" {
		t.Errorf("Resolve() = %q, want %q", got, "@")
	}
}

func TestClientIPMiddlewarePlacesAddressInContext(t *testing.T) {
	resolver, err := NewClientIPResolver("header", "X-Forwarded-For", []string{"192.0.2.1"})
	if err != nil {
		t.Fatalf("failed to build resolver: %v", err)
	}

	var seen string
	handler := ClientIPMiddleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetIPAddressFromContext(r.Context())
	}))

	req := newRequest("192.0.2.1:44444", map[string][]string{
		"X-Forwarded-For": {"9.9.9.9, 198.51.100.7"},
	})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "198.51.100.7" {
		t.Errorf("context address = %q, want %q", seen, "198.51.100.7")
	}
}

func TestClientIPMiddlewareWithNilResolver(t *testing.T) {
	var seen string
	handler := ClientIPMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetIPAddressFromContext(r.Context())
	}))

	req := newRequest("192.0.2.10:54321", map[string][]string{"X-Forwarded-For": {"9.9.9.9"}})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "192.0.2.10" {
		t.Errorf("context address = %q, want %q", seen, "192.0.2.10")
	}
}
